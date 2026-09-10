package main

import (
	"context"
	"encoding/json"
	"net/http"

	"go.mau.fi/whatsmeow"
)

func setupStatusHandler() {
	http.HandleFunc("/api/wa/status", handleStatusRequest)
}

func handleStatusRequest(w http.ResponseWriter, r *http.Request) {
	tenantID := extractTenantID(r)
	w.Header().Set(headerContentType, contentTypeJSON)

	if tenantID == "" {
		writeStatus(w, "disconnected", "", "", "Missing tenant ID")
		return
	}

	if owner, isOther := checkOtherInstanceOwner(tenantID); isOther {
		jid := getSessionJIDFromDB(tenantID)
		writeStatus(w, "connected", jid, owner, "Session handled by another instance")
		return
	}

	if client, ok := getClientByTenant(tenantID); ok && client.Store.ID != nil {
		if client.IsConnected() {
			writeStatus(w, "connected", client.Store.ID.String(), "", "")
			return
		}
		// If client is reconnecting right after pair success, check DB
		if jid := getSessionJIDFromDB(tenantID); jid != "" {
			writeStatus(w, "connected", jid, "", "Session paired and reconnecting")
			return
		}
		writeStatus(w, "connecting", client.Store.ID.String(), "", "Session reconnecting")
		return
	}

	// Fallback: check if session is recorded in DB (e.g. freshly paired or restored)
	if jid := getSessionJIDFromDB(tenantID); jid != "" {
		writeStatus(w, "connected", jid, "", "Session active")
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
	if db == nil || tenantID == "" {
		return ""
	}
	var jid string
	err := db.QueryRow("SELECT jid FROM wa_tenant_sessions WHERE tenant_id = $1", tenantID).Scan(&jid)
	if err == nil && jid != "" {
		return jid
	}
	// Fallback to wa_sessions if status is connected
	var waNum string
	err = db.QueryRow("SELECT wa_number FROM wa_sessions WHERE tenant_id = $1::uuid AND status = 'connected' LIMIT 1", tenantID).Scan(&waNum)
	if err == nil && waNum != "" {
		return waNum + "@s.whatsapp.net"
	}
	return ""
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
