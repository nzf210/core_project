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

	"core_project/shared/observability"
	"core_project/shared/sdk/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	DB *pgxpool.Pool

	// Business metrics
	subscriptionWorkerRunsTotal = observability.NewCounter(
		"subscription_worker_runs_total",
		"Total subscription worker runs by action",
		[]string{"action"},
	)
)

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

	// Prometheus metrics endpoint
	mux.Handle("/metrics", observability.PrometheusHandler())

	// Wrap handler with observability middleware
	handler := observability.Middleware("subscription-worker")(mux)

	go http.ListenAndServe(":8006", handler)

	slog.Info("Subscription worker started", "interval", interval.String(), "grace_hours", graceHours)

	// Monthly quota_counters cleanup — 1st of each month at 00:00 UTC.
	// Stdlib timer (no cron dep), runs alongside the freeze ticker.
	go scheduleMonthly(runQuotaCleanup)

	// Daily anomaly detection — runs at 02:00 UTC every day via scheduleDaily.
	go scheduleDaily(runAnomalyDetection)

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

// scheduleDaily runs fn at 02:00 UTC every day, then re-arms itself.
func scheduleDaily(fn func()) {
	for {
		now := time.Now().UTC()
		next := time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, time.UTC)
		if next.Before(now) {
			next = next.Add(24 * time.Hour)
		}
		d := time.Until(next)
		slog.Info("Next anomaly detection scheduled", "at", next.Format(time.RFC3339), "in", d.String())
		time.Sleep(d)
		fn()
	}
}

// runAnomalyDetection runs the F039 anomaly detection queries across all active tenants.
// Mirrors the logic in apps/campaign/api/handlers/anomaly.go but uses the worker's DB pool.
func runAnomalyDetection() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	rows, err := DB.Query(ctx, "SELECT DISTINCT tenant_id FROM endorsements WHERE status = 'valid'")
	if err != nil {
		slog.Error("Anomaly detection: failed to fetch tenants", "error", err)
		return
	}
	defer rows.Close()

	var tenants []string
	for rows.Next() {
		var tid string
		if err := rows.Scan(&tid); err == nil {
			tenants = append(tenants, tid)
		}
	}

	totalAnomalies := 0
	for _, tenantID := range tenants {
		// Age > 100 (siluman)
		_, _ = DB.Exec(ctx, `
			UPDATE endorsements e SET is_anomaly=TRUE, anomaly_reason='Usia Terindikasi > 100 Tahun (Siluman)'
			FROM citizens c WHERE e.citizen_id=c.id AND e.tenant_id=$1 AND e.is_anomaly=FALSE
			AND CAST(SUBSTRING(c.nik FROM 9 FOR 2) AS INTEGER) >= 20
			AND CAST(SUBSTRING(c.nik FROM 9 FOR 2) AS INTEGER) <= 25
		`, tenantID)

		// Burst > 500/jam
		_, _ = DB.Exec(ctx, `
			WITH burst_recruiters AS (
				SELECT recruiter_id FROM endorsements
				WHERE tenant_id=$1 GROUP BY recruiter_id, date_trunc('hour',created_at)
				HAVING count(id)>500
			)
			UPDATE endorsements SET is_anomaly=TRUE, anomaly_reason='Terdeteksi Joki/Bot: Burst Insert > 500/jam'
			WHERE tenant_id=$1 AND recruiter_id IN (SELECT recruiter_id FROM burst_recruiters) AND is_anomaly=FALSE
		`, tenantID)

		// Regional mismatch
		_, _ = DB.Exec(ctx, `
			UPDATE endorsements e SET is_anomaly=TRUE, anomaly_reason='NIK regional code mismatch dengan TPS region'
			FROM citizens c LEFT JOIN tps_records t ON t.id=e.tps_id
			WHERE e.tenant_id=$1 AND e.is_anomaly=FALSE AND c.nik IS NOT NULL
			AND t.region_code IS NOT NULL AND LENGTH(c.nik)>=6
			AND SUBSTRING(c.nik FROM 1 FOR 6) != t.region_code
		`, tenantID)

		var count int
		_ = DB.QueryRow(ctx, "SELECT COUNT(*) FROM endorsements WHERE tenant_id=$1 AND is_anomaly=TRUE", tenantID).Scan(&count)
		totalAnomalies += count
	}

	slog.Info("Anomaly detection finished", "tenants_checked", len(tenants), "total_anomalies", totalAnomalies)
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
		subscriptionWorkerRunsTotal.WithLabelValues("check_pending").Inc()
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

	subscriptionWorkerRunsTotal.WithLabelValues("hold_expired").Inc()
	slog.Info("Freeze pass: tenants frozen", "count", len(items))
}
