package handlers

import (
	"context"
	"net/http"

	"core_project/apps/campaign/api/repository"
)

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
