package handlers

import (
	"context"
	"net/http"

	"core_project/apps/campaign/api/repository"
)

type Province struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func HandleProvinces(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		rows, err := repository.DB.Query(context.Background(), "SELECT id, name FROM provinces")
		if err != nil {
			WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Database error"})
			return
		}
		defer rows.Close()

		var items []Province
		for rows.Next() {
			var p Province
			if err := rows.Scan(&p.ID, &p.Name); err == nil {
				items = append(items, p)
			}
		}

		if items == nil {
			items = []Province{}
		}

		WriteJSON(w, http.StatusOK, APIResponse{Success: true, Data: items})
		return
	}

	WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
}
