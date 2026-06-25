package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)


func handleFaqs(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	ctx := r.Context()

	if r.Method == http.MethodGet {
		rows, err := DB.Query(ctx, "SELECT id, question, answer FROM tenant_faqs WHERE tenant_id = $1 ORDER BY created_at ASC", tenantID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB error"})
			return
		}
		defer rows.Close()

		var faqs []map[string]string
		for rows.Next() {
			var id, question, answer string
			if err := rows.Scan(&id, &question, &answer); err == nil {
				faqs = append(faqs, map[string]string{"id": id, "question": question, "answer": answer})
			}
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: faqs})
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			Question string `json:"question"`
			Answer   string `json:"answer"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid input"})
			return
		}
		var newID string
		err := DB.QueryRow(ctx, "INSERT INTO tenant_faqs (tenant_id, question, answer) VALUES ($1, $2, $3) RETURNING id", tenantID, req.Question, req.Answer).Scan(&newID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Insert error"})
			return
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]string{"id": newID}})
		return
	}

	if r.Method == http.MethodPut {
		var req struct {
			ID       string `json:"id"`
			Question string `json:"question"`
			Answer   string `json:"answer"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid input"})
			return
		}
		if req.ID == "" {
			writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing id"})
			return
		}
		_, err := DB.Exec(ctx, "UPDATE tenant_faqs SET question = $1, answer = $2, updated_at = NOW() WHERE id = $3 AND tenant_id = $4", req.Question, req.Answer, req.ID, tenantID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Update error"})
			return
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "FAQ updated"})
		return
	}

	if r.Method == http.MethodDelete {
		id := r.URL.Query().Get("id")
		DB.Exec(ctx, "DELETE FROM tenant_faqs WHERE id = $1 AND tenant_id = $2", id, tenantID)
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Deleted"})
		return
	}

	writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
}

func handleFaqsGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
		return
	}
	tenantID := r.Header.Get("X-Tenant-ID")

	// Get tenant profile
	var tenantName string
	DB.QueryRow(r.Context(), "SELECT name FROM tenants WHERE id = $1", tenantID).Scan(&tenantName)

	prompt := fmt.Sprintf("Buatkan 3 pertanyaan FAQ dan jawabannya untuk toko bernama '%s'. Outputkan HANYA dalam format JSON array seperti: [{\"question\": \"...\", \"answer\": \"...\"}] tanpa markdown tambahan.", tenantName)

	aiReqBody := map[string]interface{}{
		"provider":   "minimax",
		"message":    prompt,
		"system_msg": "Anda adalah asisten pembuat FAQ.",
		"tenant_id":  tenantID,
	}

	payloadBytes, _ := json.Marshal(aiReqBody)
	reqHTTP, err := http.NewRequestWithContext(r.Context(), http.MethodPost, "http://ai-gateway:8002/v1/chat", bytes.NewBuffer(payloadBytes))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Message: "Gagal menyiapkan request ke AI"})
		return
	}
	reqHTTP.Header.Set("Content-Type", "application/json")
	reqHTTP.Header.Set("X-Tenant-ID", tenantID)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(reqHTTP)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Message: "AI Gateway tidak merespon"})
		return
	}
	defer resp.Body.Close()

	var aiResp struct {
		Success bool   `json:"success"`
		Text    string `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&aiResp); err != nil || !aiResp.Success {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Message: "Gagal memproses respon dari AI Gateway"})
		return
	}

	var generated []map[string]string
	if err := json.Unmarshal([]byte(aiResp.Text), &generated); err != nil {
		// Fallback to simple parse or default if AI returned malformed JSON
		generated = []map[string]string{
			{"question": "Berapa jam operasional toko?", "answer": "Kami buka dari jam 08:00 pagi hingga 20:00 malam."},
		}
	}

	for _, f := range generated {
		DB.Exec(r.Context(), "INSERT INTO tenant_faqs (tenant_id, question, answer) VALUES ($1, $2, $3)", tenantID, f["question"], f["answer"])
	}

	writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "FAQ awal berhasil di-generate."})
}