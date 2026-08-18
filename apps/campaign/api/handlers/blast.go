package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"core_project/apps/campaign/api/repository"
	"core_project/shared/sdk/response"
)

type BlastTargetRequest struct {
	VillageID  string `json:"village_id"`
	Gender     string `json:"gender"`     // "L" | "P" | "" (any)
	AgeRange   string `json:"age_range"`  // "18-25" | "26-40" | "41-60" | "60+" | "" (any)
	TemplateID string `json:"template_id"`
}

type BlastTargetResponse struct {
	Phones      []string `json:"phones"`
	TargetCount int      `json:"target_count"`
}

func validateBlastRequest(req BlastTargetRequest) (ageMin, ageMax int, err error) {
	ageMin, ageMax, err = parseAgeRange(req.AgeRange)
	if err != nil {
		return
	}
	if req.Gender != "" && req.Gender != "L" && req.Gender != "P" {
		return 0, 0, fmt.Errorf("gender must be 'L', 'P', or empty")
	}
	return
}

func queryBlastPhones(ctx context.Context, tenantID, gender, villageID string, ageMin, ageMax int) ([]string, error) {
	const query = `
		SELECT c.phone
		FROM citizens c
		JOIN endorsements e ON e.citizen_id = c.id AND e.tenant_id = $1 AND e.is_anomaly = FALSE
		LEFT JOIN dpt_records d ON d.nik = c.nik
		WHERE c.tenant_id = $1
		  AND c.phone IS NOT NULL AND c.phone != ''
		  AND ($2 = '' OR c.gender = $2)
		  AND ($3 = 0 OR c.age >= $3)
		  AND ($4 = 0 OR c.age <= $4)
		  AND ($5 = '' OR d.village_id = $5::uuid)
		GROUP BY c.id, c.phone
	`
	rows, err := repository.DB.Query(ctx, query, tenantID, gender, ageMin, ageMax, villageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	phones := []string{}
	for rows.Next() {
		var phone string
		if rows.Scan(&phone) == nil && phone != "" {
			phones = append(phones, phone)
		}
	}
	return phones, nil
}

func HandleBlastTarget(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusUnauthorized, APIResponse{Success: false, Message: "Unauthorized - Tenant ID missing"})
		return
	}
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: response.MethodNotAllowed})
		return
	}

	var req BlastTargetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Invalid JSON payload"})
		return
	}

	ageMin, ageMax, err := validateBlastRequest(req)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: err.Error()})
		return
	}

	phones, err := queryBlastPhones(context.Background(), tenantID, req.Gender, req.VillageID, ageMin, ageMax)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Message: "Failed to query target audience: " + err.Error()})
		return
	}

	WriteJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Target audience generated and dispatched to WA Gateway",
		Data:    BlastTargetResponse{Phones: phones, TargetCount: len(phones)},
	})
}

// parseAgeRange converts "18-25", "60+" syntax into min/max ints. Empty → (0,0) = no filter.
func parseAgeRange(s string) (min, max int, err error) {
	if s == "" {
		return 0, 0, nil
	}
	var a, b int
	if n, _ := fmt.Sscanf(s, "%d-%d", &a, &b); n == 2 {
		if a < 0 || b < a {
			return 0, 0, fmt.Errorf("invalid age_range: %s", s)
		}
		return a, b, nil
	}
	if n, _ := fmt.Sscanf(s, "%d+", &a); n == 1 {
		if a < 0 {
			return 0, 0, fmt.Errorf("invalid age_range: %s", s)
		}
		return a, 0, nil
	}
	return 0, 0, fmt.Errorf("age_range must be 'min-max' or 'min+', got: %s", s)
}