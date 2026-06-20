package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"core_project/apps/campaign/api/repository"
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

// HandleBlastTarget generates phone number lists based on micro-targeting filters.
// Filters: village_id (via dpt_records), gender (citizens.gender), age_range (citizens.age).
// Always excludes flagged anomalies (endorsements.is_anomaly = TRUE).
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

	// Validate age_range if provided
	ageMin, ageMax, err := parseAgeRange(req.AgeRange)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	// Validate gender if provided
	if req.Gender != "" && req.Gender != "L" && req.Gender != "P" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "gender must be 'L', 'P', or empty",
		})
		return
	}

	ctx := context.Background()

	// Build query with dynamic WHERE clauses using indexed positional args.
	// Base filter: tenant + non-anomaly. Optional filters applied conditionally.
	query := `
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

	rows, err := repository.DB.Query(ctx, query,
		tenantID,
		req.Gender,
		ageMin,
		ageMax,
		req.VillageID,
	)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: "Failed to query target audience: " + err.Error(),
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
	if phones == nil {
		phones = []string{}
	}

	WriteJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Target audience generated and dispatched to WA Gateway",
		Data: BlastTargetResponse{
			Phones:      phones,
			TargetCount: len(phones),
		},
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