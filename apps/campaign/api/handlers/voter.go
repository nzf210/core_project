package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	"core_project/apps/campaign/api/repository"
)

type Voter struct {
	ID                string `json:"id"`
	Status            string `json:"status"`
	PotentialLevel    string `json:"potential_level"`
	CompetitorSupport string `json:"competitor_support"`
	PicID             string `json:"pic_id"`
}

func HandleVoters(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	userID := ExtractUserID(r)
	userRole := ExtractUserRole(r)

	if tenantID == "" || userID == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing context"})
		return
	}

	if r.Method == http.MethodGet {
		query := `SELECT id, COALESCE(status, 'uncontacted'), COALESCE(potential_level, ''), COALESCE(competitor_support, ''), COALESCE(pic_id::text, '') FROM voters WHERE tenant_id = $1`
		args := []interface{}{tenantID}

		if userRole != "admin" {
			query = repository.GetHierarchyCTE(2) + query + ` AND pic_id IN (SELECT id FROM subordinates)`
			args = append(args, userID)
		}

		rows, err := repository.DB.Query(context.Background(), query, args...)
		if err != nil {
			WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Database error"})
			return
		}
		defer rows.Close()

		var voters []Voter
		for rows.Next() {
			var v Voter
			if err := rows.Scan(&v.ID, &v.Status, &v.PotentialLevel, &v.CompetitorSupport, &v.PicID); err == nil {
				voters = append(voters, v)
			}
		}

		if voters == nil {
			voters = []Voter{}
		}

		WriteJSON(w, http.StatusOK, APIResponse{Success: true, Data: voters})
		return
	}

	if r.Method == http.MethodPut {
		var req struct {
			ID                string `json:"id"`
			Status            string `json:"status"`
			PotentialLevel    string `json:"potential_level"`
			CompetitorSupport string `json:"competitor_support"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid request payload"})
			return
		}

		query := "UPDATE voters SET updated_at = now()"
		args := []interface{}{}
		argId := 1

		if req.Status != "" {
			query += fmt.Sprintf(", status = $%d", argId)
			args = append(args, req.Status)
			argId++
		}
		if req.PotentialLevel != "" {
			query += fmt.Sprintf(", potential_level = $%d", argId)
			args = append(args, req.PotentialLevel)
			argId++
		}
		if req.CompetitorSupport != "" {
			query += fmt.Sprintf(", competitor_support = $%d", argId)
			args = append(args, req.CompetitorSupport)
			argId++
		}

		query += fmt.Sprintf(" WHERE id = $%d AND tenant_id = $%d", argId, argId+1)
		args = append(args, req.ID, tenantID)

		_, err := repository.DB.Exec(context.Background(), query, args...)
		if err != nil {
			WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to update voter"})
			return
		}

		WriteJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Voter updated"})
		return
	}

	WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
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
		VALUES ($1, $2, $3, $4, $5) RETURNING id
	`, citizenID, tenantID, req.CampaignID, recruiterArg, endorsementStatus).Scan(&endorsementID)
	
	if err != nil {
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

func HandleVoterStats(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	userID := ExtractUserID(r)
	userRole := ExtractUserRole(r)

	if tenantID == "" || userID == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing context"})
		return
	}

	query := "SELECT status, COALESCE(potential_level, ''), COALESCE(competitor_support, ''), COUNT(*) FROM voters WHERE tenant_id = $1"
	args := []interface{}{tenantID}

	if userRole != "admin" {
		query = repository.GetHierarchyCTE(2) + query + ` AND pic_id IN (SELECT id FROM subordinates)`
		args = append(args, userID)
	}

	query += " GROUP BY status, potential_level, competitor_support"

	rows, err := repository.DB.Query(context.Background(), query, args...)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Database error"})
		return
	}
	defer rows.Close()

	stats := map[string]int{
		"uncontacted": 0,
		"contacted":   0,
	}
	potential := map[string]int{
		"high":   0,
		"medium": 0,
		"low":    0,
	}
	competitors := map[string]int{}

	total := 0
	for rows.Next() {
		var status, pLevel, compSupport string
		var count int
		if err := rows.Scan(&status, &pLevel, &compSupport, &count); err == nil {
			stats[status] += count
			if pLevel != "" {
				potential[pLevel] += count
			}
			if compSupport != "" {
				competitors[compSupport] += count
			}
			total += count
		}
	}

	data := map[string]interface{}{
		"total_voters": total,
		"by_status":    stats,
		"potential":    potential,
		"competitors":  competitors,
	}

	WriteJSON(w, http.StatusOK, APIResponse{Success: true, Data: data})
}
