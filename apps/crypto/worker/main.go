package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"core_project/shared/sdk/cache"
	"core_project/shared/sdk/config"
	"core_project/shared/sdk/db"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Load Config
	cfg := config.LoadConfig("")

	// Initialize DB
	if err := db.InitDB(cfg); err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer db.CloseDB()

	// Initialize Redis
	if err := cache.InitRedis(cfg); err != nil {
		slog.Error("Failed to initialize redis", "error", err)
		os.Exit(1)
	}
	defer cache.CloseRedis()

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	slog.Info("Starting Crypto Worker Engines...")

	// 1. Price Monitor
	wg.Add(1)
	go func() {
		defer wg.Done()
		StartPriceMonitor(ctx)
	}()

	// 2. DCA Engine
	wg.Add(1)
	go func() {
		defer wg.Done()
		StartDCAEngine(ctx, cfg)
	}()

	// 3. Grid Engine
	wg.Add(1)
	go func() {
		defer wg.Done()
		StartGridEngine(ctx, cfg)
	}()

	// 4. Signal Engine
	wg.Add(1)
	go func() {
		defer wg.Done()
		StartSignalEngine(ctx, cfg)
	}()

	// Wait for termination signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	slog.Info("Shutting down Crypto Worker...")
	cancel()
	wg.Wait()
	slog.Info("Shutdown complete")
}
