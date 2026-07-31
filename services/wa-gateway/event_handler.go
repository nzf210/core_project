package main

import (
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
)

var globalContainer *sqlstore.Container

func setContainer(c *sqlstore.Container) {
	globalContainer = c
}

// eventHandler handles WhatsApp events for a tenant
func eventHandler(tenantID string, evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		handleMessageEvent(tenantID, v)
	case *events.Connected:
		handleConnectedEvent(tenantID)
	case *events.Disconnected:
		handleDisconnectedEvent(tenantID)
	case *events.LoggedOut:
		handleLoggedOutEvent(tenantID)
	case *events.StreamReplaced:
		handleStreamReplacedEvent(tenantID)
	case *events.TemporaryBan:
		handleTemporaryBanEvent(tenantID, v)
	}
}

func handleStreamReplacedEvent(tenantID string) {
	slog.Warn("Stream replaced - device logged in elsewhere", "tenant_id", tenantID)
}

func handleTemporaryBanEvent(tenantID string, v *events.TemporaryBan) {
	slog.Error("Temporary ban received", "tenant_id", tenantID, "code", v.Code, "reason", v.Reason)
}
