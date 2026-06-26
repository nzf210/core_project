package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"regexp"

	"core_project/apps/campaign/api/repository"
)

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

	// 1. Validasi Format NIK (harus 16 digit angka)
	nikRegex := regexp.MustCompile(`^\d{16}$`)
	endorsementStatus := "valid"
	if !nikRegex.MatchString(req.Nik) {
		endorsementStatus = "invalid_nik"
	}

	ctx := context.Background()

	// Mulai Transaksi Database
	tx, err := repository.DB.Begin(ctx)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Transaction start failed"})
		return
	}
	defer tx.Rollback(ctx)

	// 2. Insert or Get Citizen (Master Data)
	var citizenID string
	var isDPTVerified bool
	var existingTPS *string

	// Cek apakah NIK ini ada di DPT
	err = tx.QueryRow(ctx, "SELECT true, tps_name FROM dpt_records WHERE nik = $1", req.Nik).Scan(&isDPTVerified, &existingTPS)
	if err != nil {
		isDPTVerified = false // Not in DPT
	}

	// Insert into citizens (ON CONFLICT DO NOTHING)
	_, err = tx.Exec(ctx, `
		INSERT INTO citizens (nik, name, address, is_dpt_verified) 
		VALUES ($1, $2, $3, $4) 
		ON CONFLICT (nik) DO NOTHING
	`, req.Nik, req.Name, req.Address, isDPTVerified)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to sync citizen records"})
		return
	}

	// Ambil citizen_id (karena ON CONFLICT DO NOTHING tidak return ID di PostgreSQL)
	err = tx.QueryRow(ctx, "SELECT id FROM citizens WHERE nik = $1", req.Nik).Scan(&citizenID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to retrieve citizen ID"})
		return
	}

	// 3. Deteksi Ganda (Internal vs External) jika NIK valid
	if endorsementStatus != "invalid_nik" {
		// Cek apakah citizen ini sudah didukung di sistem
		var existingClaims []struct {
			TenantID   string
			CampaignID string
		}
		
		rows, err := tx.Query(ctx, "SELECT tenant_id::text, campaign_id::text FROM endorsements WHERE citizen_id = $1", citizenID)
		if err == nil {
			for rows.Next() {
				var tID, cID string
				if err := rows.Scan(&tID, &cID); err == nil {
					existingClaims = append(existingClaims, struct{TenantID, CampaignID string}{tID, cID})
				}
			}
			rows.Close()
		}

		for _, claim := range existingClaims {
			if claim.TenantID == tenantID {
				// Sudah didukung di kubu/paslon yang sama -> Konflik Internal (Nyolong teman)
				endorsementStatus = "conflict_internal"
			} else {
				// Didukung oleh paslon lawan -> Konflik Eksternal (Sengketa silang)
				// Tetap validkan jika internal belum conflict, tapi utamakan external conflict
				if endorsementStatus != "conflict_internal" {
					endorsementStatus = "conflict_external"
				}
			}
		}
	}

	// 4. Insert Relasi Endorsement
	var recruiterArg interface{}
	if req.RecruiterID != "" {
		recruiterArg = req.RecruiterID
	} else {
		recruiterArg = nil
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
		Data: map[string]interface{}{
			"endorsement_id": endorsementID,
			"citizen_id": citizenID,
			"status": endorsementStatus,
			"is_dpt_verified": isDPTVerified,
		},
	})
}
