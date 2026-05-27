package handlers

import (
	"context"
	"net/http"

	"core_project/apps/campaign/api/repository"
)

func HandleReports(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
		return
	}

	if r.Method == http.MethodGet {
		// Mocked aggregation for reports
		var totalVolunteers int
		_ = repository.DB.QueryRow(context.Background(), "SELECT count(*) FROM volunteers WHERE tenant_id = $1", tenantID).Scan(&totalVolunteers)
		
		var totalVoters int
		_ = repository.DB.QueryRow(context.Background(), "SELECT count(*) FROM voters WHERE tenant_id = $1", tenantID).Scan(&totalVoters)

		reportData := map[string]interface{}{
			"summary": map[string]interface{}{
				"total_volunteers": totalVolunteers,
				"total_voters": totalVoters,
				"target_voters": 100000,
			},
			"regions": []map[string]interface{}{
				{"region": "Kecamatan A", "voters": 450},
				{"region": "Kecamatan B", "voters": 320},
			},
		}

		WriteJSON(w, http.StatusOK, APIResponse{Success: true, Data: reportData})
		return
	}

	WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
}
