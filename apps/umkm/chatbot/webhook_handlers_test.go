package main

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestEnqueueChatJob(t *testing.T) {
	job := ChatJob{TenantID: "tenant-1", Sender: "6281234567890", Message: "Halo"}
	wantErr := errors.New("redis unavailable")

	t.Run("sends job to queue", func(t *testing.T) {
		called := false
		err := enqueueChatJob(context.Background(), job, func(_ context.Context, payload []byte) error {
			called = true
			var got ChatJob
			if err := json.Unmarshal(payload, &got); err != nil {
				t.Fatalf("unmarshal queued payload: %v", err)
			}
			if got != job {
				t.Fatalf("queued job = %#v, want %#v", got, job)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("enqueueChatJob returned error: %v", err)
		}
		if !called {
			t.Fatal("queue callback was not called")
		}
	})

	t.Run("propagates queue failure", func(t *testing.T) {
		err := enqueueChatJob(context.Background(), job, func(context.Context, []byte) error {
			return wantErr
		})
		if !errors.Is(err, wantErr) {
			t.Fatalf("error = %v, want %v", err, wantErr)
		}
	})
}
