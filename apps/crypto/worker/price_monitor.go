package main

import (
	"context"
	"log/slog"
	"time"

	"core_project/shared/sdk/cache"

	"github.com/adshao/go-binance/v2"
)

// StartPriceMonitor subscribes to Binance WebSocket for prices and publishes to Redis channel crypto:prices:{symbol}
func StartPriceMonitor(ctx context.Context) {
	// For MVP, monitor popular pairs
	symbols := []string{"BTCUSDT", "ETHUSDT", "SOLUSDT"}

	slog.Info("Starting Price Monitor", "symbols", symbols)

	for _, symbol := range symbols {
		go func(sym string) {
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				doneC, stopC, err := binance.WsBookTickerServe(sym, func(event *binance.WsBookTickerEvent) {
					channel := "crypto:prices:" + sym
					cache.Client.Publish(context.Background(), channel, event.BestBidPrice)
				}, func(err error) {
					slog.Error("Binance WS Error", "symbol", sym, "error", err)
				})

				if err != nil {
					slog.Error("Failed to start WS", "symbol", sym, "error", err)
					time.Sleep(5 * time.Second)
					continue
				}

				select {
				case <-ctx.Done():
					stopC <- struct{}{}
					return
				case <-doneC:
					// Reconnect
					time.Sleep(1 * time.Second)
				}
			}
		}(symbol)
	}

	<-ctx.Done()
}
