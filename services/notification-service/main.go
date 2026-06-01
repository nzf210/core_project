package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type NotificationRequest struct {
	Target  string `json:"target"` // Phone number for WA or Chat ID for Telegram
	Message string `json:"message"`
	Type    string `json:"type"`   // 'wa' or 'telegram'
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/notification/send", handleSendNotification)

	port := "8005"
	slog.Info("Notification Service listening", "port", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		slog.Error("Failed to start server", "error", err)
	}
}

func handleSendNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "global"
	}

	var req NotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Type == "telegram" {
		err := sendTelegram(req.Target, req.Message)
		if err != nil {
			slog.Error("Failed to send telegram", "error", err)
			http.Error(w, "Failed to send telegram", http.StatusInternalServerError)
			return
		}
	} else if req.Type == "wa" {
		err := sendWA(tenantID, req.Target, req.Message)
		if err != nil {
			slog.Error("Failed to send WA", "error", err)
			http.Error(w, "Failed to send WA", http.StatusInternalServerError)
			return
		}
	} else {
		http.Error(w, "Invalid notification type", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

func sendTelegram(chatID, message string) error {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		slog.Warn("TELEGRAM_BOT_TOKEN not set, skipping telegram")
		return nil
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	
	payload := map[string]interface{}{
		"chat_id": chatID,
		"text":    message,
	}
	body, _ := json.Marshal(payload)
	
	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram api returned status %d", resp.StatusCode)
	}
	return nil
}

func sendWA(tenantID, target, message string) error {
	waGatewayURL := "http://localhost:8202/api/wa/send"
	if os.Getenv("APP_ENV") == "production" || os.Getenv("DB_HOST") == "postgres" {
		waGatewayURL = "http://wa-gateway:8202/api/wa/send"
	}

	// Assuming the custom WA gateway accepts JID like 628xxx@s.whatsapp.net
	targetJID := target
	if !strings.Contains(targetJID, "@s.whatsapp.net") {
		if strings.HasPrefix(targetJID, "0") {
			targetJID = "62" + targetJID[1:]
		}
		targetJID = strings.TrimPrefix(targetJID, "+")
		targetJID = targetJID + "@s.whatsapp.net"
	}

	data := url.Values{}
	data.Set("target", targetJID)
	data.Set("message", message)
	data.Set("tenant_id", tenantID)

	reqWA, _ := http.NewRequest("POST", waGatewayURL, strings.NewReader(data.Encode()))
	reqWA.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	resp, err := http.DefaultClient.Do(reqWA)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("wa-gateway returned status %d", resp.StatusCode)
	}
	return nil
}
