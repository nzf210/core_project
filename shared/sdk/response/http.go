package response

import (
	"encoding/json"
	"net/http"
)

type APIResponse struct {
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func JSON(w http.ResponseWriter, status int, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(APIResponse{
		Status:  status,
		Message: message,
		Data:    data,
	})
}

func Error(w http.ResponseWriter, status int, message string, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}

	_ = json.NewEncoder(w).Encode(APIResponse{
		Status:  status,
		Message: message,
		Error:   errStr,
	})
}
