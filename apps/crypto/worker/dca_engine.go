package main

import (
	"context"
	"log/slog"
	"time"

	"core_project/apps/crypto/domain"
	"core_project/shared/sdk/config"
	"core_project/shared/sdk/db"
)

func StartDCAEngine(ctx context.Context, cfg *config.Config) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			processDCABots(ctx, cfg)
		}
	}
}

func processDCABots(ctx context.Context, cfg *config.Config) {
	query := `
		SELECT b.id, b.pair, b.dca_amount_per_order, b.dca_interval, b.last_executed_at, k.exchange, k.encrypted_api_key, k.encrypted_api_secret, b.tenant_id, b.user_id
		FROM bots b
		JOIN exchange_api_keys k ON b.api_key_id = k.id
		WHERE b.bot_type = $1 AND b.status = $2
	`
	rows, err := db.Pool.Query(ctx, query, domain.BotTypeDCA, domain.BotStatusRunning)
	if err != nil {
		slog.Error("Failed to query DCA bots", "error", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var botID, pair, exchange, encKey, encSecret, dcaInterval, tenantID, userID string
		var dcaAmount int64
		var lastExecutedAt *time.Time
		
		if err := rows.Scan(&botID, &pair, &dcaAmount, &dcaInterval, &lastExecutedAt, &exchange, &encKey, &encSecret, &tenantID, &userID); err != nil {
			slog.Error("Failed to scan bot row", "error", err)
			continue
		}

		if !ShouldRunDCA(lastExecutedAt, dcaInterval) {
			continue
		}

		apiKey, err := domain.Decrypt(encKey, cfg.EncryptionKey)
		if err != nil {
			slog.Error("Failed to decrypt api key", "bot_id", botID, "error", err)
			continue
		}
		apiSecret, err := domain.Decrypt(encSecret, cfg.EncryptionKey)
		if err != nil {
			slog.Error("Failed to decrypt api secret", "bot_id", botID, "error", err)
			continue
		}

		client, err := domain.NewExchangeClient(exchange, apiKey, apiSecret)
		if err != nil {
			slog.Error("Failed to create exchange client", "bot_id", botID, "error", err)
			continue
		}

		priceCents, err := client.GetPrice(ctx, pair)
		if err != nil {
			slog.Error("Failed to get price", "bot_id", botID, "error", err)
			continue
		}

		priceFloat := domain.CentsToUSDT(priceCents)
		amountFloat := domain.CentsToUSDT(dcaAmount)
		
		balance, err := client.GetBalance(ctx, "USDT") // Assuming USDT pair
		if err != nil {
			slog.Error("Failed to get balance", "bot_id", botID, "error", err)
			continue
		}
		if balance < amountFloat {
			slog.Warn("Insufficient balance for DCA", "bot_id", botID, "balance", balance, "required", amountFloat)
			continue
		}

		qty := amountFloat / priceFloat

		orderID, err := client.PlaceOrder(ctx, pair, domain.OrderSideBuy, qty, 0)
		if err != nil {
			slog.Error("Failed to place DCA order", "bot_id", botID, "error", err)
			db.Pool.Exec(ctx, "INSERT INTO crypto_notifications (tenant_id, user_id, title, message, type, is_read, created_at) VALUES ($1, $2, $3, $4, $5, $6, NOW())",
				tenantID, userID, "DCA Order Failed", "Failed to place order for "+pair+": "+err.Error(), "error", false)
			continue
		}

		slog.Info("DCA Order Placed", "bot_id", botID, "order_id", orderID, "pair", pair)
		db.Pool.Exec(ctx, "INSERT INTO crypto_notifications (tenant_id, user_id, title, message, type, is_read, created_at) VALUES ($1, $2, $3, $4, $5, $6, NOW())",
			tenantID, userID, "DCA Order Executed", "Successfully placed BUY order for "+pair, "success", false)
		
		_, err = db.Pool.Exec(ctx, "UPDATE bots SET last_executed_at = NOW(), updated_at = NOW() WHERE id = $1", botID)
		if err != nil {
			slog.Error("Failed to update last_executed_at", "bot_id", botID, "error", err)
		}
	}
}

// ShouldRunDCA determines if the DCA bot should execute based on its last execution time and interval
func ShouldRunDCA(lastExecutedAt *time.Time, interval string) bool {
	if lastExecutedAt == nil {
		return true // Never run before
	}

	var nextRun time.Time
	switch interval {
	case domain.DCAIntervalHourly:
		nextRun = lastExecutedAt.Add(1 * time.Hour)
	case domain.DCAIntervalDaily:
		nextRun = lastExecutedAt.Add(24 * time.Hour)
	case domain.DCAIntervalWeekly:
		nextRun = lastExecutedAt.Add(7 * 24 * time.Hour)
	case domain.DCAIntervalMonthly:
		nextRun = lastExecutedAt.Add(30 * 24 * time.Hour)
	default:
		nextRun = lastExecutedAt.Add(24 * time.Hour)
	}

	return time.Now().After(nextRun) || time.Now().Equal(nextRun)
}
