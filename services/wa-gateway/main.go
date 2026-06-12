package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"

	_ "github.com/lib/pq"
)

// Redis key prefixes
const (
	sessionLockPrefix    = "wa:lock:"      // Distributed lock per tenant
	sessionOwnerPrefix   = "wa:owner:"     // Which instance owns the session
	sessionTTL           = 5 * time.Minute // Lock TTL
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
	// Map to store connected clients per tenant_id (local cache)
	clientMap = make(map[string]*whatsmeow.Client)
	clientMu  sync.RWMutex

	// Redis client for distributed coordination
	redisClient *redis.Client
	instanceID  string

	// Postgres connection
	db     *sql.DB
	dbURI  string

	// Rate limiter for whatsmeow messages (per-tenant token bucket)
	rateLimiter = NewTenantRateLimiter(5) // 5 messages per minute per tenant

	// Reconnect backoff state
	reconnectMu       sync.Mutex
	reconnectAttempts = make(map[string]int)     // tenant_id → attempt count
	reconnectBackoff  = make(map[string]time.Time) // tenant_id → last reconnect time
)

func init() {
	// Generate unique instance ID
	hostname, _ := os.Hostname()
	instanceID = fmt.Sprintf("%s-%d", hostname, os.Getpid())
}

// ─────────────────────────────────────────────
// Rate Limiter (Token Bucket per Tenant)
// ─────────────────────────────────────────────

type TenantRateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*tokenBucket
	rate    int // messages per minute
}

type tokenBucket struct {
	tokens   float64
	lastTime time.Time
}

