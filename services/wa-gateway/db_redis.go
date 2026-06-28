package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

func getDBURI() string {
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "127.0.0.1"
	}
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}
	user := os.Getenv("DB_USER")
	if user == "" {
		user = "wch_admin"
	}
	pass := os.Getenv("DB_PASSWORD")
	if pass == "" {
		pass = "secure_postgres_password_123"
	}
	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		dbname = "wch_platform"
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

	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password: redisPassword,
		DB:       9, // Use DB 9 for WA Gateway coordination
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		slog.Warn("Redis not available for distributed coordination", "error", err)
		return nil
	}

	slog.Info("Connected to Redis for distributed session coordination")
	return client
}

// initRedisWithDB creates a Redis client connected to the specified database index.
// Used by redisShared (DB 0) to access shared cross-service keys.
func initRedisWithDB(db int) *redis.Client {
	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "localhost"
	}
	redisPort := os.Getenv("REDIS_PORT")
	if redisPort == "" {
		redisPort = "6379"
	}
	redisPassword := os.Getenv("REDIS_PASSWORD")

	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password: redisPassword,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		slog.Warn("Redis DB not available", "db", db, "error", err)
		return nil
	}

	slog.Info("Connected to Redis DB", "db", db)
	return client
}

func AcquireSessionLock(ctx context.Context, tenantID string) (bool, error) {
	if redisClient == nil {
		return false, nil
	}
	lockKey := sessionLockPrefix + tenantID
	ownerKey := sessionOwnerPrefix + tenantID
	acquired, err := redisClient.SetNX(ctx, lockKey, instanceID, sessionTTL).Result()
	if err != nil {
		return false, err
	}
	if acquired {
		redisClient.Set(ctx, ownerKey, instanceID, sessionTTL)
		return true, nil
	}
	currentOwner, err := redisClient.Get(ctx, ownerKey).Result()
	if err != nil {
		return false, nil
	}
	if currentOwner == instanceID {
		redisClient.Expire(ctx, lockKey, sessionTTL)
		redisClient.Expire(ctx, ownerKey, sessionTTL)
		return true, nil
	}
	return false, nil
}

func ReleaseSessionLock(ctx context.Context, tenantID string) {
	if redisClient == nil {
		return
	}
	lockKey := sessionLockPrefix + tenantID
	ownerKey := sessionOwnerPrefix + tenantID
	currentOwner, err := redisClient.Get(ctx, ownerKey).Result()
	if err != nil {
		return
	}
	if currentOwner == instanceID {
		redisClient.Del(ctx, lockKey, ownerKey)
	}
}

func Heartbeat(ctx context.Context) {
	if redisClient == nil {
		return
	}
	key := fmt.Sprintf(instanceHeartbeatKey, instanceID)
	redisClient.Set(ctx, key, time.Now().Unix(), 2*time.Minute)
}
