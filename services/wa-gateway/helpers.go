package main

import (
	"context"
	"log/slog"
	"strings"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
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
	// Strip agent/device part (e.g. "user:9@lid" → "user@lid"). WhatsApp menolak
	// balasan ke JID yang punya device part: "message recipient must be a user JID
	// with no device part".
	parsed = parsed.ToNonAD()
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

// resolveSenderPhone resolves the actual phone number of the sender from a WhatsApp message event.
// WhatsApp modern multi-device protocol often uses @lid addressing instead of phone numbers.
// This function checks:
// 1. Direct phone JID (@s.whatsapp.net or @c.us)
// 2. SenderAlt JID provided in event info
// 3. whatsmeow_lid_map table in database
// 4. users table via wa_jid
func resolveSenderPhone(ctx context.Context, tenantID string, v *events.Message) string {
	sender := v.Info.Sender

	// 1. If sender is standard phone JID (@s.whatsapp.net or @c.us), User is the phone number
	if sender.Server == "s.whatsapp.net" || sender.Server == "c.us" {
		return sender.User
	}

	// 2. If sender is @lid or other server, check SenderAlt (alternative address provided by WhatsApp)
	if (v.Info.SenderAlt.Server == "s.whatsapp.net" || v.Info.SenderAlt.Server == "c.us") && v.Info.SenderAlt.User != "" {
		slog.Info("resolveSenderPhone: resolved via SenderAlt", "lid", sender.String(), "phone", v.Info.SenderAlt.User)
		return v.Info.SenderAlt.User
	}

	// 3. Try to query whatsmeow_lid_map directly in DB
	if db != nil {
		var pn string
		lidUser := strings.Split(strings.TrimSuffix(sender.User, "@lid"), ":")[0]
		err := db.QueryRowContext(ctx, "SELECT pn FROM whatsmeow_lid_map WHERE lid = $1", lidUser).Scan(&pn)
		if err == nil && pn != "" {
			slog.Info("resolveSenderPhone: resolved via whatsmeow_lid_map", "lid", lidUser, "phone", pn)
			return pn
		}
	}

	// 4. Try to query users table by wa_jid
	if db != nil {
		var registeredPhone string
		senderJID := sender.ToNonAD().String()
		err := db.QueryRowContext(ctx, "SELECT phone_number FROM users WHERE wa_jid = $1", senderJID).Scan(&registeredPhone)
		if err == nil && registeredPhone != "" {
			slog.Info("resolveSenderPhone: resolved via users table wa_jid", "jid", senderJID, "phone", registeredPhone)
			return registeredPhone
		}
	}

	// Fallback to sender.User
	return sender.User
}
