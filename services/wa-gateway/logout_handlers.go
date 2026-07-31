package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"core_project/shared/sdk/response"
)

func handleLogoutRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	tenantID := r.Header.Get(response.XTenantID)
	if tenantID == "" {
		tenantID = r.URL.Query().Get("tenant_id")
	}
	if tenantID == "" {
		http.Error(w, `{"error":"tenant_id required"}`, http.StatusBadRequest)
		return
	}

	clientMu.Lock()
	client, exists := clientMap[tenantID]
	if exists {
		client.Disconnect()
		delete(clientMap, tenantID)
	}
	clientMu.Unlock()

	if db != nil {
		if _, err := db.Exec(`DELETE FROM wa_tenant_sessions WHERE tenant_id = $1`, tenantID); err != nil {
			slog.Error("Failed to delete wa_tenant_sessions on logout", "tenant_id", tenantID, "error", err)
		}
	}

	invalidatePlatformWAProviderCache()
	ReleaseSessionLock(r.Context(), tenantID)

	w.Header().Set(headerContentType, contentTypeJSON)
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

func setupLogoutHandler() {
	http.HandleFunc("/api/wa/logout", handleLogoutRequest)
}
