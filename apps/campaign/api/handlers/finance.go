package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"core_project/apps/campaign/api/repository"
)

type ExpensePayload struct {
	CampaignID       string  `json:"campaign_id"`
	ExpenseCategory  string  `json:"expense_category"`
	Amount           float64 `json:"amount"`
	TargetRegionType string  `json:"target_region_type,omitempty"`
	TargetRegionID   string  `json:"target_region_id,omitempty"`
	Description      string  `json:"description,omitempty"`
}

func HandleCampaignFinance(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing context"})
		return
	}

	ctx := context.Background()

	// GET: Calculate Cost-per-Vote
	if r.Method == http.MethodGet {
		campaignID := r.URL.Query().Get("campaign_id")
		if campaignID == "" {
			WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "campaign_id required"})
			return
		}

		// Calculate total expenses
		var totalExpense float64
		_ = repository.DB.QueryRow(ctx, "SELECT COALESCE(SUM(amount), 0) FROM campaign_expenses WHERE campaign_id = $1", campaignID).Scan(&totalExpense)

		// Calculate total valid endorsements (KTP)
		var totalEndorsements float64
		_ = repository.DB.QueryRow(ctx, "SELECT COUNT(*) FROM endorsements WHERE campaign_id = $1 AND status = 'valid'", campaignID).Scan(&totalEndorsements)

		costPerVote := 0.0
		if totalEndorsements > 0 {
			costPerVote = totalExpense / totalEndorsements
		}

		WriteJSON(w, http.StatusOK, APIResponse{
			Success: true,
			Data: map[string]interface{}{
				"total_expense":      totalExpense,
				"total_endorsements": totalEndorsements,
				"cost_per_vote":      costPerVote,
			},
		})
		return
	}

	// POST: Record Expense (and optionally push to UMKM accounting engine)
	if r.Method == http.MethodPost {
		var req ExpensePayload
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid payload"})
			return
		}

		query := `
			INSERT INTO campaign_expenses 
			(tenant_id, campaign_id, expense_category, amount, target_region_type, target_region_id, description)
			VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id
		`
		var expID string
		err := repository.DB.QueryRow(ctx, query,
			tenantID, req.CampaignID, req.ExpenseCategory, req.Amount,
			req.TargetRegionType, req.TargetRegionID, req.Description,
		).Scan(&expID)

		if err != nil {
			WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to record expense"})
			return
		}

		// TODO: Integration with apps/umkm/accounting via internal HTTP call or direct DB insertion 
		// if they share the same physical database.
		// e.g. Insert into journal_entries

		WriteJSON(w, http.StatusOK, APIResponse{
			Success: true,
			Data:    map[string]string{"expense_id": expID},
		})
		return
	}

	WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
}
