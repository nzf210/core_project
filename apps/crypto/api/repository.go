package main

import (
	"context"
	"errors"

	"core_project/apps/crypto/domain"
	"core_project/shared/sdk/db"
	"github.com/jackc/pgx/v5"
)

type Repository struct{}

func NewRepository() *Repository {
	return &Repository{}
}

func (r *Repository) CreateAPIKey(ctx context.Context, apiKey *domain.ExchangeAPIKey) error {
	query := `
		INSERT INTO exchange_api_keys (tenant_id, user_id, exchange, label, encrypted_api_key, encrypted_api_secret, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`
	err := db.Pool.QueryRow(ctx, query,
		apiKey.TenantID, apiKey.UserID, apiKey.Exchange, apiKey.Label, apiKey.EncryptedAPIKey, apiKey.EncryptedAPISecret, apiKey.IsActive,
	).Scan(&apiKey.ID, &apiKey.CreatedAt, &apiKey.UpdatedAt)
	return err
}

func (r *Repository) ListAPIKeys(ctx context.Context, tenantID, userID string) ([]domain.ExchangeAPIKey, error) {
	query := `
		SELECT id, tenant_id, user_id, exchange, label, encrypted_api_key, encrypted_api_secret, is_active, created_at, updated_at
		FROM exchange_api_keys
		WHERE tenant_id = $1 AND user_id = $2
		ORDER BY created_at DESC
	`
	rows, err := db.Pool.Query(ctx, query, tenantID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []domain.ExchangeAPIKey
	for rows.Next() {
		var k domain.ExchangeAPIKey
		if err := rows.Scan(&k.ID, &k.TenantID, &k.UserID, &k.Exchange, &k.Label, &k.EncryptedAPIKey, &k.EncryptedAPISecret, &k.IsActive, &k.CreatedAt, &k.UpdatedAt); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, nil
}

func (r *Repository) DeleteAPIKey(ctx context.Context, id, tenantID, userID string) error {
	query := `DELETE FROM exchange_api_keys WHERE id = $1 AND tenant_id = $2 AND user_id = $3`
	cmd, err := db.Pool.Exec(ctx, query, id, tenantID, userID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return errors.New("api key not found")
	}
	return nil
}

func (r *Repository) CreateBot(ctx context.Context, b *domain.Bot) error {
	query := `
		INSERT INTO bots (tenant_id, user_id, api_key_id, name, bot_type, pair, status, is_paper_trading, dca_interval, dca_amount_per_order, grid_lower_price, grid_upper_price, grid_count, grid_investment, total_invested, total_profit, total_trades, has_open_position, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`
	err := db.Pool.QueryRow(ctx, query,
		b.TenantID, b.UserID, b.APIKeyID, b.Name, b.BotType, b.Pair, b.Status, b.IsPaperTrading, b.DCAInterval, b.DCAAmountPerOrder, b.GridLowerPrice, b.GridUpperPrice, b.GridCount, b.GridInvestment, b.TotalInvested, b.TotalProfit, b.TotalTrades, b.HasOpenPosition,
	).Scan(&b.ID, &b.CreatedAt, &b.UpdatedAt)
	return err
}

func (r *Repository) ListBots(ctx context.Context, tenantID, userID string) ([]domain.Bot, error) {
	query := `
		SELECT id, tenant_id, user_id, api_key_id, name, bot_type, pair, status, is_paper_trading, dca_interval, dca_amount_per_order, grid_lower_price, grid_upper_price, grid_count, grid_investment, total_invested, total_profit, total_trades, has_open_position, last_executed_at, created_at, updated_at
		FROM bots
		WHERE tenant_id = $1 AND user_id = $2
		ORDER BY created_at DESC
	`
	rows, err := db.Pool.Query(ctx, query, tenantID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bots []domain.Bot
	for rows.Next() {
		var b domain.Bot
		if err := rows.Scan(&b.ID, &b.TenantID, &b.UserID, &b.APIKeyID, &b.Name, &b.BotType, &b.Pair, &b.Status, &b.IsPaperTrading, &b.DCAInterval, &b.DCAAmountPerOrder, &b.GridLowerPrice, &b.GridUpperPrice, &b.GridCount, &b.GridInvestment, &b.TotalInvested, &b.TotalProfit, &b.TotalTrades, &b.HasOpenPosition, &b.LastExecutedAt, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		bots = append(bots, b)
	}
	return bots, nil
}

func (r *Repository) GetBot(ctx context.Context, id, tenantID, userID string) (*domain.Bot, error) {
	query := `
		SELECT id, tenant_id, user_id, api_key_id, name, bot_type, pair, status, is_paper_trading, dca_interval, dca_amount_per_order, grid_lower_price, grid_upper_price, grid_count, grid_investment, total_invested, total_profit, total_trades, has_open_position, last_executed_at, created_at, updated_at
		FROM bots
		WHERE id = $1 AND tenant_id = $2 AND user_id = $3
	`
	var b domain.Bot
	err := db.Pool.QueryRow(ctx, query, id, tenantID, userID).Scan(
		&b.ID, &b.TenantID, &b.UserID, &b.APIKeyID, &b.Name, &b.BotType, &b.Pair, &b.Status, &b.IsPaperTrading, &b.DCAInterval, &b.DCAAmountPerOrder, &b.GridLowerPrice, &b.GridUpperPrice, &b.GridCount, &b.GridInvestment, &b.TotalInvested, &b.TotalProfit, &b.TotalTrades, &b.HasOpenPosition, &b.LastExecutedAt, &b.CreatedAt, &b.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, errors.New("bot not found")
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *Repository) UpdateBotStatus(ctx context.Context, id, tenantID, userID, status string) error {
	query := `UPDATE bots SET status = $1, updated_at = NOW() WHERE id = $2 AND tenant_id = $3 AND user_id = $4`
	cmd, err := db.Pool.Exec(ctx, query, status, id, tenantID, userID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return errors.New("bot not found")
	}
	return nil
}

func (r *Repository) CreateOrder(ctx context.Context, o *domain.BotOrder) error {
	query := `
		INSERT INTO bot_orders (bot_id, side, price, quantity, total, fee, exchange_order_id, status, is_paper, error_message, executed_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
		RETURNING id, created_at
	`
	err := db.Pool.QueryRow(ctx, query,
		o.BotID, o.Side, o.Price, o.Quantity, o.Total, o.Fee, o.ExchangeOrderID, o.Status, o.IsPaper, o.ErrorMessage, o.ExecutedAt,
	).Scan(&o.ID, &o.CreatedAt)
	return err
}

func (r *Repository) ListOrders(ctx context.Context, botID string) ([]domain.BotOrder, error) {
	query := `
		SELECT id, bot_id, side, price, quantity, total, fee, exchange_order_id, status, is_paper, error_message, executed_at, created_at
		FROM bot_orders
		WHERE bot_id = $1
		ORDER BY created_at DESC
	`
	rows, err := db.Pool.Query(ctx, query, botID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []domain.BotOrder
	for rows.Next() {
		var o domain.BotOrder
		if err := rows.Scan(&o.ID, &o.BotID, &o.Side, &o.Price, &o.Quantity, &o.Total, &o.Fee, &o.ExchangeOrderID, &o.Status, &o.IsPaper, &o.ErrorMessage, &o.ExecutedAt, &o.CreatedAt); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, nil
}

func (r *Repository) CreatePnLSnapshot(ctx context.Context, pnl *domain.PnLSnapshot) error {
	query := `
		INSERT INTO bot_pnl_snapshots (bot_id, total_invested, current_value, realized_pnl, unrealized_pnl, snapshot_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		RETURNING id, snapshot_at
	`
	err := db.Pool.QueryRow(ctx, query,
		pnl.BotID, pnl.TotalInvested, pnl.CurrentValue, pnl.RealizedPnL, pnl.UnrealizedPnL,
	).Scan(&pnl.ID, &pnl.SnapshotAt)
	return err
}

func (r *Repository) GetPnLSnapshots(ctx context.Context, botID string) ([]domain.PnLSnapshot, error) {
	query := `
		SELECT id, bot_id, total_invested, current_value, realized_pnl, unrealized_pnl, snapshot_at
		FROM bot_pnl_snapshots
		WHERE bot_id = $1
		ORDER BY snapshot_at ASC
	`
	rows, err := db.Pool.Query(ctx, query, botID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snaps []domain.PnLSnapshot
	for rows.Next() {
		var s domain.PnLSnapshot
		if err := rows.Scan(&s.ID, &s.BotID, &s.TotalInvested, &s.CurrentValue, &s.RealizedPnL, &s.UnrealizedPnL, &s.SnapshotAt); err != nil {
			return nil, err
		}
		snaps = append(snaps, s)
	}
	return snaps, nil
}

func (r *Repository) GetDashboardStats(ctx context.Context, tenantID, userID string) (*domain.DashboardResponse, error) {
	// Simple aggregation query
	query := `
		SELECT 
			COALESCE(SUM(total_invested + total_profit), 0) AS total_portfolio_value,
			COUNT(*) AS total_bots,
			COUNT(*) FILTER (WHERE status = 'running') AS active_bots,
			COALESCE(SUM(total_profit), 0) AS total_profit,
			COALESCE(SUM(total_trades), 0) AS total_trades
		FROM bots
		WHERE tenant_id = $1 AND user_id = $2
	`
	var stats struct {
		PortfolioValue int64
		TotalBots      int
		ActiveBots     int
		TotalProfit    int64
		TotalTrades    int
	}

	err := db.Pool.QueryRow(ctx, query, tenantID, userID).Scan(
		&stats.PortfolioValue, &stats.TotalBots, &stats.ActiveBots, &stats.TotalProfit, &stats.TotalTrades,
	)
	if err != nil {
		return nil, err
	}

	winRate := 0.0 // Requires more complex query if we really want to compute it based on orders, just return 0 for now or fake it.
	
	resp := &domain.DashboardResponse{
		TotalPortfolioValue: domain.CentsToUSDT(stats.PortfolioValue),
		ActiveBots:          stats.ActiveBots,
		TotalBots:           stats.TotalBots,
		TotalProfit:         domain.CentsToUSDT(stats.TotalProfit),
		TotalTrades:         stats.TotalTrades,
		WinRate:             winRate,
	}

	return resp, nil
}

func (r *Repository) CreateNotification(ctx context.Context, notif *domain.CryptoNotification) error {
	query := `
		INSERT INTO crypto_notifications (tenant_id, user_id, title, message, type, is_read, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		RETURNING id, created_at
	`
	err := db.Pool.QueryRow(ctx, query,
		notif.TenantID, notif.UserID, notif.Title, notif.Message, notif.Type, notif.IsRead,
	).Scan(&notif.ID, &notif.CreatedAt)
	return err
}

func (r *Repository) ListNotifications(ctx context.Context, tenantID, userID string) ([]domain.CryptoNotification, error) {
	query := `
		SELECT id, tenant_id, user_id, title, message, type, is_read, created_at
		FROM crypto_notifications
		WHERE tenant_id = $1 AND user_id = $2
		ORDER BY created_at DESC
		LIMIT 50
	`
	rows, err := db.Pool.Query(ctx, query, tenantID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifs []domain.CryptoNotification
	for rows.Next() {
		var n domain.CryptoNotification
		if err := rows.Scan(&n.ID, &n.TenantID, &n.UserID, &n.Title, &n.Message, &n.Type, &n.IsRead, &n.CreatedAt); err != nil {
			return nil, err
		}
		notifs = append(notifs, n)
	}
	return notifs, nil
}

func (r *Repository) MarkNotificationRead(ctx context.Context, id, tenantID, userID string) error {
	query := `UPDATE crypto_notifications SET is_read = TRUE WHERE id = $1 AND tenant_id = $2 AND user_id = $3`
	cmd, err := db.Pool.Exec(ctx, query, id, tenantID, userID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return errors.New("notification not found")
	}
	return nil
}

