# Redis Usage Documentation

This document describes all Redis usage patterns across the WCH (Warung Cerdas Haji) platform.

## Table of Contents

1. [Redis Connection Configuration](#1-redis-connection-configuration)
2. [Redis Keys Reference](#2-redis-keys-reference)
3. [Pub/Sub Channels](#3-pubsub-channels)
4. [AI Semantic Cache](#4-ai-semantic-cache)
5. [Chatbot Job Queue](#5-chatbot-job-queue)
6. [Rate Limiting Implementation](#6-rate-limiting-implementation)

---

## 1. Redis Connection Configuration

### Shared SDK (`shared/sdk/cache/redis.go`)

Used by: `apps/crypto/worker/price_monitor.go`

```go
Client = redis.NewClient(&redis.Options{
    Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
    Password: cfg.Redis.Password,
    DB:       0,
})
```

### AI Gateway (`services/ai-gateway/`)

Used for: Semantic cache + Rate limiting

```go
client := redis.NewClient(&redis.Options{
    Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
    Password: cfg.Redis.Password,
    DB:       0,
})
```

### Auth Service (`services/auth-service/`)

Used for: Session management / token validation

```go
client := redis.NewClient(&redis.Options{
    Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
    Password: cfg.Redis.Password,
    DB:       0,
})
```

### Chatbot Service (`apps/umkm/chatbot/main.go`)

Used for: Job queue processing

```go
redisClient = redis.NewClient(&redis.Options{
    Addr: os.Getenv("REDIS_HOST") + ":6379",
})
// Falls back to "localhost" if REDIS_HOST not set
```

### Automation Worker (`apps/umkm/automation/main.go`)

Used for: Pub/Sub subscription (tenant events)

```go
client := redis.NewClient(&redis.Options{
    Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
    Password: cfg.Redis.Password,
    DB:       0,
})
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `REDIS_HOST` | `127.0.0.1` | Redis server hostname |
| `REDIS_PORT` | `6379` | Redis server port |
| `REDIS_PASSWORD` | `secure_redis_password_123` | Redis auth password |

### Docker Configuration

```yaml
redis:
  image: redis:7-alpine
  container_name: wch-redis
  command: redis-server --requirepass ${REDIS_PASSWORD:-secure_redis_password_123}
  ports:
    - "${REDIS_PORT:-6379}:6379"
  volumes:
    - redisdata:/data
```

---

## 2. Redis Keys Reference

| Key Pattern | Type | TTL | Service | Description |
|-------------|------|-----|---------|-------------|
| `ai:cache:{hash}` | String | Configurable (default 600s) | AI Gateway | Semantic cache for AI responses |
| `rate_limit:tenant:{tenant_id}` | Sorted Set | 2 minutes | AI Gateway | Sliding window rate limiter |
| `chatbot:queue` | List | None (queue) | Chatbot | Job queue for async WA message processing |
| `crypto:prices:{symbol}` | Pub/Sub Channel | N/A | Price Monitor | Real-time crypto price broadcasts |
| `tenant_events` | Pub/Sub Channel | N/A | Automation Worker | Tenant lifecycle event notifications |

---

## 3. Pub/Sub Channels

### 3.1 Crypto Price Updates (`crypto:prices:{symbol}`)

**Publisher**: `apps/crypto/worker/price_monitor.go`

```go
channel := "crypto:prices:" + sym
cache.Client.Publish(context.Background(), channel, event.BestBidPrice)
```

| Field | Value |
|-------|-------|
| Channel Pattern | `crypto:prices:{SYMBOL}` |
| Examples | `crypto:prices:BTCUSDT`, `crypto:prices:ETHUSDT`, `crypto:prices:SOLUSDT` |
| Message Content | Best bid price (string) |
| Subscribers | Any service interested in real-time crypto prices |

**Subscribed Symbols (MVP)**:
- `BTCUSDT`
- `ETHUSDT`
- `SOLUSDT`

### 3.2 Tenant Events (`tenant_events`)

**Publisher**: Internal services (publish on tenant lifecycle events)

**Subscriber**: `apps/umkm/automation/main.go`

```go
pubsub := client.Subscribe(context.Background(), "tenant_events")
ch := pubsub.Channel()
for msg := range ch {
    var payload EventPayload
    json.Unmarshal([]byte(msg.Payload), &payload)
    // payload = { tenant_id: string, event: string }
}
```

| Field | Value |
|-------|-------|
| Channel Name | `tenant_events` |
| Payload Format | `{"tenant_id": "uuid", "event": "event_type"}` |

**Known Event Types**:
| Event | Description |
|-------|-------------|
| `monthly_report` | Triggers monthly report generation via AI |

---

## 4. AI Semantic Cache

**Service**: `services/ai-gateway/main.go`

### Cache Key Generation

```go
func buildCacheKey(provider, system, message string) string {
    raw := provider + "|" + system + "|" + message
    hash := sha256.Sum256([]byte(raw))
    return fmt.Sprintf("ai:cache:%x", hash[:16])
}
```

**Key Format**: `ai:cache:{sha256_hash_16_bytes}`

### Cache Check (Read)

```go
func checkCache(ctx context.Context, key string) (string, bool) {
    if Redis != nil {
        val, err := Redis.Get(ctx, key).Result()
        if err == nil && val != "" {
            return val, true
        }
    }
    return "", false
}
```

### Cache Write

```go
func storeCache(ctx context.Context, key, value string, ttlSeconds int) {
    if Redis != nil {
        Redis.Set(ctx, key, value, time.Duration(ttlSeconds)*time.Second)
    }
}
```

### Cache Configuration

| Setting | Environment Variable | Default | Description |
|---------|---------------------|---------|-------------|
| Enabled | `AI_CACHE_ENABLED` | `true` | Enable/disable semantic caching |
| TTL | `AI_CACHE_TTL_SECONDS` | `600` (10 min) | Cache expiration in seconds |

### Cache Flow

1. Request arrives at `/v1/chat`
2. Generate cache key from `provider + system_msg + message` hash
3. Check Redis for existing entry
4. If hit: return cached response with `cache_hit: true`
5. If miss: call LLM, store result with configured TTL

---

## 5. Chatbot Job Queue

**Service**: `apps/umkm/chatbot/main.go`

### Queue Key

```go
const redisQueueKey = "chatbot:queue"
```

### Job Structure

```go
type ChatJob struct {
    Sender   string `json:"sender"`
    Message  string `json:"message"`
    TenantID string `json:"tenant_id"`
}
```

### Enqueue (Producer)

Called when WA webhook receives a message:

```go
job := ChatJob{Sender: sender, Message: message, TenantID: tenantID}
jobBytes, _ := json.Marshal(job)
redisClient.LPush(r.Context(), redisQueueKey, jobBytes)
```

### Dequeue (Consumer - Worker Pool)

Workers use BRPOP for blocking pop (wait for work):

```go
for i := 0; i < numWorkers; i++ {
    go func(workerID int) {
        for {
            res, err := redisClient.BRPop(ctx, 0, redisQueueKey).Result()
            if err != nil {
                slog.Error("Redis BRPOP error", "worker", workerID, "error", err)
                continue
            }
            // res = [key, value]
            var job ChatJob
            json.Unmarshal([]byte(res[1]), &job)
            processChatJob(job)
        }
    }(i)
}
```

| Parameter | Value | Description |
|-----------|-------|-------------|
| Timeout | `0` | Block indefinitely (wait forever) |
| Worker Count | `100` | Concurrent workers |

---

## 6. Rate Limiting Implementation

**Service**: `services/ai-gateway/main.go`

### Algorithm: Sliding Window

Uses Redis Sorted Set with timestamp as score.

### Configuration

| Parameter | Value |
|-----------|-------|
| Limit | 100 requests |
| Window | 60 seconds (1 minute) |

### Implementation

```go
func rateLimitMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if Redis == nil {
            next.ServeHTTP(w, r)
            return
        }

        tenantID := r.Header.Get("X-Tenant-ID")
        if tenantID == "" {
            tenantID = "anonymous"
        }

        key := fmt.Sprintf("rate_limit:tenant:%s", tenantID)
        now := time.Now().UnixMilli()
        windowStart := now - 60000 // 1 minute ago

        ctx := r.Context()

        pipe := Redis.TxPipeline()
        pipe.ZRemRangeByScore(ctx, key, "-inf", strconv.FormatInt(windowStart, 10))  // Remove old entries
        pipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: now})                   // Add current request
        countCmd := pipe.ZCard(ctx, key)                                                 // Count requests in window
        pipe.Expire(ctx, key, 2*time.Minute)                                            // Key expiration

        _, err := pipe.Exec(ctx)

        if countCmd.Val() > 100 {
            writeJSON(w, http.StatusTooManyRequests, APIResponse{Success: false, Message: "Rate limit exceeded"})
            return
        }

        next.ServeHTTP(w, r)
    })
}
```

### Rate Limit Key Pattern

| Pattern | Description |
|---------|-------------|
| `rate_limit:tenant:{tenant_id}` | Per-tenant rate limit |

### Algorithm Steps

1. **Cleanup**: Remove all entries older than 60 seconds
2. **Record**: Add current timestamp as a new sorted set member
3. **Count**: Get total members in the sorted set
4. **Check**: If count > 100, reject with HTTP 429
5. **Expire**: Set 2-minute TTL on key (allows for slight clock skew)

---

## Summary

| Use Case | Redis Commands | Key Pattern |
|----------|---------------|-------------|
| AI Semantic Cache | `GET`, `SET` | `ai:cache:{hash}` |
| Rate Limiting | `ZREMRANGEBYSCORE`, `ZADD`, `ZCARD`, `EXPIRE` | `rate_limit:tenant:{id}` |
| Chatbot Queue | `LPUSH`, `BRPOP` | `chatbot:queue` |
| Crypto Prices | `PUBLISH` | `crypto:prices:{symbol}` |
| Tenant Events | `SUBSCRIBE` | `tenant_events` |

---

## Configuration Reference

All services use the same Redis connection parameters from environment:

```bash
REDIS_HOST=127.0.0.1        # or 'redis' in Docker
REDIS_PORT=6379
REDIS_PASSWORD=secure_redis_password_123
```

AI-specific cache settings (AI Gateway only):

```bash
AI_CACHE_ENABLED=true
AI_CACHE_TTL_SECONDS=600
```