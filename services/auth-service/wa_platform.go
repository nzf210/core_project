package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
)

type PlatformWAProviderData struct {
	WAProvider        string                `json:"wa_provider"`
	EffectiveProvider string                `json:"effective_provider"`
	Reason            string                `json:"reason"`
	Connections       PlatformWAConnections `json:"connections"`
}

type PlatformWAConnections struct {
	Whatsmeow PlatformWAConnectionStatus `json:"whatsmeow"`
	CloudAPI  PlatformWACloudAPIStatus   `json:"cloud_api"`
}

type PlatformWAConnectionStatus struct {
	Connected bool   `json:"connected"`
	JID       string `json:"jid,omitempty"`
}

type PlatformWACloudAPIStatus struct {
	Active        bool   `json:"active"`
	PhoneNumberID string `json:"phone_number_id,omitempty"`
}

type SetPlatformProviderRequest struct {
	WAProvider string `json:"wa_provider"`
}

// getPlatformWAProvider detects which WA provider the platform (superadmin) is using.
// Priority: Redis manual override > auto-detect (Cloud API > whatsmeow).
// Auto-detect runs fresh each call — no caching needed (EXISTS queries are fast).
func getPlatformWAProvider(ctx context.Context) (string, string) {
	// 1. Manual override via Redis
	provider, err := Redis.Get(ctx, "platform:wa:provider").Result()
	if err == nil && provider != "" && provider != "auto" {
		return provider, "manual-override"
	}

	// 2. Auto-detect — fresh each call
	var whatsmeowConnected bool
	DB.QueryRow(ctx, `SELECT EXISTS(
		SELECT 1 FROM wa_tenant_sessions WHERE tenant_id IN ('verifier','system')
	)`).Scan(&whatsmeowConnected)

	var cloudAPIActive bool
	DB.QueryRow(ctx, `SELECT EXISTS(
		SELECT 1 FROM wa_cloud_api_credentials WHERE is_active = true
		AND (verification_status IS NULL OR verification_status = 'verified')
	)`).Scan(&cloudAPIActive)

	if cloudAPIActive && whatsmeowConnected {
		return "cloud_api", "auto-detect:both-connected-cloud-priority"
	}
	if cloudAPIActive {
		return "cloud_api", "auto-detect:cloud-api-only"
	}
	if whatsmeowConnected {
		return "whatsmeow", "auto-detect:whatsmeow-only"
	}
	return "whatsmeow", "auto-detect:no-connection-fallback"
}

func handleGetSetPlatformProvider(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleGetPlatformProvider(w, r)
	case http.MethodPut:
		handleSetPlatformProvider(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: msgMethodNotAllowed})
	}
}

func handleGetPlatformProvider(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Read Redis override
	overrideProvider, _ := Redis.Get(ctx, "platform:wa:provider").Result()
	effectiveProvider, reason := getPlatformWAProvider(ctx)

	var wsConnected bool
	var wsJID string
	DB.QueryRow(ctx, `SELECT EXISTS(
		SELECT 1 FROM wa_tenant_sessions WHERE tenant_id IN ('verifier','system')
	)`).Scan(&wsConnected)
	if wsConnected {
		DB.QueryRow(ctx, `SELECT COALESCE(jid, '') FROM wa_tenant_sessions WHERE tenant_id IN ('verifier','system') LIMIT 1`).Scan(&wsJID)
	}

	var caActive bool
	var caPhoneID string
	DB.QueryRow(ctx, `SELECT EXISTS(
		SELECT 1 FROM wa_cloud_api_credentials WHERE is_active = true
	)`).Scan(&caActive)
	if caActive {
		DB.QueryRow(ctx, `SELECT COALESCE(phone_number_id, '') FROM wa_cloud_api_credentials WHERE is_active = true LIMIT 1`).Scan(&caPhoneID)
	}

	waProvider := overrideProvider
	if waProvider == "" {
		waProvider = "auto"
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data: PlatformWAProviderData{
			WAProvider:        waProvider,
			EffectiveProvider: effectiveProvider,
			Reason:            reason,
			Connections: PlatformWAConnections{
				Whatsmeow: PlatformWAConnectionStatus{Connected: wsConnected, JID: wsJID},
				CloudAPI:  PlatformWACloudAPIStatus{Active: caActive, PhoneNumberID: caPhoneID},
			},
		},
	})
}

func handleSetPlatformProvider(w http.ResponseWriter, r *http.Request) {
	var req SetPlatformProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: msgInvalidPayload})
		return
	}

	if req.WAProvider != "auto" && req.WAProvider != "whatsmeow" && req.WAProvider != "cloud_api" {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Provider must be auto, whatsmeow, or cloud_api"})
		return
	}

	ctx := context.Background()
	if req.WAProvider == "auto" {
		Redis.Del(ctx, "platform:wa:provider")
	} else {
		Redis.Set(ctx, "platform:wa:provider", req.WAProvider, 0)
	}

	slog.Info("Platform WA provider updated", "provider", req.WAProvider)
	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Message: "Platform WA provider updated to " + req.WAProvider,
	})
}
