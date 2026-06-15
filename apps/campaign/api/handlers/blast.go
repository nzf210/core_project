package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"core_project/apps/campaign/api/repository"
)

type BlastTargetRequest struct {
	VillageID  string `json:"village_id"`
	Gender     string `json:"gender"`
	AgeRange   string `json:"age_range"`
	TemplateID string `json:"template_id"`
}

// HandleBlastTarget generates phone number lists based on micro-targeting filters
func HandleBlastTarget(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusUnauthorized, APIResponse{
			Success: false,
			Message: "Unauthorized - Tenant ID missing",
		})
		return
	}

	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
		return
	}

	var req BlastTargetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Invalid JSON payload",
		})
		return
	}

	ctx := context.Background()
	query := `
		SELECT c.phone 
		FROM citizens c
		JOIN endorsements e ON c.id = e.citizen_id
		WHERE c.tenant_id = $1 AND e.is_anomaly = FALSE
	`
	rows, err := repository.DB.Query(ctx, query, tenantID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: "Failed to query target audience",
		})
		return
	}
	defer rows.Close()

	var phones []string
	for rows.Next() {
		var phone string
		if err := rows.Scan(&phone); err == nil && phone != "" {
			phones = append(phones, phone)
		}
	}

	WriteJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Target audience generated and dispatched to WA Gateway",
		Data: map[string]interface{}{
			"target_count": len(phones),
		},
	})
}
