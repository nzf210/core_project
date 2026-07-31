package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"core_project/shared/sdk/response"
)

const (
	errDBNotInitInternal = "DB not initialized"
	errDBInternal        = response.DBError
)


func handleInternalScheduledReports(w http.ResponseWriter, r *http.Request) {
	timeParam := r.URL.Query().Get("time")
	if timeParam == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing time parameter"})
		return
	}

	if DB == nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: errDBNotInitInternal})
		return
	}

	rows, err := DB.Query(r.Context(), "SELECT id, name, wa_number FROM tenants WHERE report_enabled = true AND report_time = $1", timeParam)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: errDBInternal})
		return
	}
	defer rows.Close()

	var tenants []map[string]interface{}
	for rows.Next() {
		var id, name string
		var waNumber *string
		if err := rows.Scan(&id, &name, &waNumber); err == nil {
			wn := ""
			if waNumber != nil {
				wn = *waNumber
			}
			tenants = append(tenants, map[string]interface{}{
				"id":        id,
				"name":      name,
				"wa_number": wn,
			})
		}
	}

	if tenants == nil {
		tenants = []map[string]interface{}{}
	}

	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: tenants})
}

func handleInternalReportsSummary(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing tenant_id"})
		return
	}

	if DB == nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: errDBNotInitInternal})
		return
	}

	// Today's boundaries
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local).Format("2006-01-02")

	query := `
		SELECT c.type, SUM(l.credit - l.debit) as balance
		FROM journal_lines l
		JOIN journal_entries e ON l.entry_id = e.id
		JOIN chart_of_accounts c ON l.account_id = c.id
		WHERE e.tenant_id = $1 AND c.type = 'revenue' AND e.date = $2
		GROUP BY c.type
	`
	rows, err := DB.Query(r.Context(), query, tenantID, startOfDay)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: errDBInternal})
		return
	}
	defer rows.Close()

	var totalRevenue int64
	for rows.Next() {
		var typ string
		var balance int64
		if err := rows.Scan(&typ, &balance); err == nil {
			totalRevenue += balance
		}
	}

	// Count transactions
	var txCount int
	err = DB.QueryRow(r.Context(), "SELECT COUNT(id) FROM journal_entries WHERE tenant_id = $1 AND date = $2", tenantID, startOfDay).Scan(&txCount)
	if err != nil {
		txCount = 0
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"tenant_id":     tenantID,
			"date":          startOfDay,
			"total_revenue": totalRevenue,
			"tx_count":      txCount,
		},
	})
}

func handleAutomations(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: response.MissingXTenantID})
		return
	}
	if DB == nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: errDBNotInitInternal})
		return
	}

	ctx := r.Context()

	switch r.Method {
	case http.MethodGet:
		listAutomationsGET(w, ctx, tenantID)
	case http.MethodPost:
		listAutomationsPOST(w, r, ctx, tenantID)
	case http.MethodPut:
		listAutomationsPUT(w, r, ctx, tenantID)
	case http.MethodDelete:
		listAutomationsDELETE(w, r, ctx, tenantID)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: response.MethodNotAllowed})
	}
}

