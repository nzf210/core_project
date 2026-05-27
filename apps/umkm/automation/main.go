package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"core_project/shared/sdk/config"
	"github.com/redis/go-redis/v9"
)

var AIGatewayURL = "http://localhost:8002/v1/chat"

type EventPayload struct {
	TenantID string `json:"tenant_id"`
	Event    string `json:"event"`
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := config.LoadConfig(".env")
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       0,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		slog.Error("Failed to connect to Redis", "error", err)
		os.Exit(1)
	}

	pubsub := client.Subscribe(context.Background(), "tenant_events")
	defer pubsub.Close()

	slog.Info("UMKM Automation Worker started. Listening on channel: tenant_events")

	ch := pubsub.Channel()
	for msg := range ch {
		slog.Info("Received event", "payload", msg.Payload)
		var payload EventPayload
		if err := json.Unmarshal([]byte(msg.Payload), &payload); err != nil {
			slog.Error("Failed to parse event", "error", err)
			continue
		}

		if payload.Event == "monthly_report" {
			go handleMonthlyReport(payload.TenantID)
		}
	}
}

func handleMonthlyReport(tenantID string) {
	slog.Info("Processing monthly_report", "tenant_id", tenantID)

	// In a real scenario, this would fetch accounting data for the month
	// and feed it to the AI. For simulation, we send a prompt directly.
	prompt := "Buatkan ringkasan performa bisnis bulanan secara profesional untuk laporan PDF yang akan dikirim via email ke pemilik UMKM."

	aiReqBody := map[string]interface{}{
		"provider":   "minimax",
		"message":    prompt,
		"system_msg": "Anda adalah mesin generator laporan keuangan.",
		"tenant_id":  tenantID,
	}
	jsonBody, _ := json.Marshal(aiReqBody)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "POST", AIGatewayURL, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("Failed to call AI Gateway for report", "error", err)
		return
	}
	defer resp.Body.Close()

	var aiResp struct {
		Success bool   `json:"success"`
		Text    string `json:"text"`
	}
	json.NewDecoder(resp.Body).Decode(&aiResp)

	slog.Info("✅ AI PDF Report Generated", "snippet", aiResp.Text[:min(50, len(aiResp.Text))]+"...")
	slog.Info("📧 Simulated Email Sent", "to", "owner_"+tenantID+"@umkm.local", "subject", "Laporan Keuangan Bulanan")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
