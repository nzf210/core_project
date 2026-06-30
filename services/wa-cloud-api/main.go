package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"core_project/shared/observability"
	"core_project/shared/sdk/config"
)

// ─────────────────────────────────────────────
// Types
// ─────────────────────────────────────────────

type SendRequest struct {
	To      string `json:"to"`       // Recipient phone (international format: 6281234567890)
	Type    string `json:"type"`     // "text" or "template"
	Text    string `json:"text"`     // For type=text: message body
	Template string `json:"template"` // For type=template: template name
	Params  []string `json:"params"` // Template parameters
}

type CloudAPICredential struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	PhoneNumberID string    `json:"phone_number_id"`
	WABAID        string    `json:"waba_id,omitempty"`
	AccessToken   string    `json:"-"`
	VerifyToken   string    `json:"-"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type SendResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	WAMsgID string `json:"wa_message_id,omitempty"`
}

type MetaSendPayload struct {
	MessagingProduct string       `json:"messaging_product"`
	RecipientType    string       `json:"recipient_type"`
	To               string       `json:"to"`
	Type             string       `json:"type"`
	Text             *MetaText    `json:"text,omitempty"`
	Template         *MetaTemplate `json:"template,omitempty"`
}

type MetaText struct {
	Body string `json:"body"`
}

type MetaTemplate struct {
	Name     string                `json:"name"`
	Language MetaTemplateLanguage  `json:"language"`
	Components []MetaTemplateComp  `json:"components,omitempty"`
}

type MetaTemplateLanguage struct {
	Code string `json:"code"`
}

type MetaTemplateComp struct {
	Type       string              `json:"type"`
	Parameters []MetaTemplateParam `json:"parameters,omitempty"`
}

type MetaTemplateParam struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type MetaResponse struct {
	MessagingProduct string          `json:"messaging_product"`
	Contacts    []MetaContact        `json:"contacts"`
	Messages    []MetaMessage        `json:"messages"`
	Error       *MetaError           `json:"error,omitempty"`
}

type MetaContact struct {
	Input string `json:"input"`
	WAID  string `json:"wa_id"`
}

type MetaMessage struct {
	ID string `json:"id"`
}

type MetaError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Type    string `json:"type"`
}

// ─────────────────────────────────────────────
// Global state
// ─────────────────────────────────────────────

var (
	DB           *pgxpool.Pool
	graphBaseURL string
	graphVersion string

	// Business metrics
	waCloudMessagesTotal = observability.NewCounter(
		"wa_cloud_messages_total",
		"Total WhatsApp Cloud API messages by template and status",
		[]string{"template", "status"},
	)
)

// ─────────────────────────────────────────────
// Main
// ─────────────────────────────────────────────

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := config.LoadConfig("")
	if config.GlobalConfig == nil {
		config.GlobalConfig = cfg
	}

	graphBaseURL = cfg.WhatsApp.CloudAPIURL
	graphVersion = os.Getenv("WA_CLOUD_API_VERSION")
	if graphVersion == "" {
		graphVersion = "v22.0"
	}

	// Database connection
	dbURI := buildDBURI(cfg)
	var err error
	DB, err = pgxpool.New(context.Background(), dbURI)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer DB.Close()

	slog.Info("Connected to database", "host", cfg.DB.Host, "name", cfg.DB.Name)

	// Auto-migration
	if err := runMigrations(DB); err != nil {
		slog.Warn("Migration warning (non-fatal)", "error", err)
	}

	// Routes
	mux := http.NewServeMux()
	mux.HandleFunc("/send", handleSend)
	mux.HandleFunc("/webhook", handleWebhook)
	mux.HandleFunc("/healthz", handleHealthz)
	mux.HandleFunc("/admin/credentials", handleAdminCredentials)
	mux.HandleFunc("/admin/credentials/", handleAdminCredentialsItem)
	mux.HandleFunc("/validate", handleValidateCredential)

	// Prometheus metrics endpoint
	mux.Handle("/metrics", observability.PrometheusHandler())

	// Wrap handler with observability middleware
	handler := observability.Middleware("wa-cloud-api")(mux)

	port := cfg.WhatsApp.CloudAPIPort
	if port == "" {
		port = "8210"
	}

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      corsMiddleware(handler),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	slog.Info("WA Cloud API listening", "port", port, "graph_api", graphBaseURL+"/"+graphVersion)
	if err := server.ListenAndServe(); err != nil {
		slog.Error("Failed to start server", "error", err)
	}
}

// ─────────────────────────────────────────────
// Handlers
// ─────────────────────────────────────────────

// POST /send
// Kirim pesan WhatsApp via Meta Cloud API.
// Headers: X-Tenant-ID (required), Content-Type: application/json
