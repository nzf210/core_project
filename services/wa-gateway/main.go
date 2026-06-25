package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"

	_ "github.com/lib/pq"
)

// Redis key prefixes
const (
	sessionLockPrefix    = "wa:lock:"       // Distributed lock per tenant
	sessionOwnerPrefix   = "wa:owner:"      // Which instance owns the session
	sessionTTL           = 5 * time.Minute  // Lock TTL
	instanceHeartbeatKey = "wa:instance:%s" // This instance's heartbeat
)

// Metrics counters
var (
	waMessagesSent   int64
	waMessagesRecv   int64
	waSessionsActive int64
	waErrorsTotal    int64
)

var (
	clientMap    = make(map[string]*whatsmeow.Client)
	clientMu     sync.RWMutex
	redisClient  *redis.Client
	instanceID   string
	db           *sql.DB
	dbURI        string
	rateLimiter  = NewTenantRateLimiter(5)

	reconnectMu       sync.Mutex
	reconnectAttempts = make(map[string]int)
	reconnectBackoff  = make(map[string]time.Time)
)

func init() {
	hostname, _ := os.Hostname()
	instanceID = fmt.Sprintf("%s-%d", hostname, os.Getpid())
}

// ─────────────────────────────────────────────
// Rate Limiter
// ─────────────────────────────────────────────

type TenantRateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*tokenBucket
	rate    int
}

type tokenBucket struct {
	tokens   float64
	lastTime time.Time
}

func NewTenantRateLimiter(msgPerMinute int) *TenantRateLimiter {
	return &TenantRateLimiter{buckets: make(map[string]*tokenBucket), rate: msgPerMinute}
}

func (rl *TenantRateLimiter) Allow(tenantID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, exists := rl.buckets[tenantID]
	now := time.Now()

	if !exists {
		b = &tokenBucket{tokens: float64(rl.rate), lastTime: now}
		rl.buckets[tenantID] = b
	}

	elapsed := now.Sub(b.lastTime).Minutes()
	b.tokens = math.Min(float64(rl.rate), b.tokens+elapsed*float64(rl.rate))
	b.lastTime = now

	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

// ─────────────────────────────────────────────
// Reconnect Backoff
// ─────────────────────────────────────────────

func shouldReconnect(tenantID string) bool {
	reconnectMu.Lock()
	defer reconnectMu.Unlock()

	if lastAttempt, ok := reconnectBackoff[tenantID]; ok {
		if time.Since(lastAttempt) < 5*time.Minute {
			if attempts := reconnectAttempts[tenantID]; attempts > 0 {
				return false
			}
		}
	}

	attempts := reconnectAttempts[tenantID]
	if attempts > 5 {
		attempts = 5
	}

	backoff := time.Duration(30*(1<<attempts)) * time.Second
	if lastAttempt, ok := reconnectBackoff[tenantID]; ok {
		if time.Since(lastAttempt) < backoff {
			return false
		}
	}

	reconnectAttempts[tenantID] = attempts + 1
	reconnectBackoff[tenantID] = time.Now()
	return true
}

func resetReconnectBackoff(tenantID string) {
	reconnectMu.Lock()
	defer reconnectMu.Unlock()
	delete(reconnectAttempts, tenantID)
	delete(reconnectBackoff, tenantID)
}

func main() {
	_ = godotenv.Load(".env")
	_ = godotenv.Load("../../.env")

	dbURI = getDBURI()
	redisClient = initRedis()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				Heartbeat(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()

	dbLog := waLog.Stdout("Database", "WARN", true)
	container, err := sqlstore.New(context.Background(), "postgres", dbURI, dbLog)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer container.Close()

	db, err = sql.Open("postgres", dbURI)
	if err == nil {
		_, err = db.Exec(`CREATE TABLE IF NOT EXISTS wa_tenant_sessions (tenant_id TEXT PRIMARY KEY, jid VARCHAR NOT NULL)`)
		if err != nil {
			slog.Error("Failed to create wa_tenant_sessions", "error", err)
		}
		db.Exec(`ALTER TABLE wa_tenant_sessions ALTER COLUMN tenant_id TYPE TEXT`)
	} else {
		slog.Error("Failed to open DB for mapping", "error", err)
	}

	setContainer(container)
	restoreSessions(ctx, container)

	originalMux := http.NewServeMux()
	http.DefaultServeMux = originalMux
	setupRoutes(ctx, container)

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		slog.Info("Shutting down WA Gateway", "instance_id", instanceID)
		cancel()

		clientMu.Lock()
		for tenantID, client := range clientMap {
			client.Disconnect()
			slog.Info("Disconnected tenant", "tenant_id", tenantID)
		}
		clientMu.Unlock()

		if redisClient != nil {
			clientMu.RLock()
			for tenantID := range clientMap {
				ReleaseSessionLock(context.Background(), tenantID)
			}
			clientMu.RUnlock()
		}

		os.Exit(0)
	}()

	port := ":8202"
	slog.Info("WA Gateway running", "port", port, "instance_id", instanceID)
	slog.Info("Active WhatsApp sessions", "count", len(clientMap))
	go func() {
		if err := http.ListenAndServe(port, corsMiddleware(http.DefaultServeMux)); err != nil {
			slog.Error("WA Gateway server error", "error", err)
			os.Exit(1)
		}
	}()
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Tenant-ID")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
