package main

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
)

func restoreSingleSession(tenantID string) {
	if globalContainer == nil || db == nil {
		slog.Warn("restoreSingleSession: cannot restore, container or db nil")
		return
	}

	var jidStr string
	err := db.QueryRow(`SELECT jid FROM wa_tenant_sessions WHERE tenant_id = $1`, tenantID).Scan(&jidStr)
	if err != nil || jidStr == "" {
		slog.Warn("restoreSingleSession: no session in DB", "tenant_id", tenantID)
		return
	}

	ctx := context.Background()
	if owned, _ := AcquireSessionLock(ctx, tenantID); !owned {
		slog.Info("restoreSingleSession: session owned by another instance, skipping", "tenant_id", tenantID)
		return
	}

	jid, _ := types.ParseJID(jidStr)
	device, _ := globalContainer.GetDevice(ctx, jid)
	if device == nil {
		slog.Warn("restoreSingleSession: no device in whatsmeow store", "tenant_id", tenantID)
		ReleaseSessionLock(ctx, tenantID)
		return
	}

	client := whatsmeow.NewClient(device, waLog.Stdout("Client-"+tenantID, "INFO", true))
	client.AddEventHandler(func(evt interface{}) { eventHandler(tenantID, evt) })
	if err := client.Connect(); err == nil {
		clientMu.Lock()
		clientMap[tenantID] = client
		clientMu.Unlock()
		slog.Info("restoreSingleSession: session restored", "tenant_id", tenantID)
	} else {
		slog.Error("restoreSingleSession: failed to connect", "tenant_id", tenantID, "error", err)
		ReleaseSessionLock(ctx, tenantID)
	}
}

func handleConnectedEvent(tenantID string) {
	slog.Info("Connected to WhatsApp", "tenant_id", tenantID)
	invalidatePlatformWAProviderCache()
	clientMu.RLock()
	c := clientMap[tenantID]
	clientMu.RUnlock()
	if c != nil && c.Store.ID != nil && db != nil {
		if _, err := db.Exec(`INSERT INTO wa_tenant_sessions (tenant_id, jid) VALUES ($1, $2) ON CONFLICT (tenant_id) DO UPDATE SET jid = EXCLUDED.jid`, tenantID, c.Store.ID.String()); err != nil {
			slog.Error("Failed to save wa_tenant_sessions on connect", "tenant_id", tenantID, "error", err)
		}
	}
}

func handleDisconnectedEvent(tenantID string) {
	slog.Warn("Disconnected from WhatsApp", "tenant_id", tenantID)
	invalidatePlatformWAProviderCache()
}

func handleLoggedOutEvent(tenantID string) {
	slog.Warn("Logged out from WhatsApp", "tenant_id", tenantID)
	invalidatePlatformWAProviderCache()
	clientMu.Lock()
	delete(clientMap, tenantID)
	clientMu.Unlock()
	if db != nil {
		db.Exec(`DELETE FROM wa_tenant_sessions WHERE tenant_id = $1`, tenantID)
	}
	ctx := context.Background()
	ReleaseSessionLock(ctx, tenantID)
}

func mapUserJIDIfNeeded(senderJID, senderPhone string) {
	if db == nil || senderJID == "" || senderPhone == "" {
		return
	}
	phone := senderPhone
	if strings.HasPrefix(phone, "62") {
		phone = "0" + phone[2:]
	}
	if _, err := db.Exec(`UPDATE users SET wa_jid = $1 WHERE phone_number = $2 AND (wa_jid IS NULL OR wa_jid = '')`,
		senderJID, phone); err != nil {
		slog.Error("Failed to update user wa_jid", "phone", phone, "error", err)
	}
}

func isSixDigitOTP(s string) bool {
	if len(s) != 6 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
