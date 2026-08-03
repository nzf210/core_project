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

	"core_project/shared/sdk/auth"
)

func activateSubscription(ctx context.Context, tenantID, planID, planName string, validityDays int, activatedBy string, voucherCodeID *string, systemVoucherCode string) string {
	now := time.Now()
	ticketNumber := generateTicketNumber()

	// F027 AC-4: Proration — preserve remaining days from previous active subscription
	var proratedDays int
	row := DB.QueryRow(ctx, `
		SELECT GREATEST(0,
			EXTRACT(EPOCH FROM (current_plan_expires_at - NOW())) / 86400
		)::INTEGER
		FROM tenant_subscriptions
		WHERE tenant_id = $1 AND status = 'active'
	`, tenantID)
	if err := row.Scan(&proratedDays); err == nil && proratedDays > 0 {
		validityDays += proratedDays
		slog.Info("prorated subscription", "tenant_id", tenantID, "prorated_days", proratedDays, "new_validity", validityDays)
	}

	// 1. Day-duration accumulator logic via voucher_subscriptions
	// If same plan already exists → accumulate remaining_days
	// If different plan → create new row (both co-exist; priority decides which is active)
	var newOrUpdatedVoucherSubID string
	err := DB.QueryRow(ctx, `
		INSERT INTO voucher_subscriptions (tenant_id, plan_id, validity_days, remaining_days, is_system_generated, source_voucher_code)
		VALUES ($1, $2, $3, $3, $4, $5)
		ON CONFLICT (tenant_id, plan_id) DO UPDATE SET
			validity_days = voucher_subscriptions.validity_days + EXCLUDED.validity_days,
			remaining_days = voucher_subscriptions.remaining_days + EXCLUDED.remaining_days,
			updated_at = NOW(),
			is_system_generated = COALESCE($4, voucher_subscriptions.is_system_generated),
			source_voucher_code = COALESCE($5, voucher_subscriptions.source_voucher_code)
		RETURNING id
	`, tenantID, planID, validityDays, systemVoucherCode != "", systemVoucherCode).Scan(&newOrUpdatedVoucherSubID)
	if err != nil {
		slog.Error("Failed to upsert voucher_subscription", "error", err)
	}

	// 2. Recalculate effective plan_priority and update tenants
	// Priority: business=4, pro=3, lite=2, free=1 (highest wins)
	var effectivePlanID string
	var maxPriority int
	err = DB.QueryRow(ctx, `
		SELECT vs.plan_id,
			   CASE vs.plan_id
				   WHEN 'ultimate' THEN 4 WHEN 'pro' THEN 3 WHEN 'lite' THEN 2
				   ELSE 1
			   END AS priority,
			   vs.remaining_days
		FROM voucher_subscriptions vs
		WHERE vs.tenant_id = $1 AND vs.remaining_days > 0
		ORDER BY priority DESC, remaining_days DESC
		LIMIT 1
	`, tenantID).Scan(&effectivePlanID, &maxPriority, new(int))
	if err != nil || effectivePlanID == "" {
		effectivePlanID = "inactive"
		maxPriority = 0
	}

	// 3. Upsert subscription record
	_, err = DB.Exec(ctx, `
		INSERT INTO tenant_subscriptions (tenant_id, plan_id, plan_tier, status, period_days, remaining_days, activated_by, voucher_code_id, system_voucher_code, updated_at)
		VALUES ($1, $2, $2, 'active', $3, $3, $4, $5, $6, NOW())
		ON CONFLICT (tenant_id)
		DO UPDATE SET
			plan_id = $2,
			plan_tier = $2,
			status = 'active',
			period_days = $3,
			remaining_days = $3,
			activated_by = $4,
			voucher_code_id = COALESCE($5, tenant_subscriptions.voucher_code_id),
			system_voucher_code = COALESCE($6, tenant_subscriptions.system_voucher_code),
			updated_at = NOW()
	`, tenantID, effectivePlanID, validityDays, activatedBy, voucherCodeID, systemVoucherCode)
	if err != nil {
		slog.Error("Failed to upsert subscription", "error", err)
	}

	// 4. Update tenant (un-freeze, set plan + priority)
	_, _ = DB.Exec(ctx, `
		UPDATE tenants SET
			plan = $1,
			plan_priority = $2,
			is_frozen = false,
			frozen_at = NULL,
			current_plan_expires_at = NOW() + (SELECT COALESCE(SUM(remaining_days), 0) || ' days'::interval FROM voucher_subscriptions WHERE tenant_id = $3)
		WHERE id = $3
	`, effectivePlanID, maxPriority, tenantID)

	// Sync Redis cache for quota gates
	auth.SetTenantPlan(ctx, tenantID, effectivePlanID)

	// 5. Create subscription ticket
	var ticketID string
	err = DB.QueryRow(ctx, `
		INSERT INTO subscription_tickets (tenant_id, plan_id, plan_name, ticket_number, expires_at, activated_by, notify_wa, notify_telegram, notify_email)
		VALUES ($1, $2, $3, $4, GREATEST(COALESCE((SELECT expires_at FROM subscription_tickets WHERE tenant_id = $1), NOW()), NOW()) + ($5 || ' days')::interval, $6, true, true, true)
		ON CONFLICT (tenant_id) DO UPDATE SET
			plan_id = EXCLUDED.plan_id,
			plan_name = EXCLUDED.plan_name,
			ticket_number = EXCLUDED.ticket_number,
			status = 'active',
			expires_at = GREATEST(COALESCE(subscription_tickets.expires_at, NOW()), NOW()) + ($5 || ' days')::interval,
			activated_at = NOW(),
			updated_at = NOW()
		RETURNING id
	`, tenantID, effectivePlanID, planName, ticketNumber, fmt.Sprintf("%d", validityDays), activatedBy).Scan(&ticketID)
	if err != nil {
		slog.Error("Failed to create ticket", "error", err)
		return ""
	}

	// 6. Update subscription with ticket ID
	_, _ = DB.Exec(ctx, "UPDATE tenant_subscriptions SET ticket_id = $1 WHERE tenant_id = $2", ticketID, tenantID)

	// 7. Async notification
	go sendTicketNotifications(tenantID, TicketPayload{
		TicketNumber:  ticketNumber,
		TenantName:    "",
		PlanName:      planName,
		PlanID:        planID,
		ActivatedAt:   now.Format(timeFormatWIB),
		ExpiresAt:     fmt.Sprintf("%d hari dari sekarang", validityDays),
		AmountPaid:    0,
		PaymentMethod: activatedBy,
		VoucherCode:   systemVoucherCode,
	})

	slog.Info("Subscription activated (day-duration)", "tenant_id", tenantID, "plan", effectivePlanID, "validity_days", validityDays, "ticket", ticketNumber, "system_voucher", systemVoucherCode)
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

	// Increment usage count
	DB.Exec(ctx, `UPDATE voucher_programs SET uses_count = uses_count + 1 WHERE id = $1`, programID)

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
		go sendTelegramNotification(telegramChatID, msg)
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

	waBase := Cfg.WhatsApp.GatewayURL
	if waBase == "" {
		waBase = "http://wa-gateway:8202"
	}
	waURL := waBase + "/api/wa/send"

	data := url.Values{}
	data.Set("tenant_id", tenantID)
	data.Set("target", targetJID)
	data.Set("message", message)

	req, err := http.NewRequest("POST", waURL, strings.NewReader(data.Encode()))
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

func sendTelegramNotification(chatID, message string) {
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
