package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"core_project/shared/sdk/response"
)

// Send conflict alert to campaign admin via wa-gateway
func sendConflictAlert(tenantID string, citizenName, citizenNIK, ourRecruiter, enemyName string) {
	adminPhone := os.Getenv("CAMPAIGN_ADMIN_PHONE") // Get from config or DB
	if adminPhone == "" {
		slog.Warn("CAMPAIGN_ADMIN_PHONE not set, skipping conflict alert")
		return
	}

	msg := fmt.Sprintf("⚠️ *SENGKETA DATA PEMILIH*\n\nWarga bernama *%s* (NIK: %s) yang diinput oleh Relawan kita (*%s*) ternyata juga *diklaim* oleh kubu lawan (*%s*).\n\nMohon cross-check dan siapkan bukti dukungan fisik.", citizenName, citizenNIK, ourRecruiter, enemyName)

	sendWA(tenantID, adminPhone, msg)
}

// Handler exposed for internal Campaign backend to trigger alerts
func handleConflictAlertTrigger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, response.MethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		TenantID     string `json:"tenant_id"`
		CitizenName  string `json:"citizen_name"`
		CitizenNIK   string `json:"citizen_nik"`
		OurRecruiter string `json:"our_recruiter"`
		EnemyName    string `json:"enemy_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	go sendConflictAlert(payload.TenantID, payload.CitizenName, payload.CitizenNIK, payload.OurRecruiter, payload.EnemyName)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"success": true}`))
}
