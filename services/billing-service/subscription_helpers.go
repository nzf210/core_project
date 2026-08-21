package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"core_project/shared/sdk/queue"
	"github.com/google/uuid"
)
type voucherActivationOpts struct {
	VoucherCodeID     *string
	SystemVoucherCode string
}

func activateSubscription(ctx context.Context, tenantID, planID, planName string, validityDays int, activatedBy string, voucher voucherActivationOpts) string {
	now := time.Now()
	ticketNumber := generateTicketNumber()

	proratedDays := calculateProratedDays(ctx, tenantID)
	validityDays += proratedDays

	upsertVoucherSubscription(ctx, tenantID, planID, validityDays, voucher.SystemVoucherCode)

	effectivePlanID, maxPriority := calculateEffectivePlan(ctx, tenantID)

	upsertTenantSubscription(ctx, tenantID, effectivePlanID, validityDays, activatedBy, voucher.VoucherCodeID, voucher.SystemVoucherCode)

	updateTenantPlanAndCache(ctx, tenantID, effectivePlanID, maxPriority)

	ticketID, err := createSubscriptionTicket(ctx, tenantID, effectivePlanID, planName, ticketNumber, validityDays, activatedBy)
	if err != nil {
		return ""
	}

	linkTicketToSubscription(ctx, ticketID, tenantID)

	go sendTicketNotifications(tenantID, TicketPayload{
		TicketNumber:  ticketNumber,
		TenantName:    "",
		PlanName:      planName,
		PlanID:        planID,
		ActivatedAt:   now.Format(timeFormatWIB),
		ExpiresAt:     fmt.Sprintf("%d hari dari sekarang", validityDays),
		AmountPaid:    0,
		PaymentMethod: activatedBy,
		VoucherCode:   voucher.SystemVoucherCode,
	})

	slog.Info("Subscription activated (day-duration)", "tenant_id", tenantID, "plan", effectivePlanID, "validity_days", validityDays, "ticket", ticketNumber, "system_voucher", voucher.SystemVoucherCode)
	return ticketID
}

// ─────────────────────────────────────────────
// Voucher Application
// ─────────────────────────────────────────────

// validateVoucherOnly checks if a voucher code is valid without redeeming it.

func validateVoucherOnly(ctx context.Context, code, planID string) bool {
	var id string
	err := DB.QueryRow(ctx, `
		SELECT vc.code FROM voucher_codes vc
		JOIN voucher_programs vp ON vc.program_id = vp.id
		WHERE vc.code = $1 AND vc.is_redeemed = false
		  AND vp.is_active = true
		  AND (vp.expires_at IS NULL OR vp.expires_at > NOW())
		  AND (vp.target_plan_id IS NULL OR vp.target_plan_id = '' OR vp.target_plan_id = $2)
	`, code, planID).Scan(&id)
	return err == nil
}

func applyVoucher(ctx context.Context, code, planID, tenantID string) (bool, *string) {
	// Atomically mark voucher as redeemed and fetch its program rules
	var programID string
	var targetPlanID *string

	err := DB.QueryRow(ctx, `
		WITH valid_voucher AS (
			SELECT vc.code, vp.id as program_id, vp.target_plan_id
			FROM voucher_codes vc
			JOIN voucher_programs vp ON vc.program_id = vp.id
			WHERE vc.code = $1 AND vc.is_redeemed = false
			  AND vp.is_active = true
			  AND (vp.expires_at IS NULL OR vp.expires_at > NOW())
			  AND (vp.target_plan_id IS NULL OR vp.target_plan_id = '' OR vp.target_plan_id = $3)
		),
		updated_voucher AS (
			UPDATE voucher_codes
			SET is_redeemed = true, used_by = $2, used_at = NOW()
			FROM valid_voucher
			WHERE voucher_codes.code = valid_voucher.code
			RETURNING valid_voucher.program_id, valid_voucher.target_plan_id
		)
		SELECT program_id, target_plan_id FROM updated_voucher
	`, code, tenantID, planID).Scan(&programID, &targetPlanID)

	if err != nil { // No rows = invalid, expired, or already used (Race Condition prevented)
		return false, nil
	}

	incrUsageSQL := `UPDATE voucher_programs SET uses_count = uses_count + 1 WHERE id = $1`
	DB.Exec(ctx, incrUsageSQL, programID)

	return true, nil
}

// ─────────────────────────────────────────────
// Notification Dispatcher
// ─────────────────────────────────────────────

func sendTicketNotifications(tenantID string, payload TicketPayload) {
	// Get tenant contact info
	// Note: TEST_* env vars are intentional test overrides, not regular config
	waNumber := os.Getenv("TEST_WA_NUMBER")
	telegramChatID := os.Getenv("TEST_TELEGRAM_CHAT_ID")
	email := os.Getenv("TEST_EMAIL")

	// Try to fetch from DB
	var tenantName string
	DB.QueryRow(context.Background(), "SELECT name FROM tenants WHERE id = $1", tenantID).Scan(&tenantName)
	if tenantName == "" {
		tenantName = "Pelanggan"
	}
	payload.TenantName = tenantName

	// Get notification preferences
	var notifyWA, notifyTelegram, notifyEmail bool
	DB.QueryRow(context.Background(), `
		SELECT COALESCE(notify_wa, true), COALESCE(notify_telegram, false), COALESCE(notify_email, true)
		FROM tenant_notification_settings WHERE tenant_id = $1
	`, tenantID).Scan(&notifyWA, &notifyTelegram, &notifyEmail)

	if notifyWA && waNumber != "" {
		msg := buildTicketMessage(payload)
		sendWANotification(tenantID, waNumber, msg)
		DB.Exec(context.Background(), `
			UPDATE subscription_tickets SET wa_sent_at = NOW() WHERE ticket_number = $1
		`, payload.TicketNumber)
	}

	if notifyTelegram && telegramChatID != "" {
		msg := buildTicketMessage(payload)
		go sendTelegramNotification(tenantID, telegramChatID, msg)
		DB.Exec(context.Background(), `
			UPDATE subscription_tickets SET telegram_sent_at = NOW() WHERE ticket_number = $1
		`, payload.TicketNumber)
	}

	if notifyEmail && email != "" {
		go sendEmailNotification(email, payload)
		DB.Exec(context.Background(), `
			UPDATE subscription_tickets SET email_sent_at = NOW() WHERE ticket_number = $1
		`, payload.TicketNumber)
	}
}

