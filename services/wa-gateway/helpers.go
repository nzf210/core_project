package main

import (
	"context"
	"log/slog"
	"strings"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

const helperContentTypeJSON = "application/json"

// sendWAMessage sends a plain text WhatsApp message to a JID via the tenant's whatsmeow client.
func sendWAMessage(tenantID, jid, text string) {
	client := getWAClient(tenantID)
	if client == nil {
		slog.Warn("sendWAMessage: no client for tenant", "tenant_id", tenantID)
		return
	}
	parsed, err := types.ParseJID(jid)
	if err != nil {
		slog.Error("sendWAMessage: invalid JID", "jid", jid, "error", err)
		return
	}
	_, err = client.SendMessage(context.Background(), parsed, &waE2E.Message{
		Conversation: proto.String(text),
	})
	if err != nil {
		slog.Error("sendWAMessage: failed", "tenant_id", tenantID, "jid", jid, "error", err)
	}
}

// extractPhoneFromJID extracts the phone number from a WhatsApp JID.
// "62812123456789@s.whatsapp.net" → "0812123456789"
func extractPhoneFromJID(jid string) string {
	if jid == "" {
		return ""
	}
	at := strings.Index(jid, "@")
	var user string
	if at >= 0 {
		user = jid[:at]
	} else {
		user = jid
	}
	if strings.HasPrefix(user, "62") {
		return "0" + user[2:]
	}
	return user
}

// invalidatePlatformWAProviderCache clears the cached WA provider preference for all tenants
// so that the next request re-reads from DB (e.g. after a connect/disconnect event).
func invalidatePlatformWAProviderCache() {
	if redisShared == nil {
		return
	}
	ctx := context.Background()
	keys, err := redisShared.Keys(ctx, "wa:provider:*").Result()
	if err != nil || len(keys) == 0 {
		return
	}
	redisShared.Del(ctx, keys...)
}
