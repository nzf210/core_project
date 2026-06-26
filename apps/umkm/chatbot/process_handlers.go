package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"core_project/shared/sdk/auth"
	"log/slog"
)

const headerContentTypeForm = "application/x-www-form-urlencoded"

func processChatJob(job ChatJob) {
	ctx := context.Background()
	plan := auth.GetPlan(job.TenantID)

	job.Message = preprocessMedia(plan, job)

	sender := job.Sender
	tenantID := job.TenantID
	userRole := "customer"
	tenantName := "UMKM WCH"

	if tenantID != "" {
		tenantName, userRole = resolveTenantRole(ctx, tenantID, sender)
	} else {
		tenantID, userRole = handleGlobalWebhook(ctx, sender)
		if tenantID == "" {
			return
		}
	}

	chatCfg := loadChatbotConfig(ctx, tenantID)
	if !sendOutsideHoursReply(ctx, tenantID, sender, chatCfg) {
		return
	}

	callAIAndSendReply(ctx, tenantID, sender, job.Message, userRole, tenantName, chatCfg)
}

func preprocessMedia(plan auth.PlanFeaturesRow, job ChatJob) string {
	switch {
	case job.MsgType == "audio" && job.MediaPath != "":
		if plan.MaxAIAudioMinutes == 0 {
			return "[Sistem] Maaf, layanan pesan suara (Voice Note) belum diaktifkan oleh Toko ini. Harap ketik pesan Anda."
		}
		if text, err := transcribeAudio(job.TenantID, job.MediaPath); err == nil {
			return text
		}
		return "[Pesan Suara tidak dapat diproses]"

	case job.MsgType == "image" && job.MediaPath != "":
		if plan.MaxAIVision == 0 {
			if job.Message != "" {
				return job.Message + "\n[Sistem] Maaf, layanan analisa gambar belum diaktifkan oleh Toko ini. Harap ketik pertanyaan Anda secara detail."
			}
			return "[Sistem] Maaf, layanan analisa gambar belum diaktifkan oleh Toko ini. Harap ketik pertanyaan Anda secara detail."
		}
		if text, err := analyzeImage(job.TenantID, job.MediaPath, job.Message); err == nil {
			if job.Message != "" {
				return job.Message + "\n[Analisis Gambar: " + text + "]"
			}
			return "[Analisis Gambar: " + text + "]"
		}
		return job.Message + "\n[Gambar tidak dapat diproses]"

	default:
		return job.Message
	}
}

func resolveTenantRole(ctx context.Context, tenantID, sender string) (tenantName, userRole string) {
	tenantName, userRole = "UMKM WCH", "customer"
	if DB == nil {
		return
	}
	_ = DB.QueryRow(ctx, "SELECT name FROM tenants WHERE id = $1", tenantID).Scan(&tenantName)

	cleanSender := strings.Split(sender, "@")[0]
	rows, err := DB.Query(ctx, "SELECT phone_number, role FROM users WHERE tenant_id = $1", tenantID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var dbPhone, dbRole string
			if rows.Scan(&dbPhone, &dbRole) == nil {
				dbPhone = normalizePhone(dbPhone)
				if dbPhone == cleanSender {
					userRole = dbRole
					break
				}
			}
		}
	}
	_, _ = DB.Exec(ctx, "INSERT INTO tenant_contacts (tenant_id, phone_number) VALUES ($1, $2) ON CONFLICT (tenant_id, phone_number) DO NOTHING", tenantID, cleanSender)
	return
}

func handleGlobalWebhook(ctx context.Context, sender string) (tenantID string, role string) {
	if DB == nil {
		slog.Error("Database unavailable while processing async chat")
		return "", ""
	}
	err := DB.QueryRow(ctx, "SELECT tenant_id FROM users WHERE phone_number = $1 LIMIT 1", sender).Scan(&tenantID)
	if err != nil {
		slog.Warn("Unregistered phone number attempted to chat", "sender", sender)
		sendUnregisteredReply(ctx, sender)
		return "", ""
	}
	return tenantID, "customer"
}

