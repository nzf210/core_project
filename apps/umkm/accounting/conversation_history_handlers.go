package main

import (
	"net/http"
	"strconv"
	"time"

	"core_project/shared/sdk/response"
)

const (
	defaultConversationHistoryLimit = 10
	maxConversationHistoryLimit     = 50
)

type conversationHistoryMessage struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

func handleInternalConversationHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: response.MethodNotAllowed})
		return
	}

	tenantID := r.URL.Query().Get("tenant_id")
	customerID := r.URL.Query().Get("customer_id")
	if tenantID == "" || customerID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "tenant_id and customer_id required"})
		return
	}

	limit, err := parseConversationHistoryLimit(r.URL.Query().Get("limit"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "limit must be between 1 and 50"})
		return
	}

	rows, err := DB.Query(r.Context(), `
		WITH latest_session AS (
			SELECT id
			FROM conversation_sessions
			WHERE tenant_id = $1 AND customer_id = $2 AND status = 'active'
			ORDER BY last_message_at DESC
			LIMIT 1
		), recent_messages AS (
			SELECT logs.id, logs.role, logs.content, logs.created_at
			FROM conversation_logs AS logs
			JOIN latest_session ON latest_session.id = logs.session_id
			ORDER BY logs.created_at DESC, logs.id DESC
			LIMIT $3
		)
		SELECT role, content, created_at
		FROM recent_messages
		ORDER BY created_at ASC, id ASC
	`, tenantID, customerID, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: response.DBError})
		return
	}
	defer rows.Close()

	history := make([]conversationHistoryMessage, 0, limit)
	for rows.Next() {
		var message conversationHistoryMessage
		if err := rows.Scan(&message.Role, &message.Content, &message.CreatedAt); err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: response.DBError})
			return
		}
		history = append(history, message)
	}
	if err := rows.Err(); err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: response.DBError})
		return
	}

	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: history})
}

func parseConversationHistoryLimit(raw string) (int, error) {
	if raw == "" {
		return defaultConversationHistoryLimit, nil
	}

	limit, err := strconv.Atoi(raw)
	if err != nil || limit < 1 || limit > maxConversationHistoryLimit {
		return 0, strconv.ErrSyntax
	}
	return limit, nil
}
