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
	"core_project/shared/sdk/queue"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

var AIGatewayURL = "http://localhost:8002/v1/chat"
var NotificationServiceURL = "http://localhost:8005/api/notification/send"
var WAGatewayURL = "http://localhost:8202/api/wa/send"
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

	if cfg.Env == "production" || cfg.DB.Host == "postgres" {
		NotificationServiceURL = "http://notification-service:8005/api/notification/send"
		WAGatewayURL = "http://wa-gateway:8202/api/wa/send"
		AIGatewayURL = "http://ai-gateway:8002/v1/chat"
	}

	var err error
	DB, err = initDB(cfg)
	if err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer DB.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       0,
	})

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		slog.Error("Failed to connect to Redis", "error", err)
		os.Exit(1)
	}

	// Start metrics + health HTTP server
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
		slog.Info("UMKM Automation metrics server starting", "port", "8204")
		if err := http.ListenAndServe(":8204", mux); err != nil {
			slog.Error("Metrics server failed", "error", err)
		}
	}()

	// Start RabbitMQ consumers if URL is configured
	if cfg.RabbitMQ.URL != "" {
		go startRabbitMQConsumers(cfg.RabbitMQ.URL)
	} else {
		slog.Warn("RABBITMQ_URL not set — async queue consumers disabled")
	}

	// Redis pub/sub loop (existing functionality)
	pubsub := redisClient.Subscribe(context.Background(), "tenant_events")
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

// startRabbitMQConsumers connects to RabbitMQ and starts queue consumers.
// Runs in a goroutine; logs and exits on fatal error.
func startRabbitMQConsumers(url string) {
	client, err := queue.NewClient(url)
	if err != nil {
		slog.Error("RabbitMQ connection failed — async consumers disabled", "error", err)
		return
	}
	defer client.Close()

	queues := []string{
		"notifications.wa",
		"notifications.telegram",
		"chatbot.replies",
	}
	for _, q := range queues {
		if err := client.DeclareQueue(q); err != nil {
			slog.Error("Failed to declare queue", "queue", q, "error", err)
			return
		}
	}

	slog.Info("RabbitMQ consumers started", "queues", queues)

	// notifications.wa — forward to wa-gateway
	go client.Consume("notifications.wa", func(job queue.Job) error {
		return processWANotification(job)
	})

	// notifications.telegram — forward to notification-service
	go client.Consume("notifications.telegram", func(job queue.Job) error {
		return processTelegramNotification(job)
	})

	// chatbot.replies — forward AI-generated reply to wa-gateway
	go client.Consume("chatbot.replies", func(job queue.Job) error {
		return processChatbotReply(job)
	})

	// Keep goroutine alive until broker disconnects all channels.
	select {}
}

func processWANotification(job queue.Job) error {
	target, _ := job.Data["target"].(string)
	message, _ := job.Data["message"].(string)
	if target == "" || message == "" {
		slog.Warn("Invalid notifications.wa job — missing target or message", "job_id", job.JobID)
		return nil
	}

	payload := map[string]interface{}{
		"target":  target,
		"message": message,
		"type":    "wa",
	}
	body, _ := json.Marshal(payload)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", WAGatewayURL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", job.TenantID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("wa-gateway send: %w", err)
	}
	defer resp.Body.Close()

	slog.Info("WA notification sent", "job_id", job.JobID, "tenant_id", job.TenantID, "target", target)
	return nil
}

func processTelegramNotification(job queue.Job) error {
	target, _ := job.Data["target"].(string)
	message, _ := job.Data["message"].(string)
	if target == "" || message == "" {
		slog.Warn("Invalid notifications.telegram job — missing target or message", "job_id", job.JobID)
		return nil
	}

	payload := map[string]interface{}{
		"target":  target,
		"message": message,
		"type":    "telegram",
	}
	body, _ := json.Marshal(payload)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", NotificationServiceURL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", job.TenantID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("notification-service send: %w", err)
	}
	defer resp.Body.Close()

	slog.Info("Telegram notification sent", "job_id", job.JobID, "tenant_id", job.TenantID, "target", target)
	return nil
}

func processChatbotReply(job queue.Job) error {
	waNumber, _ := job.Data["wa_number"].(string)
	reply, _ := job.Data["reply"].(string)
	if waNumber == "" || reply == "" {
		slog.Warn("Invalid chatbot.replies job — missing wa_number or reply", "job_id", job.JobID)
		return nil
	}

	payload := map[string]interface{}{
		"target":  waNumber,
		"message": reply,
	}
	body, _ := json.Marshal(payload)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", WAGatewayURL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", job.TenantID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("wa-gateway chatbot reply: %w", err)
	}
	defer resp.Body.Close()

	slog.Info("Chatbot reply sent", "job_id", job.JobID, "tenant_id", job.TenantID, "wa_number", waNumber)
	return nil
}

func handleMonthlyReport(tenantID string) {
	slog.Info("Processing monthly_report", "tenant_id", tenantID)

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
	snippet := reportText
	if len(snippet) > 50 {
		snippet = snippet[:50]
	}
	slog.Info("AI Report Generated", "snippet", snippet+"...")

	if DB == nil {
		if isTest {
			slog.Info("AI Report Generated (test mode, skip DB query)", "tenant_id", tenantID)
			return
		}
		slog.Error("Database connection not available")
		return
	}

	var notifyEmail, notifyWA, notifyTelegram bool
	var telegramChatID string
	var waNumber string
	var email string

	DB.QueryRow(context.Background(), `
		SELECT s.notify_email, s.notify_wa, s.notify_telegram, s.telegram_chat_id, t.wa_number, u.email
		FROM tenants t
		LEFT JOIN tenant_notification_settings s ON t.id = s.tenant_id
		LEFT JOIN users u ON t.id = u.tenant_id AND u.role = 'owner'
		WHERE t.id = $1 LIMIT 1
	`, tenantID).Scan(&notifyEmail, &notifyWA, &notifyTelegram, &telegramChatID, &waNumber, &email)

	if notifyEmail && email != "" {
		slog.Info("Simulated Email Sent", "to", email, "subject", "Laporan Keuangan Bulanan")
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
