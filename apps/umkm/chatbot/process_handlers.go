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

// processChatJob handles the heavy logic asynchronously
func processChatJob(job ChatJob) {
	ctx := context.Background()

	// F029: Dynamic Multimodal Guardrails (Feature Toggles based on Plan limits)
	plan := auth.GetPlan(job.TenantID)

	// Handle Multimedia First
	if job.MsgType == "audio" && job.MediaPath != "" {
		if plan.MaxAIAudioMinutes == 0 {
			// Feature disabled
			job.Message = "[Sistem] Maaf, layanan pesan suara (Voice Note) belum diaktifkan oleh Toko ini. Harap ketik pesan Anda."
		} else {
			text, err := transcribeAudio(job.TenantID, job.MediaPath)
			if err == nil {
				job.Message = text // replace voice note with transcribed text
			} else {
				job.Message = "[Pesan Suara tidak dapat diproses]"
			}
		}
	} else if job.MsgType == "image" && job.MediaPath != "" {
		if plan.MaxAIVision == 0 {
			// Feature disabled
			if job.Message != "" {
				job.Message = job.Message + "\n[Sistem] Maaf, layanan analisa gambar belum diaktifkan oleh Toko ini. Harap ketik pertanyaan Anda secara detail."
			} else {
				job.Message = "[Sistem] Maaf, layanan analisa gambar belum diaktifkan oleh Toko ini. Harap ketik pertanyaan Anda secara detail."
			}
		} else {
			text, err := analyzeImage(job.TenantID, job.MediaPath, job.Message)
			if err == nil {
				if job.Message != "" {
					job.Message = job.Message + "\n[Analisis Gambar: " + text + "]"
				} else {
					job.Message = "[Analisis Gambar: " + text + "]"
				}
			} else {
				job.Message = job.Message + "\n[Gambar tidak dapat diproses]"
			}
		}
	}

	sender := job.Sender
	message := job.Message
	tenantID := job.TenantID

	userRole := "customer"
	tenantName := "UMKM WCH"

	if tenantID != "" {
		if DB != nil {
			// Fetch tenant name
			err := DB.QueryRow(ctx, "SELECT name FROM tenants WHERE id = $1", tenantID).Scan(&tenantName)
			if err != nil {
				slog.Warn("Failed to get tenant name", "error", err)
			}

			// Check user role
			cleanSender := strings.Split(sender, "@")[0]
			rows, err := DB.Query(ctx, "SELECT phone_number, role FROM users WHERE tenant_id = $1", tenantID)
			if err == nil {
				for rows.Next() {
					var dbPhone, dbRole string
					if err := rows.Scan(&dbPhone, &dbRole); err == nil {
						if strings.HasPrefix(dbPhone, "0") {
							dbPhone = "62" + dbPhone[1:]
						}
						dbPhone = strings.TrimPrefix(dbPhone, "+")
						if dbPhone == cleanSender {
							userRole = dbRole
							break
						}
					}
				}
				rows.Close()
			}

			// Auto-save contact
			_, errSave := DB.Exec(ctx, "INSERT INTO tenant_contacts (tenant_id, phone_number) VALUES ($1, $2) ON CONFLICT (tenant_id, phone_number) DO NOTHING", tenantID, cleanSender)
			if errSave != nil {
				slog.Warn("Failed to auto-save contact", "error", errSave, "phone", cleanSender)
			}
		}
	} else {
		// Global webhook (Tenant owner chatting with the central bot)
		if DB != nil {
			err := DB.QueryRow(ctx, "SELECT tenant_id FROM users WHERE phone_number = $1 LIMIT 1", sender).Scan(&tenantID)
			if err != nil {
				slog.Warn("Unregistered phone number attempted to chat", "sender", sender)

				// Auto-reply to unregistered user via WA Gateway
				waGatewayURL := waSendURL()
				data := url.Values{}
				data.Set("target", sender)
				data.Set("message", "Mohon maaf, nomor WhatsApp Anda belum terdaftar sebagai pengguna sistem UMKM WCH. Silakan mendaftar melalui aplikasi web kami terlebih dahulu.")
				data.Set("tenant_id", "global")

				reqWA, _ := http.NewRequestWithContext(ctx, "POST", waGatewayURL, strings.NewReader(data.Encode()))
				reqWA.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				respWA, errF := http.DefaultClient.Do(reqWA)
				if errF == nil {
					respWA.Body.Close()
				}
				return
			}
		} else {
			// DB is down
			slog.Error("Database unavailable while processing async chat")
			return
		}
	}

	// 2. Load per-tenant chatbot config (cached) and enforce business hours.
	chatCfg := loadChatbotConfig(ctx, tenantID)
	withinHours, outsideMsg := isWithinBusinessHours(chatCfg)
	if !withinHours {
		// Skip LLM call to save cost; reply with outside-hours message
		if outsideMsg == "" {
			outsideMsg = "Terima kasih telah menghubungi kami. Saat ini di luar jam operasional. Pesan Anda akan dibalas saat jam kerja."
		}
		waGatewayURL := waSendURL()
		data := url.Values{}
		data.Set("target", sender)
		data.Set("message", outsideMsg)
		data.Set("tenant_id", tenantID)
		reqWA, _ := http.NewRequestWithContext(ctx, "POST", waGatewayURL, strings.NewReader(data.Encode()))
		reqWA.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if respWA, errWA := http.DefaultClient.Do(reqWA); errWA == nil {
			respWA.Body.Close()
			if tenantID != "" {
				auth.IncrementQuota(ctx, tenantID, "chatbot_messages", 1)
			}
		}
		return
	}

	// 3. Call AI Gateway (system prompt is built from base + per-tenant overrides)
	systemPrompt := buildSystemPrompt(ctx, tenantID, tenantName, message, userRole, chatCfg)
	aiReqBody := map[string]interface{}{
		"provider":   "minimax",
		"message":    message,
		"system_msg": systemPrompt,
		"tenant_id":  tenantID,
	}
	jsonBody, _ := json.Marshal(aiReqBody)
	aiReqHTTP, _ := http.NewRequestWithContext(ctx, "POST", AIGatewayURL, bytes.NewBuffer(jsonBody))
	aiReqHTTP.Header.Set("Content-Type", "application/json")
	aiReqHTTP.Header.Set("X-Tenant-ID", tenantID)

	aiRespHTTP, err := http.DefaultClient.Do(aiReqHTTP)
	if err == nil {
		defer aiRespHTTP.Body.Close()
		var aiGatewayResp struct {
			Success bool   `json:"success"`
			Text    string `json:"text"`
		}
		json.NewDecoder(aiRespHTTP.Body).Decode(&aiGatewayResp)

		if aiGatewayResp.Success && aiGatewayResp.Text != "" {
			if chatCfg.FallbackMessage != "" && strings.Contains(aiGatewayResp.Text, chatCfg.FallbackMessage) {
				aiGatewayResp.Text = "[FORWARD_TO_ADMIN] " + aiGatewayResp.Text
			}
			finalText := processAIAnswer(ctx, tenantID, aiGatewayResp.Text, sender, userRole)
			// 3. Post reply back to WA Gateway API
			waGatewayURL := waSendURL()
			data := url.Values{}
			data.Set("target", sender)
			data.Set("message", finalText)
			data.Set("tenant_id", tenantID)

			reqWA, _ := http.NewRequestWithContext(ctx, "POST", waGatewayURL, strings.NewReader(data.Encode()))
			reqWA.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			respWA, errWA := http.DefaultClient.Do(reqWA)
			if errWA == nil {
				respWA.Body.Close()
				if tenantID != "" {
					auth.IncrementQuota(ctx, tenantID, "chatbot_messages", 1)
				}
			} else {
				slog.Error("Failed to send WA reply", "error", errWA)
			}
		}
	} else {
		slog.Error("Failed to contact AI Gateway in worker", "error", err)
	}
}