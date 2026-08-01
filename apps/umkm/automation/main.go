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

	"core_project/shared/observability"
	"core_project/shared/sdk/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

var AIGatewayURL = "http://localhost:8002/v1/chat"
var NotificationServiceURL = "http://localhost:8005/api/notification/send"
var DB *pgxpool.Pool
var isTest bool

type EventPayload struct {
	TenantID string `json:"tenant_id"`
	Event    string `json:"event"`
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := config.LoadConfig(".env")

	// Set service URLs based on environment
	if cfg.Env == "production" || cfg.DB.Host == "postgres" {
		NotificationServiceURL = "http://notification-service:8005/api/notification/send"
	}

	var err error
	DB, err = initDB(cfg)
	if err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer DB.Close()

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

	// Start metrics HTTP server in background
	mux := http.NewServeMux()
	mux.Handle("/metrics", observability.PrometheusHandler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		dbStatus := "disconnected"
		if DB != nil {
			if err := DB.Ping(r.Context()); err == nil {
				dbStatus = "connected"
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":   "ok",
			"database": dbStatus,
		})
	})
	go func() {
		port := "8204"
		slog.Info("UMKM Automation metrics server starting", "port", port)
		if err := http.ListenAndServe(":"+port, mux); err != nil {
			slog.Error("Metrics server failed", "error", err)
		}
	}()

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
	req.Header.Set("X-Tenant-ID", tenantID)

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

	reportText := aiResp.Text
	slog.Info("✅ AI Report Generated", "snippet", reportText[:min(50, len(reportText))]+"...")

	// Fetch Notification Settings
	var notifyEmail, notifyWA, notifyTelegram bool
	var telegramChatID string
	var waNumber string
	var email string

	// DB guard untuk testing
	if DB == nil {
		if isTest {
			slog.Info("✅ AI Report Generated (test mode, skip DB query)", "tenant_id", tenantID)
			return
		}
		slog.Error("Database connection not available")
		return
	}

	// Defaults and queries
	DB.QueryRow(context.Background(), `
		SELECT s.notify_email, s.notify_wa, s.notify_telegram, s.telegram_chat_id, t.wa_number, u.email
		FROM tenants t
		LEFT JOIN tenant_notification_settings s ON t.id = s.tenant_id
		LEFT JOIN users u ON t.id = u.tenant_id AND u.role = 'owner'
		WHERE t.id = $1 LIMIT 1
	`, tenantID).Scan(&notifyEmail, &notifyWA, &notifyTelegram, &telegramChatID, &waNumber, &email)

	// Send Notifications
	if notifyEmail && email != "" {
		slog.Info("📧 Simulated Email Sent", "to", email, "subject", "Laporan Keuangan Bulanan")
	}

	if notifyWA && waNumber != "" {
		sendNotification(tenantID, "wa", waNumber, reportText)
	}

	if notifyTelegram && telegramChatID != "" {
		sendNotification(tenantID, "telegram", telegramChatID, reportText)
	}
}

func sendNotification(tenantID, channel, target, message string) {
	payload := map[string]interface{}{
		"target":  target,
		"message": message,
		"type":    channel,
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", NotificationServiceURL, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenantID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("Failed to send notification request", "channel", channel, "error", err)
		return
	}
	defer resp.Body.Close()
	slog.Info("Notification sent via service", "channel", channel, "target", target)
}

func initDB(cfg *config.Config) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.DB.User, cfg.DB.Password, cfg.DB.Host, cfg.DB.Port, cfg.DB.Name, cfg.DB.SSLMode)

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
