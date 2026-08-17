package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"

	"core_project/apps/campaign/api/repository"

	"github.com/jackc/pgx/v5"
	"core_project/shared/sdk/response"
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
		WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: response.MethodNotAllowed})
		return
	}

	req, err := parseEndorsementRequest(r)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: err.Error()})
		return
	}

	result, err := processEndorsement(tenantID, req)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: err.Error()})
		return
	}

	WriteJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Endorsement recorded",
		Data:    result,
	})
}

type endorsementRequest struct {
	CampaignID  string
	RecruiterID string
	Nik         string
	Name        string
	Address     string
	Phone       string
}

func parseEndorsementRequest(r *http.Request) (*endorsementRequest, error) {
	var req endorsementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, fmt.Errorf("Invalid request payload")
	}

	if req.CampaignID == "" || req.Nik == "" || req.Name == "" {
		return nil, fmt.Errorf("campaign_id, nik, and name are required")
	}

	return &req, nil
}

func processEndorsement(tenantID string, req *endorsementRequest) (map[string]any, error) {
	ctx := context.Background()
	tx, err := repository.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("Transaction start failed")
	}
	defer tx.Rollback(ctx)

	endorsementStatus := validateNIK(req.Nik)
	isDPTVerified := checkDPTVerification(ctx, tx, req.Nik)

	citizenID, err := upsertCitizen(ctx, tx, req, isDPTVerified)
	if err != nil {
		return nil, err
	}

	if endorsementStatus == "valid" {
		if dupStatus := checkDuplicateEndorsement(ctx, tx, tenantID, citizenID); dupStatus != "" {
			endorsementStatus = dupStatus
		}
	}

	endorsementID, err := recordEndorsement(ctx, tx, tenantID, citizenID, req, endorsementStatus)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("Transaction commit failed")
	}

	return map[string]any{
		"endorsement_id":   endorsementID,
		"citizen_id":       citizenID,
		"status":           endorsementStatus,
		"is_dpt_verified":  isDPTVerified,
	}, nil
}

func checkDPTVerification(ctx context.Context, tx pgx.Tx, nik string) bool {
	var isDPTVerified bool
	err := tx.QueryRow(ctx, "SELECT true, tps_name FROM dpt_records WHERE nik = $1", nik).Scan(&isDPTVerified, new(string))
	return err == nil
}

func upsertCitizen(ctx context.Context, tx pgx.Tx, req *endorsementRequest, isDPTVerified bool) (string, error) {
	_, err := tx.Exec(ctx, `
		INSERT INTO citizens (nik, name, address, is_dpt_verified)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (nik) DO NOTHING
	`, req.Nik, req.Name, req.Address, isDPTVerified)
	if err != nil {
		return "", fmt.Errorf("Failed to sync citizen records")
	}

	var citizenID string
	err = tx.QueryRow(ctx, "SELECT id FROM citizens WHERE nik = $1", req.Nik).Scan(&citizenID)
	if err != nil {
		return "", fmt.Errorf("Failed to retrieve citizen ID")
	}

	return citizenID, nil
}

func recordEndorsement(ctx context.Context, tx pgx.Tx, tenantID, citizenID string, req *endorsementRequest, status string) (string, error) {
	var recruiterArg interface{}
	if req.RecruiterID != "" {
		recruiterArg = req.RecruiterID
	}

	var endorsementID string
	err := tx.QueryRow(ctx, `
		INSERT INTO endorsements (citizen_id, tenant_id, campaign_id, recruiter_id, status)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (citizen_id, campaign_id) DO NOTHING
		RETURNING id
	`, citizenID, tenantID, req.CampaignID, recruiterArg, status).Scan(&endorsementID)

	if err != nil {
		slog.Error("Failed to record endorsement", "tenant_id", tenantID, "citizen_id", citizenID)
		return "", fmt.Errorf("Failed to record endorsement")
	}

	return endorsementID, nil
}
