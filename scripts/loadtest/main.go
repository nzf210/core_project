package main

import (
	"bytes"
	"fmt"
	"net/http"
	"sync"
	"time"
)

func main() {
	fmt.Println("Starting Load Test on AI Gateway...")

	url := "http://localhost:8002/v1/chat"
	payload := []byte(`{"message": "Hello world, give me a quick tip.", "provider": "gemini"}`)

	concurrency := 50
	requestsPerWorker := 20
	totalRequests := concurrency * requestsPerWorker

	var wg sync.WaitGroup
	start := time.Now()

	successCount := 0
	failCount := 0
	var mu sync.Mutex

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := &http.Client{Timeout: 5 * time.Second}
			for j := 0; j < requestsPerWorker; j++ {
				req, _ := http.NewRequest("POST", url, bytes.NewBuffer(payload))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Tenant-ID", "tenant-test-load")

				resp, err := client.Do(req)
				if err != nil {
					mu.Lock()
					failCount++
					mu.Unlock()
					continue
				}
				
				resp.Body.Close()
				
				mu.Lock()
				if resp.StatusCode == 200 {
					successCount++
				} else {
					failCount++
				}
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	duration := time.Since(start)

	fmt.Println("=== Load Test Results ===")
	fmt.Printf("Total Requests: %d\n", totalRequests)
	fmt.Printf("Concurrency: %d\n", concurrency)
	fmt.Printf("Duration: %s\n", duration)
	fmt.Printf("Success: %d\n", successCount)
	fmt.Printf("Failures: %d\n", failCount)
	fmt.Printf("Req/Sec: %.2f\n", float64(totalRequests)/duration.Seconds())
}
