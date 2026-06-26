package main

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"time"

	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
)

func setContainer(c *sqlstore.Container) {
	// Container is managed by routes.go; this stub satisfies the call site
	_ = c
}

// eventHandler handles WhatsApp events for a tenant
func eventHandler(tenantID string, evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		handleMessageEvent(tenantID, v)
	case *events.Connected:
		handleConnectedEvent(tenantID)
	}
}

func handleMessageEvent(tenantID string, v *events.Message) {
	// Ignore group messages
	if v.Info.IsGroup {
		return
	}
	// Ignore messages from ourselves
	if v.Info.IsFromMe {
		return
	}

	// Skip initial sync history (timestamp too old)
	if time.Since(v.Info.Timestamp) > 5*time.Minute {
		return
	}

	var text string
	if v.Message.GetConversation() != "" {
		text = v.Message.GetConversation()
	} else if v.Message.GetExtendedTextMessage() != nil {
		text = v.Message.GetExtendedTextMessage().GetText()
	}
	if text == "" {
		return // Ignore non-text messages for now
	}

	senderJID := v.Info.Sender.ToNonAD().String()
	senderName := v.Info.PushName

	// Build standard webhook payload for UMKM Chatbot
	payload := map[string]interface{}{
		"tenant_id":   tenantID,
		"sender_jid":  senderJID,
		"sender_name": senderName,
		"message":     text,
		"timestamp":   v.Info.Timestamp.Unix(),
		"source":      "whatsmeow",
	}

	jsonBody, _ := json.Marshal(payload)

	// Check if it's the N8N Chatbot or Go Chatbot
	var n8nWebhookURL string
	if db != nil {
		err := db.QueryRow(`
			SELECT n8n_webhook_url 
			FROM tenant_chatbot_configs 
			WHERE tenant_id = $1 AND is_active = true
		`, tenantID).Scan(&n8nWebhookURL)
		
		if err == nil && n8nWebhookURL != "" {
			// Forward to N8N
			resp, err := http.Post(n8nWebhookURL, "application/json", bytes.NewBuffer(jsonBody))
			if err != nil {
				slog.Error("Failed to forward message to N8N", "error", err)
			} else {
				resp.Body.Close()
			}
			return
		}
	}

	// Default Go Chatbot route
	chatbotURL := os.Getenv("UMKM_CHATBOT_URL")
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
		slog.Error("Failed to forward message to chatbot", "error", err)
	} else {
		resp.Body.Close()
	}
}

func handleConnectedEvent(tenantID string) {
	slog.Info("Connected to WhatsApp", "tenant_id", tenantID)
	clientMu.RLock()
	c := clientMap[tenantID]
	clientMu.RUnlock()
	if c != nil && c.Store.ID != nil && db != nil {
		db.Exec(`INSERT INTO wa_tenant_sessions (tenant_id, jid) VALUES ($1, $2) ON CONFLICT (tenant_id) DO UPDATE SET jid = EXCLUDED.jid`, tenantID, c.Store.ID.String())
	}
}
