package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// JobStatusResponse represents async job status
type JobStatusResponse struct {
	JobID       string                 `json:"job_id"`
	Type        string                 `json:"type"`
	Status      string                 `json:"status"`
	Data        map[string]interface{} `json:"data,omitempty"`
	Result      map[string]interface{} `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	CreatedAt   string                 `json:"created_at"`
	StartedAt   *string                `json:"started_at,omitempty"`
	CompletedAt *string                `json:"completed_at,omitempty"`
}

// APIResponse standard response format
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func writeJobJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// handleJobStatus returns status of an async job
func handleJobStatus(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJobJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
			return
		}

		tenantID := r.Header.Get("X-Tenant-ID")
		if tenantID == "" {
			writeJobJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID header"})
			return
		}

		// Extract job_id from path: /api/jobs/{job_id}
		pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(pathParts) < 3 {
			writeJobJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing job_id in path"})
			return
		}
		jobID := pathParts[2]

		var status JobStatusResponse
		var dataJSON, resultJSON []byte
		err := db.QueryRow(r.Context(), `
			SELECT job_id, type, status, data, result, error,
			       created_at, started_at, completed_at
			FROM async_jobs
			WHERE job_id = $1 AND tenant_id = $2
		`, jobID, tenantID).Scan(
			&status.JobID,
			&status.Type,
			&status.Status,
			&dataJSON,
			&resultJSON,
			&status.Error,
			&status.CreatedAt,
			&status.StartedAt,
			&status.CompletedAt,
		)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeJobJSON(w, http.StatusNotFound, APIResponse{Message: "Job not found"})
			} else {
				writeJobJSON(w, http.StatusInternalServerError, APIResponse{Message: "Internal server error"})
			}
			return
		}

		// Parse JSONB columns
		if len(dataJSON) > 0 {
			json.Unmarshal(dataJSON, &status.Data)
		}
		if len(resultJSON) > 0 {
			json.Unmarshal(resultJSON, &status.Result)
		}

		writeJobJSON(w, http.StatusOK, APIResponse{
			Success: true,
			Data:    status,
		})
	}
}
