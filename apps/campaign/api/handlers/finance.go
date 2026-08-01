package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"core_project/apps/campaign/api/repository"
	"core_project/shared/sdk/response"
)

type ExpensePayload struct {
	CampaignID       string  `json:"campaign_id"`
	ExpenseCategory  string  `json:"expense_category"`
	Amount           float64 `json:"amount"`
	TargetRegionType string  `json:"target_region_type,omitempty"`
	TargetRegionID   string  `json:"target_region_id,omitempty"`
	Description      string  `json:"description,omitempty"`
}

// F034: CostPerVoteAlert threshold — alert jika CPV > Rp 200.000
const costPerVoteAlertThreshold = 200000.0

type CostPerVoteResult struct {
	TotalExpense      float64 `json:"total_expense"`
	TotalEndorsements int64   `json:"total_endorsements"`
	CostPerVote       float64 `json:"cost_per_vote"`
	Alert             bool    `json:"alert"`
	AlertMessage      string  `json:"alert_message,omitempty"`
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

		var totalExpense float64
		_ = repository.DB.QueryRow(ctx, "SELECT COALESCE(SUM(amount), 0) FROM campaign_expenses WHERE campaign_id = $1", campaignID).Scan(&totalExpense)

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

		// F034 AC-1: Sync to UMKM Accounting engine
		go syncExpenseToAccounting(req.CampaignID, tenantID, expID, req.Amount, req.ExpenseCategory)

		// F034 AC-3: Auto-check CPV after recording expense
		go func() {
			_ = checkAndAlertCPV(ctx, req.CampaignID, tenantID)
		}()

		WriteJSON(w, http.StatusOK, APIResponse{
			Success: true,
			Data:    map[string]string{"expense_id": expID},
		})
		return
	}

	WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: response.MethodNotAllowed})
}

// syncExpenseToAccounting pushes expense record to UMKM Accounting engine via HTTP.
func syncExpenseToAccounting(campaignID, tenantID, expenseID string, amount float64, category string) {
	accountingURL := "http://localhost:8201/expenses"
	if AppConfig.Env == "production" || AppConfig.DB.Host == "postgres" {
		accountingURL = "http://umkm-accounting:8201/expenses"
	}

	payload := map[string]interface{}{
		"campaign_id":  campaignID,
		"expense_id":   expenseID,
		"amount":       amount,
		"category":     category,
		"tenant_id":    tenantID,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest(http.MethodPost, accountingURL, bytes.NewReader(body))
	if err != nil {
		slog.Error("Failed to create accounting sync request", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenantID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Warn("Accounting sync unreachable", "url", accountingURL, "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Warn("Accounting sync failed", "status", resp.StatusCode, "campaign", campaignID)
	}
}

// checkAndAlertCPV hitung CPV dan kirim alert jika melebihi threshold.
func checkAndAlertCPV(ctx context.Context, campaignID string, tenantID string) error {
	var totalExpense float64
	err := repository.DB.QueryRow(ctx,
		"SELECT COALESCE(SUM(amount), 0) FROM campaign_expenses WHERE campaign_id = $1",
		campaignID,
	).Scan(&totalExpense)
	if err != nil {
		return err
	}

	var totalEndorsements int64
	err = repository.DB.QueryRow(ctx,
		"SELECT COUNT(*) FROM endorsements WHERE campaign_id = $1 AND status = 'valid'",
		campaignID,
	).Scan(&totalEndorsements)
	if err != nil {
		return err
	}

	var cpv float64
	if totalEndorsements > 0 {
		cpv = totalExpense / float64(totalEndorsements)
	}

	if cpv > costPerVoteAlertThreshold {
		alertMsg := fmt.Sprintf("⚠️ CPV %.0f > threshold %.0f | Expense: %.0f | Voters: %d",
			cpv, costPerVoteAlertThreshold, totalExpense, totalEndorsements)
		slog.Warn("CPV ALERT", "campaign", campaignID, "cpv", cpv)

		_, err = repository.DB.Exec(ctx,
			"INSERT INTO notifications (tenant_id, title, message, type) VALUES ($1, $2, $3, $4)",
			tenantID, "CPV Alert", alertMsg, "cpv_alert",
		)
		if err != nil {
			slog.Error("Failed to save CPV alert", "error", err)
		}
	}

	return nil
}