package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"core_project/shared/sdk/config"
	"core_project/shared/sdk/response"
)

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(response.ContentType, response.ApplicationJSON)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleN8NStatus(w http.ResponseWriter, r *http.Request) {
	status := N8NStatus{
		Status:          "unknown",
		Version:         "unknown",
		ActiveWorkflows: 0,
		QueueMode:       false,
		LastHealthCheck: time.Now().Format(time.RFC3339),
	}

	n8nURL := "http://localhost:5678/rest/healthz"
	client := &http.Client{Timeout: 5 * time.Second}

	resp, err := client.Get(n8nURL)
	if err != nil {
		status.Status = "disconnected"
		response.JSON(w, http.StatusOK, "ok", status)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		status.Status = "connected"
		var n8nResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&n8nResp); err == nil {
			if v, ok := n8nResp["version"].(string); ok {
				status.Version = v
			}
			if qm, ok := n8nResp["queueMode"].(bool); ok {
				status.QueueMode = qm
			}
		}
	} else {
		status.Status = "disconnected"
	}

	workflowsURL := "http://localhost:5678/rest/workflows?active=true"
	wfResp, err := client.Get(workflowsURL)
	if err == nil {
		defer wfResp.Body.Close()
		if wfResp.StatusCode == http.StatusOK {
			var workflows []map[string]interface{}
			if err := json.NewDecoder(wfResp.Body).Decode(&workflows); err == nil {
				status.ActiveWorkflows = len(workflows)
			}
		}
	}

	response.JSON(w, http.StatusOK, "ok", status)
}

func handleN8NExecutions(w http.ResponseWriter, r *http.Request) {
	n8nURL := "http://localhost:5678/rest/executions?limit=10&includeData=true"
	client := &http.Client{Timeout: 10 * time.Second}

	req, _ := http.NewRequest("GET", n8nURL, nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("admin:n8n_secure_admin_password_123")))

	resp, err := client.Do(req)
	if err != nil {
		response.JSON(w, http.StatusServiceUnavailable, "N8N unavailable", nil)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		response.JSON(w, http.StatusServiceUnavailable, "Failed to fetch executions", nil)
		return
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		response.JSON(w, http.StatusInternalServerError, "Failed to parse response", nil)
		return
	}

	data := result["data"]
	if data == nil {
		data = []interface{}{}
	}

	response.JSON(w, http.StatusOK, "ok", data)
}

func handleHealthStatus(w http.ResponseWriter, r *http.Request) {
	type svcHealth struct {
		Name    string `json:"name"`
		Port    string `json:"port"`
		Status  string `json:"status"`
		Metrics string `json:"metrics,omitempty"`
	}

	services := []struct {
		name string
		port string
	}{
		{"api-gateway", "8000"},
		{"auth-service", "8001"},
		{"ai-gateway", "8002"},
		{"billing-service", "8003"},
		{"notification-service", "8005"},
		{"wa-gateway", "8202"},
		{"umkm-accounting", "8201"},
		{"umkm-chatbot", "8202"},
		{"umkm-business", "9001"},
		{"campaign-api", "9002"},
	}

	getTargetURL := func(svcName, port, endpoint string) string {
		if Cfg.Env == "production" {
			return fmt.Sprintf("http://%s:%s%s", svcName, port, endpoint)
		}
		return fmt.Sprintf("http://localhost:%s%s", port, endpoint)
	}

	_, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	results := make([]svcHealth, 0, len(services))
	allUp := true

	for _, svc := range services {
		sh := svcHealth{Name: svc.name, Port: svc.port}

		metricsURL := getTargetURL(svc.name, svc.port, "/metrics")
		client := &http.Client{Timeout: 3 * time.Second}

		if resp, err := client.Get(metricsURL); err == nil && resp.StatusCode < 500 {
			defer resp.Body.Close()
			if body, err := io.ReadAll(resp.Body); err == nil {
				sh.Status = "up"
				sh.Metrics = string(body)
			}
		} else {
			healthURL := getTargetURL(svc.name, svc.port, "/health")
			if resp, err := client.Get(healthURL); err == nil {
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					sh.Status = "up"
				} else {
					sh.Status = "degraded"
					allUp = false
				}
			} else {
				sh.Status = "down"
				allUp = false
			}
		}

		results = append(results, sh)
	}

	overall := "healthy"
	if !allUp {
		overall = "degraded"
	}

	response.JSON(w, http.StatusOK, "ok", map[string]interface{}{
		"status":     overall,
		"env":        config.LoadConfig(".env").Env,
		"services":   results,
		"checked_at": time.Now().UTC().Format(time.RFC3339),
	})
}
