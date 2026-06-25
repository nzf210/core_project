package main

import (
	"context"
	"core_project/shared/sdk/auth"
	"core_project/shared/sdk/config"
	"core_project/shared/sdk/response"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type VoucherLinkRedeemReq struct {
	Token    string `json:"token"`
	TenantID string `json:"tenant_id"`
}

type VoucherLinkRedeemResp struct {
	PlanID         string `json:"plan_id"`
	PlanName       string `json:"plan_name"`
	ActivatedAt    string `json:"activated_at"`
	ExpiresAt      string `json:"expires_at"`
	DurationMonths int    `json:"duration_months"`
	TicketNumber   string `json:"ticket_number"`
}

func signVoucherToken(programID, planID string, durationMonths int, expiresAt time.Time) (string, error) {
	cfg := config.GlobalConfig
	secret := []byte(cfg.JWTSecret)
	claims := jwt.MapClaims{
		"program_id":      programID,
		"plan_id":         planID,
		"duration_months": durationMonths,
		"exp":             expiresAt.Unix(),
		"iat":             time.Now().Unix(),
		"jti":             uuid.NewString(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString(secret)
}
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

type voucherLinkClaims struct {
	ProgramID      string
	PlanID         string
	DurationMonths int
}

func parseVoucherLinkToken(token string, secret string) (voucherLinkClaims, error) {
	var result voucherLinkClaims
	parsed, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil || !parsed.Valid {
		return result, fmt.Errorf("invalid or expired voucher link")
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return result, fmt.Errorf("invalid claims")
	}
	programID, _ := claims["program_id"].(string)
	planID, _ := claims["plan_id"].(string)
	durationMonthsF, _ := claims["duration_months"].(float64)
	durationMonths := int(durationMonthsF)
	if programID == "" || planID == "" {
		return result, fmt.Errorf("malformed voucher link")
	}
	result.ProgramID = programID
	result.PlanID = planID
	result.DurationMonths = durationMonths
	return result, nil
}

type voucherLinkDB struct {
	ID         string
	ProgramID  string
	ExpiresAt  time.Time
	RedeemedBy *string
	RedeemedAt *time.Time
	IsActive   bool
}

func lookupVoucherLink(ctx context.Context, tokenHash string) (voucherLinkDB, error) {
	var link voucherLinkDB
	err := DB.QueryRow(ctx, `
		SELECT id, program_id, expires_at, redeemed_by, redeemed_at, is_active
		FROM voucher_links WHERE token_hash = $1
	`, tokenHash).Scan(&link.ID, &link.ProgramID, &link.ExpiresAt, &link.RedeemedBy, &link.RedeemedAt, &link.IsActive)
	return link, err
}
func validateVoucherLink(link voucherLinkDB, programID string) error {
	if !link.IsActive {
		return fmt.Errorf("voucher link has been deactivated")
	}
	if link.RedeemedBy != nil {
		return fmt.Errorf("voucher link already redeemed")
	}
	if time.Now().After(link.ExpiresAt) {
		return fmt.Errorf("voucher link has expired")
	}
	if link.ProgramID != programID {
		return fmt.Errorf("token/program mismatch")
	}
	return nil
}

type voucherProgramInfo struct {
	PlanName         string
	ProgramDuration  int
	MaxUsesPerTenant int
}

func lookupVoucherProgramInfo(ctx context.Context, planID, programID string) (voucherProgramInfo, error) {
	var info voucherProgramInfo
	err := DB.QueryRow(ctx, `
		SELECT sp.name, COALESCE(vp.duration_months, 1), COALESCE(vp.max_uses_per_tenant, 1)
		FROM voucher_programs vp
		JOIN saas_plans sp ON sp.id = $1
		WHERE vp.id = $2 AND vp.is_active = true
	`, planID, programID).Scan(&info.PlanName, &info.ProgramDuration, &info.MaxUsesPerTenant)
	return info, err
}
func checkVoucherUsageQuota(ctx context.Context, programID, tenantID string, maxUsesPerTenant int) error {
	var usesByTenant int
	err := DB.QueryRow(ctx, `
		SELECT COUNT(*) FROM voucher_links
		WHERE program_id = $1 AND redeemed_by = $2
	`, programID, tenantID).Scan(&usesByTenant)
	if err != nil {
		return err
	}
	if usesByTenant >= maxUsesPerTenant {
		return fmt.Errorf("Voucher quota per tenant exceeded")
	}
	return nil
}
func processVoucherRedemptionTx(ctx context.Context, req VoucherLinkRedeemReq, linkDB voucherLinkDB, claims *voucherLinkClaims, progInfo voucherProgramInfo, durationMonths int, r *http.Request) (time.Time, string, error) {
	tx, err := DB.Begin(ctx)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("failed to start tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		UPDATE voucher_links
		SET redeemed_by = $1, redeemed_at = NOW(), is_active = false, ip_address = $2, user_agent = $3
		WHERE id = $4 AND is_active = true AND redeemed_by IS NULL
	`, req.TenantID, r.RemoteAddr, r.UserAgent(), linkDB.ID)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("failed to redeem link: %w", err)
	}

	_, _ = tx.Exec(ctx, `UPDATE voucher_programs SET uses_count = uses_count + 1 WHERE id = $1`, claims.ProgramID)

	var existingPeriodEnd *time.Time
	var existingStatus string
	_ = tx.QueryRow(ctx, `SELECT current_period_end, status FROM tenant_subscriptions WHERE tenant_id = $1`, req.TenantID).Scan(&existingPeriodEnd, &existingStatus)

	now := time.Now()
	var newPeriodEnd time.Time
	if existingPeriodEnd != nil && existingStatus == "active" && existingPeriodEnd.After(now) {
		newPeriodEnd = existingPeriodEnd.AddDate(0, durationMonths, 0)
	} else {
		newPeriodEnd = now.AddDate(0, durationMonths, 0)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO tenant_subscriptions (tenant_id, plan_id, plan_tier, status, current_period_end, period_days, activated_by, updated_at)
		VALUES ($1, $2, $3, 'active', $4, $5, 'voucher', NOW())
		ON CONFLICT (tenant_id)
		DO UPDATE SET
			plan_id = EXCLUDED.plan_id,
			plan_tier = EXCLUDED.plan_tier,
			status = 'active',
			current_period_end = EXCLUDED.current_period_end,
			period_days = EXCLUDED.period_days,
			frozen_at = NULL,
			frozen_reason = NULL,
			updated_at = NOW()
	`, req.TenantID, claims.PlanID, claims.PlanID, newPeriodEnd, durationMonths*30)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("failed to update subscription: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE tenants SET plan = $1, is_frozen = false, frozen_at = NULL, current_plan_expires_at = $2
		WHERE id = $3
	`, claims.PlanID, newPeriodEnd, req.TenantID)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("failed to update tenant: %w", err)
	}

	ticketNumber := generateTicketNumber()
	var ticketID string
	err = tx.QueryRow(ctx, `
		INSERT INTO subscription_tickets (tenant_id, plan_id, plan_name, ticket_number, expires_at, activated_by, notify_wa, notify_telegram, notify_email)
		VALUES ($1, $2, $3, $4, $5, 'voucher', true, true, true)
		ON CONFLICT (tenant_id) DO UPDATE SET
			plan_id = EXCLUDED.plan_id,
			plan_name = EXCLUDED.plan_name,
			ticket_number = EXCLUDED.ticket_number,
			status = 'active',
			expires_at = EXCLUDED.expires_at,
			activated_at = NOW(),
			updated_at = NOW()
		RETURNING id
	`, req.TenantID, claims.PlanID, progInfo.PlanName, ticketNumber, newPeriodEnd).Scan(&ticketID)
	if err != nil {
		slog.Warn("Failed to create ticket", "error", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return time.Time{}, "", fmt.Errorf("failed to commit: %w", err)
	}
	return newPeriodEnd, ticketNumber, nil
}
func handleRedeemVoucherLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	var req VoucherLinkRedeemReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.InvalidRequest, err)
		return
	}
	if req.Token == "" || req.TenantID == "" {
		response.Error(w, http.StatusBadRequest, "token and tenant_id are required", nil)
		return
	}

	// 1. Verify JWT signature & extract claims
	claims, err := parseVoucherLinkToken(req.Token, config.GlobalConfig.JWTSecret)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	ctx := r.Context()
	tokenHash := hashToken(req.Token)

	// 2. Lookup link in DB
	linkDB, err := lookupVoucherLink(ctx, tokenHash)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Voucher link not found", nil)
		return
	}
	if err := validateVoucherLink(linkDB, claims.ProgramID); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// 3. Lookup program & plan
	progInfo, err := lookupVoucherProgramInfo(ctx, claims.PlanID, claims.ProgramID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Program inactive or not found", nil)
		return
	}

	durationMonths := claims.DurationMonths
	if durationMonths == 0 {
		durationMonths = progInfo.ProgramDuration
	}

	// 4. Check max_uses_per_tenant (default 1)
	if err := checkVoucherUsageQuota(ctx, claims.ProgramID, req.TenantID, progInfo.MaxUsesPerTenant); err != nil {
		w.Header().Set(response.ContentType, response.ApplicationJSON)
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Voucher quota per tenant exceeded",
			"data":    map[string]interface{}{"max_uses_per_tenant": progInfo.MaxUsesPerTenant},
		})
		return
	}

	// 5. Begin tx: mark link redeemed, activate/extend subscription
	newPeriodEnd, ticketNumber, err := processVoucherRedemptionTx(ctx, req, linkDB, &claims, progInfo, durationMonths, r)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	now := time.Now()

	// Sync Redis cache so quota gates read the correct plan tier.
	auth.SetTenantPlan(ctx, req.TenantID, claims.PlanID)

	// Async notification
	go sendTicketNotifications(req.TenantID, TicketPayload{
		TicketNumber:  ticketNumber,
		PlanName:      progInfo.PlanName,
		PlanID:        claims.PlanID,
		ActivatedAt:   now.Format(timeFormatWIB),
		ExpiresAt:     newPeriodEnd.Format(timeFormatWIB),
		AmountPaid:    0,
		PaymentMethod: "voucher",
	})

	slog.Info("Voucher link redeemed", "tenant_id", req.TenantID, "plan", claims.PlanID, "duration_months", durationMonths, "new_expires", newPeriodEnd)

	response.JSON(w, http.StatusOK, "Voucher redeemed successfully", VoucherLinkRedeemResp{
		PlanID:         claims.PlanID,
		PlanName:       progInfo.PlanName,
		ActivatedAt:    now.Format(time.RFC3339),
		ExpiresAt:      newPeriodEnd.Format(time.RFC3339),
		DurationMonths: durationMonths,
		TicketNumber:   ticketNumber,
	})
}
