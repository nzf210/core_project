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
	if v.Info.IsGroup || v.Info.IsFromMe || time.Since(v.Info.Timestamp) > 5*time.Minute {
		return
	}

	text := extractMessageText(v)
	if text == "" {
		return
	}

	jsonBody, _ := json.Marshal(map[string]interface{}{
		"tenant_id":   tenantID,
		"sender_jid":  v.Info.Sender.ToNonAD().String(),
		"sender_name": v.Info.PushName,
		"message":     text,
		"timestamp":   v.Info.Timestamp.Unix(),
		"source":      "whatsmeow",
	})

	if tryForwardToN8N(tenantID, jsonBody) {
		return
	}
	forwardToChatbot(tenantID, jsonBody)
}

func extractMessageText(v *events.Message) string {
	if v.Message.GetConversation() != "" {
		return v.Message.GetConversation()
	}
	if v.Message.GetExtendedTextMessage() != nil {
		return v.Message.GetExtendedTextMessage().GetText()
	}
	return ""
}

func tryForwardToN8N(tenantID string, jsonBody []byte) bool {
	if db == nil {
		return false
	}
	var n8nURL string
	err := db.QueryRow(`SELECT n8n_webhook_url FROM tenant_chatbot_configs WHERE tenant_id = $1 AND is_active = true`, tenantID).Scan(&n8nURL)
	if err != nil || n8nURL == "" {
		return false
	}
	resp, err := http.Post(n8nURL, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		slog.Error("Failed to forward message to N8N", "error", err)
		return true // still handled, don't fall through
	}
	resp.Body.Close()
	return true
}

func forwardToChatbot(tenantID string, jsonBody []byte) {
	chatbotURL := os.Getenv("UMKM_CHATBOT_URL")
	if chatbotURL == "" {
		if os.Getenv("APP_ENV") == "production" || os.Getenv("DB_HOST") == "postgres" {
			chatbotURL = "http://umkm-chatbot:8203"
		} else {
			chatbotURL = "http://localhost:8203"
		}
	}
	resp, err := http.Post(chatbotURL+"/webhook/wa?tenant_id="+tenantID, "application/json", bytes.NewBuffer(jsonBody))
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