func NewTenantRateLimiter(msgPerMinute int) *TenantRateLimiter {
	return &TenantRateLimiter{
		buckets: make(map[string]*tokenBucket),
		rate:    msgPerMinute,
	}
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

	// Refill tokens based on elapsed time
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
// Reconnect Backoff (Exponential)
// ─────────────────────────────────────────────

func shouldReconnect(tenantID string) bool {
	reconnectMu.Lock()
	defer reconnectMu.Unlock()

	// Max 1 reconnect attempt per 5 minutes per tenant
	if lastAttempt, ok := reconnectBackoff[tenantID]; ok {
		if time.Since(lastAttempt) < 5*time.Minute {
			// Check if this is a new attempt window
			attempts := reconnectAttempts[tenantID]
			if attempts > 0 {
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

// ─────────────────────────────────────────────
// Cloud API Routing
// ─────────────────────────────────────────────

// isTransactional determines if a message should go via Meta Cloud API
func isTransactional(r *http.Request) bool {
	msgType := r.Header.Get("X-Message-Type")
	switch msgType {
	case "otp", "invoice", "payment", "subscription", "system":
		return true
	}
	source := r.Header.Get("X-Source")
	switch source {
	case "auth-service", "billing-service", "notification-service":
		return true
	}
	return false
}

// routeToCloudAPI sends a message via the wa-cloud-api service (Meta Cloud API)
func routeToCloudAPI(tenantID, target, message, msgType string) (string, error) {
	cloudAPIURL := os.Getenv("WA_CLOUD_API_URL_PORT")
	cloudAPIHost := "http://localhost:8210"
	if os.Getenv("APP_ENV") == "production" || os.Getenv("DB_HOST") == "postgres" {
		cloudAPIHost = "http://wa-cloud-api:8210"
	}
	_ = cloudAPIURL

	payload := map[string]interface{}{
		"to":   target,
		"type": "text",
		"text": message,
	}
	if msgType != "" {
		payload["type"] = msgType
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, cloudAPIHost+"/send", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenantID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("Cloud API: failed to parse response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		errMsg := "unknown error"
		if m, ok := result["message"].(string); ok {
			errMsg = m
		}
		return "", fmt.Errorf("Cloud API: %s", errMsg)
	}

	waMsgID := ""
	if id, ok := result["wa_message_id"].(string); ok {
		waMsgID = id
	}

	log.Printf("Routed message via Cloud API for tenant %s, wa_msg_id=%s", tenantID, waMsgID)
	return waMsgID, nil
}

func getDBURI() string {
	user := os.Getenv("DB_USER")
	if user == "" { user = "postgres" }
	pass := os.Getenv("DB_PASSWORD")
	if pass == "" { pass = "postgres" }
	host := os.Getenv("DB_HOST")
	if host == "" { host = "localhost" }
	port := os.Getenv("DB_PORT")
	if port == "" { port = "5433" }
	dbname := os.Getenv("DB_NAME")
	if dbname == "" { dbname = "wch_core" }
	sslmode := os.Getenv("DB_SSLMODE")
	if sslmode == "" { sslmode = "disable" }

	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, pass, host, port, dbname, sslmode)
}

func initRedis() *redis.Client {
	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" { redisHost = "localhost" }
	redisPort := os.Getenv("REDIS_PORT")
	if redisPort == "" { redisPort = "6379" }
	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisDB := os.Getenv("REDIS_DB")
	if redisDB == "" { redisDB = "0" }

	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password: redisPassword,
		DB:       9, // Use DB 9 for WA Gateway coordination
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis not available for distributed coordination: %v", err)
		return nil
	}

	log.Printf("Connected to Redis for distributed session coordination")
	return client
}

// AcquireSessionLock tries to acquire a distributed lock for a tenant's WA session
// Returns true if this instance owns the session, false otherwise
func AcquireSessionLock(ctx context.Context, tenantID string) (bool, error) {
	if redisClient == nil {
		// Fallback: no Redis, assume we own it
		return true, nil
	}

	lockKey := sessionLockPrefix + tenantID
	ownerKey := sessionOwnerPrefix + tenantID

	// Try to acquire lock using SET NX with expiry
	acquired, err := redisClient.SetNX(ctx, lockKey, instanceID, sessionTTL).Result()
	if err != nil {
		return true, nil // Fallback to local on error
	}

	if acquired {
		// We got the lock, update owner
		redisClient.Set(ctx, ownerKey, instanceID, sessionTTL)
		return true, nil
	}

	// Check if we already own it
	currentOwner, err := redisClient.Get(ctx, ownerKey).Result()
	if err == nil && currentOwner == instanceID {
		// Refresh TTL
		redisClient.Expire(ctx, lockKey, sessionTTL)
		redisClient.Expire(ctx, ownerKey, sessionTTL)
		return true, nil
	}

	return false, nil
}

// ReleaseSessionLock releases the distributed lock for a tenant
func ReleaseSessionLock(ctx context.Context, tenantID string) error {
	if redisClient == nil {
		return nil
	}

	ownerKey := sessionOwnerPrefix + tenantID
	lockKey := sessionLockPrefix + tenantID

	// Only release if we own it
	currentOwner, err := redisClient.Get(ctx, ownerKey).Result()
	if err == nil && currentOwner == instanceID {
		redisClient.Del(ctx, lockKey, ownerKey)
	}

	return nil
}

// Heartbeat updates the instance's heartbeat in Redis
func Heartbeat(ctx context.Context) {
	if redisClient == nil {
		return
	}

	key := fmt.Sprintf(instanceHeartbeatKey, instanceID)
	redisClient.Set(ctx, key, time.Now().Unix(), 2*time.Minute)
}

func getTenantID(r *http.Request) string {
	// 1. Try URL Query parameter
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID != "" {
		return tenantID
	}

	// 2. Try Form value
	tenantID = r.FormValue("tenant_id")
	if tenantID != "" {
		return tenantID
	}

	// 3. Try X-Tenant-ID Header
	tenantID = r.Header.Get("X-Tenant-ID")
	if tenantID != "" {
		return tenantID
	}

	// 4. Try JSON body if Content-Type is application/json
	if r.Body != nil && strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		bodyBytes, err := io.ReadAll(r.Body)
		if err == nil {
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			var data map[string]interface{}
			if err := json.Unmarshal(bodyBytes, &data); err == nil {
				if tID, ok := data["tenant_id"].(string); ok {
					return tID
				}
			}
		}
	}

	return ""
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	clientMu.RLock()
	connected := len(clientMap)
	clientMu.RUnlock()

	// Check Redis connectivity
	redisStatus := "disconnected"
	if redisClient != nil {
		if err := redisClient.Ping(r.Context()).Err(); err == nil {
			redisStatus = "connected"
		}
	}

	// Check DB connectivity
	dbStatus := "disconnected"
	if db != nil {
		if err := db.Ping(); err == nil {
			dbStatus = "connected"
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":         "ok",
		"instance_id":    instanceID,
		"connected_sessions": connected,
		"redis":          redisStatus,
		"database":       dbStatus,
	})
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	handleHealth(w, r)
}

// handleMetrics returns Prometheus-compatible metrics for monitoring
func handleMetrics(w http.ResponseWriter, r *http.Request) {
	clientMu.RLock()
	connectedSessions := len(clientMap)
	clientMu.RUnlock()

	// Get total sessions from Redis
	totalTenants := int64(0)
	if redisClient != nil {
		keys, _ := redisClient.Keys(r.Context(), "wa:owner:*").Result()
		totalTenants = int64(len(keys))
	}

	// Get active instances
	activeInstances := int64(0)
	if redisClient != nil {
		keys, _ := redisClient.Keys(r.Context(), "wa:instance:*").Result()
		activeInstances = int64(len(keys))
	}

	// Output in Prometheus exposition format
	metrics := fmt.Sprintf(`# HELP wa_gateway_info WA Gateway instance info
# TYPE wa_gateway_info gauge
wa_gateway_info{instance="%s"} 1

# HELP wa_gateway_connected_sessions Current connected WhatsApp sessions
# TYPE wa_gateway_connected_sessions gauge
wa_gateway_connected_sessions{instance="%s"} %d

# HELP wa_gateway_total_tenants Total tenants with sessions
# TYPE wa_gateway_total_tenants gauge
wa_gateway_total_tenants %d

# HELP wa_gateway_active_instances Number of active gateway instances
# TYPE wa_gateway_active_instances gauge
wa_gateway_active_instances %d

# HELP wa_gateway_messages_sent_total Total messages sent
# TYPE wa_gateway_messages_sent_total counter
wa_gateway_messages_sent_total{instance="%s"} %d

# HELP wa_gateway_messages_received_total Total messages received
# TYPE wa_gateway_messages_received_total counter
wa_gateway_messages_received_total{instance="%s"} %d

# HELP wa_gateway_errors_total Total errors
# TYPE wa_gateway_errors_total counter
wa_gateway_errors_total{instance="%s"} %d
`, instanceID, instanceID, connectedSessions, totalTenants, activeInstances, instanceID, waMessagesSent, instanceID, waMessagesRecv, instanceID, waErrorsTotal)

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(metrics))
}

func main() {
	// Load .env file
	_ = godotenv.Load(".env")
	_ = godotenv.Load("../../.env")

	dbURI = getDBURI()

	// Initialize Redis for distributed coordination
	redisClient = initRedis()

	// Start heartbeat goroutine
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

	// Initialize Postgres container store (whatsmeow's own storage)
	dbLog := waLog.Stdout("Database", "WARN", true)
	container, err := sqlstore.New(context.Background(), "postgres", dbURI, dbLog)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer container.Close()

	// Initialize our custom mapping table
	db, err = sql.Open("postgres", dbURI)
	if err == nil {
		_, err = db.Exec(`CREATE TABLE IF NOT EXISTS wa_tenant_sessions (tenant_id TEXT PRIMARY KEY, jid VARCHAR NOT NULL)`)
		if err != nil {
			log.Printf("Failed to create wa_tenant_sessions: %v", err)
		}
		db.Exec(`ALTER TABLE wa_tenant_sessions ALTER COLUMN tenant_id TYPE TEXT`)
	} else {
		log.Printf("Failed to open DB for mapping: %v", err)
	}

	// Restore sessions on startup
	restoreSessions(ctx, container)

	// Setup HTTP handlers
	originalMux := http.NewServeMux()
	http.DefaultServeMux = originalMux
	setupRoutes(ctx, container)

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Printf("Shutting down WA Gateway (instance %s)...", instanceID)
		cancel()

		// Disconnect all clients
		clientMu.Lock()
		for tenantID, client := range clientMap {
			client.Disconnect()
			log.Printf("Disconnected tenant %s", tenantID)
		}
		clientMu.Unlock()

		// Release all our locks
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
	log.Printf("WA Gateway running on port %s (instance: %s)", port, instanceID)
	log.Printf("Active WhatsApp sessions: %d", len(clientMap))
	log.Fatal(http.ListenAndServe(port, nil))
}

// wrapWithCORS wraps an http.Handler with CORS headers
func wrapWithCORS(handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Tenant-ID")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		handler.ServeHTTP(w, r)
	}
}

// restoreSessions loads and reconnects existing sessions from DB
func restoreSessions(ctx context.Context, container *sqlstore.Container) {
	if db == nil {
		return
	}

	// Add jitter delay to avoid race condition between replicas
	// First instance gets a short delay, second gets longer
	delayMs := 1000 + (time.Now().UnixNano()%3000)
	time.Sleep(time.Duration(delayMs) * time.Millisecond)

	rows, err := db.Query(`SELECT tenant_id, jid FROM wa_tenant_sessions`)
	if err != nil {
		log.Printf("Failed to query sessions: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var tID, jidStr string
		if err := rows.Scan(&tID, &jidStr); err != nil {
			continue
		}

		// Try to acquire lock for this tenant
		if owned, _ := AcquireSessionLock(ctx, tID); !owned {
			log.Printf("Session for tenant %s owned by another instance, skipping", tID)
			continue
		}

		jid, _ := types.ParseJID(jidStr)
		device, _ := container.GetDevice(context.Background(), jid)
		if device != nil {
			client := whatsmeow.NewClient(device, waLog.Stdout("Client-"+tID, "INFO", true))
			client.AddEventHandler(func(evt interface{}) { eventHandler(ctx, tID, evt) })
			if err := client.Connect(); err == nil {
				clientMu.Lock()
				clientMap[tID] = client
				clientMu.Unlock()
				log.Printf("Restored session for tenant %s", tID)
			} else {
				log.Printf("Failed to restore session for tenant %s: %v", tID, err)
				ReleaseSessionLock(ctx, tID)
			}
		}
	}
}

func setupRoutes(ctx context.Context, container *sqlstore.Container) {
	// Health endpoints
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/healthz", handleHealthz)

	// Prometheus-compatible metrics endpoint
	http.HandleFunc("/metrics", handleMetrics)

	// QR code generation
	http.HandleFunc("/api/wa/qr", func(w http.ResponseWriter, r *http.Request) {
		tenantID := getTenantID(r)
		if tenantID == "" {
			http.Error(w, `{"error":"tenant_id required"}`, http.StatusBadRequest)
			return
		}

		// Distributed lock check
		owned, err := AcquireSessionLock(r.Context(), tenantID)
		if err != nil || !owned {
			// Another instance is handling this tenant
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "busy",
				"message": "Another gateway instance is handling this tenant. Please retry.",
			})
			return
		}

		clientMu.Lock()
		client, exists := clientMap[tenantID]
		if exists && client.Store.ID == nil {
			client.Disconnect()
			delete(clientMap, tenantID)
			exists = false
		}

		if !exists {
			newStore := container.NewDevice()
			clientLog := waLog.Stdout("Client-"+tenantID, "INFO", true)
			client = whatsmeow.NewClient(newStore, clientLog)
			client.AddEventHandler(func(evt interface{}) { eventHandler(ctx, tenantID, evt) })
			clientMap[tenantID] = client
		}
		clientMu.Unlock()

		if client.Store.ID != nil {
			// Already logged in
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "connected",
				"message": "Already connected",
			})
			return
		}

		// Generate QR
		qrChan, _ := client.GetQRChannel(context.Background())
		if err := client.Connect(); err != nil {
			log.Printf("Failed to connect for QR: %v", err)
			http.Error(w, `{"error":"failed to connect"}`, http.StatusInternalServerError)
			return
		}

		for evt := range qrChan {
			if evt.Event == "code" {
				png, err := qrcode.Encode(evt.Code, qrcode.Medium, 256)
				if err != nil {
					http.Error(w, `{"error":"failed to generate qr"}`, http.StatusInternalServerError)
					return
				}
				base64QR := "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"status":   "qr",
					"qr_code":  base64QR,
					"raw_code": evt.Code,
				})

				// Timeout: release lock if not connected in 60s
				go func(c *whatsmeow.Client, tid string) {
					time.Sleep(60 * time.Second)
					if c.Store.ID == nil {
						c.Disconnect()
						clientMu.Lock()
						if clientMap[tid] == c {
							delete(clientMap, tid)
						}
						clientMu.Unlock()
						ReleaseSessionLock(context.Background(), tid)
					}
				}(client, tenantID)
				return
			}
			log.Printf("QR channel event: %s", evt.Event)
		}
	})

	// Status check
	http.HandleFunc("/api/wa/status", func(w http.ResponseWriter, r *http.Request) {
		tenantID := getTenantID(r)
		w.Header().Set("Content-Type", "application/json")

		// Check if this instance owns the session
		if redisClient != nil {
			ownerKey := sessionOwnerPrefix + tenantID
			owner, err := redisClient.Get(r.Context(), ownerKey).Result()
			if err == nil && owner != instanceID {
				var jid string
				if db != nil {
					db.QueryRow("SELECT jid FROM wa_tenant_sessions WHERE tenant_id = $1", tenantID).Scan(&jid)
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"status":  "connected",
					"jid":     jid,
					"owner":   owner,
					"message": "Session handled by another instance",
				})
				return
			}
		}

		clientMu.RLock()
		client, exists := clientMap[tenantID]
		clientMu.RUnlock()

		if !exists || client.Store.ID == nil {
			// Fallback: check if we have a valid session in DB that hasn't been initialized in this map yet
			if db != nil {
				var jid string
				err := db.QueryRow("SELECT jid FROM wa_tenant_sessions WHERE tenant_id = $1", tenantID).Scan(&jid)
				if err == nil && jid != "" {
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"status": "connected",
						"jid":    jid,
					})
					return
				}
			}
			
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "disconnected",
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "connected",
			"jid":    client.Store.ID.String(),
		})
	})

	// Send message
	http.HandleFunc("/api/wa/send", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, `{"error":"Failed to parse form"}`, http.StatusBadRequest)
			return
		}

		tenantID := getTenantID(r)
		target := r.FormValue("target")
		message := r.FormValue("message")

		if tenantID == "" || target == "" || message == "" {
			http.Error(w, `{"error":"Missing tenant_id, target, or message"}`, http.StatusBadRequest)
			return
		}

		// ── Hybrid routing: transactional messages via Cloud API ──
		if isTransactional(r) {
			msgType := r.Header.Get("X-Message-Type")
			if msgType == "" {
				msgType = "text"
			}
			waMsgID, err := routeToCloudAPI(tenantID, target, message, msgType)
			if err == nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success":       true,
					"routed":        "cloud_api",
					"wa_message_id": waMsgID,
				})
				return
			}
			log.Printf("Cloud API failed, falling back to whatsmeow for tenant %s: %v", tenantID, err)
			// Fall through to whatsmeow
		}

		// ── Rate limiting for whatsmeow ──
		if !rateLimiter.Allow(tenantID) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":   "Rate limit exceeded",
				"message": "Too many WhatsApp messages. Please slow down to avoid blocking.",
			})
			return
		}

		// Verify ownership
		if redisClient != nil {
			ownerKey := sessionOwnerPrefix + tenantID
			owner, err := redisClient.Get(r.Context(), ownerKey).Result()
			if err == nil && owner != instanceID {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"status":  "delegated",
					"message": "Session handled by another instance",
				})
				return
			}
		}

		clientMu.RLock()
		client, exists := clientMap[tenantID]
		clientMu.RUnlock()

		if !exists || client.Store.ID == nil {
			http.Error(w, `{"error":"WhatsApp not connected for this tenant"}`, http.StatusBadGateway)
			return
		}

		jid, err := types.ParseJID(target)
		if err != nil {
			http.Error(w, `{"error":"Invalid target JID"}`, http.StatusBadRequest)
			return
		}

		msg := &waE2E.Message{
			Conversation: proto.String(message),
		}

		_, err = client.SendMessage(context.Background(), jid, msg)
		if err != nil {
			log.Printf("[Tenant %s] Failed to send message to %s: %v", tenantID, jid.String(), err)
			http.Error(w, `{"error":"Failed to send message"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
	})

	// Logout
	http.HandleFunc("/api/wa/logout", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		tenantID := getTenantID(r)
		if tenantID == "" {
			http.Error(w, `{"error":"tenant_id required"}`, http.StatusBadRequest)
			return
		}

		clientMu.Lock()
		client, exists := clientMap[tenantID]
		if exists {
			client.Disconnect()
			delete(clientMap, tenantID)
		}
		clientMu.Unlock()

		if db != nil {
			db.Exec(`DELETE FROM wa_tenant_sessions WHERE tenant_id = $1`, tenantID)
		}

		ReleaseSessionLock(context.Background(), tenantID)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
	})
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

// eventHandler handles WhatsApp events for a tenant
func eventHandler(ctx context.Context, tenantID string, evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		text := v.Message.GetConversation()
		if text == "" {
			text = v.Message.GetExtendedTextMessage().GetText()
		}
		if text == "" {
			return
		}
		senderJID := v.Info.Sender.ToNonAD().String()

		if v.Info.IsFromMe {
			return
		}

		log.Printf("[Tenant %s] Received message from %s: %s", tenantID, senderJID, text)

		// Forward to Chatbot
		payload := map[string]interface{}{
			"sender":  senderJID,
			"message": text,
		}
		jsonBody, _ := json.Marshal(payload)

		webhookURL := "http://umkm-chatbot:8202/webhook/wa?tenant_id=" + tenantID
		resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonBody))
		if err != nil {
			log.Printf("Failed to forward message to chatbot: %v", err)
		} else {
			resp.Body.Close()
		}

	case *events.Connected:
		log.Printf("[Tenant %s] Connected to WhatsApp!", tenantID)
		clientMu.RLock()
		c := clientMap[tenantID]
		clientMu.RUnlock()
		if c != nil && c.Store.ID != nil && db != nil {
			db.Exec(`INSERT INTO wa_tenant_sessions (tenant_id, jid) VALUES ($1, $2) ON CONFLICT (tenant_id) DO UPDATE SET jid = EXCLUDED.jid`, tenantID, c.Store.ID.String())
		}
	}
}