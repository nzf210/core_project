package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"core_project/apps/crypto/domain"
	"core_project/shared/sdk/config"
	"core_project/shared/sdk/db"
)

func StartGridEngine(ctx context.Context, cfg *config.Config) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			processGridBots(ctx, cfg)
		}
	}
}

func processGridBots(ctx context.Context, cfg *config.Config) {
	query := `
		SELECT b.id, b.pair, b.grid_lower_price, b.grid_upper_price, b.grid_count, b.grid_investment, k.exchange, k.encrypted_api_key, k.encrypted_api_secret, b.tenant_id, b.user_id
		FROM bots b
		JOIN exchange_api_keys k ON b.api_key_id = k.id
		WHERE b.bot_type = $1 AND b.status = $2
	`
	rows, err := db.Pool.Query(ctx, query, domain.BotTypeGrid, domain.BotStatusRunning)
	if err != nil {
		slog.Error("Failed to query Grid bots", "error", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var botID, pair, exchange, encKey, encSecret, tenantID, userID string
		var lowerPrice, upperPrice, investment int64
		var gridCount int

		if err := rows.Scan(&botID, &pair, &lowerPrice, &upperPrice, &gridCount, &investment, &exchange, &encKey, &encSecret, &tenantID, &userID); err != nil {
			slog.Error("Failed to scan bot row", "error", err)
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
		currentPrice := domain.CentsToUSDT(priceCents)

		// 1. Fetch existing grid lines
		lineRows, err := db.Pool.Query(ctx, "SELECT id, price, side, status, exchange_order_id FROM bot_grid_lines WHERE bot_id = $1", botID)
		if err != nil {
			slog.Error("Failed to get grid lines", "bot_id", botID, "error", err)
			continue
		}
		
		type GridLine struct {
			ID              string
			Price           int64
			Side            string
			Status          string
			ExchangeOrderID *string
		}
		var lines []GridLine
		for lineRows.Next() {
			var l GridLine
			lineRows.Scan(&l.ID, &l.Price, &l.Side, &l.Status, &l.ExchangeOrderID)
			lines = append(lines, l)
		}
		lineRows.Close()

		gridStep := float64(0)
		if gridCount > 1 {
			gridStep = float64(upperPrice-lowerPrice) / float64(gridCount-1)
		}

		// 2. Initialize if empty
		if len(lines) == 0 {
			slog.Info("Initializing grid lines", "bot_id", botID)
			
			for i := 0; i < gridCount; i++ {
				linePriceCents := float64(lowerPrice) + float64(i)*gridStep
				linePriceUSDT := domain.CentsToUSDT(int64(linePriceCents))
				
				side := domain.OrderSideSell
				if linePriceUSDT < currentPrice {
					side = domain.OrderSideBuy
				}
				
				var newID string
				err := db.Pool.QueryRow(ctx, "INSERT INTO bot_grid_lines (bot_id, price, side, status) VALUES ($1, $2, $3, $4) RETURNING id", 
					botID, int64(linePriceCents), side, "pending").Scan(&newID)
				if err == nil {
					lines = append(lines, GridLine{
						ID: newID, Price: int64(linePriceCents), Side: side, Status: "pending",
					})
				}
			}
		}

		// 3. Process grid lines
		qtyPerGrid := (domain.CentsToUSDT(investment) / float64(gridCount)) / currentPrice
		
		for _, line := range lines {
			linePriceUSDT := domain.CentsToUSDT(line.Price)
			
			if line.Status == "pending" {
				// Place order
				orderID, err := client.PlaceOrder(ctx, pair, line.Side, qtyPerGrid, linePriceUSDT)
				if err != nil {
					slog.Error("Failed to place grid limit order", "bot_id", botID, "error", err)
					db.Pool.Exec(ctx, "INSERT INTO crypto_notifications (tenant_id, user_id, title, message, type, is_read, created_at) VALUES ($1, $2, $3, $4, $5, $6, NOW())",
						tenantID, userID, "Grid Order Failed", "Failed to place "+line.Side+" limit order for "+pair+": "+err.Error(), "error", false)
					continue
				}
				
				// Update status
				db.Pool.Exec(ctx, "UPDATE bot_grid_lines SET status = 'active', exchange_order_id = $1 WHERE id = $2", orderID, line.ID)
				slog.Info("Placed grid order", "bot_id", botID, "side", line.Side, "price", linePriceUSDT)
				
			} else if line.Status == "active" && line.ExchangeOrderID != nil {
				// Check status
				status, err := client.GetOrderStatus(ctx, pair, *line.ExchangeOrderID)
				if err != nil {
					slog.Error("Failed to get order status", "order_id", *line.ExchangeOrderID, "error", err)
					continue
				}
				
				if status == domain.OrderStatusFilled {
					db.Pool.Exec(ctx, "UPDATE bot_grid_lines SET status = 'filled' WHERE id = $1", line.ID)
					slog.Info("Grid order filled", "bot_id", botID, "side", line.Side, "price", linePriceUSDT)
					
					db.Pool.Exec(ctx, "INSERT INTO crypto_notifications (tenant_id, user_id, title, message, type, is_read, created_at) VALUES ($1, $2, $3, $4, $5, $6, NOW())",
						tenantID, userID, "Grid Order Filled", fmt.Sprintf("Grid %s order filled at %f %s", line.Side, domain.CentsToUSDT(line.Price), pair), "success", false)
					
					// Create opposite order
					newSide := domain.OrderSideSell
					newPriceCents := float64(line.Price) + gridStep
					if line.Side == domain.OrderSideSell {
						newSide = domain.OrderSideBuy
						newPriceCents = float64(line.Price) - gridStep
					}
					
					// Add boundary checks here if needed in future
					db.Pool.Exec(ctx, "INSERT INTO bot_grid_lines (bot_id, price, side, status) VALUES ($1, $2, $3, $4)", 
						botID, int64(newPriceCents), newSide, "pending")
				} else if status == domain.OrderStatusCancelled {
					db.Pool.Exec(ctx, "UPDATE bot_grid_lines SET status = 'pending' WHERE id = $1", line.ID)
				}
			}
		}
	}
}
