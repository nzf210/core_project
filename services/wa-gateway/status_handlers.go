package main

import (
	"encoding/json"
	"net/http"
)

func setupStatusHandler() {
	http.HandleFunc("/api/wa/status", func(w http.ResponseWriter, r *http.Request) {
		tenantID := getTenantID(r)
		w.Header().Set("Content-Type", "application/json")

		if redisClient != nil {
			ownerKey := sessionOwnerPrefix + tenantID
			owner, err := redisClient.Get(r.Context(), ownerKey).Result()
			if err == nil && owner != instanceID {
				var jid string
				if db != nil {
					db.QueryRow("SELECT jid FROM wa_tenant_sessions WHERE tenant_id = $1", tenantID).Scan(&jid)
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"status":  "connected",
					"jid":     jid,
					"owner":   owner,
					"message": "Session handled by another instance",
				})
				return
			}
		}

		clientMu.RLock()
		client, exists := clientMap[tenantID]
		clientMu.RUnlock()

		if !exists || client.Store.ID == nil {
			if db != nil {
				var jid string
				err := db.QueryRow("SELECT jid FROM wa_tenant_sessions WHERE tenant_id = $1", tenantID).Scan(&jid)
				if err == nil && jid != "" {
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"status": "connected",
						"jid":    jid,
					})
					return
				}
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "disconnected",
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "connected",
			"jid":    client.Store.ID.String(),
		})
	})
}
