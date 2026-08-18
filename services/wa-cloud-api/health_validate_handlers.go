package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"core_project/shared/sdk/response"
)

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	if DB == nil {
		response.Error(w, http.StatusServiceUnavailable, "Database not connected", nil)
		return
	}
	if err := DB.Ping(r.Context()); err != nil {
		response.Error(w, http.StatusServiceUnavailable, "Database ping failed", err)
		return
	}
	response.JSON(w, http.StatusOK, "OK", map[string]string{"status": "healthy"})
}

func handleValidateCredential(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	var req struct {
		PhoneNumberID string `json:"phone_number_id"`
		AccessToken   string `json:"access_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.InvalidRequest, err)
		return
	}

	url := fmt.Sprintf("%s/%s/%s", graphBaseURL, graphVersion, req.PhoneNumberID)
	httpReq, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, url, nil)
	httpReq.Header.Set("Authorization", "Bearer "+req.AccessToken)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		response.Error(w, http.StatusBadGateway, "Failed to validate credential", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		response.Error(w, http.StatusUnauthorized, "Invalid credentials", nil)
		return
	}

	var result struct {
		DisplayPhoneNumber string `json:"display_phone_number"`
		VerifiedName       string `json:"verified_name"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	response.JSON(w, http.StatusOK, "Credential valid", map[string]string{
		"phone_number":  result.DisplayPhoneNumber,
		"verified_name": result.VerifiedName,
	})
}
