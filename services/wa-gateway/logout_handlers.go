package main

import (
	"encoding/json"
	"net/http"
)

func handleLogoutRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	tenantID := getTenantID(r)
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
		db.Exec(`DELETE FROM wa_tenant_sessions WHERE tenant_id = $1`, tenantID)
	}

	invalidatePlatformWAProviderCache()
	ReleaseSessionLock(r.Context(), tenantID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

func setupLogoutHandler() {
	http.HandleFunc("/api/wa/logout", handleLogoutRequest)
}
