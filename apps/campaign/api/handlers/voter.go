package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"log/slog"
	"regexp"
	"bytes"

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

func HandleKTPScan(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing context"})
		return
	}

	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
		return
	}

	var req struct {
		ImageURL   string `json:"image_url"`
		CampaignID string `json:"campaign_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid request"})
		return
	}

	// Call AI Gateway Vision OCR
	// TODO: Replace with real service discovery / config URL
	visionURL := "http://localhost:8002/v1/vision"
	visionPayload := map[string]string{
		"image_url": req.ImageURL,
		"prompt":    "Extract KTP data",
	}
	body, _ := json.Marshal(visionPayload)
	
	resp, err := http.Post(visionURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "AI Gateway unreachable"})
		return
	}
	defer resp.Body.Close()

	var visionResp struct {
		Success bool `json:"success"`
		Data    struct {
			Text string `json:"text"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&visionResp)

	if !visionResp.Success {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "AI OCR failed"})
		return
	}

	// Parse the structured JSON returned from mock vision
	var ktpData struct {
		NIK     string `json:"nik"`
		Name    string `json:"name"`
		Address string `json:"address"`
		Gender  string `json:"gender"`
		Age     int    `json:"age"`
	}
	if err := json.Unmarshal([]byte(visionResp.Data.Text), &ktpData); err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "AI returned malformed data"})
		return
	}

	// Auto-Register the citizen and endorsement — wrap in transaction to prevent orphan records
	ctx := context.Background()
	tx, err := repository.DB.Begin(ctx)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Transaction start failed"})
		return
	}
	defer tx.Rollback(ctx)

	var citizenID string
	queryCitizen := `
		INSERT INTO citizens (nik, name, address, gender, age)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (nik) DO UPDATE SET
			name = EXCLUDED.name,
			address = EXCLUDED.address,
			updated_at = NOW()
		RETURNING id
	`
	err = tx.QueryRow(ctx, queryCitizen, ktpData.NIK, ktpData.Name, ktpData.Address, ktpData.Gender, ktpData.Age).Scan(&citizenID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to save citizen data"})
		return
	}

	// Record endorsement
	queryEndorsement := `
		INSERT INTO endorsements (citizen_id, tenant_id, campaign_id, proof_image_url, status)
		VALUES ($1, $2, $3, $4, 'valid')
	`
	_, err = tx.Exec(ctx, queryEndorsement, citizenID, tenantID, req.CampaignID, req.ImageURL)
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
		Message: "KTP Scanned and Registered",
		Data:    ktpData,
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

func HandleEndorsementConflicts(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing context"})
		return
	}

	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
		return
	}

	// Parameter: conflict_type = 'internal' | 'external'
	conflictType := r.URL.Query().Get("type")
	if conflictType == "" {
		conflictType = "external" // default ke musuh
	}

	var query string
	if conflictType == "internal" {
		query = `
			SELECT c.nik, c.name, e.status, v.name as recruiter_name
			FROM citizens c
			JOIN endorsements e ON c.id = e.citizen_id
			LEFT JOIN volunteers v ON e.recruiter_id = v.id
			WHERE e.tenant_id = $1 AND e.status = 'conflict_internal'
			ORDER BY e.created_at DESC LIMIT 100
		`
	} else {
		query = `
			SELECT c.nik, c.name, e.status, v.name as recruiter_name
			FROM citizens c
			JOIN endorsements e ON c.id = e.citizen_id
			LEFT JOIN volunteers v ON e.recruiter_id = v.id
			WHERE e.tenant_id = $1 AND e.status = 'conflict_external'
			ORDER BY e.created_at DESC LIMIT 100
		`
	}

	rows, err := repository.DB.Query(context.Background(), query, tenantID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Database error"})
		return
	}
	defer rows.Close()

	var conflicts []map[string]interface{}
	for rows.Next() {
		var nik, name, status string
		var recruiterName *string
		if err := rows.Scan(&nik, &name, &status, &recruiterName); err == nil {
			rName := "Unknown/System"
			if recruiterName != nil {
				rName = *recruiterName
			}
			conflicts = append(conflicts, map[string]interface{}{
				"nik":            nik,
				"name":           name,
				"status":         status,
				"recruiter_name": rName,
			})
		}
	}

	if conflicts == nil {
		conflicts = []map[string]interface{}{}
	}

	WriteJSON(w, http.StatusOK, APIResponse{Success: true, Data: conflicts})
}

func HandleCrossLevelEndorsements(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing context"})
		return
	}

	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
		return
	}

	// target_tenant_id adalah ID paslon tandem/partner (misal ID calon gubernur)
	targetTenantID := r.URL.Query().Get("target_tenant_id")
	if targetTenantID == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing target_tenant_id parameter"})
		return
	}

	query := `
		SELECT c.nik, c.name, e1.created_at
		FROM citizens c
		JOIN endorsements e1 ON c.id = e1.citizen_id
		JOIN endorsements e2 ON c.id = e2.citizen_id
		WHERE e1.tenant_id = $1 AND e2.tenant_id = $2
		ORDER BY e1.created_at DESC
	`

	rows, err := repository.DB.Query(context.Background(), query, tenantID, targetTenantID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Database error"})
		return
	}
	defer rows.Close()

	var irisans []map[string]interface{}
	for rows.Next() {
		var nik, name, createdAt string
		if err := rows.Scan(&nik, &name, &createdAt); err == nil {
			irisans = append(irisans, map[string]interface{}{
				"nik":        nik,
				"name":       name,
				"created_at": createdAt,
			})
		}
	}

	if irisans == nil {
		irisans = []map[string]interface{}{}
	}

	WriteJSON(w, http.StatusOK, APIResponse{
		Success: true, 
		Message: fmt.Sprintf("Found %d intersecting supporters", len(irisans)),
		Data: irisans,
	})
}
