package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"core_project/apps/campaign/api/repository"
	"core_project/shared/sdk/response"
)

type LogisticItemPayload struct {
	Name          string `json:"name"`
	TotalQuantity int    `json:"total_quantity"`
	Unit          string `json:"unit"`
	CampaignID    string `json:"campaign_id"`
}

type DistributePayload struct {
	ItemID           string   `json:"item_id"`
	SenderID         string   `json:"sender_id"`
	ReceiverID       string   `json:"receiver_id"`
	TargetRegionType string   `json:"target_region_type"`
	TargetRegionID   string   `json:"target_region_id"`
	Quantity         int      `json:"quantity"`
	ProofImageURL    string   `json:"proof_image_url,omitempty"`
	Lat              *float64 `json:"lat,omitempty"`
	Lng              *float64 `json:"lng,omitempty"`
}

func HandleLogistics(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing context"})
		return
	}

	ctx := context.Background()

	if r.Method == http.MethodGet {
		// List logistic items
		rows, err := repository.DB.Query(ctx, 
			"SELECT id, name, total_quantity, unit, campaign_id FROM logistic_items WHERE tenant_id = $1", 
			tenantID,
		)
		if err != nil {
			WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB error"})
			return
		}
		defer rows.Close()

		var items []map[string]interface{}
		for rows.Next() {
			var id, name, unit, campaignID string
			var qty int
			if err := rows.Scan(&id, &name, &qty, &unit, &campaignID); err == nil {
				items = append(items, map[string]interface{}{
					"id":             id,
					"name":           name,
					"total_quantity": qty,
					"unit":           unit,
					"campaign_id":    campaignID,
				})
			}
		}
		if items == nil {
			items = []map[string]interface{}{}
		}
		WriteJSON(w, http.StatusOK, APIResponse{Success: true, Data: items})
		return
	}

	if r.Method == http.MethodPost {
		// Create logistic item
		var req LogisticItemPayload
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteJSON(w, http.StatusBadRequest, APIResponse{Message: response.InvalidRequest})
			return
		}

		query := `
			INSERT INTO logistic_items (tenant_id, campaign_id, name, total_quantity, unit)
			VALUES ($1, $2, $3, $4, $5) RETURNING id
		`
		var newID string
		err := repository.DB.QueryRow(ctx, query, tenantID, req.CampaignID, req.Name, req.TotalQuantity, req.Unit).Scan(&newID)
		if err != nil {
			WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to create item"})
			return
		}

		WriteJSON(w, http.StatusCreated, APIResponse{Success: true, Data: map[string]string{"id": newID}})
		return
	}

	WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: response.MethodNotAllowed})
}

func HandleDistributeLogistics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: response.MethodNotAllowed})
		return
	}

	var req DistributePayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: response.InvalidRequest})
		return
	}

	ctx := context.Background()

	// Dedopt/Verify quantity in stock
	var stock int
	err := repository.DB.QueryRow(ctx, "SELECT total_quantity FROM logistic_items WHERE id = $1", req.ItemID).Scan(&stock)
	if err != nil {
		WriteJSON(w, http.StatusNotFound, APIResponse{Message: "Item not found"})
		return
	}

	if stock < req.Quantity {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Insufficient stock"})
		return
	}

	// Record distribution
	query := `
		INSERT INTO logistic_distributions
		(item_id, sender_id, receiver_id, target_region_type, target_region_id, quantity, status, proof_image_url, location_lat, location_lng)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id
	`
	// If receiver is WA bot or auto-registered volunteer with proof, status received, else in_transit
	status := "in_transit"
	if req.ProofImageURL != "" {
		status = "received"
	}

	var distID string
	err = repository.DB.QueryRow(ctx, query,
		req.ItemID, req.SenderID, req.ReceiverID,
		req.TargetRegionType, req.TargetRegionID, req.Quantity,
		status, req.ProofImageURL, req.Lat, req.Lng,
	).Scan(&distID)

	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to record distribution"})
		return
	}

	// Update stock
	_, _ = repository.DB.Exec(ctx, "UPDATE logistic_items SET total_quantity = total_quantity - $1 WHERE id = $2", req.Quantity, req.ItemID)

	WriteJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"distribution_id": distID,
			"status":          status,
		},
	})
}

// HandleStalledDistributions — F033 AC-3: Cek distribusi yang terhenti > 2 hari
// GET /logistics/stalled?since_days=2
func HandleStalledDistributions(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing context"})
		return
	}

	sinceDays := 2
	if d := r.URL.Query().Get("since_days"); d != "" {
		fmt.Sscanf(d, "%d", &sinceDays)
	}

	ctx := context.Background()

	query := `
		SELECT d.id, d.item_id, li.name as item_name, d.quantity, d.status,
		       d.target_region_type, d.target_region_id, d.created_at,
		       v.name as receiver_name
		FROM logistic_distributions d
		JOIN logistic_items li ON li.id = d.item_id
		LEFT JOIN volunteers v ON v.id = d.receiver_id
		WHERE d.tenant_id = $1
		  AND d.status IN ('in_transit', 'pending')
		  AND d.created_at < NOW() - INTERVAL '$1 days'
		ORDER BY d.created_at ASC
	`

	rows, err := repository.DB.Query(ctx, query, tenantID, sinceDays)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB error"})
		return
	}
	defer rows.Close()

	var stalled []map[string]interface{}
	for rows.Next() {
		var id, itemID, itemName, status, regionType, regionID, receiverName string
		var quantity int
		var createdAt time.Time
		if err := rows.Scan(&id, &itemID, &itemName, &quantity, &status, &regionType, &regionID, &createdAt, &receiverName); err == nil {
			stalled = append(stalled, map[string]interface{}{
				"id":                id,
				"item_id":           itemID,
				"item_name":         itemName,
				"quantity":          quantity,
				"status":            status,
				"target_region_type": regionType,
				"target_region_id":  regionID,
				"receiver_name":     receiverName,
				"days_stalled":      int(time.Since(createdAt).Hours() / 24),
				"created_at":        createdAt.Format(time.RFC3339),
			})
		}
	}

	if stalled == nil {
		stalled = []map[string]interface{}{}
	}

	WriteJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"threshold_days": sinceDays,
			"stalled_count":  len(stalled),
			"distributions":  stalled,
		},
	})
}
