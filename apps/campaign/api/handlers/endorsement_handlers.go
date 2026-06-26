package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"regexp"

	"core_project/apps/campaign/api/repository"

	"github.com/jackc/pgx/v5"
)

var nikRegex = regexp.MustCompile(`^\d{16}$`)

func validateNIK(nik string) string {
	if nikRegex.MatchString(nik) {
		return "valid"
	}
	return "invalid_nik"
}

func checkDuplicateEndorsement(ctx context.Context, tx pgx.Tx, tenantID, citizenID string) string {
	rows, err := tx.Query(ctx, "SELECT tenant_id::text, campaign_id::text FROM endorsements WHERE citizen_id = $1", citizenID)
	if err != nil {
		return ""
	}
	defer rows.Close()

	for rows.Next() {
		var tID, cID string
		if err := rows.Scan(&tID, &cID); err == nil {
			if tID == tenantID {
				return "conflict_internal"
			}
			return "conflict_external"
		}
	}
	return ""
}

// HandleEndorsements replaces the old POST /voters logic
// Supports F031 Anti-Double Validation and Citizen-Centric Schema
func HandleEndorsements(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing tenant context"})
		return
	}

	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
		return
	}

	var req struct {
		CampaignID  string `json:"campaign_id"`
		RecruiterID string `json:"recruiter_id"` // can be empty
		Nik         string `json:"nik"`
		Name        string `json:"name"`
		Address     string `json:"address"`
		Phone       string `json:"phone"` // legacy phone compat
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid request payload"})
		return
	}

	if req.CampaignID == "" || req.Nik == "" || req.Name == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "campaign_id, nik, and name are required"})
		return
	}

	endorsementStatus := validateNIK(req.Nik)
	ctx := context.Background()

	tx, err := repository.DB.Begin(ctx)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Transaction start failed"})
		return
	}
	defer tx.Rollback(ctx)

	// Check if NIK exists in DPT
	var isDPTVerified bool
	err = tx.QueryRow(ctx, "SELECT true, tps_name FROM dpt_records WHERE nik = $1", req.Nik).Scan(&isDPTVerified, new(string))
	if err != nil {
		isDPTVerified = false
	}

	// Insert citizen (ON CONFLICT DO NOTHING)
	_, err = tx.Exec(ctx, `
		INSERT INTO citizens (nik, name, address, is_dpt_verified)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (nik) DO NOTHING
	`, req.Nik, req.Name, req.Address, isDPTVerified)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to sync citizen records"})
		return
	}

	var citizenID string
	err = tx.QueryRow(ctx, "SELECT id FROM citizens WHERE nik = $1", req.Nik).Scan(&citizenID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to retrieve citizen ID"})
		return
	}

	// Check for duplicate endorsements if NIK is valid
	if endorsementStatus == "valid" {
		if dupStatus := checkDuplicateEndorsement(ctx, tx, tenantID, citizenID); dupStatus != "" {
			endorsementStatus = dupStatus
		}
	}

	var recruiterArg interface{}
	if req.RecruiterID != "" {
		recruiterArg = req.RecruiterID
	}

	var endorsementID string
	err = tx.QueryRow(ctx, `
		INSERT INTO endorsements (citizen_id, tenant_id, campaign_id, recruiter_id, status)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (citizen_id, campaign_id) DO NOTHING
		RETURNING id
	`, citizenID, tenantID, req.CampaignID, recruiterArg, endorsementStatus).Scan(&endorsementID)
	if err != nil {
		slog.Error("Failed to record endorsement", "tenant_id", tenantID, "citizen_id", citizenID)
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to record endorsement"})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Transaction commit failed"})
		return
	}

	WriteJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Endorsement recorded",
		Data: map[string]any{
			"endorsement_id":   endorsementID,
			"citizen_id":      citizenID,
			"status":          endorsementStatus,
			"is_dpt_verified": isDPTVerified,
		},
	})
}
