package handlers

import (
	"context"
	"encoding/json"
	"math"
	"net/http"

	"core_project/apps/campaign/api/repository"
	"core_project/shared/sdk/response"
)

type SimulationRequest struct {
	CampaignID      string  `json:"campaign_id"`
	BudgetAllocated float64 `json:"budget_allocated"`
	SelectedRegion  string  `json:"selected_region"`
}

func HandleSimulationWargame(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing context"})
		return
	}

	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: response.MethodNotAllowed})
		return
	}

	var req SimulationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid payload"})
		return
	}

	ctx := context.Background()
	histExp, histEnd, target := fetchHistoricalData(ctx, req.CampaignID)
	cpv := calculateCPV(histExp, histEnd)
	simVotes := simulateVotes(req.BudgetAllocated, target, cpv)
	totalVotes := float64(histEnd) + simVotes
	if totalVotes > float64(target) {
		totalVotes = float64(target)
	}

	winProb := calculateWinProbability(totalVotes, target)
	projCPV := 0.0
	if totalVotes > 0 {
		projCPV = (histExp + req.BudgetAllocated) / totalVotes
	}

	WriteJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]any{
			"historical_cpv":          cpv,
			"projected_votes_added":   math.Round(simVotes),
			"total_projected_votes":   math.Round(totalVotes),
			"win_probability_percent": math.Round(winProb),
			"projected_cost_per_vote": math.Round(projCPV),
			"target_voters_total":     target,
		},
	})
}

func fetchHistoricalData(ctx context.Context, campaignID string) (float64, int64, int64) {
	var histExp float64
	repository.DB.QueryRow(ctx, "SELECT COALESCE(SUM(amount), 0) FROM campaign_expenses WHERE campaign_id = $1", campaignID).Scan(&histExp)

	var histEnd int64
	repository.DB.QueryRow(ctx, "SELECT COUNT(*) FROM endorsements WHERE campaign_id = $1 AND status = 'valid'", campaignID).Scan(&histEnd)

	var target int64 = 1000
	repository.DB.QueryRow(ctx, "SELECT COALESCE(SUM(target_voters), 1000) FROM campaigns WHERE id = $1", campaignID).Scan(&target)
	if target == 0 {
		target = 1000
	}
	return histExp, histEnd, target
}

func calculateCPV(histExp float64, histEnd int64) float64 {
	if histEnd > 0 && histExp > 0 {
		return histExp / float64(histEnd)
	}
	return 50000.0
}

func simulateVotes(budget float64, target int64, cpv float64) float64 {
	if cpv <= 0 {
		return 0
	}
	scale := budget / (float64(target) * cpv)
	return float64(target) * (1 - math.Exp(-scale))
}

func calculateWinProbability(totalVotes float64, target int64) float64 {
	if totalVotes == 0 {
		return 0
	}
	winProb := (totalVotes / float64(target)) * 100
	if totalVotes >= float64(target)/2 {
		winProb += 15
	}
	if winProb > 100 {
		return 100
	}
	return winProb
}
