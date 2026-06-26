package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

const (
	headerContentType    = "Content-Type"
	mimeApplicationJSON   = "application/json"
)

func setupQRHandler(container *sqlstore.Container) {
	http.HandleFunc("/api/wa/qr", func(w http.ResponseWriter, r *http.Request) {
		handleQRRequest(w, r, container)
	})
}

func handleQRRequest(w http.ResponseWriter, r *http.Request, container *sqlstore.Container) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	tenantID := extractTenantID(r)
	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "Missing X-Tenant-ID header or tenant_id query param")
		return
	}

	client := getOrCreateClient(tenantID, container)
	if client.Store.ID != nil {
		writeJSON(w, http.StatusOK, map[string]any{"status": "connected", "message": "Already connected"})
		return
	}

	qrChan, _ := client.GetQRChannel(context.Background())
	if err := client.Connect(); err != nil {
		slog.Error("Failed to connect for QR", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to connect")
		return
	}

	handleQRChannel(w, client, tenantID, qrChan)
}

func extractTenantID(r *http.Request) string {
	if tenantID := r.Header.Get("X-Tenant-ID"); tenantID != "" {
		return tenantID
	}
	return r.URL.Query().Get("tenant_id")
}

func handleQRChannel(w http.ResponseWriter, client *whatsmeow.Client, tenantID string, qrChan <-chan whatsmeow.QRChannelItem) {
	for evt := range qrChan {
		if evt.Event == "code" {
			handleQRCode(w, client, tenantID, evt.Code)
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "timeout", "message": "Failed to get QR code"})
}

func handleQRCode(w http.ResponseWriter, client *whatsmeow.Client, tenantID, code string) {
	png, err := qrcode.Encode(code, qrcode.Medium, 256)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate qr")
		return
	}
	base64QR := "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)

	writeJSON(w, http.StatusOK, map[string]any{
		"status":   "qr",
		"qr_code":  base64QR,
		"raw_code": code,
	})

	go scheduleQRExpiry(client, tenantID)
}

func scheduleQRExpiry(client *whatsmeow.Client, tenantID string) {
	time.Sleep(10 * time.Minute)
	if client.Store.ID == nil {
		client.Disconnect()
		clientMu.Lock()
		delete(clientMap, tenantID)
		clientMu.Unlock()
	}
}

func getOrCreateClient(tenantID string, container *sqlstore.Container) *whatsmeow.Client {
	clientMu.Lock()
	defer clientMu.Unlock()
	client, exists := clientMap[tenantID]
	if exists && client.Store.ID == nil {
		client.Disconnect()
		delete(clientMap, tenantID)
		exists = false
	}

	if !exists {
		newStore := container.NewDevice()
		clientLog := waLog.Stdout("Client-"+tenantID, "INFO", true)
		client = whatsmeow.NewClient(newStore, clientLog)
		client.AddEventHandler(func(evt any) { eventHandler(tenantID, evt) })
		clientMap[tenantID] = client
	}
	return client
}

func writeError(w http.ResponseWriter, status int, msg string) {
	http.Error(w, `{"error":"`+msg+`"}`, status)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
