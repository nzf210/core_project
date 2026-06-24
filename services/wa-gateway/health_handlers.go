package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
)

func handleHealth(w http.ResponseWriter, r *http.Request) {
	clientMu.RLock()
	connected := len(clientMap)
	clientMu.RUnlock()

	redisStatus := "disconnected"
	if redisClient != nil {
		if err := redisClient.Ping(r.Context()).Err(); err == nil {
			redisStatus = "connected"
		}
	}

	dbStatus := "disconnected"
	if db != nil {
		if err := db.Ping(); err == nil {
			dbStatus = "connected"
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":             "ok",
		"instance_id":        instanceID,
		"connected_sessions": connected,
		"redis":              redisStatus,
		"database":           dbStatus,
	})
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	handleHealth(w, r)
}

// handleMetrics returns Prometheus-compatible metrics
func handleMetrics(w http.ResponseWriter, r *http.Request) {
	clientMu.RLock()
	connectedSessions := len(clientMap)
	clientMu.RUnlock()

	totalTenants := int64(0)
	if redisClient != nil {
		keys, _ := redisClient.Keys(r.Context(), "wa:owner:*").Result()
		totalTenants = int64(len(keys))
	}

	activeInstances := int64(0)
	if redisClient != nil {
		keys, _ := redisClient.Keys(r.Context(), "wa:instance:*").Result()
		activeInstances = int64(len(keys))
	}

	metrics := fmt.Sprintf(`# HELP wa_gateway_info WA Gateway instance info
# TYPE wa_gateway_info gauge
wa_gateway_info{instance="%s"} 1

# HELP wa_gateway_connected_sessions Current connected WhatsApp sessions
# TYPE wa_gateway_connected_sessions gauge
wa_gateway_connected_sessions{instance="%s"} %d

# HELP wa_gateway_total_tenants Total tenants with sessions
# TYPE wa_gateway_total_tenants gauge
wa_gateway_total_tenants %d

# HELP wa_gateway_active_instances Number of active gateway instances
# TYPE wa_gateway_active_instances gauge
wa_gateway_active_instances %d

# HELP wa_gateway_messages_sent_total Total messages sent
# TYPE wa_gateway_messages_sent_total counter
wa_gateway_messages_sent_total{instance="%s"} %d

# HELP wa_gateway_messages_received_total Total messages received
# TYPE wa_gateway_messages_received_total counter
wa_gateway_messages_received_total{instance="%s"} %d

# HELP wa_gateway_errors_total Total errors
# TYPE wa_gateway_errors_total counter
wa_gateway_errors_total{instance="%s"} %d
`, instanceID, instanceID, connectedSessions, totalTenants, activeInstances, instanceID, waMessagesSent, instanceID, waMessagesRecv, instanceID, waErrorsTotal)

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(metrics))
}

// restoreSessions loads and reconnects existing sessions from DB
func restoreSessions(ctx context.Context, container *sqlstore.Container) {
	if db == nil {
		return
	}

	// Add jitter delay to avoid race condition between replicas
	delayMs := 1000 + (time.Now().UnixNano() % 3000)
	time.Sleep(time.Duration(delayMs) * time.Millisecond)

	rows, err := db.Query(`SELECT tenant_id, jid FROM wa_tenant_sessions`)
	if err != nil {
		log.Printf("Failed to query sessions: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var tID, jidStr string
		if err := rows.Scan(&tID, &jidStr); err != nil {
			continue
		}

		if owned, _ := AcquireSessionLock(ctx, tID); !owned {
			log.Printf("Session for tenant %s owned by another instance, skipping", tID)
			continue
		}

		jid, _ := types.ParseJID(jidStr)
		device, _ := container.GetDevice(context.Background(), jid)
		if device != nil {
			client := whatsmeow.NewClient(device, waLog.Stdout("Client-"+tID, "INFO", true))
			client.AddEventHandler(func(evt interface{}) { eventHandler(tID, evt) })
			if err := client.Connect(); err == nil {
				clientMu.Lock()
				clientMap[tID] = client
				clientMu.Unlock()
				log.Printf("Restored session for tenant %s", tID)
			} else {
				log.Printf("Failed to restore session for tenant %s: %v", tID, err)
				ReleaseSessionLock(ctx, tID)
			}
		}
	}
}
