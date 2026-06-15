package handlers

import (
	"context"
	"encoding/json"
	"math"
	"net/http"

	"core_project/apps/campaign/api/repository"
)

type SimulationRequest struct {
	CampaignID      string  `json:"campaign_id"`
	BudgetAllocated float64 `json:"budget_allocated"` // Slider Anggaran Baru (Rupiah)
	SelectedRegion  string  `json:"selected_region"`  // Level wilayah (misal Kecamatan ID)
}

func HandleSimulationWargame(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing context"})
		return
	}

	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
		return
	}

	var req SimulationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid payload"})
		return
	}

	ctx := context.Background()

	// 1. Ambil data historis pengeluaran
	var historicalExpense float64
	err := repository.DB.QueryRow(ctx, 
		"SELECT COALESCE(SUM(amount), 0) FROM campaign_expenses WHERE campaign_id = $1", 
		req.CampaignID,
	).Scan(&historicalExpense)
	if err != nil {
		historicalExpense = 0
	}

	// 2. Ambil data historis dukungan valid (suara terjamin)
	var historicalEndorsements int64
	err = repository.DB.QueryRow(ctx, 
		"SELECT COUNT(*) FROM endorsements WHERE campaign_id = $1 AND status = 'valid'", 
		req.CampaignID,
	).Scan(&historicalEndorsements)
	if err != nil {
		historicalEndorsements = 0
	}

	// 3. Ambil target voter regional
	var targetVoters int64
	err = repository.DB.QueryRow(ctx, 
		"SELECT COALESCE(SUM(target_voters), 1000) FROM campaigns WHERE id = $1", 
		req.CampaignID,
	).Scan(&targetVoters)
	if err != nil || targetVoters == 0 {
		targetVoters = 1000
	}

	// HITUNG HISTORICAL COST PER VOTE (CPV)
	cpv := 50000.0 // Default baseline Rp 50.000 per suara
	if historicalEndorsements > 0 && historicalExpense > 0 {
		cpv = historicalExpense / float64(historicalEndorsements)
	}

	// ALGORITMA SIMULASI WARGAME (Logarithmic Saturation Effect)
	// Hubungan Budget vs Penambahan Suara tidak linier (ada titik jenuh / diminishing returns).
	// Formula: Penambahan Suara = Target * (1 - e ^ (-Budget / (Target * CPV)))
	// Kita modifikasi agar budget baru memengaruhi probabilitas kemenangan.
	
	simulatedVotes := 0.0
	if cpv > 0 {
		scaleFactor := req.BudgetAllocated / (float64(targetVoters) * cpv)
		simulatedVotes = float64(targetVoters) * (1 - math.Exp(-scaleFactor))
	}

	// Pastikan ada baseline dukungan organik tanpa budget
	organicVotes := float64(historicalEndorsements)
	totalProjectedVotes := organicVotes + simulatedVotes
	if totalProjectedVotes > float64(targetVoters) {
		totalProjectedVotes = float64(targetVoters) // Cap di target DPT maks
	}

	// Hitung probabilitas menang (kemenangan 50% + 1 dari target)
	winThreshold := float64(targetVoters) / 2
	winProbability := 0.0
	if totalProjectedVotes > 0 {
		winProbability = (totalProjectedVotes / float64(targetVoters)) * 100
		if totalProjectedVotes >= winThreshold {
			// Boost probability jika melewati garis threshold 50%
			winProbability += 15
		}
		if winProbability > 100 {
			winProbability = 100
		}
	}

	// Hitung efisiensi biaya proyeksi (Projected Cost per Vote)
	projectedCPV := 0.0
	if totalProjectedVotes > 0 {
		projectedCPV = (historicalExpense + req.BudgetAllocated) / totalProjectedVotes
	}

	WriteJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"historical_cpv":          cpv,
			"projected_votes_added":   math.Round(simulatedVotes),
			"total_projected_votes":   math.Round(totalProjectedVotes),
			"win_probability_percent": math.Round(winProbability),
			"projected_cost_per_vote": math.Round(projectedCPV),
			"target_voters_total":     targetVoters,
		},
	})
}