func sendUnregisteredReply(ctx context.Context, sender string) {
	data := url.Values{}
	data.Set("target", sender)
	data.Set("message", "Mohon maaf, nomor WhatsApp Anda belum terdaftar sebagai pengguna sistem UMKM WCH. Silakan mendaftar melalui aplikasi web kami terlebih dahulu.")
	data.Set("tenant_id", "global")
	req, _ := http.NewRequestWithContext(ctx, "POST", waSendURL(), strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", headerContentTypeForm)
	if resp, err := http.DefaultClient.Do(req); err == nil {
		resp.Body.Close()
	}
}

func normalizePhone(phone string) string {
	if strings.HasPrefix(phone, "0") {
		phone = "62" + phone[1:]
	}
	return strings.TrimPrefix(phone, "+")
}

func sendOutsideHoursReply(ctx context.Context, tenantID, sender string, chatCfg *chatConfigCache) bool {
	withinHours, outsideMsg := isWithinBusinessHours(chatCfg)
	if withinHours {
		return true
	}
	if outsideMsg == "" {
		outsideMsg = "Terima kasih telah menghubungi kami. Saat ini di luar jam operasional. Pesan Anda akan dibalas saat jam kerja."
	}
	data := url.Values{}
	data.Set("target", sender)
	data.Set("message", outsideMsg)
	data.Set("tenant_id", tenantID)
	req, _ := http.NewRequestWithContext(ctx, "POST", waSendURL(), strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", headerContentTypeForm)
	if resp, err := http.DefaultClient.Do(req); err == nil {
		resp.Body.Close()
		if tenantID != "" {
			auth.IncrementQuota(ctx, tenantID, "chatbot_messages", 1)
		}
	}
	return false
}

func callAIAndSendReply(ctx context.Context, tenantID, sender, message, userRole, tenantName string, chatCfg *chatConfigCache) {
	systemPrompt := buildSystemPrompt(ctx, tenantID, tenantName, message, userRole, chatCfg)
	aiReqBody := map[string]any{
		"provider":   "Anthropic",
		"message":    message,
		"system_msg": systemPrompt,
		"tenant_id":  tenantID,
	}
	jsonBody, _ := json.Marshal(aiReqBody)
	aiReq, _ := http.NewRequestWithContext(ctx, "POST", AIGatewayURL, bytes.NewBuffer(jsonBody))
	aiReq.Header.Set("Content-Type", "application/json")
	aiReq.Header.Set("X-Tenant-ID", tenantID)

	aiResp, err := http.DefaultClient.Do(aiReq)
	if err != nil {
		slog.Error("Failed to contact AI Gateway in worker", "error", err)
		return
	}
	defer aiResp.Body.Close()

	var aiResult struct {
		Success bool   `json:"success"`
		Text    string `json:"text"`
	}
	json.NewDecoder(aiResp.Body).Decode(&aiResult)
	if !aiResult.Success || aiResult.Text == "" {
		return
	}

	if chatCfg != nil && chatCfg.FallbackMessage != "" && strings.Contains(aiResult.Text, chatCfg.FallbackMessage) {
		aiResult.Text = "[FORWARD_TO_ADMIN] " + aiResult.Text
	}
	finalText := processAIAnswer(ctx, tenantID, aiResult.Text, sender, userRole)
	sendWAMessage(ctx, tenantID, sender, finalText)
}

func sendWAMessage(ctx context.Context, tenantID, target, message string) {
	data := url.Values{}
	data.Set("target", target)
	data.Set("message", message)
	data.Set("tenant_id", tenantID)
	req, _ := http.NewRequestWithContext(ctx, "POST", waSendURL(), strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", headerContentTypeForm)
	if resp, err := http.DefaultClient.Do(req); err == nil {
		resp.Body.Close()
		if tenantID != "" {
			auth.IncrementQuota(ctx, tenantID, "chatbot_messages", 1)
		}
	}
}
