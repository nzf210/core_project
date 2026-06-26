package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)


func validateWAConnectionForChatbot(ctx context.Context, pool *pgxpool.Pool, tenantID string) error {
	var exists bool

	// Check whatsmeow
	var whatsmeowExists bool
	err := pool.QueryRow(ctx, `SELECT EXISTS (
		SELECT 1 FROM wa_sessions 
		WHERE tenant_id = $1 AND status = 'connected'
	)`, tenantID).Scan(&whatsmeowExists)
	if err != nil {
		return fmt.Errorf("DB error wa_sessions: %v", err)
	}

	// Check cloud_api
	var cloudAPIExists bool
	err = pool.QueryRow(ctx, `SELECT EXISTS (
		SELECT 1 FROM wa_cloud_api_credentials 
		WHERE tenant_id = $1 AND is_active = true
	)`, tenantID).Scan(&cloudAPIExists)

	// Table wa_cloud_api_credentials might not exist if migration failed, but we assume it does based on schema
	if err != nil {
		return fmt.Errorf("DB error wa_cloud_api_credentials: %v", err)
	}

	exists = whatsmeowExists || cloudAPIExists

	if !exists {
		return fmt.Errorf("nomor WhatsApp (CS) belum terhubung, silakan hubungkan WhatsApp terlebih dahulu sebelum mengaktifkan Chatbot")
	}

	return nil
}