package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

func setupQRHandler(container *sqlstore.Container) {
	http.HandleFunc("/api/wa/qr", func(w http.ResponseWriter, r *http.Request) {
		tenantID := getTenantID(r)
		if tenantID == "" {
			http.Error(w, `{"error":"tenant_id required"}`, http.StatusBadRequest)
			return
		}

		owned, err := AcquireSessionLock(r.Context(), tenantID)
		if err != nil || !owned {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "busy",
				"message": "Another gateway instance is handling this tenant. Please retry.",
			})
			return
		}

		clientMu.Lock()
		client, exists := clientMap[tenantID]
		if exists && client.Store.ID == nil {
			client.Disconnect()
			delete(clientMap, tenantID)
			exists = false
		}

		if !exists {
			newStore := container.NewDevice()
			clientLog := waLog.Stdout("Client-"+tenantID, "INFO", true)
			client = whatsmeow.NewClient(newStore, clientLog)
			client.AddEventHandler(func(evt interface{}) { eventHandler(tenantID, evt) })
			clientMap[tenantID] = client
		}
		clientMu.Unlock()

		if client.Store.ID != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "connected",
				"message": "Already connected",
			})
			return
		}

		qrChan, _ := client.GetQRChannel(context.Background())
		if err := client.Connect(); err != nil {
			log.Printf("Failed to connect for QR: %v", err)
			http.Error(w, `{"error":"failed to connect"}`, http.StatusInternalServerError)
			return
		}

		for evt := range qrChan {
			if evt.Event == "code" {
				png, err := qrcode.Encode(evt.Code, qrcode.Medium, 256)
				if err != nil {
					http.Error(w, `{"error":"failed to generate qr"}`, http.StatusInternalServerError)
					return
				}
				base64QR := "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"status":   "qr",
					"qr_code":  base64QR,
					"raw_code": evt.Code,
				})

				go func(c *whatsmeow.Client, tid string) {
					time.Sleep(60 * time.Second)
					if c.Store.ID == nil {
						c.Disconnect()
						clientMu.Lock()
						if clientMap[tid] == c {
							delete(clientMap, tid)
						}
						clientMu.Unlock()
						ReleaseSessionLock(context.Background(), tid)
					}
				}(client, tenantID)
				return
			}
			log.Printf("QR channel event: %s", evt.Event)
		}
	})
}
