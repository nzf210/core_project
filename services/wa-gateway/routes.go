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
}
