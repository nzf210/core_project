package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

func getDBURI() string {
	user := os.Getenv("DB_USER")
	if user == "" {
		user = "postgres"
	}
	pass := os.Getenv("DB_PASSWORD")
	if pass == "" {
		pass = "postgres"
	}
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5433"
	}
	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		dbname = "wch_core"
	}
	sslmode := os.Getenv("DB_SSLMODE")
	if sslmode == "" {
		sslmode = "disable"
	}

	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, pass, host, port, dbname, sslmode)
}

func initRedis() *redis.Client {
	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "localhost"
	}
	redisPort := os.Getenv("REDIS_PORT")
	if redisPort == "" {
		redisPort = "6379"
	}
	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisDB := os.Getenv("REDIS_DB")
	if redisDB == "" {
		redisDB = "0"
	}

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
func AcquireSessionLock(ctx context.Context, tenantID string) (bool, error) {
	if redisClient == nil {
		return true, nil
	}

	lockKey := sessionLockPrefix + tenantID
	ownerKey := sessionOwnerPrefix + tenantID

	acquired, err := redisClient.SetNX(ctx, lockKey, instanceID, sessionTTL).Result()
	if err != nil {
		return true, nil // Fallback to local on error
	}

	if acquired {
		redisClient.Set(ctx, ownerKey, instanceID, sessionTTL)
		return true, nil
	}

	currentOwner, err := redisClient.Get(ctx, ownerKey).Result()
	if err == nil && currentOwner == instanceID {
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
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID != "" {
		return tenantID
	}

	tenantID = r.FormValue("tenant_id")
	if tenantID != "" {
		return tenantID
	}

	tenantID = r.Header.Get("X-Tenant-ID")
	if tenantID != "" {
		return tenantID
	}

	return ""
}
