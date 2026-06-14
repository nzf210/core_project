// Package main — subscription-worker
//
// Background worker: cek tenant_subscriptions setiap interval (default 1 jam).
// Kalau current_period_end < NOW() dan status masih 'active' → freeze akun
// (status='frozen', tenants.is_frozen=true, kirim notifikasi).
//
// Grace period: 0 hari (langsung freeze). Bisa di-override via env GRACE_PERIOD_HOURS.
//
// Endpoint: GET /healthz untuk liveness check (port 8006).
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"core_project/shared/sdk/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

var DB *pgxpool.Pool

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := config.LoadConfig(".env")

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.DB.User, cfg.DB.Password, cfg.DB.Host, cfg.DB.Port, cfg.DB.Name, cfg.DB.SSLMode)
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		slog.Error("Failed to connect DB", "error", err)
		os.Exit(1)
	}
	if err := pool.Ping(context.Background()); err != nil {
		slog.Error("Failed to ping DB", "error", err)
		os.Exit(1)
	}
	DB = pool
	defer DB.Close()

	intervalStr := os.Getenv("FREEZE_CHECK_INTERVAL")
	interval := 1 * time.Hour
	if intervalStr != "" {
		if d, err := time.ParseDuration(intervalStr); err == nil {
			interval = d
		}
	}

	graceHours := 0
	if v := os.Getenv("GRACE_PERIOD_HOURS"); v != "" {
		fmt.Sscanf(v, "%d", &graceHours)
	}

	// Health endpoint
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := DB.Ping(r.Context()); err != nil {
			http.Error(w, "db down", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	go http.ListenAndServe(":8006", mux)

	slog.Info("Subscription worker started", "interval", interval.String(), "grace_hours", graceHours)

	// Monthly quota_counters cleanup — 1st of each month at 00:00 UTC.
	// Stdlib timer (no cron dep), runs alongside the freeze ticker.
	go scheduleMonthly(runQuotaCleanup)

	// Run once at startup, then on ticker
	runFreezePass(graceHours)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		runFreezePass(graceHours)
	}
}

// runQuotaCleanup deletes quota_counters rows whose reset_at is older than 60 days.
// 60 days = 2 months grace: a row whose reset_at is in the past by >60 days has been
// superseded by at least one full reset cycle. Redis cache has its own TTL, so this
// is safe to purge. Runs on the 1st of each month at 00:00 UTC via scheduleMonthly.
func runQuotaCleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	result, err := DB.Exec(ctx,
		`DELETE FROM quota_counters WHERE reset_at < NOW() - INTERVAL '60 days'`)
	if err != nil {
		slog.Error("Quota cleanup failed", "error", err)
		return
	}
	if rows := result.RowsAffected(); rows > 0 {
		slog.Info("Quota cleanup: archived old quota_counters rows", "rows", rows)
	} else {
		slog.Info("Quota cleanup: no old quota_counters rows to archive")
	}
}

// scheduleMonthly runs fn at 00:00 UTC on the 1st of every month, then re-arms itself.
// No external cron lib — stdlib time only, matching the existing ticker style.
func scheduleMonthly(fn func()) {
	for {
		now := time.Now().UTC()
		next := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)
		d := time.Until(next)
		slog.Info("Next quota cleanup scheduled", "at", next.Format(time.RFC3339), "in", d.String())
		time.Sleep(d)
		fn()
	}
}

func runFreezePass(graceHours int) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cutoff := time.Now().Add(-time.Duration(graceHours) * time.Hour)

	// Find subscriptions that should be frozen (paid subscriptions only)
	rows, err := DB.Query(ctx, `
		SELECT ts.tenant_id, t.name, t.plan, ts.current_period_end
		FROM tenant_subscriptions ts
		JOIN tenants t ON t.id = ts.tenant_id
		WHERE ts.status = 'active'
		  AND ts.current_period_end IS NOT NULL
		  AND ts.current_period_end < $1
		LIMIT 500
	`, cutoff)
	if err != nil {
		slog.Error("Freeze pass query failed", "error", err)
		return
	}

	type frozenItem struct {
		tenantID string
		name     string
		plan     string
		expiredAt time.Time
	}
	items := []frozenItem{}
	for rows.Next() {
		var it frozenItem
		var exp *time.Time
		if err := rows.Scan(&it.tenantID, &it.name, &it.plan, &exp); err == nil && exp != nil {
			it.expiredAt = *exp
			items = append(items, it)
		}
	}
	rows.Close()

	if len(items) == 0 {
		slog.Info("Freeze pass: nothing to freeze")
		return
	}

	// Batch update: set status=frozen, set tenants.is_frozen
	tx, err := DB.Begin(ctx)
	if err != nil {
		slog.Error("Freeze pass tx failed", "error", err)
		return
	}
	defer tx.Rollback(ctx)

	ids := make([]string, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.tenantID)
	}

	_, err = tx.Exec(ctx, `
		UPDATE tenant_subscriptions
		SET status = 'frozen', frozen_at = NOW(), frozen_reason = 'subscription period ended',
		    updated_at = NOW()
		WHERE tenant_id = ANY($1) AND status = 'active'
	`, ids)
	if err != nil {
		slog.Error("Freeze update subscriptions failed", "error", err)
		return
	}

	_, err = tx.Exec(ctx, `
		UPDATE tenants
		SET is_frozen = true, frozen_at = NOW()
		WHERE id = ANY($1)
	`, ids)
	if err != nil {
		slog.Error("Freeze update tenants failed", "error", err)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		slog.Error("Freeze commit failed", "error", err)
		return
	}

	slog.Info("Freeze pass: tenants frozen", "count", len(items))
}
