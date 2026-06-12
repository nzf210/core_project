package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"core_project/shared/sdk/config"
	"core_project/shared/sdk/response"
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

	port := cfg.WhatsApp.CloudAPIPort
	if port == "" {
		port = "8210"
	}

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      corsMiddleware(mux),
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
func handleSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		response.Error(w, http.StatusBadRequest, "Missing X-Tenant-ID header", nil)
		return
	}

	var req SendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if req.To == "" {
		response.Error(w, http.StatusBadRequest, "Missing 'to' field", nil)
		return
	}

	// Query credential per-tenant
	cred, err := getCredential(r.Context(), tenantID)
	if err != nil {
		slog.Error("Failed to get credential", "tenant", tenantID, "error", err)
		response.Error(w, http.StatusNotFound, "No Cloud API credentials configured for this tenant", err)
		return
	}

	// Build Meta payload
	payload := MetaSendPayload{
		MessagingProduct: "whatsapp",
		RecipientType:    "individual",
		To:               normalizeTo(req.To),
	}

	if req.Type == "template" && req.Template != "" {
		payload.Type = "template"
		payload.Template = &MetaTemplate{
			Name:     req.Template,
			Language: MetaTemplateLanguage{Code: "id"},
		}
		if len(req.Params) > 0 {
			var params []MetaTemplateParam
			for _, p := range req.Params {
				params = append(params, MetaTemplateParam{Type: "text", Text: p})
			}
			payload.Template.Components = []MetaTemplateComp{
				{Type: "body", Parameters: params},
			}
		}
	} else {
		payload.Type = "text"
		payload.Text = &MetaText{Body: req.Text}
	}

	// Send to Meta Graph API
	result, err := sendToMeta(r.Context(), cred.PhoneNumberID, cred.AccessToken, payload)
	if err != nil {
		slog.Error("Failed to send via Cloud API", "tenant", tenantID, "error", err)
		response.Error(w, http.StatusBadGateway, "Failed to send via Cloud API", err)
		return
	}

	if result.Error != nil {
		slog.Error("Meta API error", "tenant", tenantID,
			"meta_code", result.Error.Code,
			"meta_message", result.Error.Message)
		response.Error(w, http.StatusBadGateway,
			fmt.Sprintf("Meta API error: %s", result.Error.Message), nil)
		return
	}

	waMsgID := ""
	if len(result.Messages) > 0 {
		waMsgID = result.Messages[0].ID
	}

	slog.Info("Message sent via Cloud API",
		"tenant", tenantID,
		"to", req.To,
		"wa_message_id", waMsgID,
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(SendResponse{
		Success: true,
		Message: "Message sent via WhatsApp Cloud API",
		WAMsgID: waMsgID,
	})
}

// POST /webhook
// Menerima incoming message + status callback dari Meta
func handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		// Meta webhook verification challenge
		mode := r.URL.Query().Get("hub.mode")
		token := r.URL.Query().Get("hub.verify_token")
		challenge := r.URL.Query().Get("hub.challenge")

		if mode == "subscribe" {
			// Verify against stored tokens in DB
			if verifyWebhookToken(r.Context(), token) {
				slog.Info("Webhook verified", "mode", mode)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(challenge))
				return
			}
		}

		slog.Warn("Webhook verification failed", "mode", mode)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Failed to read body", err)
		return
	}
	defer r.Body.Close()

	slog.Info("Webhook received", "body_length", len(body))

	// Parse incoming webhook
	var webhook map[string]interface{}
	if err := json.Unmarshal(body, &webhook); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON", err)
		return
	}

	// Process entries (status updates, incoming messages)
	if entries, ok := webhook["entry"].([]interface{}); ok {
		for _, entry := range entries {
			e, _ := entry.(map[string]interface{})
			changes, _ := e["changes"].([]interface{})
			for _, change := range changes {
				c, _ := change.(map[string]interface{})
				value, _ := c["value"].(map[string]interface{})

				// Log status updates
				if statuses, ok := value["statuses"].([]interface{}); ok {
					for _, s := range statuses {
						status, _ := s.(map[string]interface{})
						slog.Info("Message status update",
							"message_id", status["id"],
							"status", status["status"],
							"timestamp", status["timestamp"],
						)
					}
				}

				// Log incoming messages (for future chatbot integration)
				if messages, ok := value["messages"].([]interface{}); ok {
					slog.Info("Incoming messages count", "count", len(messages))
					for _, m := range messages {
						msg, _ := m.(map[string]interface{})
						slog.Info("Incoming message", "from", msg["from"], "id", msg["id"])
					}
				}
			}
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "received"})
}

