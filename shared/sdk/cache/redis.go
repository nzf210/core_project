package cache

import (
	"context"
	"fmt"
	"log"
	"time"

	"core_project/shared/sdk/config"
	"github.com/redis/go-redis/v9"
)

var Client *redis.Client

// InitRedis initializes Redis connection
func InitRedis(cfg *config.Config) error {
	Client = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       0, // Use default DB
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := Client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("unable to connect to redis: %w", err)
	}

	log.Println("✅ Redis connected successfully")
	return nil
}

// CloseRedis closes Redis connection
func CloseRedis() {
	if Client != nil {
		Client.Close()
		log.Println("Redis connection closed")
	}
}
