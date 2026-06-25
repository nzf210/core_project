package main

import (
	"net/http"
	"time"

	"core_project/shared/sdk/auth"
	"core_project/shared/sdk/response"
)

func handleListTickets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	tenantID, ok := r.Context().Value(auth.TenantIDKey).(string)
	if !ok || tenantID == "" {
		response.Error(w, http.StatusUnauthorized, response.MissingTenantID, nil)
		return
	}

	ctx := r.Context()
	rows, err := DB.Query(ctx, `
		SELECT id, ticket_number, plan_id, plan_name, status,
		       activated_at, expires_at,
		       notify_wa, notify_telegram, notify_email,
		       wa_sent_at, telegram_sent_at, email_sent_at
		FROM subscription_tickets
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT 50
	`, tenantID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to fetch tickets", err)
		return
	}
	defer rows.Close()

	var tickets []map[string]interface{}
	for rows.Next() {
		var id, ticketNum, planID, planName, status string
		var activatedAt, expiresAt *time.Time
		var notifyWA, notifyTelegram, notifyEmail bool
		var waSentAt, tgSentAt, emailSentAt *time.Time

		if rows.Scan(&id, &ticketNum, &planID, &planName, &status, &activatedAt, &expiresAt,
			&notifyWA, &notifyTelegram, &notifyEmail, &waSentAt, &tgSentAt, &emailSentAt) != nil {
			continue
		}

		tickets = append(tickets, map[string]interface{}{
			"id":               id,
			"ticket_number":    ticketNum,
			"plan_id":          planID,
			"plan_name":        planName,
			"status":           status,
			"activated_at":     formatTime(activatedAt),
			"expires_at":       formatTime(expiresAt),
			"notify_wa":        notifyWA,
			"notify_telegram":  notifyTelegram,
			"notify_email":     notifyEmail,
			"wa_sent_at":       formatTime(waSentAt),
			"telegram_sent_at": formatTime(tgSentAt),
			"email_sent_at":    formatTime(emailSentAt),
		})
	}

	response.JSON(w, http.StatusOK, "Tickets retrieved", tickets)
}
