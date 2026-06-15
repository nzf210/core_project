package handlers

import (
	"context"
	"net/http"
	"sort"

	"core_project/apps/campaign/api/repository"
)

// HandleSainteLague Simulator untuk Dapil (F043)
func HandleSainteLague(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusUnauthorized, APIResponse{
			Success: false,
			Message: "Unauthorized - Tenant ID missing",
		})
		return
	}

	dapilID := r.URL.Query().Get("dapil_id")
	if dapilID == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "dapil_id is required"})
		return
	}

	ctx := context.Background()

	// 1. Get Dapil total seats
	var totalSeats int
	err := repository.DB.QueryRow(ctx, "SELECT total_seats FROM dapils WHERE id = $1 AND tenant_id = $2", dapilID, tenantID).Scan(&totalSeats)
	if err != nil {
		WriteJSON(w, http.StatusNotFound, APIResponse{Message: "Dapil not found"})
		return
	}

	// 2. Get Real Our Valid Votes (App Data)
	var ourRealVotes int
	_ = repository.DB.QueryRow(ctx, `
		SELECT count(id) FROM endorsements 
		WHERE tenant_id = $1 AND is_anomaly = FALSE 
		-- in real prod, join with citizens to filter by dapil bounds
	`, tenantID).Scan(&ourRealVotes)

	// 3. Get Competitor Parties
	type Party struct {
		Name  string
		Votes int
		Seats int
	}
	parties := []Party{
		{Name: "Partai Kita (App)", Votes: ourRealVotes, Seats: 0},
	}

	rows, _ := repository.DB.Query(ctx, "SELECT party_name, estimated_votes FROM competitor_parties WHERE dapil_id = $1", dapilID)
	defer rows.Close()
	for rows.Next() {
		var p Party
		_ = rows.Scan(&p.Name, &p.Votes)
		parties = append(parties, p)
	}

	// 4. Sainte-Lague Calculation loop
	type SeatAllocation struct {
		SeatNumber int    `json:"seat_number"`
		PartyName  string `json:"party_name"`
		Divisor    int    `json:"divisor"`
		Score      int    `json:"score"`
	}
	allocations := []SeatAllocation{}

	for i := 1; i <= totalSeats; i++ {
		bestPartyIdx := 0
		maxScore := 0
		bestDivisor := 1

		for j, p := range parties {
			divisor := (p.Seats * 2) + 1 // 1, 3, 5, 7
			score := p.Votes / divisor
			if score > maxScore {
				maxScore = score
				bestPartyIdx = j
				bestDivisor = divisor
			}
		}

		allocations = append(allocations, SeatAllocation{
			SeatNumber: i,
			PartyName:  parties[bestPartyIdx].Name,
			Divisor:    bestDivisor,
			Score:      maxScore,
		})
		parties[bestPartyIdx].Seats++
	}

	// Sort final result
	sort.Slice(parties, func(i, j int) bool {
		return parties[i].Seats > parties[j].Seats
	})

	WriteJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"dapil_id":        dapilID,
			"total_seats":     totalSeats,
			"seat_allocations": allocations,
			"party_standings":  parties,
		},
	})
}