func buildTicketMessage(p TicketPayload) string {
	return fmt.Sprintf(`🎫 *Tiket Langganan WCH Platform*

Halo %s!

Langganan kamu telah aktif:

📋 *Detail Tiket:*
• Nomor Tiket: %s
• Paket: %s
• Status: ✅ Aktif
• Aktivasi: %s
• Kadaluarsa: %s

Terima kasih sudah percaya WCH Platform! 🚀`, p.TenantName, p.TicketNumber, p.PlanName, p.ActivatedAt, p.ExpiresAt)
}

// ─────────────────────────────────────────────
// Send WA Notification
// ─────────────────────────────────────────────

func sendWANotification(tenantID, target, message string) {
	targetJID := target
	if !strings.Contains(targetJID, "@s.whatsapp.net") {
		if strings.HasPrefix(targetJID, "0") {
			targetJID = "62" + targetJID[1:]
		}
		targetJID = strings.TrimPrefix(targetJID, "+")
		targetJID = targetJID + "@s.whatsapp.net"
	}

	// Publish async via RabbitMQ if available; fall back to direct HTTP.
	if MQ != nil {
		job := buildNotifJob(tenantID, "notifications.wa", map[string]interface{}{
			"target":  targetJID,
			"message": message,
		})
		if err := MQ.Publish(context.Background(), "notifications.wa", job); err != nil {
			slog.Warn("RabbitMQ publish failed, falling back to direct HTTP", "error", err)
		} else {
			slog.Info("WA notification queued", "tenant_id", tenantID, "target", targetJID)
			return
		}
	}

	waBase := Cfg.WhatsApp.GatewayURL
	if waBase == "" {
		waBase = "http://wa-gateway:8202"
	}
	waURL := waBase + "/api/wa/send"

	data := url.Values{}
	data.Set("tenant_id", tenantID)
	data.Set("target", targetJID)
	data.Set("message", message)

	req, err := http.NewRequestWithContext(context.Background(), "POST", waURL, strings.NewReader(data.Encode()))
	if err != nil {
		slog.Error("WA notification: failed to create request", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Message-Type", "invoice")
	req.Header.Set("X-Source", "billing-service")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("WA notification failed", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		slog.Info("WA ticket notification sent", "tenant_id", tenantID, "target", targetJID)
	}
}

// ─────────────────────────────────────────────
// Send Telegram Notification
// ─────────────────────────────────────────────

func sendTelegramNotification(tenantID, chatID, message string) {
	// Publish async via RabbitMQ if available; fall back to direct Telegram API.
	if MQ != nil {
		job := buildNotifJob(tenantID, "notifications.telegram", map[string]interface{}{
			"target":  chatID,
			"message": message,
		})
		if err := MQ.Publish(context.Background(), "notifications.telegram", job); err != nil {
			slog.Warn("RabbitMQ publish failed, falling back to direct Telegram API", "error", err)
		} else {
			slog.Info("Telegram notification queued", "tenant_id", tenantID, "chat_id", chatID)
			return
		}
	}

	botToken := Cfg.Telegram.BotToken
	if botToken == "" {
		slog.Warn("TELEGRAM_BOT_TOKEN not set, skipping")
		return
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	payload := map[string]interface{}{
		"chat_id": chatID,
		"text":    message,
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		slog.Error("Telegram notification failed", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		slog.Info("Telegram ticket notification sent", "chat_id", chatID)
	}
}

// ─────────────────────────────────────────────
// Send Email Notification
// ─────────────────────────────────────────────

func sendEmailNotification(email string, payload TicketPayload) {
	// Simple SMTP email via notification-service
	notifURL := "http://localhost:8005/api/notification/send"
	if Cfg.Env == "production" || Cfg.DB.Host == "postgres" {
		notifURL = "http://notification-service:8005/api/notification/send"
	}

	bodyBytes, _ := json.Marshal(map[string]interface{}{
		"type":    "email",
		"target":  email,
		"message": buildTicketMessage(payload),
		"subject": fmt.Sprintf("Tiket Langganan WCH Platform — %s", payload.TicketNumber),
	})

	resp, err := http.Post(notifURL, "application/json", bytes.NewBuffer(bodyBytes))
	if err != nil {
		slog.Error("Email notification failed", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		slog.Info("Email ticket notification sent", "email", email)
	}
}

// ─────────────────────────────────────────────
// Helpers

// buildNotifJob creates a queue.Job for notification queues.
func buildNotifJob(tenantID, jobType string, data map[string]interface{}) queue.Job {
	return queue.Job{
		JobID:     uuid.New().String(),
		TenantID:  tenantID,
		Type:      jobType,
		Data:      data,
		CreatedAt: time.Now(),
	}
}
