package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"core_project/apps/crypto/domain"
	"core_project/shared/sdk/config"
	"core_project/shared/sdk/db"
)

func StartSignalEngine(ctx context.Context, cfg *config.Config) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			processSignalBots(ctx, cfg)
		}
	}
}

func processSignalBots(ctx context.Context, cfg *config.Config) {
	query := `
		SELECT b.id, b.pair, b.has_open_position, b.dca_amount_per_order, k.exchange, k.encrypted_api_key, k.encrypted_api_secret, b.tenant_id, b.user_id
		FROM bots b
		JOIN exchange_api_keys k ON b.api_key_id = k.id
		WHERE b.bot_type = $1 AND b.status = $2
	`
	rows, err := db.Pool.Query(ctx, query, domain.BotTypeSignal, domain.BotStatusRunning)
	if err != nil {
		slog.Error("Failed to query signal bots", "error", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var botID, pair, exchange, encKey, encSecret, tenantID, userID string
		var hasOpenPosition bool
		var baseInvestment int64
		if err := rows.Scan(&botID, &pair, &hasOpenPosition, &baseInvestment, &exchange, &encKey, &encSecret, &tenantID, &userID); err != nil {
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

		priceFloat := domain.CentsToUSDT(priceCents)

		// Ask AI Gateway
		reqBody := map[string]interface{}{
			"provider":   "minimax",
			"message":    "Current price is " + strconv.FormatFloat(priceFloat, 'f', 2, 64) + " for " + pair + ". Answer only BUY, SELL, or HOLD. Current position: " + strconv.FormatBool(hasOpenPosition),
			"system_msg": "You are a quant trading bot. If there is an open position, look for SELL signals. If no position, look for BUY signals. Answer ONLY with BUY, SELL, or HOLD.",
			"tenant_id":  "crypto-worker",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		aiReq, _ := http.NewRequestWithContext(ctx, "POST", "http://localhost:8002/v1/chat", bytes.NewBuffer(bodyBytes))
		aiReq.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(aiReq)
		if err != nil {
			slog.Error("AI Gateway request failed", "bot_id", botID, "error", err)
			continue
		}

		var aiResp struct {
			Success bool   `json:"success"`
			Text    string `json:"text"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&aiResp); err != nil {
			slog.Error("Failed to decode AI response", "bot_id", botID, "error", err)
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		decision := aiResp.Text
		slog.Info("AI Decision", "bot_id", botID, "decision", decision)

		// Base investment (use DCA amount or fallback to $10)
		invFloat := domain.CentsToUSDT(baseInvestment)
		if invFloat <= 0 {
			invFloat = 10.0 // MVP fallback
		}

		if decision == "BUY" {
			if hasOpenPosition {
				slog.Info("Signal BUY ignored, already have open position", "bot_id", botID)
				continue
			}
			
			balance, err := client.GetBalance(ctx, "USDT") // Assuming USDT base
			if err != nil || balance < invFloat {
				slog.Warn("Insufficient balance for Signal BUY", "bot_id", botID, "balance", balance, "required", invFloat)
				continue
			}

			qty := invFloat / priceFloat
			orderID, err := client.PlaceOrder(ctx, pair, domain.OrderSideBuy, qty, 0)
			if err != nil {
				slog.Error("Failed to place signal BUY order", "bot_id", botID, "error", err)
				db.Pool.Exec(ctx, "INSERT INTO crypto_notifications (tenant_id, user_id, title, message, type, is_read, created_at) VALUES ($1, $2, $3, $4, $5, $6, NOW())",
					tenantID, userID, "Signal Order Failed", "Failed to place BUY order for "+pair+": "+err.Error(), "error", false)
			} else {
				slog.Info("Signal BUY Order Placed", "bot_id", botID, "order_id", orderID)
				db.Pool.Exec(ctx, "UPDATE bots SET has_open_position = TRUE, last_executed_at = NOW() WHERE id = $1", botID)
				
				db.Pool.Exec(ctx, "INSERT INTO crypto_notifications (tenant_id, user_id, title, message, type, is_read, created_at) VALUES ($1, $2, $3, $4, $5, $6, NOW())",
					tenantID, userID, "Signal Order Executed", "AI placed BUY order for "+pair, "success", false)
			}
		} else if decision == "SELL" {
			if !hasOpenPosition {
				slog.Info("Signal SELL ignored, no open position", "bot_id", botID)
				continue
			}
			
			// In MVP, we sell the same amount we bought. In reality, we should check asset balance (e.g. BTC)
			qty := invFloat / priceFloat 
			orderID, err := client.PlaceOrder(ctx, pair, domain.OrderSideSell, qty, 0)
			if err != nil {
				slog.Error("Failed to place signal SELL order", "bot_id", botID, "error", err)
				db.Pool.Exec(ctx, "INSERT INTO crypto_notifications (tenant_id, user_id, title, message, type, is_read, created_at) VALUES ($1, $2, $3, $4, $5, $6, NOW())",
					tenantID, userID, "Signal Order Failed", "Failed to place SELL order for "+pair+": "+err.Error(), "error", false)
			} else {
				slog.Info("Signal SELL Order Placed", "bot_id", botID, "order_id", orderID)
				db.Pool.Exec(ctx, "UPDATE bots SET has_open_position = FALSE, last_executed_at = NOW() WHERE id = $1", botID)
				
				db.Pool.Exec(ctx, "INSERT INTO crypto_notifications (tenant_id, user_id, title, message, type, is_read, created_at) VALUES ($1, $2, $3, $4, $5, $6, NOW())",
					tenantID, userID, "Signal Order Executed", "AI placed SELL order for "+pair, "success", false)
			}
		}
	}
}
