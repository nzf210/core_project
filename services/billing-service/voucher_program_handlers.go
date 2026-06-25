package main

import (
	"encoding/json"
	"net/http"
	"time"

	"core_project/shared/sdk/response"
)

type CreateVoucherProgramReq struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	VoucherType    string `json:"voucher_type"`
	DiscountValue  int    `json:"discount_value"`
	TargetPlanID   string `json:"target_plan_id"`
	DurationMonths int    `json:"duration_months"`
	MaxUses        int    `json:"max_uses"`
	StartsAt       string `json:"starts_at"`
	ExpiresAt      string `json:"expires_at"`
}

func handleAdminVoucherProgramsCollection(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get(response.XUserRole)
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, response.SuperadminOnly, nil)
		return
	}

	switch r.Method {
	case http.MethodGet:
		listVoucherPrograms(w, r)
	case http.MethodPost:
		createVoucherProgram(w, r)
	default:
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
	}
}

func listVoucherPrograms(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rows, err := DB.Query(ctx, `
		SELECT id, name, description, voucher_type, discount_value, COALESCE(target_plan_id, ''), duration_months, max_uses, uses_count, starts_at, expires_at, is_active
		FROM voucher_programs ORDER BY created_at DESC
	`)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to list voucher programs", err)
		return
	}
	defer rows.Close()

	type programRow struct {
		ID             string     `json:"id"`
		Name           string     `json:"name"`
		Description    string     `json:"description"`
		VoucherType    string     `json:"voucher_type"`
		DiscountValue  int        `json:"discount_value"`
		TargetPlanID   string     `json:"target_plan_id"`
		DurationMonths int        `json:"duration_months"`
		MaxUses        int        `json:"max_uses"`
		UsesCount      int        `json:"uses_count"`
		StartsAt       time.Time  `json:"starts_at"`
		ExpiresAt      *time.Time `json:"expires_at"`
		IsActive       bool       `json:"is_active"`
	}

	var programs []programRow
	for rows.Next() {
		var p programRow
		err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.VoucherType, &p.DiscountValue,
			&p.TargetPlanID, &p.DurationMonths, &p.MaxUses, &p.UsesCount,
			&p.StartsAt, &p.ExpiresAt, &p.IsActive,
		)
		if err == nil {
			programs = append(programs, p)
		}
	}

	response.JSON(w, http.StatusOK, "Voucher programs retrieved", programs)
}

func createVoucherProgram(w http.ResponseWriter, r *http.Request) {
	var req CreateVoucherProgramReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.InvalidRequest, err)
		return
	}

	if req.Name == "" || req.VoucherType == "" {
		response.Error(w, http.StatusBadRequest, "name and voucher_type are required", nil)
		return
	}

	startsAt := time.Now()
	if req.StartsAt != "" {
		if t, err := time.Parse(time.RFC3339, req.StartsAt); err == nil {
			startsAt = t
		}
	}

	var expiresAt *time.Time
	if req.ExpiresAt != "" {
		if t, err := time.Parse(time.RFC3339, req.ExpiresAt); err == nil {
			expiresAt = &t
		}
	}

	ctx := r.Context()
	var id string
	err := DB.QueryRow(ctx, `
		INSERT INTO voucher_programs (name, description, voucher_type, discount_value, target_plan_id, duration_months, max_uses, starts_at, expires_at, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, true)
		RETURNING id
	`, req.Name, req.Description, req.VoucherType, req.DiscountValue, req.TargetPlanID, req.DurationMonths, req.MaxUses, startsAt, expiresAt).Scan(&id)

	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to create voucher program", err)
		return
	}

	response.JSON(w, http.StatusOK, "Voucher program created", map[string]interface{}{
		"id": id,
	})
}

func handleAdminVoucherAnalytics(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get(response.XUserRole)
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, response.SuperadminOnly, nil)
		return
	}

	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	ctx := r.Context()
	programID := r.URL.Query().Get("program_id")

	var (
		totalGenerated int
		totalRedeemed  int
	)

	if programID != "" {
		err := DB.QueryRow(ctx, `
			SELECT
				COUNT(*),
				COUNT(CASE WHEN redeemed_by IS NOT NULL THEN 1 END)
			FROM voucher_links
			WHERE program_id = $1
		`, programID).Scan(&totalGenerated, &totalRedeemed)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to fetch program analytics", err)
			return
		}

		response.JSON(w, http.StatusOK, "Program analytics retrieved", map[string]interface{}{
			"program_id":              programID,
			"total_links_generated":   totalGenerated,
			"total_links_redeemed":    totalRedeemed,
			"redemption_rate_percent": calculateRate(totalGenerated, totalRedeemed),
		})
	} else {
		var totalPrograms, activePrograms int
		err := DB.QueryRow(ctx, `
			SELECT
				COUNT(*),
				COUNT(CASE WHEN is_active = true THEN 1 END)
			FROM voucher_programs
		`).Scan(&totalPrograms, &activePrograms)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to fetch global program stats", err)
			return
		}

		err = DB.QueryRow(ctx, `
			SELECT
				COUNT(*),
				COUNT(CASE WHEN redeemed_by IS NOT NULL THEN 1 END)
			FROM voucher_links
		`).Scan(&totalGenerated, &totalRedeemed)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to fetch global links stats", err)
			return
		}

		response.JSON(w, http.StatusOK, "Global voucher analytics retrieved", map[string]interface{}{
			"total_programs":          totalPrograms,
			"active_programs":         activePrograms,
			"total_links_generated":   totalGenerated,
			"total_links_redeemed":    totalRedeemed,
			"redemption_rate_percent": calculateRate(totalGenerated, totalRedeemed),
		})
	}
}

func calculateRate(total, redeemed int) float64 {
	if total == 0 {
		return 0.0
	}
	return float64(redeemed) / float64(total) * 100.0
}
