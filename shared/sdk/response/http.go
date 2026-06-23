package response

import (
	"encoding/json"
	"net/http"
)

// SonarQube compliant constants
const (
	XUserID = "X-User-Id"
	XTenantID = "X-Tenant-ID"
	XUserRole = "X-User-Role"
	TimeFormatWIB = "02 Jan 2006, 15:04 WIB"
	ContentType     = "Content-Type"
	ApplicationJSON = "application/json"
	MethodNotAllowed = "Method not allowed"
	MissingTenantID = "Missing Tenant ID"
	MissingXTenantID = "Missing X-Tenant-ID"
	SuperadminOnly  = "Superadmin only"
	InvalidRequest  = "Invalid request"
	DBError         = "DB error"
)

type APIResponse struct {
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func JSON(w http.ResponseWriter, status int, message string, data interface{}) {
	w.Header().Set(ContentType, ApplicationJSON)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(APIResponse{
		Status:  status,
		Message: message,
		Data:    data,
	})
}

func Error(w http.ResponseWriter, status int, message string, err error) {
	w.Header().Set(ContentType, ApplicationJSON)
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
