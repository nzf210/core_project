package handlers

import (
	"context"
	"fmt"
	"net/http"
	"sort"

	"core_project/apps/campaign/api/repository"
)

// parliamentaryThreshold — ambang batas parlemen (4%) per UU Pileg Indonesia.
// Parties below this threshold are excluded from seat allocation.
const parliamentaryThreshold = 0.04

// HandleSainteLague Simulator untuk Dapil (F043).
// GET /wargame/sainte-lague?dapil_id=...  → single dapil detailed simulation
// GET /wargame/sainte-lague                → multi-dapil dashboard (returns list of all tenant dapils)
func HandleSainteLague(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusUnauthorized, APIResponse{
			Success: false,
			Message: "Unauthorized - Tenant ID missing",
		})
		return
	}

	ctx := context.Background()
	dapilID := r.URL.Query().Get("dapil_id")

	// Multi-dapil mode: no dapil_id → return all dapils for tenant
	if dapilID == "" {
		simulateAllDapils(w, ctx, tenantID)
		return
	}

	// Single dapil detailed mode
	simulateSingleDapil(w, ctx, tenantID, dapilID)
}

func simulateAllDapils(w http.ResponseWriter, ctx context.Context, tenantID string) {
	rows, err := repository.DB.Query(ctx, `
		SELECT id, name, total_seats
		FROM dapils
		WHERE tenant_id = $1
		ORDER BY name ASC
	`, tenantID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to load dapils"})
		return
	}
	defer rows.Close()

	type DapilSummary struct {
		DapilID    string `json:"dapil_id"`
		Name       string `json:"name"`
		TotalSeats int    `json:"total_seats"`
	}

	var summaries []DapilSummary
	for rows.Next() {
		var d DapilSummary
		if err := rows.Scan(&d.DapilID, &d.Name, &d.TotalSeats); err == nil {
			summaries = append(summaries, d)
		}
	}
	if summaries == nil {
		summaries = []DapilSummary{}
	}

	WriteJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"mode":         "multi_dapil",
			"dapil_count":  len(summaries),
			"dapils":       summaries,
			"hint":         "Pass ?dapil_id=<id> for detailed Sainte-Laguë simulation",
		},
	})
}

func simulateSingleDapil(w http.ResponseWriter, ctx context.Context, tenantID, dapilID string) {
	dapilName, totalSeats, err := getDapilInfo(ctx, tenantID, dapilID)
	if err != nil {
		WriteJSON(w, http.StatusNotFound, APIResponse{Message: "Dapil not found"})
		return
	}

	parties := loadParties(ctx, tenantID, dapilID)
	totalVotes := sumVotes(parties)

	if totalVotes == 0 {
		WriteJSON(w, http.StatusOK, APIResponse{
			Success: true,
			Data: map[string]interface{}{
				"mode":        "single_dapil",
				"dapil_id":    dapilID,
				"name":        dapilName,
				"total_seats": totalSeats,
				"message":     "No votes recorded yet",
			},
		})
		return
	}

	eligible := filterEligibleParties(parties, totalVotes)
	allocations := allocateSeats(eligible, totalSeats)
	standings := buildStandings(parties, totalVotes)

	sort.Slice(standings, func(i, j int) bool {
		if standings[i].Seats != standings[j].Seats {
			return standings[i].Seats > standings[j].Seats
		}
		return standings[i].Votes > standings[j].Votes
	})

	allocatedSeats := sumSeats(parties)

	WriteJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"mode":              "single_dapil",
			"dapil_id":          dapilID,
			"name":              dapilName,
			"total_seats":       totalSeats,
			"allocated_seats":   allocatedSeats,
			"remaining_seats":   totalSeats - allocatedSeats,
			"total_votes":       totalVotes,
			"threshold_percent": parliamentaryThreshold * 100,
			"seat_allocations":  allocations,
			"party_standings":   standings,
			"algorithm":         fmt.Sprintf("Sainte-Laguë (divisor: 1, 3, 5, 7, ... ; threshold: %.0f%%)", parliamentaryThreshold*100),
		},
	})
}

