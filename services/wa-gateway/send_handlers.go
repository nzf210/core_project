package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

func setupSendHandler() {
	http.HandleFunc("/api/wa/send", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, `{"error":"Failed to parse form"}`, http.StatusBadRequest)
			return
		}

		tenantID := getTenantID(r)
		target := r.FormValue("target")
		message := r.FormValue("message")
		mediaURL := r.FormValue("media_url")
		mediaName := r.FormValue("media_name")

		if tenantID == "" || target == "" || message == "" {
			http.Error(w, `{"error":"Missing tenant_id, target, or message"}`, http.StatusBadRequest)
			return
		}

		preference := resolveProviderPreference(r, tenantID)
		forceCloud := preference == "cloud_api"
		forceWhatsmeow := preference == "whatsmeow"

		if forceCloud {
			clientMu.Lock()
			if client, exists := clientMap[tenantID]; exists {
				client.Disconnect()
				delete(clientMap, tenantID)
			}
			clientMu.Unlock()
			_, _ = db.Exec(`DELETE FROM wa_tenant_sessions WHERE tenant_id = $1`, tenantID)
			ReleaseSessionLock(context.Background(), tenantID)
		}

		if forceCloud || (!forceWhatsmeow && isTransactional(r)) {
			msgType := r.Header.Get("X-Message-Type")
			if msgType == "" {
				msgType = "text"
			}
			waMsgID, err := routeToCloudAPI(tenantID, target, message, msgType)
			if err == nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success":       true,
					"routed":        "cloud_api",
					"wa_message_id": waMsgID,
				})
				return
			}
			if forceCloud {
				log.Printf("Cloud API failed (forced) for tenant %s: %v", tenantID, err)
				http.Error(w, fmt.Sprintf(`{"error":"Cloud API failed: %v"}`, err), http.StatusBadGateway)
				return
			}
			if r.Header.Get("X-Message-Type") == "broadcast" {
				log.Printf("Cloud API failed for broadcast, blocking fallback to QR for tenant %s: %v", tenantID, err)
				http.Error(w, fmt.Sprintf(`{"error":"Broadcast must use Cloud API, setup required or insufficient balance: %v"}`, err), http.StatusPaymentRequired)
				return
			}
			log.Printf("Cloud API failed, falling back to whatsmeow for tenant %s: %v", tenantID, err)
		}

		if !rateLimiter.Allow(tenantID) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":   "Rate limit exceeded",
				"message": "Too many WhatsApp messages. Please slow down to avoid blocking.",
			})
			return
		}

		if redisClient != nil {
			ownerKey := sessionOwnerPrefix + tenantID
			owner, err := redisClient.Get(r.Context(), ownerKey).Result()
			if err == nil && owner != instanceID {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"status":  "delegated",
					"message": "Session handled by another instance",
				})
				return
			}
		}

		clientMu.RLock()
		client, exists := clientMap[tenantID]
		clientMu.RUnlock()

		if !exists || client.Store.ID == nil {
			http.Error(w, `{"error":"WhatsApp not connected for this tenant"}`, http.StatusBadGateway)
			return
		}

		jid, err := types.ParseJID(target)
		if err != nil {
			http.Error(w, `{"error":"Invalid target JID"}`, http.StatusBadRequest)
			return
		}

		var msg *waE2E.Message
		if mediaURL != "" {
			resp, err := http.Get(mediaURL)
			if err != nil {
				log.Printf("[Tenant %s] Failed to download media %s: %v", tenantID, mediaURL, err)
				http.Error(w, `{"error":"Failed to download media"}`, http.StatusInternalServerError)
				return
			}
			defer resp.Body.Close()
			mediaData, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Printf("[Tenant %s] Failed to read media: %v", tenantID, err)
				http.Error(w, `{"error":"Failed to read media"}`, http.StatusInternalServerError)
				return
			}
			mimetype := resp.Header.Get("Content-Type")
			if mimetype == "" {
				mimetype = "application/octet-stream"
			}
			if mediaName == "" {
				mediaName = "document"
			}

			uploadResp, err := client.Upload(context.Background(), mediaData, whatsmeow.MediaDocument)
			if err != nil {
				log.Printf("[Tenant %s] Failed to upload media to WA: %v", tenantID, err)
				http.Error(w, `{"error":"Failed to upload media"}`, http.StatusInternalServerError)
				return
			}

			msg = &waE2E.Message{
				DocumentMessage: &waE2E.DocumentMessage{
					URL:           proto.String(uploadResp.URL),
					DirectPath:    proto.String(uploadResp.DirectPath),
					MediaKey:      uploadResp.MediaKey,
					Mimetype:      proto.String(mimetype),
					FileEncSHA256: uploadResp.FileEncSHA256,
					FileSHA256:    uploadResp.FileSHA256,
					FileLength:    proto.Uint64(uint64(len(mediaData))),
					Title:         proto.String(mediaName),
					FileName:      proto.String(mediaName),
					Caption:       proto.String(message),
				},
			}
		} else {
			msg = &waE2E.Message{
				Conversation: proto.String(message),
			}
		}

		_, err = client.SendMessage(context.Background(), jid, msg)
		if err != nil {
			log.Printf("[Tenant %s] Failed to send message to %s: %v", tenantID, jid.String(), err)
			http.Error(w, `{"error":"Failed to send message"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
	})
}
