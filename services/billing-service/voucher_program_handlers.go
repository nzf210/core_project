package main

import (
	"encoding/json"
	"net/http"
	"strings"
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

type UpdateVoucherProgramReq struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	VoucherType    string `json:"voucher_type"`
	DiscountValue  int    `json:"discount_value"`
	TargetPlanID   string `json:"target_plan_id"`
	DurationMonths int    `json:"duration_months"`
	MaxUses        int    `json:"max_uses"`
	StartsAt       string `json:"starts_at"`
	ExpiresAt      string `json:"expires_at"`
	IsActive       *bool  `json:"is_active"`
}

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

func handleAdminVoucherProgramsItem(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get(response.XUserRole)
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, response.SuperadminOnly, nil)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/admin/voucher-programs/")
	id = strings.TrimSpace(id)
	if id == "" || id == "/admin/voucher-programs" {
		response.Error(w, http.StatusBadRequest, "Missing voucher program ID", nil)
		return
	}

	switch r.Method {
	case http.MethodGet:
		getVoucherProgramItem(w, r, id)
	case http.MethodPut:
		updateVoucherProgramItem(w, r, id)
	case http.MethodDelete:
		deleteVoucherProgramItem(w, r, id)
	default:
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
	}
}

func getVoucherProgramItem(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	var p programRow
	err := DB.QueryRow(ctx, `
		SELECT id, name, description, voucher_type, discount_value, COALESCE(target_plan_id, ''), duration_months, max_uses, uses_count, starts_at, expires_at, is_active
		FROM voucher_programs WHERE id = $1
	`, id).Scan(
		&p.ID, &p.Name, &p.Description, &p.VoucherType, &p.DiscountValue,
		&p.TargetPlanID, &p.DurationMonths, &p.MaxUses, &p.UsesCount,
		&p.StartsAt, &p.ExpiresAt, &p.IsActive,
	)
	if err != nil {
		response.Error(w, http.StatusNotFound, "Voucher program not found", err)
		return
	}

	response.JSON(w, http.StatusOK, "Voucher program retrieved", p)
}

func updateVoucherProgramItem(w http.ResponseWriter, r *http.Request, id string) {
	var req UpdateVoucherProgramReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.InvalidRequest, err)
		return
	}

	if req.Name == "" || req.VoucherType == "" {
		response.Error(w, http.StatusBadRequest, "name and voucher_type are required", nil)
		return
	}

	ctx := r.Context()

	var startsAt time.Time
	if req.StartsAt != "" {
		if t, err := time.Parse(time.RFC3339, req.StartsAt); err == nil {
			startsAt = t
		} else if t2, err2 := time.Parse("2006-01-02T15:04", req.StartsAt); err2 == nil {
			startsAt = t2
		}
	}
	if startsAt.IsZero() {
		startsAt = time.Now()
	}

	var expiresAt *time.Time
	if req.ExpiresAt != "" {
		if t, err := time.Parse(time.RFC3339, req.ExpiresAt); err == nil {
			expiresAt = &t
		} else if t2, err2 := time.Parse("2006-01-02T15:04", req.ExpiresAt); err2 == nil {
			expiresAt = &t2
		}
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	res, err := DB.Exec(ctx, `
		UPDATE voucher_programs
		SET name = $1, description = $2, voucher_type = $3, discount_value = $4,
		    target_plan_id = $5, duration_months = $6, max_uses = $7,
		    starts_at = $8, expires_at = $9, is_active = $10, updated_at = NOW()
		WHERE id = $11
	`, req.Name, req.Description, req.VoucherType, req.DiscountValue,
		req.TargetPlanID, req.DurationMonths, req.MaxUses,
		startsAt, expiresAt, isActive, id)

	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to update voucher program", err)
		return
	}
	if res.RowsAffected() == 0 {
		response.Error(w, http.StatusNotFound, "Voucher program not found", nil)
		return
	}

	response.JSON(w, http.StatusOK, "Voucher program updated", map[string]interface{}{
		"id": id,
	})
}

func deleteVoucherProgramItem(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	var redeemedCodes, redeemedLinks int
	_ = DB.QueryRow(ctx, `SELECT COUNT(*) FROM voucher_codes WHERE program_id = $1 AND is_redeemed = true`, id).Scan(&redeemedCodes)
	_ = DB.QueryRow(ctx, `SELECT COUNT(*) FROM voucher_links WHERE program_id = $1 AND redeemed_by IS NOT NULL`, id).Scan(&redeemedLinks)

	if redeemedCodes > 0 || redeemedLinks > 0 {
		_, err := DB.Exec(ctx, `UPDATE voucher_programs SET is_active = false, updated_at = NOW() WHERE id = $1`, id)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to deactivate voucher program", err)
			return
		}
		response.JSON(w, http.StatusOK, "Voucher program deactivated (has redeemed vouchers)", map[string]interface{}{
			"id":        id,
			"is_active": false,
		})
		return
	}

	res, err := DB.Exec(ctx, `DELETE FROM voucher_programs WHERE id = $1`, id)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to delete voucher program", err)
		return
	}
	if res.RowsAffected() == 0 {
		response.Error(w, http.StatusNotFound, "Voucher program not found", nil)
		return
	}

	response.JSON(w, http.StatusOK, "Voucher program deleted", map[string]interface{}{
		"id": id,
	})
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
