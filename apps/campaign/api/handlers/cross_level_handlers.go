package handlers

import (
	"context"
	"fmt"
	"net/http"

	"core_project/apps/campaign/api/repository"
	"core_project/shared/sdk/response"
)

// /endorsements/conflicts
func HandleEndorsementConflicts(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing context"})
		return
	}

	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: response.MethodNotAllowed})
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
		WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: response.MethodNotAllowed})
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
		Data:    irisans,
	})
}
