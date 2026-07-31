package handlers

import (
	"encoding/json"
	"net/http"
	"core_project/shared/sdk/response"
)

const errMissingTenantID = "Missing X-Tenant-ID"

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func ExtractTenantID(r *http.Request) string {
	return r.Header.Get(response.XTenantID)
}

func ExtractUserID(r *http.Request) string {
	return r.Header.Get("X-User-ID")
}

func ExtractUserRole(r *http.Request) string {
	return r.Header.Get("X-User-Role")
}
