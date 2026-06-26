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
		go workerLoop(i)
	}
	slog.Info("Worker pool started", "workers", numWorkers)
}

func workerLoop(workerID int) {
	ctx := context.Background()
	consecutiveErrors := 0
	for {
		res, err := redisClient.BRPop(ctx, 5*time.Second, redisQueueKey).Result()
		if err != nil {
			consecutiveErrors = handleRedisError(workerID, err, consecutiveErrors)
			continue
		}
		consecutiveErrors = 0
		dispatchJob(ctx, workerID, res)
	}
}

func handleRedisError(workerID int, err error, consecutiveErrors int) int {
	if err == redis.Nil {
		return consecutiveErrors
	}
	consecutiveErrors++
	slog.Error("Redis BRPOP error", "worker", workerID, "error", err, "consecutive_errors", consecutiveErrors)

	backoff := time.Duration(min(consecutiveErrors*2, 30)) * time.Second
	time.Sleep(backoff)

	if err := redisClient.Ping(ctx).Err(); err != nil {
		slog.Warn("Redis reconnect failed, retrying...", "worker", workerID)
	} else {
		consecutiveErrors = 0
	}
	return consecutiveErrors
}

var ctx = context.Background()

func dispatchJob(ctx context.Context, workerID int, res []string) {
	if len(res) != 2 {
		return
	}
	var job ChatJob
	if err := json.Unmarshal([]byte(res[1]), &job); err != nil {
		slog.Error("Failed to unmarshal chat job", "error", err, "data", res[1])
		atomicAddInt64(&chatbotErrors, 1)
		return
	}
	atomicAddInt64(&chatbotMessagesProcessed, 1)
	processJobWithTimeout(ctx, workerID, job)
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