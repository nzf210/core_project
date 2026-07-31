package main

import (
	"net/http"
	"time"
	"core_project/shared/sdk/response"
)


func handleOCR(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: response.MethodNotAllowed})
		return
	}

	err := r.ParseMultipartForm(10 << 20) // 10 MB limit
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Failed to parse form"})
		return
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "No image provided"})
		return
	}
	defer file.Close()

	// Simulate AI Vision OCR processing latency
	time.Sleep(1500 * time.Millisecond)

	draft := map[string]interface{}{
		"date":        time.Now().Format("2006-01-02"),
		"description": "Pembelian Bahan Baku (Hasil Scan OCR)",
		"reference":   "OCR-" + time.Now().Format("150405"),
		"lines": []map[string]interface{}{
			{"account_id": "beban_bahan_baku", "debit": 150000, "credit": 0},
			{"account_id": "kas_kecil", "debit": 0, "credit": 150000},
		},
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Nota berhasil dipindai oleh AI",
		Data:    draft,
	})
}