type Party struct {
	Name  string
	Votes int
	Seats int
}

type SeatAllocation struct {
	SeatNumber int    `json:"seat_number"`
	PartyName  string `json:"party_name"`
	Divisor    int    `json:"divisor"`
	Score      int    `json:"score"`
}

type PartyStanding struct {
	Name           string  `json:"name"`
	Votes          int     `json:"votes"`
	Seats          int     `json:"seats"`
	VoteShare      float64 `json:"vote_share"`
	AboveThreshold bool    `json:"above_threshold"`
}

func getDapilInfo(ctx context.Context, tenantID, dapilID string) (string, int, error) {
	var dapilName string
	var totalSeats int
	err := repository.DB.QueryRow(ctx,
		"SELECT name, total_seats FROM dapils WHERE id = $1 AND tenant_id = $2",
		dapilID, tenantID,
	).Scan(&dapilName, &totalSeats)
	return dapilName, totalSeats, err
}

func loadParties(ctx context.Context, tenantID, dapilID string) []Party {
	var ourRealVotes int
	_ = repository.DB.QueryRow(ctx, `
		SELECT count(id) FROM endorsements
		WHERE tenant_id = $1 AND is_anomaly = FALSE
	`, tenantID).Scan(&ourRealVotes)

	parties := []Party{
		{Name: "Partai Kita (App)", Votes: ourRealVotes, Seats: 0},
	}

	compRows, _ := repository.DB.Query(ctx,
		"SELECT party_name, estimated_votes FROM competitor_parties WHERE dapil_id = $1",
		dapilID,
	)
	if compRows != nil {
		defer compRows.Close()
		for compRows.Next() {
			var p Party
			if err := compRows.Scan(&p.Name, &p.Votes); err == nil {
				parties = append(parties, p)
			}
		}
	}
	return parties
}

func sumVotes(parties []Party) int {
	total := 0
	for _, p := range parties {
		total += p.Votes
	}
	return total
}

func sumSeats(parties []Party) int {
	total := 0
	for _, p := range parties {
		total += p.Seats
	}
	return total
}

func filterEligibleParties(parties []Party, totalVotes int) []Party {
	eligible := []Party{}
	for _, p := range parties {
		if float64(p.Votes)/float64(totalVotes) >= parliamentaryThreshold {
			eligible = append(eligible, p)
		}
	}
	return eligible
}

func allocateSeats(eligible []Party, totalSeats int) []SeatAllocation {
	allocations := []SeatAllocation{}

	for i := 1; i <= totalSeats; i++ {
		bestIdx := -1
		bestScore := 0
		bestDivisor := 1

		for j := range eligible {
			divisor := (eligible[j].Seats * 2) + 1
			score := eligible[j].Votes / divisor
			if score > bestScore {
				bestScore = score
				bestIdx = j
				bestDivisor = divisor
			}
		}

		if bestIdx == -1 {
			break
		}

		allocations = append(allocations, SeatAllocation{
			SeatNumber: i,
			PartyName:  eligible[bestIdx].Name,
			Divisor:    bestDivisor,
			Score:      bestScore,
		})
		eligible[bestIdx].Seats++
	}
	return allocations
}

func buildStandings(parties []Party, totalVotes int) []PartyStanding {
	standings := []PartyStanding{}
	for _, p := range parties {
		share := 0.0
		if totalVotes > 0 {
			share = float64(p.Votes) / float64(totalVotes) * 100
		}
		standings = append(standings, PartyStanding{
			Name:           p.Name,
			Votes:          p.Votes,
			Seats:          p.Seats,
			VoteShare:      share,
			AboveThreshold: share/100 >= parliamentaryThreshold,
		})
	}
	return standings
}