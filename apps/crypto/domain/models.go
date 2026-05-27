package domain

import "time"

// =============================================
// Constants
// =============================================

// Bot types
const (
	BotTypeDCA    = "dca"
	BotTypeGrid   = "grid"
	BotTypeSignal = "signal"
)

// Bot statuses
const (
	BotStatusRunning = "running"
	BotStatusPaused  = "paused"
	BotStatusStopped = "stopped"
)

// DCA intervals
const (
	DCAIntervalHourly  = "hourly"
	DCAIntervalDaily   = "daily"
	DCAIntervalWeekly  = "weekly"
	DCAIntervalMonthly = "monthly"
)

// Order sides
const (
	OrderSideBuy  = "buy"
	OrderSideSell = "sell"
)

// Order statuses
const (
	OrderStatusPending   = "pending"
	OrderStatusFilled    = "filled"
	OrderStatusFailed    = "failed"
	OrderStatusCancelled = "cancelled"
)

// Supported exchanges
const (
	ExchangeBinance    = "binance"
	ExchangeTokocrypto = "tokocrypto"
	ExchangeIndodax    = "indodax"
)

// =============================================
// Models (Database entities)
// =============================================

// ExchangeAPIKey represents a stored exchange API key
type ExchangeAPIKey struct {
	ID                 string    `json:"id"`
	UserID             string    `json:"user_id"`
	TenantID           string    `json:"tenant_id"`
	Exchange           string    `json:"exchange"`
	Label              string    `json:"label"`
	EncryptedAPIKey    string    `json:"-"` // Never expose encrypted data in JSON
	EncryptedAPISecret string    `json:"-"` // Never expose encrypted data in JSON
	IsActive           bool      `json:"is_active"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// Bot represents a trading bot configuration
type Bot struct {
	ID               string     `json:"id"`
	UserID           string     `json:"user_id"`
	TenantID         string     `json:"tenant_id"`
	APIKeyID         *string    `json:"api_key_id,omitempty"`
	Name             string     `json:"name"`
	BotType          string     `json:"bot_type"`
	Pair             string     `json:"pair"`
	Status           string     `json:"status"`
	IsPaperTrading   bool       `json:"is_paper_trading"`
	DCAInterval      *string    `json:"dca_interval,omitempty"`
	DCAAmountPerOrder int64     `json:"dca_amount_per_order,omitempty"` // USDT cents
	GridLowerPrice   int64      `json:"grid_lower_price,omitempty"`    // USDT cents
	GridUpperPrice   int64      `json:"grid_upper_price,omitempty"`    // USDT cents
	GridCount        int        `json:"grid_count,omitempty"`
	GridInvestment   int64      `json:"grid_investment,omitempty"`     // USDT cents
	TotalInvested    int64      `json:"total_invested"`                // USDT cents
	TotalProfit      int64      `json:"total_profit"`                  // USDT cents
	TotalTrades      int        `json:"total_trades"`
	HasOpenPosition  bool       `json:"has_open_position"`
	LastExecutedAt   *time.Time `json:"last_executed_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// BotOrder represents an executed trade order
type BotOrder struct {
	ID              string     `json:"id"`
	BotID           string     `json:"bot_id"`
	Side            string     `json:"side"`
	Price           int64      `json:"price"`             // USDT cents
	Quantity        int64      `json:"quantity"`           // satoshi-level (x10^8)
	Total           int64      `json:"total"`              // USDT cents
	Fee             int64      `json:"fee"`                // USDT cents
	ExchangeOrderID *string    `json:"exchange_order_id,omitempty"`
	Status          string     `json:"status"`
	IsPaper         bool       `json:"is_paper"`
	ErrorMessage    *string    `json:"error_message,omitempty"`
	ExecutedAt      *time.Time `json:"executed_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// PnLSnapshot represents a point-in-time performance snapshot
type PnLSnapshot struct {
	ID            string    `json:"id"`
	BotID         string    `json:"bot_id"`
	TotalInvested int64     `json:"total_invested"` // USDT cents
	CurrentValue  int64     `json:"current_value"`  // USDT cents
	RealizedPnL   int64     `json:"realized_pnl"`   // USDT cents
	UnrealizedPnL int64     `json:"unrealized_pnl"` // USDT cents
	SnapshotAt    time.Time `json:"snapshot_at"`
}

// BotGridLine represents a single order line in a grid bot
type BotGridLine struct {
	ID              string    `json:"id"`
	BotID           string    `json:"bot_id"`
	Price           int64     `json:"price"` // USDT cents
	Side            string    `json:"side"`  // 'buy' or 'sell'
	Status          string    `json:"status"` // 'pending', 'active', 'filled'
	ExchangeOrderID *string   `json:"exchange_order_id,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// CryptoNotification represents a notification for the user
type CryptoNotification struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	UserID    string    `json:"user_id"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Type      string    `json:"type"` // e.g. "info", "success", "error", "trade"
	IsRead    bool      `json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
}

// =============================================
// Request DTOs
// =============================================

// CreateAPIKeyRequest is the request to add a new exchange API key
type CreateAPIKeyRequest struct {
	Exchange  string `json:"exchange"`
	Label     string `json:"label"`
	APIKey    string `json:"api_key"`
	APISecret string `json:"api_secret"`
}

// Validate validates the CreateAPIKeyRequest
func (r *CreateAPIKeyRequest) Validate() error {
	if r.Exchange == "" {
		return ErrExchangeRequired
	}
	if r.Exchange != ExchangeBinance && r.Exchange != ExchangeTokocrypto && r.Exchange != ExchangeIndodax {
		return ErrUnsupportedExchange
	}
	if r.Label == "" {
		return ErrLabelRequired
	}
	if r.APIKey == "" || r.APISecret == "" {
		return ErrAPIKeyRequired
	}
	return nil
}

// CreateBotRequest is the request to create a new trading bot
type CreateBotRequest struct {
	Name            string  `json:"name"`
	BotType         string  `json:"bot_type"`
	Pair            string  `json:"pair"`
	APIKeyID        *string `json:"api_key_id,omitempty"`
	IsPaperTrading  bool    `json:"is_paper_trading"`
	// DCA fields
	DCAInterval     string  `json:"dca_interval,omitempty"`
	DCAAmount       float64 `json:"dca_amount,omitempty"`      // USDT (will be converted to cents internally)
	// Grid fields
	GridLowerPrice  float64 `json:"grid_lower_price,omitempty"` // USDT
	GridUpperPrice  float64 `json:"grid_upper_price,omitempty"` // USDT
	GridCount       int     `json:"grid_count,omitempty"`
	GridInvestment  float64 `json:"grid_investment,omitempty"`  // USDT
}

// QuickTradeRequest is the request to place a quick manual trade
type QuickTradeRequest struct {
	Pair     string  `json:"pair"`
	Side     string  `json:"side"` // "buy" or "sell"
	Quantity float64 `json:"quantity"` // in base coin (e.g. BTC) or USDT amount? If amount, we should specify. Let's use USDT amount for simplicity
	Amount   float64 `json:"amount"` // USDT
}

// Validate validates the QuickTradeRequest
func (r *QuickTradeRequest) Validate() error {
	if r.Pair == "" {
		return ErrPairRequired
	}
	if r.Side != OrderSideBuy && r.Side != OrderSideSell {
		return ErrInvalidBotType // reuse or create new error
	}
	if r.Amount <= 0 {
		return ErrDCAAmountRequired // reuse or create new error
	}
	return nil
}

// Validate validates the CreateBotRequest
func (r *CreateBotRequest) Validate() error {
	if r.Name == "" {
		return ErrBotNameRequired
	}
	if r.BotType != BotTypeDCA && r.BotType != BotTypeGrid && r.BotType != BotTypeSignal {
		return ErrInvalidBotType
	}
	if r.Pair == "" {
		return ErrPairRequired
	}
	if r.BotType == BotTypeDCA {
		if r.DCAInterval == "" {
			return ErrDCAIntervalRequired
		}
		if r.DCAAmount <= 0 {
			return ErrDCAAmountRequired
		}
	}
	if r.BotType == BotTypeGrid {
		if r.GridLowerPrice <= 0 || r.GridUpperPrice <= 0 {
			return ErrGridPriceRequired
		}
		if r.GridLowerPrice >= r.GridUpperPrice {
			return ErrGridPriceInvalid
		}
		if r.GridCount < 2 {
			return ErrGridCountInvalid
		}
	}
	return nil
}

// USDTToCents converts a USDT float value to int64 cents (x100)
func USDTToCents(usdt float64) int64 {
	return int64(usdt * 100)
}

// CentsToUSDT converts cents (int64) to USDT float for display
func CentsToUSDT(cents int64) float64 {
	return float64(cents) / 100.0
}

// =============================================
// Response DTOs
// =============================================

// APIKeyResponse is the masked response for an API key
type APIKeyResponse struct {
	ID        string    `json:"id"`
	Exchange  string    `json:"exchange"`
	Label     string    `json:"label"`
	MaskedKey string    `json:"masked_key"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// BotResponse is the response for bot data (with float USDT values for frontend)
type BotResponse struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	BotType          string     `json:"bot_type"`
	Pair             string     `json:"pair"`
	Status           string     `json:"status"`
	IsPaperTrading   bool       `json:"is_paper_trading"`
	DCAInterval      *string    `json:"dca_interval,omitempty"`
	DCAAmountPerOrder float64   `json:"dca_amount_per_order,omitempty"`
	GridLowerPrice   float64    `json:"grid_lower_price,omitempty"`
	GridUpperPrice   float64    `json:"grid_upper_price,omitempty"`
	GridCount        int        `json:"grid_count,omitempty"`
	GridInvestment   float64    `json:"grid_investment,omitempty"`
	TotalInvested    float64    `json:"total_invested"`
	TotalProfit      float64    `json:"total_profit"`
	TotalTrades      int        `json:"total_trades"`
	LastExecutedAt   *time.Time `json:"last_executed_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

// ToBotResponse converts a Bot to a BotResponse (converting cents to USDT)
func (b *Bot) ToBotResponse() BotResponse {
	return BotResponse{
		ID:                b.ID,
		Name:              b.Name,
		BotType:           b.BotType,
		Pair:              b.Pair,
		Status:            b.Status,
		IsPaperTrading:    b.IsPaperTrading,
		DCAInterval:       b.DCAInterval,
		DCAAmountPerOrder: CentsToUSDT(b.DCAAmountPerOrder),
		GridLowerPrice:    CentsToUSDT(b.GridLowerPrice),
		GridUpperPrice:    CentsToUSDT(b.GridUpperPrice),
		GridCount:         b.GridCount,
		GridInvestment:    CentsToUSDT(b.GridInvestment),
		TotalInvested:     CentsToUSDT(b.TotalInvested),
		TotalProfit:       CentsToUSDT(b.TotalProfit),
		TotalTrades:       b.TotalTrades,
		LastExecutedAt:    b.LastExecutedAt,
		CreatedAt:         b.CreatedAt,
	}
}

// DashboardResponse is the aggregated dashboard stats
type DashboardResponse struct {
	TotalPortfolioValue float64 `json:"total_portfolio_value"`
	ActiveBots          int     `json:"active_bots"`
	TotalBots           int     `json:"total_bots"`
	TotalProfit         float64 `json:"total_profit"`
	TotalTrades         int     `json:"total_trades"`
	WinRate             float64 `json:"win_rate"`
}

// OrderResponse is the response for a bot order (with float USDT values)
type OrderResponse struct {
	ID              string     `json:"id"`
	BotID           string     `json:"bot_id"`
	Side            string     `json:"side"`
	Price           float64    `json:"price"`
	Quantity        float64    `json:"quantity"`
	Total           float64    `json:"total"`
	Fee             float64    `json:"fee"`
	ExchangeOrderID *string    `json:"exchange_order_id,omitempty"`
	Status          string     `json:"status"`
	IsPaper         bool       `json:"is_paper"`
	ErrorMessage    *string    `json:"error_message,omitempty"`
	ExecutedAt      *time.Time `json:"executed_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// ToOrderResponse converts a BotOrder to an OrderResponse
func (o *BotOrder) ToOrderResponse() OrderResponse {
	return OrderResponse{
		ID:              o.ID,
		BotID:           o.BotID,
		Side:            o.Side,
		Price:           CentsToUSDT(o.Price),
		Quantity:        float64(o.Quantity) / 1e8, // satoshi to coin
		Total:           CentsToUSDT(o.Total),
		Fee:             CentsToUSDT(o.Fee),
		ExchangeOrderID: o.ExchangeOrderID,
		Status:          o.Status,
		IsPaper:         o.IsPaper,
		ErrorMessage:    o.ErrorMessage,
		ExecutedAt:      o.ExecutedAt,
		CreatedAt:       o.CreatedAt,
	}
}

// MaskAPIKey masks an API key for display (show first 6 and last 4 chars)
func MaskAPIKey(key string) string {
	if len(key) < 10 {
		return "****"
	}
	return key[:6] + "..." + key[len(key)-4:]
}
