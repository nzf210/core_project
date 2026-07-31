package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"core_project/apps/campaign/api/repository"
	"core_project/shared/sdk/response"
)

type Candidate struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	Status               string `json:"status"`
	VerificationDocument string `json:"verification_document"`
	IsVerified           bool   `json:"is_verified"`
	Suspended            bool   `json:"suspended"`
}

type Campaign struct {
	ID           string `json:"id"`
	CandidateID  string `json:"candidate_id"`
	Name         string `json:"name"`
	Status       string `json:"status"`
	TargetVoters int    `json:"target_voters"`
}

func HandleCandidates(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: errMissingTenantID})
		return
	}

	switch r.Method {
	case http.MethodGet:
		rows, err := repository.DB.Query(context.Background(),
			"SELECT id, name, status, COALESCE(verification_document, ''), is_verified, suspended FROM candidates WHERE tenant_id = $1", tenantID)
		if err != nil {
			WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Database error"})
			return
		}
		defer rows.Close()

		var candidates []Candidate
		for rows.Next() {
			var c Candidate
			if err := rows.Scan(&c.ID, &c.Name, &c.Status, &c.VerificationDocument, &c.IsVerified, &c.Suspended); err == nil {
				candidates = append(candidates, c)
			}
		}

		if candidates == nil {
			candidates = []Candidate{}
		}

		WriteJSON(w, http.StatusOK, APIResponse{Success: true, Data: candidates})

	case http.MethodPost:
		var req struct {
			Name                 string `json:"name"`
			VerificationDocument string `json:"verification_document"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid request payload"})
			return
		}

		var id string
		err := repository.DB.QueryRow(context.Background(),
			"INSERT INTO candidates (tenant_id, name, verification_document) VALUES ($1, $2, $3) RETURNING id",
			tenantID, req.Name, req.VerificationDocument).Scan(&id)
		if err != nil {
			WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to create candidate"})
			return
		}

		WriteJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Candidate created", Data: map[string]string{"id": id}})

	default:
		WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: response.MethodNotAllowed})
	}
}

func HandleCandidateVerify(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: errMissingTenantID})
		return
	}

	// FIX #2: Auth Bypass — only admin/owner may verify candidates
	userRole := ExtractUserRole(r)
	if userRole != "admin" && userRole != "owner" {
		WriteJSON(w, http.StatusForbidden, APIResponse{Message: "Only admin or owner can verify candidates"})
		return
	}

	id := r.PathValue("id")
	if id == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing candidate ID"})
		return
	}

	_, err := repository.DB.Exec(context.Background(),
		"UPDATE candidates SET is_verified = true, status = 'verified' WHERE id = $1 AND tenant_id = $2", id, tenantID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to verify candidate"})
		return
	}

	WriteJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Candidate verified successfully"})
}

func HandleCampaigns(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: errMissingTenantID})
		return
	}

	switch r.Method {
	case http.MethodGet:
		rows, err := repository.DB.Query(context.Background(),
			"SELECT id, candidate_id, name, status, target_voters FROM campaigns WHERE tenant_id = $1", tenantID)
		if err != nil {
			WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Database error"})
			return
		}
		defer rows.Close()

		var campaigns []Campaign
		for rows.Next() {
			var c Campaign
			if err := rows.Scan(&c.ID, &c.CandidateID, &c.Name, &c.Status, &c.TargetVoters); err == nil {
				campaigns = append(campaigns, c)
			}
		}

		if campaigns == nil {
			campaigns = []Campaign{}
		}

		WriteJSON(w, http.StatusOK, APIResponse{Success: true, Data: campaigns})

	case http.MethodPost:
		var req struct {
			CandidateID  string `json:"candidate_id"`
			Name         string `json:"name"`
			TargetVoters int    `json:"target_voters"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid request payload"})
			return
		}

		var id string
		err := repository.DB.QueryRow(context.Background(),
			"INSERT INTO campaigns (tenant_id, candidate_id, name, target_voters) VALUES ($1, $2, $3, $4) RETURNING id",
			tenantID, req.CandidateID, req.Name, req.TargetVoters).Scan(&id)
		if err != nil {
			WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to create campaign"})
			return
		}

		WriteJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Campaign created", Data: map[string]string{"id": id}})

	default:
		WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: response.MethodNotAllowed})
	}
}
