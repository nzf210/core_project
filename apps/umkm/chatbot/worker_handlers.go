package main

import (
	"context"
	"encoding/json"
	"time"

	"log/slog"

	"github.com/redis/go-redis/v9"
)

// Queue setup
type ChatJob struct {
	Sender    string `json:"sender"`
	Message   string `json:"message"`
	TenantID  string `json:"tenant_id"`
	MsgType   string `json:"msg_type"`
	MediaPath string `json:"media_path"`
}

const redisQueueKey = "chatbot:queue"
const chatbotConfigCacheTTL = 5 * time.Minute

func startWorkerPool(numWorkers int) {
	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			ctx := context.Background()
			consecutiveErrors := 0
			for {
				// BRPOP blocks until an item is available or timeout
				res, err := redisClient.BRPop(ctx, 5*time.Second, redisQueueKey).Result()
				if err != nil {
					// Handle Redis being temporarily unavailable
					if err == redis.Nil {
						// Timeout, no messages - continue
						continue
					}
					consecutiveErrors++
					slog.Error("Redis BRPOP error", "worker", workerID, "error", err, "consecutive_errors", consecutiveErrors)

					// Exponential backoff with max 30 seconds
					backoff := time.Duration(min(consecutiveErrors*2, 30)) * time.Second
					time.Sleep(backoff)

					// Try to reconnect
					if err := redisClient.Ping(ctx).Err(); err != nil {
						slog.Warn("Redis reconnect failed, retrying...", "worker", workerID)
					} else {
						consecutiveErrors = 0
					}
					continue
				}

				consecutiveErrors = 0

				// BRPOP returns [key, value]
				if len(res) == 2 {
					var job ChatJob
					if err := json.Unmarshal([]byte(res[1]), &job); err == nil {
						// Increment metrics
						atomicAddInt64(&chatbotMessagesProcessed, 1)
						// Process with timeout and error handling
						processJobWithTimeout(ctx, workerID, job)
					} else {
						slog.Error("Failed to unmarshal chat job", "error", err, "data", res[1])
						atomicAddInt64(&chatbotErrors, 1)
					}
				}
			}
		}(i)
	}
	slog.Info("Worker pool started", "workers", numWorkers)
}

// processJobWithTimeout processes a job with timeout and error recovery
func processJobWithTimeout(ctx context.Context, workerID int, job ChatJob) {
	jobCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		processChatJob(job)
	}()

	select {
	case <-done:
		slog.Debug("Job processed successfully", "worker", workerID, "tenant", job.TenantID)
	case <-jobCtx.Done():
		slog.Error("Job processing timed out", "worker", workerID, "tenant", job.TenantID)
		// Re-queue the job for retry
		if jobBytes, err := json.Marshal(job); err == nil {
			redisClient.LPush(ctx, redisQueueKey+":retry", jobBytes)
		}
	}
}