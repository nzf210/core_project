package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
)

const (
	headerContentType = "Content-Type"
	contentTypeJSON   = "application/json"
	contentTypeText   = "text/plain; charset=utf-8"
)

func handleHealth(w http.ResponseWriter, r *http.Request) {
	clientMu.RLock()
	connected := len(clientMap)
	clientMu.RUnlock()

	redisStatus := "disconnected"
	if redisClient != nil {
		if redisClient.Ping(r.Context()).Err() == nil {
			redisStatus = "connected"
		}
	}

	dbStatus := "disconnected"
	if db != nil {
		if db.Ping() == nil {
			dbStatus = "connected"
		}
	}

	w.Header().Set(headerContentType, contentTypeJSON)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"status":             "ok",
		"instance_id":        instanceID,
		"connected_sessions": connected,
		"redis":             redisStatus,
		"database":           dbStatus,
	})
}

func handleHealthz(w http.ResponseWriter, r *http.Request) { handleHealth(w, r) }

func handleMetrics(w http.ResponseWriter, r *http.Request) {
	clientMu.RLock()
	n := len(clientMap)
	clientMu.RUnlock()

	w.Header().Set(headerContentType, contentTypeText)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `# HELP wa_gateway_info WA Gateway instance info
# TYPE wa_gateway_info gauge
wa_gateway_info{instance=%q} 1
# HELP wa_gateway_connected_sessions Current connected sessions
# TYPE wa_gateway_connected_sessions gauge
wa_gateway_connected_sessions{instance=%q} %d
`, instanceID, instanceID, n)
}

// restoreSessions loads and reconnects existing sessions from DB.
func restoreSessions(ctx context.Context, container *sqlstore.Container) {
	if db == nil {
		return
	}
	// Jitter to avoid race between replicas
	time.Sleep(time.Duration(1000+time.Now().UnixNano()%3000) * time.Millisecond)

	rows, err := db.Query(`SELECT tenant_id, jid FROM wa_tenant_sessions`)
	if err != nil {
		slog.Error("restoreSessions: query failed", "error", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var tID, jidStr string
		if rows.Scan(&tID, &jidStr) != nil {
			continue
		}
		if owned, _ := AcquireSessionLock(ctx, tID); !owned {
			slog.Info("restoreSessions: owned by another instance, skipping", "tenant_id", tID)
			continue
		}
		jid, _ := types.ParseJID(jidStr)
		device, _ := container.GetDevice(context.Background(), jid)
		if device == nil {
			ReleaseSessionLock(ctx, tID)
			continue
		}
		client := whatsmeow.NewClient(device, waLog.Stdout("Client-"+tID, "INFO", true))
		client.AddEventHandler(func(evt any) { eventHandler(tID, evt) })
		if err := client.Connect(); err == nil {
			clientMu.Lock()
			clientMap[tID] = client
			clientMu.Unlock()
			slog.Info("restoreSessions: session restored", "tenant_id", tID)
		} else {
			slog.Error("restoreSessions: connect failed", "tenant_id", tID, "error", err)
			ReleaseSessionLock(ctx, tID)
		}
	}
	if err := rows.Err(); err != nil {
		slog.Error("restoreSessions: rows iteration error", "error", err)
	}
}
