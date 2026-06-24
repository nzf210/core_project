package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
)

var (
	containerStore *sqlstore.Container
	containerMu    sync.RWMutex
)

func setContainer(c *sqlstore.Container) {
	containerMu.Lock()
	containerStore = c
	containerMu.Unlock()
}

func getContainer() *sqlstore.Container {
	containerMu.RLock()
	defer containerMu.RUnlock()
	return containerStore
}

// eventHandler handles WhatsApp events for a tenant
func eventHandler(tenantID string, evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		text := v.Message.GetConversation()
		if text == "" {
			text = v.Message.GetExtendedTextMessage().GetText()
		}
		if text == "" {
			if v.Message.GetImageMessage() == nil && v.Message.GetAudioMessage() == nil {
				return
			}
		}
		senderJID := v.Info.Sender.ToNonAD().String()

		if v.Info.IsFromMe {
			return
		}

		msgType := "text"
		mediaPath := ""

		if imgMsg := v.Message.GetImageMessage(); imgMsg != nil {
			clientMu.RLock()
			c := clientMap[tenantID]
			clientMu.RUnlock()
			if c != nil {
				data, err := c.Download(context.Background(), imgMsg)
				if err == nil {
					os.MkdirAll("/tmp/wa-media", 0755)
					msgID := v.Info.ID
					filePath := fmt.Sprintf("/tmp/wa-media/%s.jpg", msgID)
					if err := os.WriteFile(filePath, data, 0644); err == nil {
						msgType = "image"
						mediaPath = filePath
						log.Printf("[Tenant %s] Downloaded image to %s", tenantID, filePath)
					}
				}
			}
		} else if audioMsg := v.Message.GetAudioMessage(); audioMsg != nil {
			clientMu.RLock()
			c := clientMap[tenantID]
			clientMu.RUnlock()
			if c != nil {
				data, err := c.Download(context.Background(), audioMsg)
				if err == nil {
					os.MkdirAll("/tmp/wa-media", 0755)
					msgID := v.Info.ID
					filePath := fmt.Sprintf("/tmp/wa-media/%s.ogg", msgID)
					if err := os.WriteFile(filePath, data, 0644); err == nil {
						msgType = "audio"
						mediaPath = filePath
						log.Printf("[Tenant %s] Downloaded audio to %s", tenantID, filePath)
					}
				}
			}
		}

		log.Printf("[Tenant %s] Received %s message from %s", tenantID, msgType, senderJID)

		payload := map[string]interface{}{
			"sender":     senderJID,
			"message":    text,
			"msg_type":   msgType,
			"media_path": mediaPath,
		}
		jsonBody, _ := json.Marshal(payload)

		chatbotURL := os.Getenv("CHATBOT_URL")
		if chatbotURL == "" {
			if os.Getenv("APP_ENV") == "production" || os.Getenv("DB_HOST") == "postgres" {
				chatbotURL = "http://umkm-chatbot:8203"
			} else {
				chatbotURL = "http://localhost:8203"
			}
		}
		webhookURL := chatbotURL + "/webhook/wa?tenant_id=" + tenantID
		resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonBody))
		if err != nil {
			log.Printf("Failed to forward message to chatbot: %v", err)
		} else {
			resp.Body.Close()
		}

	case *events.Connected:
		log.Printf("[Tenant %s] Connected to WhatsApp!", tenantID)
		clientMu.RLock()
		c := clientMap[tenantID]
		clientMu.RUnlock()
		if c != nil && c.Store.ID != nil && db != nil {
			db.Exec(`INSERT INTO wa_tenant_sessions (tenant_id, jid) VALUES ($1, $2) ON CONFLICT (tenant_id) DO UPDATE SET jid = EXCLUDED.jid`, tenantID, c.Store.ID.String())
		}
	}
}
