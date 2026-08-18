package main

import (
	"context"
	"fmt"
	"log/slog"

	"core_project/shared/sdk/auth"
)

func calculateProratedDays(ctx context.Context, tenantID string) int {
	var proratedDays int
	row := DB.QueryRow(ctx, `
		SELECT GREATEST(0,
			EXTRACT(EPOCH FROM (current_plan_expires_at - NOW())) / 86400
		)::INTEGER
		FROM tenant_subscriptions
		WHERE tenant_id = $1 AND status = 'active'
	`, tenantID)
	if err := row.Scan(&proratedDays); err == nil && proratedDays > 0 {
		slog.Info("prorated subscription", "tenant_id", tenantID, "prorated_days", proratedDays)
		return proratedDays
	}
	return 0
}

func upsertVoucherSubscription(ctx context.Context, tenantID, planID string, validityDays int, systemVoucherCode string) error {
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
	return err
}

func calculateEffectivePlan(ctx context.Context, tenantID string) (planID string, priority int) {
	err := DB.QueryRow(ctx, `
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
	`, tenantID).Scan(&planID, &priority, new(int))
	if err != nil || planID == "" {
		return "inactive", 0
	}
	return planID, priority
}

func upsertTenantSubscription(ctx context.Context, tenantID, planID string, validityDays int, activatedBy string, voucherCodeID *string, systemVoucherCode string) error {
	_, err := DB.Exec(ctx, `
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
	`, tenantID, planID, validityDays, activatedBy, voucherCodeID, systemVoucherCode)
	if err != nil {
		slog.Error("Failed to upsert subscription", "error", err)
	}
	return err
}

func updateTenantPlanAndCache(ctx context.Context, tenantID, planID string, priority int) {
	DB.Exec(ctx, `
		UPDATE tenants SET
			plan = $1,
			plan_priority = $2,
			is_frozen = false,
			frozen_at = NULL,
			current_plan_expires_at = NOW() + (SELECT COALESCE(SUM(remaining_days), 0) || ' days'::interval FROM voucher_subscriptions WHERE tenant_id = $3)
		WHERE id = $3
	`, planID, priority, tenantID)
	auth.SetTenantPlan(ctx, tenantID, planID)
}

func createSubscriptionTicket(ctx context.Context, tenantID, planID, planName, ticketNumber string, validityDays int, activatedBy string) (string, error) {
	var ticketID string
	err := DB.QueryRow(ctx, `
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
	`, tenantID, planID, planName, ticketNumber, fmt.Sprintf("%d", validityDays), activatedBy).Scan(&ticketID)
	if err != nil {
		slog.Error("Failed to create ticket", "error", err)
		return "", err
	}
	return ticketID, nil
}

func linkTicketToSubscription(ctx context.Context, ticketID, tenantID string) {
	DB.Exec(ctx, "UPDATE tenant_subscriptions SET ticket_id = $1 WHERE tenant_id = $2", ticketID, tenantID)
}
