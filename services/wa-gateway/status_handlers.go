package main

import (
	"context"
	"encoding/json"
	"net/http"

	"go.mau.fi/whatsmeow"
	"core_project/shared/sdk/response"
)

func setupStatusHandler() {
	http.HandleFunc("/api/wa/status", handleStatusRequest)
}

func handleStatusRequest(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get(response.XTenantID)
	w.Header().Set(headerContentType, contentTypeJSON)

	if owner, isOther := checkOtherInstanceOwner(tenantID); isOther {
		jid := getSessionJIDFromDB(tenantID)
		writeStatus(w, "connected", jid, owner, "Session handled by another instance")
		return
	}

	if client, ok := getClientByTenant(tenantID); ok && client.Store.ID != nil {
		writeStatus(w, "connected", client.Store.ID.String(), "", "")
		return
	}

	if jid := getSessionJIDFromDB(tenantID); jid != "" {
		writeStatus(w, "connected", jid, "", "")
		return
	}

	writeStatus(w, "disconnected", "", "", "")
}

func checkOtherInstanceOwner(tenantID string) (string, bool) {
	if redisClient == nil {
		return "", false
	}
	ownerKey := sessionOwnerPrefix + tenantID
	owner, err := redisClient.Get(context.Background(), ownerKey).Result()
	if err != nil || owner == "" || owner == instanceID {
		return "", false
	}
	return owner, true
}

func getClientByTenant(tenantID string) (*whatsmeow.Client, bool) {
	clientMu.RLock()
	defer clientMu.RUnlock()
	client, exists := clientMap[tenantID]
	return client, exists
}

func getSessionJIDFromDB(tenantID string) string {
	if db == nil {
		return ""
	}
	var jid string
	err := db.QueryRow("SELECT jid FROM wa_tenant_sessions WHERE tenant_id = $1", tenantID).Scan(&jid)
	if err != nil || jid == "" {
		return ""
	}
	return jid
}

func writeStatus(w http.ResponseWriter, status, jid, owner, msg string) {
	w.WriteHeader(http.StatusOK)
	resp := map[string]any{"status": status}
	if jid != "" {
		resp["jid"] = jid
	}
	if owner != "" {
		resp["owner"] = owner
	}
	if msg != "" {
		resp["message"] = msg
	}
	_ = json.NewEncoder(w).Encode(resp)
}
