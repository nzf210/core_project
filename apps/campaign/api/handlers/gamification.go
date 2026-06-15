package handlers

import (
	"context"
	"net/http"

	"core_project/apps/campaign/api/repository"
)

// HandleGamificationLeaderboard aggregates top recruiters
func HandleGamificationLeaderboard(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusUnauthorized, APIResponse{
			Success: false,
			Message: "Unauthorized - Tenant ID missing",
		})
		return
	}

	ctx := context.Background()
	query := `
		SELECT v.name, count(e.id) as count
		FROM endorsements e
		JOIN volunteers v ON e.recruiter_id = v.id
		WHERE e.tenant_id = $1
		GROUP BY v.name
		ORDER BY count DESC
		LIMIT 10
	`
	rows, err := repository.DB.Query(ctx, query, tenantID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: "Failed to fetch leaderboard",
		})
		return
	}
	defer rows.Close()

	type Entry struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	var leaderboard []Entry
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.Name, &e.Count); err == nil {
			leaderboard = append(leaderboard, e)
		}
	}

	WriteJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    leaderboard,
	})
}