func listAutomationsGET(w http.ResponseWriter, ctx context.Context, tenantID string) {
	rows, err := DB.Query(ctx, `SELECT id, type, name, enabled, cron_expression, config, target_wa, last_run_at, created_at FROM tenant_automations WHERE tenant_id = $1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		slog.Error("Failed to query tenant_automations", "error", err, "tenant_id", tenantID)
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: errDBInternal})
		return
	}
	defer rows.Close()

	var results []map[string]any
	for rows.Next() {
		var id, typ, name, cronExpr string
		var enabled bool
		var configJSON []byte
		var targetWA *string
		var lastRunAt *time.Time
		var createdAt time.Time
		if err := rows.Scan(&id, &typ, &name, &enabled, &cronExpr, &configJSON, &targetWA, &lastRunAt, &createdAt); err == nil {
			var cfg map[string]any
			json.Unmarshal(configJSON, &cfg)
			if cfg == nil {
				cfg = map[string]any{}
			}
			tw := ""
			if targetWA != nil {
				tw = *targetWA
			}
			lra := ""
			if lastRunAt != nil {
				lra = lastRunAt.Format(time.RFC3339)
			}
			results = append(results, map[string]any{
				"id": id, "type": typ, "name": name, "enabled": enabled,
				"cron_expression": cronExpr, "config": cfg, "target_wa": tw,
				"last_run_at": lra, "created_at": createdAt.Format(time.RFC3339),
			})
		}
	}
	if results == nil {
		results = []map[string]any{}
	}
	var plan string
	DB.QueryRow(ctx, "SELECT plan FROM tenants WHERE id = $1", tenantID).Scan(&plan)
	limit := getAutomationLimit(plan)
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]any{
		"automations": results, "plan": plan, "limit": limit, "count": len(results),
	}})
}

func listAutomationsPOST(w http.ResponseWriter, r *http.Request, ctx context.Context, tenantID string) {
	var plan string
	DB.QueryRow(ctx, "SELECT plan FROM tenants WHERE id = $1", tenantID).Scan(&plan)
	limit := getAutomationLimit(plan)

	var currentCount int
	DB.QueryRow(ctx, "SELECT COUNT(*) FROM tenant_automations WHERE tenant_id = $1", tenantID).Scan(&currentCount)

	if currentCount >= limit {
		msg := "Paket Anda tidak mendukung fitur automasi. Upgrade ke paket Lite atau lebih tinggi."
		if limit > 0 {
			msg = fmt.Sprintf("Batas automasi untuk paket %s adalah %d. Hapus automasi lama atau upgrade paket.", plan, limit)
		}
		writeJSON(w, http.StatusForbidden, APIResponse{Message: msg})
		return
	}

	var req struct {
		Type           string         `json:"type"`
		Name           string         `json:"name"`
		CronExpression string         `json:"cron_expression"`
		Config         map[string]any `json:"config"`
		TargetWA       string         `json:"target_wa"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid body"})
		return
	}
	if req.Type == "" || req.Name == "" || req.CronExpression == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "type, name, dan cron_expression wajib diisi"})
		return
	}

	configJSON, _ := json.Marshal(req.Config)
	if req.Config == nil {
		configJSON = []byte("{}")
	}

	var id string
	err := DB.QueryRow(ctx,
		"INSERT INTO tenant_automations (tenant_id, type, name, cron_expression, config, target_wa) VALUES ($1, $2, $3, $4, $5, NULLIF($6, '')) RETURNING id",
		tenantID, req.Type, req.Name, req.CronExpression, configJSON, req.TargetWA).Scan(&id)
	if err != nil {
		slog.Error("Failed to create automation", "error", err)
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Gagal membuat automasi"})
		return
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Automasi berhasil dibuat", Data: map[string]string{"id": id}})
}

func listAutomationsPUT(w http.ResponseWriter, r *http.Request, ctx context.Context, tenantID string) {
	automationID := r.URL.Query().Get("id")
	if automationID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing id"})
		return
	}
	var req struct {
		Name           string         `json:"name"`
		Enabled        *bool         `json:"enabled"`
		CronExpression string         `json:"cron_expression"`
		Config         map[string]any `json:"config"`
		TargetWA       string         `json:"target_wa"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid body"})
		return
	}

	sets := []string{"updated_at = NOW()"}
	args := []any{}
	argIdx := 1

	if req.Name != "" {
		sets = append(sets, fmt.Sprintf("name = $%d", argIdx)); args = append(args, req.Name); argIdx++
	}
	if req.Enabled != nil {
		sets = append(sets, fmt.Sprintf("enabled = $%d", argIdx)); args = append(args, *req.Enabled); argIdx++
	}
	if req.CronExpression != "" {
		sets = append(sets, fmt.Sprintf("cron_expression = $%d", argIdx)); args = append(args, req.CronExpression); argIdx++
	}
	if req.Config != nil {
		configJSON, _ := json.Marshal(req.Config)
		sets = append(sets, fmt.Sprintf("config = $%d", argIdx)); args = append(args, configJSON); argIdx++
	}
	if req.TargetWA != "" {
		sets = append(sets, fmt.Sprintf("target_wa = $%d", argIdx)); args = append(args, req.TargetWA); argIdx++
	}

	query := fmt.Sprintf("UPDATE tenant_automations SET %s WHERE id = $%d AND tenant_id = $%d",
		strings.Join(sets, ", "), argIdx, argIdx+1)
	args = append(args, automationID, tenantID)

	_, err := DB.Exec(ctx, query, args...)
	if err != nil {
		slog.Error("Failed to update automation", "error", err)
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Gagal mengupdate automasi"})
		return
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Automasi berhasil diupdate"})
}

func listAutomationsDELETE(w http.ResponseWriter, r *http.Request, ctx context.Context, tenantID string) {
	automationID := r.URL.Query().Get("id")
	if automationID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing id"})
		return
	}
	_, err := DB.Exec(ctx, "DELETE FROM tenant_automations WHERE id = $1 AND tenant_id = $2", automationID, tenantID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Gagal menghapus automasi"})
		return
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Automasi berhasil dihapus"})
}
