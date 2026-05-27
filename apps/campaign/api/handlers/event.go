package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"core_project/apps/campaign/api/repository"
)

type Event struct {
	ID         string    `json:"id"`
	CampaignID string    `json:"campaign_id"`
	Name       string    `json:"name"`
	EventDate  time.Time `json:"event_date"`
	Location   string    `json:"location"`
}

func HandleEvents(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
		return
	}

	if r.Method == http.MethodGet {
		rows, err := repository.DB.Query(context.Background(),
			"SELECT id, campaign_id, name, event_date, COALESCE(location, '') FROM events WHERE tenant_id = $1", tenantID)
		if err != nil {
			WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Database error"})
			return
		}
		defer rows.Close()

		var events []Event
		for rows.Next() {
			var e Event
			if err := rows.Scan(&e.ID, &e.CampaignID, &e.Name, &e.EventDate, &e.Location); err == nil {
				events = append(events, e)
			}
		}

		if events == nil {
			events = []Event{}
		}

		WriteJSON(w, http.StatusOK, APIResponse{Success: true, Data: events})
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			CampaignID string    `json:"campaign_id"`
			Name       string    `json:"name"`
			EventDate  time.Time `json:"event_date"`
			Location   string    `json:"location"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid request payload"})
			return
		}

		var id string
		err := repository.DB.QueryRow(context.Background(),
			"INSERT INTO events (tenant_id, campaign_id, name, event_date, location) VALUES ($1, $2, $3, $4, $5) RETURNING id",
			tenantID, req.CampaignID, req.Name, req.EventDate, req.Location).Scan(&id)
		
		if err != nil {
			WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to create event"})
			return
		}

		WriteJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Event created", Data: map[string]string{"id": id}})
		return
	}

	WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
}
