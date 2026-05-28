package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

var (
	AccountingURL = "http://localhost:8201"
	ChatbotURL    = "http://localhost:8203"
	TenantID      = "sim-tenant-123"
)

func main() {
	fmt.Println("====================================================")
	fmt.Println("🚀 SIMULASI PHASE 2: OCR & CONVERSATIONAL ACCOUNTING")
	fmt.Println("====================================================")

	client := &http.Client{Timeout: 30 * time.Second}

	// ---------------------------------------------------------
	// 1. Simulate OCR Scanner
	// ---------------------------------------------------------
	fmt.Println("\n[1] Mencoba Simulasi AI OCR Scanner (Upload Nota)...")
	fmt.Println("    Mengirim file gambar (dummy) ke endpoint /ocr...")

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("image", "nota_dummy.jpg")
	part.Write([]byte("dummy image content"))
	writer.Close()

	reqOCR, _ := http.NewRequest("POST", AccountingURL+"/ocr", body)
	reqOCR.Header.Set("Content-Type", writer.FormDataContentType())
	reqOCR.Header.Set("X-Tenant-ID", TenantID)

	startOCR := time.Now()
	respOCR, err := client.Do(reqOCR)
	if err != nil {
		fmt.Println("❌ Gagal memanggil OCR:", err)
	} else {
		defer respOCR.Body.Close()
		ocrBody, _ := io.ReadAll(respOCR.Body)
		fmt.Printf("✅ Respons OCR diterima (Latency: %v):\n", time.Since(startOCR))
		fmt.Println("   " + string(ocrBody))
	}

	// ---------------------------------------------------------
	// 2. Simulate Conversational Accounting
	// ---------------------------------------------------------
	fmt.Println("\n[2] Mencoba Simulasi Conversational Accounting via Chatbot...")
	pesan := "Hari ini saya baru beli token listrik seharga 50 ribu rupiah"
	fmt.Printf("    User mengirim chat: \"%s\"\n", pesan)

	chatPayload := map[string]interface{}{
		"message": pesan,
	}
	chatBytes, _ := json.Marshal(chatPayload)

	reqChat, _ := http.NewRequest("POST", ChatbotURL+"/chat", bytes.NewBuffer(chatBytes))
	reqChat.Header.Set("Content-Type", "application/json")
	reqChat.Header.Set("X-Tenant-ID", TenantID)

	startChat := time.Now()
	respChat, err := client.Do(reqChat)
	if err != nil {
		fmt.Println("❌ Gagal memanggil Chatbot:", err)
	} else {
		defer respChat.Body.Close()
		chatBody, _ := io.ReadAll(respChat.Body)
		fmt.Printf("✅ Respons AI Chatbot diterima (Latency: %v):\n", time.Since(startChat))
		
		// Pretty print
		var chatResp map[string]interface{}
		json.Unmarshal(chatBody, &chatResp)
		prettyJSON, _ := json.MarshalIndent(chatResp, "   ", "  ")
		fmt.Println(string(prettyJSON))
	}

	fmt.Println("\n====================================================")
	fmt.Println("✨ SIMULASI SELESAI ✨")
	fmt.Println("====================================================")
}