// GET/POST/PUT/DELETE /admin/credentials
// Superadmin CRUD untuk credential per-tenant
func handleAdminCredentials(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, "Superadmin only", nil)
		return
	}

	ctx := r.Context()

	switch r.Method {
	case http.MethodGet:
		rows, err := DB.Query(ctx, `
			SELECT c.id, c.tenant_id, c.phone_number_id, COALESCE(c.waba_id,''),
			       c.is_active, c.created_at, c.updated_at,
			       COALESCE(t.business_name, t.name) as tenant_name
			FROM wa_cloud_api_credentials c
			JOIN tenants t ON t.id = c.tenant_id
			ORDER BY c.created_at DESC
		`)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to fetch credentials", err)
			return
		}
		defer rows.Close()

		var credentials []map[string]interface{}
		for rows.Next() {
			var id, tenantID, phoneID, wabaID, tenantName string
			var isActive bool
			var createdAt, updatedAt time.Time
			if err := rows.Scan(&id, &tenantID, &phoneID, &wabaID, &isActive, &createdAt, &updatedAt, &tenantName); err != nil {
				continue
			}
			credentials = append(credentials, map[string]interface{}{
				"id":              id,
				"tenant_id":       tenantID,
				"tenant_name":     tenantName,
				"phone_number_id": phoneID,
				"waba_id":         wabaID,
				"is_active":       isActive,
				"created_at":      createdAt,
				"updated_at":      updatedAt,
			})
		}
		if credentials == nil {
			credentials = []map[string]interface{}{}
		}

		response.JSON(w, http.StatusOK, "Credentials fetched", credentials)

	case http.MethodPost:
		var req struct {
			TenantID      string `json:"tenant_id"`
			PhoneNumberID string `json:"phone_number_id"`
			WABAID        string `json:"waba_id"`
			AccessToken   string `json:"access_token"`
			VerifyToken   string `json:"verify_token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "Invalid request body", err)
			return
		}

		if req.TenantID == "" || req.PhoneNumberID == "" || req.AccessToken == "" {
			response.Error(w, http.StatusBadRequest,
				"tenant_id, phone_number_id, and access_token are required", nil)
			return
		}

		// Upsert
		_, err := DB.Exec(ctx, `
			INSERT INTO wa_cloud_api_credentials (tenant_id, phone_number_id, waba_id, access_token, verify_token)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (phone_number_id) DO UPDATE SET
				access_token = EXCLUDED.access_token,
				verify_token = EXCLUDED.verify_token,
				waba_id = EXCLUDED.waba_id,
				is_active = true,
				updated_at = NOW()
		`, req.TenantID, req.PhoneNumberID, req.WABAID, req.AccessToken, req.VerifyToken)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to save credential", err)
			return
		}

		response.JSON(w, http.StatusCreated, "Credential saved", nil)

	default:
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
	}
}

// PUT/DELETE /admin/credentials/{id}
func handleAdminCredentialsItem(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, "Superadmin only", nil)
		return
	}

	// Extract credential ID from path: /admin/credentials/{id}
	id := strings.TrimPrefix(r.URL.Path, "/admin/credentials/")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "Credential ID required", nil)
		return
	}

	ctx := r.Context()

	switch r.Method {
	case http.MethodPut:
		var req struct {
			IsActive bool `json:"is_active"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "Invalid request body", err)
			return
		}

		_, err := DB.Exec(ctx, `
			UPDATE wa_cloud_api_credentials SET is_active = $1, updated_at = NOW()
			WHERE id = $2
		`, req.IsActive, id)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to update credential", err)
			return
		}

		response.JSON(w, http.StatusOK, "Credential updated", nil)

	case http.MethodDelete:
		_, err := DB.Exec(ctx, "DELETE FROM wa_cloud_api_credentials WHERE id = $1", id)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to delete credential", err)
			return
		}

		response.JSON(w, http.StatusOK, "Credential deleted", nil)

	default:
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
	}
}

// GET /healthz
func handleHealthz(w http.ResponseWriter, r *http.Request) {
	healthy := true
	status := "healthy"

	if DB != nil {
		if err := DB.Ping(r.Context()); err != nil {
			healthy = false
			status = "database unreachable"
		}
	}

	code := http.StatusOK
	if !healthy {
		code = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     status,
		"service":    "wa-cloud-api",
		"graph_api":  graphBaseURL + "/" + graphVersion,
	})
}

// ─────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────

func buildDBURI(cfg *config.Config) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s&pool_max_conns=10",
		cfg.DB.User, cfg.DB.Password, cfg.DB.Host, cfg.DB.Port, cfg.DB.Name, cfg.DB.SSLMode)
}

func getCredential(ctx context.Context, tenantID string) (*CloudAPICredential, error) {
	var cred CloudAPICredential
	var wabaID, verifyToken *string

	err := DB.QueryRow(ctx, `
		SELECT id, tenant_id, phone_number_id, waba_id, access_token, verify_token, is_active
		FROM wa_cloud_api_credentials
		WHERE tenant_id = $1 AND is_active = true
		ORDER BY created_at DESC
		LIMIT 1
	`, tenantID).Scan(&cred.ID, &cred.TenantID, &cred.PhoneNumberID,
		&wabaID, &cred.AccessToken, &verifyToken, &cred.IsActive)

	if err != nil {
		return nil, err
	}

	if wabaID != nil {
		cred.WABAID = *wabaID
	}
	if verifyToken != nil {
		cred.VerifyToken = *verifyToken
	}

	return &cred, nil
}

func verifyWebhookToken(ctx context.Context, token string) bool {
	if token == "" {
		return false
	}

	var count int
	err := DB.QueryRow(ctx, `
		SELECT COUNT(*) FROM wa_cloud_api_credentials
		WHERE verify_token = $1 AND is_active = true
	`, token).Scan(&count)

	return err == nil && count > 0
}

func sendToMeta(ctx context.Context, phoneNumberID, accessToken string, payload MetaSendPayload) (*MetaResponse, error) {
	url := fmt.Sprintf("%s/%s/%s/messages", graphBaseURL, graphVersion, phoneNumberID)

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send to Meta: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var metaResp MetaResponse
	if err := json.Unmarshal(respBody, &metaResp); err != nil {
		return nil, fmt.Errorf("failed to parse Meta response: %w (body: %s)", err, string(respBody))
	}

	return &metaResp, nil
}

// normalizeTo ensures phone number is in international format without +
func normalizeTo(phone string) string {
	phone = strings.TrimPrefix(phone, "+")
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	return phone
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Tenant-ID, X-User-Role")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
