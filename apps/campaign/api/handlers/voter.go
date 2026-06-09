package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"core_project/apps/campaign/api/repository"
	"core_project/shared/sdk/encryption"
	"core_project/shared/sdk/config"
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

	if r.Method == http.MethodPost {
		var req struct {
			Nik     string `json:"nik"`
			Name    string `json:"name"`
			Address string `json:"address"`
			Phone   string `json:"phone"`
			Status  string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid request payload"})
			return
		}

		if req.Status == "" {
			req.Status = "uncontacted"
		}

		cfg := config.LoadConfig("")
		encryptionKey := cfg.EncryptionKey

		nikEnc, err := encryption.Encrypt(req.Nik, encryptionKey)
		if err != nil {
			WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Encryption error"})
			return
		}
		nameEnc, err := encryption.Encrypt(req.Name, encryptionKey)
		if err != nil {
			WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Encryption error"})
			return
		}
		addrEnc, err := encryption.Encrypt(req.Address, encryptionKey)
		if err != nil {
			WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Encryption error"})
			return
		}

		var id string
		err = repository.DB.QueryRow(context.Background(),
			"INSERT INTO voters (tenant_id, nik_encrypted, name_encrypted, address_encrypted, phone, status) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id",
			tenantID, nikEnc, nameEnc, addrEnc, req.Phone, req.Status).Scan(&id)
		
		if err != nil {
			WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to create voter"})
			return
		}

		WriteJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Voter created", Data: map[string]string{"id": id}})
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
