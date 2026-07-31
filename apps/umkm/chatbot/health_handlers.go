package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

const (
	contentTypeTextPlain = "text/plain; version=0.0.4"
)

func handleHealth(w http.ResponseWriter, r *http.Request) {
	status := "ok"
	dbStatus := "disconnected"
	redisStatus := "disconnected"
	aiStatus := "unknown"

	// Check DB
	if DB != nil {
		if err := DB.Ping(r.Context()); err == nil {
			dbStatus = "connected"
		}
	}

	// Check Redis
	if redisClient != nil {
		if err := redisClient.Ping(r.Context()).Err(); err == nil {
			redisStatus = "connected"
		}
	}

	// Check AI Gateway
	if len(AIGatewayURL) > 0 {
		client := &http.Client{Timeout: 3 * time.Second}
		if resp, err := client.Get(AIGatewayURL + "/health"); err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				aiStatus = "connected"
			}
		}
	}

	// Overall status
	if dbStatus != "connected" || redisStatus != "connected" {
		status = "degraded"
	}

	w.Header().Set(headerContentType, contentTypeJSON)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     status,
		"database":   dbStatus,
		"redis":      redisStatus,
		"ai_gateway": aiStatus,
	})
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	handleHealth(w, r)
}

// handleMetrics returns Prometheus-compatible metrics for monitoring
func handleMetrics(w http.ResponseWriter, r *http.Request) {
	// Get queue depth
	var queueDepth int64
	if redisClient != nil {
		n, err := redisClient.LLen(r.Context(), "chatbot:queue").Result()
		if err == nil {
			queueDepth = n
		}
	}

	hostname, _ := os.Hostname()
	metrics := fmt.Sprintf(`# HELP chatbot_info Chatbot instance info
# TYPE chatbot_info gauge
chatbot_info{instance="%s"} 1

# HELP chatbot_messages_processed_total Total messages processed
# TYPE chatbot_messages_processed_total counter
chatbot_messages_processed_total{instance="%s"} %d

# HELP chatbot_llm_calls_total Total LLM API calls
# TYPE chatbot_llm_calls_total counter
chatbot_llm_calls_total{instance="%s"} %d

# HELP chatbot_errors_total Total errors
# TYPE chatbot_errors_total counter
chatbot_errors_total{instance="%s"} %d

# HELP chatbot_queue_depth Current job queue depth
# TYPE chatbot_queue_depth gauge
chatbot_queue_depth{instance="%s"} %d
`, hostname, hostname, chatbotMessagesProcessed, hostname, chatbotLLMCalls, hostname, chatbotErrors, hostname, queueDepth)

	w.Header().Set(headerContentType, contentTypeTextPlain)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(metrics))
}