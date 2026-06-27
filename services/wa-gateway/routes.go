package main

import (
	"context"
	"net/http"

	"go.mau.fi/whatsmeow/store/sqlstore"
)

func setupRoutes(_ context.Context, container *sqlstore.Container) {
	setContainer(container)

	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/healthz", handleHealthz)
	http.HandleFunc("/metrics", handleMetrics)

	setupQRHandler(container)
	setupStatusHandler()
	setupSendHandler()
	setupLogoutHandler()

	// F063: Additional routes for superadmin WA Center via api-gateway proxy
	// API Gateway strips /api/superadmin → forwarded as /wa/* (but we need /api/wa/*)
	// Solution: register both /api/wa/* (original) AND /wa/* (for api-gateway strip)
	http.HandleFunc("/wa/status", handleStatusRequest)
	http.HandleFunc("/wa/qr", func(w http.ResponseWriter, r *http.Request) {
		handleQRRequest(w, r, container)
	})
	http.HandleFunc("/wa/logout", handleLogoutRequest)
}
