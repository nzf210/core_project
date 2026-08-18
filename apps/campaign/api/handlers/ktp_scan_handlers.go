package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"core_project/apps/campaign/api/repository"
	"core_project/shared/sdk/response"
)

func HandleKTPScan(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing context"})
		return
	}

	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: response.MethodNotAllowed})
		return
	}

	var req struct {
		ImageURL   string `json:"image_url"`
		CampaignID string `json:"campaign_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: response.InvalidRequest})
		return
	}

	ocrText, err := callVisionOCR(req.ImageURL)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "AI Gateway unreachable"})
		return
	}
	if ocrText == "" {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "AI OCR failed"})
		return
	}

	ktpData, err := parseKTPData(ocrText)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "AI returned malformed data"})
		return
	}

	ctx := context.Background()
	tx, err := repository.DB.Begin(ctx)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Transaction start failed"})
		return
	}
	defer tx.Rollback(ctx)

	citizenID, err := upsertCitizenFromKTP(ctx, tx, ktpData)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to save citizen data"})
		return
	}

	if err := recordKTPEndorsement(ctx, tx, citizenID, tenantID, req.CampaignID, req.ImageURL); err != nil {
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

