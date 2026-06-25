package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
)


func handleSeed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
		return
	}
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
		return
	}

	// Basic SAK-EMKM
	accounts := []struct {
		Code, Name, Type string
	}{
		{"100", "Kas", "asset"},
		{"101", "Bank / QRIS", "asset"},
		{"110", "Piutang Usaha", "asset"},
		{"120", "Persediaan", "asset"},
		{"200", "Hutang Usaha", "liability"},
		{"300", "Modal", "equity"},
		{"400", "Pendapatan Usaha", "revenue"},
		{"500", "Beban Operasional", "expense"},
	}

	ctx := context.Background()
	for _, acc := range accounts {
		_, err := DB.Exec(ctx,
			"INSERT INTO chart_of_accounts (tenant_id, code, name, type) VALUES ($1, $2, $3, $4) ON CONFLICT (tenant_id, code) DO NOTHING",
			tenantID, acc.Code, acc.Name, acc.Type,
		)
		if err != nil {
			slog.Error("Failed to seed account", "error", err)
		}
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Seeded successfully"})
}

func handleAccounts(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
		return
	}

	if r.Method == http.MethodGet {
		// List accounts
		rows, err := DB.Query(r.Context(), "SELECT id, code, name, type FROM chart_of_accounts WHERE tenant_id = $1", tenantID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB error"})
			return
		}
		defer rows.Close()

		var results []map[string]interface{}
		for rows.Next() {
			var id, code, name, typ string
			if err := rows.Scan(&id, &code, &name, &typ); err == nil {
				results = append(results, map[string]interface{}{
					"id": id, "code": code, "name": name, "type": typ,
				})
			}
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: results})
		return
	}

	if r.Method == http.MethodPost {
		var req AccountReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid body"})
			return
		}

		var parent interface{}
		if req.ParentID != "" {
			parent = req.ParentID
		}

		var id string
		err := DB.QueryRow(r.Context(),
			"INSERT INTO chart_of_accounts (tenant_id, code, name, type, parent_id) VALUES ($1, $2, $3, $4, $5) RETURNING id",
			tenantID, req.Code, req.Name, req.Type, parent).Scan(&id)

		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Insert failed"})
			return
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]string{"id": id}})
		return
	}

	if r.Method == http.MethodDelete {
		accID := r.URL.Query().Get("id")
		if accID == "" {
			writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing id parameter"})
			return
		}

		var balance float64
		err := DB.QueryRow(r.Context(), "SELECT balance FROM chart_of_accounts WHERE id = $1 AND tenant_id = $2", accID, tenantID).Scan(&balance)
		if err == nil && balance != 0 {
			writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Tidak dapat menghapus akun yang memiliki saldo"})
			return
		}
		var count int
		err = DB.QueryRow(r.Context(), "SELECT count(*) FROM journal_lines WHERE account_id = $1 AND tenant_id = $2", accID, tenantID).Scan(&count)
		if err == nil && count > 0 {
			writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Tidak dapat menghapus akun yang memiliki riwayat jurnal"})
			return
		}

		_, err = DB.Exec(r.Context(), "DELETE FROM chart_of_accounts WHERE id = $1 AND tenant_id = $2", accID, tenantID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Delete failed"})
			return
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Account deleted"})
		return
	}
}