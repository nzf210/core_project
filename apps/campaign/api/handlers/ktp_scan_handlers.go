package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"core_project/apps/campaign/api/repository"
)

func HandleKTPScan(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing context"})
		return
	}

	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
		return
	}

	var req struct {
		ImageURL   string `json:"image_url"`
		CampaignID string `json:"campaign_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid request"})
		return
	}

	// Call AI Gateway Vision OCR
	visionPayload := map[string]string{
		"image_url": req.ImageURL,
		"prompt":    "Extract KTP data",
	}
	body, _ := json.Marshal(visionPayload)
	
	resp, err := http.Post(visionGatewayURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "AI Gateway unreachable"})
		return
	}
	defer resp.Body.Close()

	var visionResp struct {
		Success bool `json:"success"`
		Data    struct {
			Text string `json:"text"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&visionResp)

	if !visionResp.Success {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "AI OCR failed"})
		return
	}

	// Parse the structured JSON returned from mock vision
	var ktpData struct {
		NIK     string `json:"nik"`
		Name    string `json:"name"`
		Address string `json:"address"`
		Gender  string `json:"gender"`
		Age     int    `json:"age"`
	}
	if err := json.Unmarshal([]byte(visionResp.Data.Text), &ktpData); err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "AI returned malformed data"})
		return
	}

	// Auto-Register the citizen and endorsement — wrap in transaction to prevent orphan records
	ctx := context.Background()
	tx, err := repository.DB.Begin(ctx)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Transaction start failed"})
		return
	}
	defer tx.Rollback(ctx)

	var citizenID string
	queryCitizen := `
		INSERT INTO citizens (nik, name, address, gender, age)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (nik) DO UPDATE SET
			name = EXCLUDED.name,
			address = EXCLUDED.address,
			updated_at = NOW()
		RETURNING id
	`
	err = tx.QueryRow(ctx, queryCitizen, ktpData.NIK, ktpData.Name, ktpData.Address, ktpData.Gender, ktpData.Age).Scan(&citizenID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to save citizen data"})
		return
	}

	// Record endorsement
	queryEndorsement := `
		INSERT INTO endorsements (citizen_id, tenant_id, campaign_id, proof_image_url, status)
		VALUES ($1, $2, $3, $4, 'valid')
	`
	_, err = tx.Exec(ctx, queryEndorsement, citizenID, tenantID, req.CampaignID, req.ImageURL)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to record endorsement"})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Transaction commit failed"})
		return
	}

	WriteJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "KTP Scanned and Registered",
		Data:    ktpData,
	})
}
