package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"core_project/shared/sdk/config"
	"core_project/shared/sdk/encryption"
	"core_project/shared/sdk/response"
)

const (
	headerContentType = "Content-Type"
	contentTypeJSON   = "application/json"
)

func handleSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	tenantID := r.Header.Get(response.XTenantID)
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

	cred, err := getCredential(r.Context(), tenantID)
	if err != nil {
		slog.Error("Failed to get credential", "tenant", tenantID, "error", err)
		response.Error(w, http.StatusNotFound, "No Cloud API credentials configured for this tenant", err)
		return
	}

	payload := MetaSendPayload{
		MessagingProduct: "whatsapp",
		RecipientType:   "individual",
		To:              normalizeTo(req.To),
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
		templateName := "custom"
		if req.Type == "template" && req.Template != "" {
			templateName = req.Template
		}
		waCloudMessagesTotal.WithLabelValues(templateName, "failed").Inc()
		response.Error(w, http.StatusBadGateway,
			fmt.Sprintf("Meta API error: %s", result.Error.Message), nil)
		return
	}

	waMsgID := ""
	if len(result.Messages) > 0 {
		waMsgID = result.Messages[0].ID
	}

	templateName := "custom"
	if req.Type == "template" && req.Template != "" {
		templateName = req.Template
	}
	waCloudMessagesTotal.WithLabelValues(templateName, "sent").Inc()

	slog.Info("Message sent via Cloud API",
		"tenant", tenantID,
		"to", req.To,
		"wa_message_id", waMsgID,
	)

	w.Header().Set(headerContentType, contentTypeJSON)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(SendResponse{
		Success: true,
		Message: "Message sent via WhatsApp Cloud API",
		WAMsgID: waMsgID,
	})
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		token := r.URL.Query().Get("hub.verify_token")
		if verifyWebhookToken(r.Context(), token) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(r.URL.Query().Get("hub.challenge")))
			return
		}
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

	var notif struct {
		Object string `json:"object"`
		Entry  []struct {
			ID      string `json:"id"`
			Changes []struct {
				Value struct {
					Messaging []struct {
						Sender    struct{ ID string `json:"id"` } `json:"sender"`
						Recipient struct{ ID string `json:"id"` } `json:"recipient"`
						Message   struct {
							ID   string `json:"id"`
							Text string `json:"text"`
						} `json:"message"`
					} `json:"messaging"`
				} `json:"value"`
			} `json:"changes"`
		} `json:"entry"`
	}

	if err := json.Unmarshal(body, &notif); err != nil {
		slog.Warn("Failed to parse webhook payload", "error", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	slog.Info("Webhook received",
		"object", notif.Object,
		"entry_count", len(notif.Entry),
	)

	for _, entry := range notif.Entry {
		for _, change := range entry.Changes {
			for _, msg := range change.Value.Messaging {
				slog.Info("Incoming message",
					"from", msg.Sender.ID,
					"to", msg.Recipient.ID,
					"message_id", msg.Message.ID,
					"text", msg.Message.Text,
				)
			}
		}
	}

	w.WriteHeader(http.StatusOK)
}

func handleAdminCredentials(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get(response.XUserRole)
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, response.SuperadminOnly, nil)
		return
	}

	switch r.Method {
	case http.MethodGet:
		tenantID := r.URL.Query().Get("tenant_id")
		if tenantID != "" {
			cred, err := getCredential(r.Context(), tenantID)
			if err != nil {
				response.Error(w, http.StatusNotFound, "Credential not found", err)
				return
			}
			response.JSON(w, http.StatusOK, "Credential retrieved", cred)
			return
		}

		rows, err := DB.Query(r.Context(), `
			SELECT id, tenant_id, phone_number_id, waba_id, is_active, created_at, updated_at
			FROM wa_cloud_api_credentials ORDER BY created_at DESC LIMIT 100
		`, nil)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to list credentials", err)
			return
		}
		defer rows.Close()

		type credRow struct {
			ID            string `json:"id"`
			TenantID      string `json:"tenant_id"`
			PhoneNumberID string `json:"phone_number_id"`
			WABAID        string `json:"waba_id"`
			IsActive      bool   `json:"is_active"`
			CreatedAt     string `json:"created_at"`
			UpdatedAt     string `json:"updated_at"`
		}
		var creds []credRow
		for rows.Next() {
			var c credRow
			if rows.Scan(&c.ID, &c.TenantID, &c.PhoneNumberID, &c.WABAID, &c.IsActive, &c.CreatedAt, &c.UpdatedAt) == nil {
				creds = append(creds, c)
			}
		}
		response.JSON(w, http.StatusOK, "Credentials retrieved", creds)

	case http.MethodPost:
		var req struct {
			TenantID      string `json:"tenant_id"`
			PhoneNumberID string `json:"phone_number_id"`
			WABAID        string `json:"waba_id"`
			AccessToken   string `json:"access_token"`
			VerifyToken   string `json:"verify_token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, response.InvalidRequest, err)
			return
		}

		// Encrypt access_token before storing
		encryptedToken, err := encryption.Encrypt(req.AccessToken, config.GlobalConfig.EncryptionKey)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to encrypt token", err)
			return
		}

		var id string
		err = DB.QueryRow(r.Context(), `
			INSERT INTO wa_cloud_api_credentials (tenant_id, phone_number_id, waba_id, access_token, verify_token, is_active)
			VALUES ($1, $2, $3, $4, $5, true)
			ON CONFLICT (tenant_id) DO UPDATE SET phone_number_id = EXCLUDED.phone_number_id, waba_id = EXCLUDED.waba_id, access_token = EXCLUDED.access_token, is_active = true, updated_at = NOW()
			RETURNING id
		`, req.TenantID, req.PhoneNumberID, req.WABAID, encryptedToken, req.VerifyToken).Scan(&id)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to save credential", err)
			return
		}
		response.JSON(w, http.StatusOK, "Credential saved", map[string]string{"id": id})

	default:
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
	}
}

func handleAdminCredentialsItem(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get(response.XUserRole)
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, response.SuperadminOnly, nil)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/admin/credentials/")
	if id == "" {
		id = r.URL.Query().Get("id")
	}

	if id == "" {
		response.Error(w, http.StatusBadRequest, "Missing credential id", nil)
		return
	}

	switch r.Method {
	case http.MethodGet:
		var cred CloudAPICredential
		err := DB.QueryRow(r.Context(), `
			SELECT id, tenant_id, phone_number_id, COALESCE(waba_id, ''), is_active, created_at, updated_at
			FROM wa_cloud_api_credentials WHERE id = $1
		`, id).Scan(&cred.ID, &cred.TenantID, &cred.PhoneNumberID, &cred.WABAID, &cred.IsActive, &cred.CreatedAt, &cred.UpdatedAt)
		if err != nil {
			response.Error(w, http.StatusNotFound, "Credential not found", err)
			return
		}
		response.JSON(w, http.StatusOK, "Credential retrieved", cred)

	case http.MethodPut:
		var req struct {
			PhoneNumberID string `json:"phone_number_id"`
			WABAID        string `json:"waba_id"`
			AccessToken   string `json:"access_token"`
			IsActive      bool   `json:"is_active"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, response.InvalidRequest, err)
			return
		}

		// Encrypt access_token if provided
		tokenToStore := req.AccessToken
		if req.AccessToken != "" {
			encrypted, err := encryption.Encrypt(req.AccessToken, config.GlobalConfig.EncryptionKey)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, "Failed to encrypt token", err)
				return
			}
			tokenToStore = encrypted
		}

		_, err := DB.Exec(r.Context(), `
			UPDATE wa_cloud_api_credentials
			SET phone_number_id = $1, waba_id = $2, access_token = $3, is_active = $4, updated_at = NOW()
			WHERE id = $5
		`, req.PhoneNumberID, req.WABAID, tokenToStore, req.IsActive, id)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to update credential", err)
			return
		}
		response.JSON(w, http.StatusOK, "Credential updated", nil)

	case http.MethodDelete:
		_, err := DB.Exec(r.Context(), "DELETE FROM wa_cloud_api_credentials WHERE id = $1", id)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to delete credential", err)
			return
		}
		response.JSON(w, http.StatusOK, "Credential deleted", nil)

	default:
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
	}
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	if DB == nil {
		response.Error(w, http.StatusServiceUnavailable, "Database not connected", nil)
		return
	}
	if err := DB.Ping(r.Context()); err != nil {
		response.Error(w, http.StatusServiceUnavailable, "Database ping failed", err)
		return
	}
	response.JSON(w, http.StatusOK, "OK", map[string]string{"status": "healthy"})
}

func handleValidateCredential(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	var req struct {
		PhoneNumberID string `json:"phone_number_id"`
		AccessToken   string `json:"access_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.InvalidRequest, err)
		return
	}

	url := fmt.Sprintf("%s/%s/%s", graphBaseURL, graphVersion, req.PhoneNumberID)
	httpReq, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, url, nil)
	httpReq.Header.Set("Authorization", "Bearer "+req.AccessToken)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		response.Error(w, http.StatusBadGateway, "Failed to validate credential", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		response.Error(w, http.StatusUnauthorized, "Invalid credentials", nil)
		return
	}

	var result struct {
		DisplayPhoneNumber string `json:"display_phone_number"`
		VerifiedName       string `json:"verified_name"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	response.JSON(w, http.StatusOK, "Credential valid", map[string]string{
		"phone_number": result.DisplayPhoneNumber,
		"verified_name": result.VerifiedName,
	})
}
