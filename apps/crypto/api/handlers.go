package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"core_project/apps/crypto/domain"
	"core_project/shared/sdk/auth"
	"core_project/shared/sdk/config"
	"core_project/shared/sdk/response"
)

type RepositoryInterface interface {
	CreateAPIKey(ctx context.Context, apiKey *domain.ExchangeAPIKey) error
	ListAPIKeys(ctx context.Context, tenantID, userID string) ([]domain.ExchangeAPIKey, error)
	GetActiveAPIKey(ctx context.Context, tenantID, userID string) (*domain.ExchangeAPIKey, error)
	DeleteAPIKey(ctx context.Context, id, tenantID, userID string) error
	CreateBot(ctx context.Context, b *domain.Bot) error
	ListBots(ctx context.Context, tenantID, userID string) ([]domain.Bot, error)
	GetBot(ctx context.Context, id, tenantID, userID string) (*domain.Bot, error)
	UpdateBotStatus(ctx context.Context, id, tenantID, userID, status string) error
	CreateOrder(ctx context.Context, o *domain.BotOrder) error
	ListOrders(ctx context.Context, botID string) ([]domain.BotOrder, error)
	CreatePnLSnapshot(ctx context.Context, pnl *domain.PnLSnapshot) error
	GetPnLSnapshots(ctx context.Context, botID string) ([]domain.PnLSnapshot, error)
	GetDashboardStats(ctx context.Context, tenantID, userID string) (*domain.DashboardResponse, error)
	CreateNotification(ctx context.Context, notif *domain.CryptoNotification) error
	ListNotifications(ctx context.Context, tenantID, userID string) ([]domain.CryptoNotification, error)
	MarkNotificationRead(ctx context.Context, id, tenantID, userID string) error
}

type Handlers struct {
	repo           RepositoryInterface
	encryptionKey  string
}

func NewHandlers(repo RepositoryInterface, encryptionKey string) *Handlers {
	return &Handlers{repo: repo, encryptionKey: encryptionKey}
}

func (h *Handlers) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	var req domain.CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	if err := req.Validate(); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error(), err)
		return
	}

	tenantID, _ := r.Context().Value(auth.TenantIDKey).(string)
	userID, _ := r.Context().Value(auth.UserIDKey).(string)

	encKey, err := domain.Encrypt(req.APIKey, config.GlobalConfig.EncryptionKey)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to encrypt API key", err)
		return
	}
	encSecret, err := domain.Encrypt(req.APISecret, config.GlobalConfig.EncryptionKey)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to encrypt API secret", err)
		return
	}

	apiKey := domain.ExchangeAPIKey{
		UserID:             userID,
		TenantID:           tenantID,
		Exchange:           req.Exchange,
		Label:              req.Label,
		EncryptedAPIKey:    encKey,
		EncryptedAPISecret: encSecret,
		IsActive:           true,
	}

	if err := h.repo.CreateAPIKey(r.Context(), &apiKey); err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to create API key", err)
		return
	}

	resp := domain.APIKeyResponse{
		ID:        apiKey.ID,
		Exchange:  apiKey.Exchange,
		Label:     apiKey.Label,
		MaskedKey: domain.MaskAPIKey(req.APIKey),
		IsActive:  apiKey.IsActive,
		CreatedAt: apiKey.CreatedAt,
	}

	response.JSON(w, http.StatusCreated, "API key created successfully", resp)
}

func (h *Handlers) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	tenantID, _ := r.Context().Value(auth.TenantIDKey).(string)
	userID, _ := r.Context().Value(auth.UserIDKey).(string)

	keys, err := h.repo.ListAPIKeys(r.Context(), tenantID, userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to list API keys", err)
		return
	}

	var resp []domain.APIKeyResponse
	for _, k := range keys {
		// Decrypt to mask
		rawKey, err := domain.Decrypt(k.EncryptedAPIKey, config.GlobalConfig.EncryptionKey)
		masked := "****"
		if err == nil {
			masked = domain.MaskAPIKey(rawKey)
		}

		resp = append(resp, domain.APIKeyResponse{
			ID:        k.ID,
			Exchange:  k.Exchange,
			Label:     k.Label,
			MaskedKey: masked,
			IsActive:  k.IsActive,
			CreatedAt: k.CreatedAt,
		})
	}
	if resp == nil {
		resp = []domain.APIKeyResponse{}
	}

	response.JSON(w, http.StatusOK, "API keys retrieved successfully", resp)
}

func (h *Handlers) DeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	id := parts[len(parts)-1]

	tenantID, _ := r.Context().Value(auth.TenantIDKey).(string)
	userID, _ := r.Context().Value(auth.UserIDKey).(string)

	if err := h.repo.DeleteAPIKey(r.Context(), id, tenantID, userID); err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to delete API key", err)
		return
	}

	response.JSON(w, http.StatusOK, "API key deleted successfully", nil)
}

func (h *Handlers) CreateBot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	var req domain.CreateBotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	if err := req.Validate(); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error(), err)
		return
	}

	tenantID, _ := r.Context().Value(auth.TenantIDKey).(string)
	userID, _ := r.Context().Value(auth.UserIDKey).(string)

	var dcaInterval *string
	if req.DCAInterval != "" {
		dcaInterval = &req.DCAInterval
	}

	bot := domain.Bot{
		UserID:            userID,
		TenantID:          tenantID,
		APIKeyID:          req.APIKeyID,
		Name:              req.Name,
		BotType:           req.BotType,
		Pair:              req.Pair,
		Status:            domain.BotStatusStopped,
		IsPaperTrading:    req.IsPaperTrading,
		DCAInterval:       dcaInterval,
		DCAAmountPerOrder: domain.USDTToCents(req.DCAAmount),
		GridLowerPrice:    domain.USDTToCents(req.GridLowerPrice),
		GridUpperPrice:    domain.USDTToCents(req.GridUpperPrice),
		GridCount:         req.GridCount,
		GridInvestment:    domain.USDTToCents(req.GridInvestment),
	}

	if err := h.repo.CreateBot(r.Context(), &bot); err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to create bot", err)
		return
	}

	response.JSON(w, http.StatusCreated, "Bot created successfully", bot.ToBotResponse())
}

func (h *Handlers) ListBots(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	tenantID, _ := r.Context().Value(auth.TenantIDKey).(string)
	userID, _ := r.Context().Value(auth.UserIDKey).(string)

	bots, err := h.repo.ListBots(r.Context(), tenantID, userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to list bots", err)
		return
	}

	var resp []domain.BotResponse
	for _, b := range bots {
		resp = append(resp, b.ToBotResponse())
	}
	if resp == nil {
		resp = []domain.BotResponse{}
	}

	response.JSON(w, http.StatusOK, "Bots retrieved successfully", resp)
}

func (h *Handlers) GetBot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	id := parts[len(parts)-1]

	tenantID, _ := r.Context().Value(auth.TenantIDKey).(string)
	userID, _ := r.Context().Value(auth.UserIDKey).(string)

	bot, err := h.repo.GetBot(r.Context(), id, tenantID, userID)
	if err != nil {
		if err.Error() == "bot not found" {
			response.Error(w, http.StatusNotFound, "Bot not found", err)
			return
		}
		response.Error(w, http.StatusInternalServerError, "Failed to get bot", err)
		return
	}

	response.JSON(w, http.StatusOK, "Bot retrieved successfully", bot.ToBotResponse())
}

func (h *Handlers) UpdateBotStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	// /api/v1/bots/{id}/status -> id is at len(parts)-2
	if len(parts) < 2 {
		response.Error(w, http.StatusBadRequest, "Invalid path", nil)
		return
	}
	id := parts[len(parts)-2]

	var payload struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	tenantID, _ := r.Context().Value(auth.TenantIDKey).(string)
	userID, _ := r.Context().Value(auth.UserIDKey).(string)

	if err := h.repo.UpdateBotStatus(r.Context(), id, tenantID, userID, payload.Status); err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to update bot status", err)
		return
	}

	response.JSON(w, http.StatusOK, "Bot status updated successfully", nil)
}

func (h *Handlers) ListOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	botID := r.URL.Query().Get("bot_id")
	if botID == "" {
		response.Error(w, http.StatusBadRequest, "bot_id query parameter is required", nil)
		return
	}

	orders, err := h.repo.ListOrders(r.Context(), botID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to list orders", err)
		return
	}

	var resp []domain.OrderResponse
	for _, o := range orders {
		resp = append(resp, o.ToOrderResponse())
	}
	if resp == nil {
		resp = []domain.OrderResponse{}
	}

	response.JSON(w, http.StatusOK, "Orders retrieved successfully", resp)
}

func (h *Handlers) GetDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	tenantID, _ := r.Context().Value(auth.TenantIDKey).(string)
	userID, _ := r.Context().Value(auth.UserIDKey).(string)

	stats, err := h.repo.GetDashboardStats(r.Context(), tenantID, userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to get dashboard stats", err)
		return
	}

	response.JSON(w, http.StatusOK, "Dashboard stats retrieved successfully", stats)
}

func (h *Handlers) ListNotifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	tenantID, _ := r.Context().Value(auth.TenantIDKey).(string)
	userID, _ := r.Context().Value(auth.UserIDKey).(string)

	notifs, err := h.repo.ListNotifications(r.Context(), tenantID, userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to list notifications", err)
		return
	}

	if notifs == nil {
		notifs = []domain.CryptoNotification{}
	}

	response.JSON(w, http.StatusOK, "Notifications retrieved successfully", notifs)
}

func (h *Handlers) MarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 2 {
		response.Error(w, http.StatusBadRequest, "Invalid path", nil)
		return
	}
	id := parts[len(parts)-2]

	tenantID, _ := r.Context().Value(auth.TenantIDKey).(string)
	userID, _ := r.Context().Value(auth.UserIDKey).(string)

	if err := h.repo.MarkNotificationRead(r.Context(), id, tenantID, userID); err != nil {
		if err.Error() == "notification not found" {
			response.Error(w, http.StatusNotFound, "Notification not found", err)
			return
		}
		response.Error(w, http.StatusInternalServerError, "Failed to update notification", err)
		return
	}

	response.JSON(w, http.StatusOK, "Notification marked as read", nil)
}

func (h *Handlers) ExecuteQuickTrade(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	var req domain.QuickTradeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	if err := req.Validate(); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error(), err)
		return
	}

	tenantID, _ := r.Context().Value(auth.TenantIDKey).(string)
	userID, _ := r.Context().Value(auth.UserIDKey).(string)
	ctx := r.Context()

	// Get active API key for the user
	apiKey, err := h.repo.GetActiveAPIKey(ctx, tenantID, userID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "No active API key found. Please add an exchange API key first.", err)
		return
	}

	// Decrypt API credentials
	apiKeyDecrypted, err := domain.Decrypt(apiKey.EncryptedAPIKey, h.encryptionKey)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to decrypt API key", err)
		return
	}
	apiSecretDecrypted, err := domain.Decrypt(apiKey.EncryptedAPISecret, h.encryptionKey)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to decrypt API secret", err)
		return
	}

	// Create exchange client
	exchangeClient, err := domain.NewExchangeClient(apiKey.Exchange, apiKeyDecrypted, apiSecretDecrypted)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Unsupported exchange: "+apiKey.Exchange, err)
		return
	}

	// Get current price
	price, err := exchangeClient.GetPrice(ctx, req.Pair)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to get current price for "+req.Pair, err)
		return
	}

	// Determine base and quote assets from pair (e.g., BTCUSDT -> base=BTC, quote=USDT)
	quoteAsset := strings.TrimPrefix(req.Pair, "BTC")
	quoteAsset = strings.TrimPrefix(quoteAsset, "ETH")
	quoteAsset = strings.TrimPrefix(quoteAsset, "BNB")
	if quoteAsset == req.Pair {
		quoteAsset = "USDT" // default
	}

	// For buying: check quote asset balance (e.g., USDT)
	// For selling: check base asset balance (e.g., BTC)
	var checkAsset string
	if req.Side == domain.OrderSideBuy {
		checkAsset = quoteAsset
	} else {
		checkAsset = strings.TrimSuffix(req.Pair, quoteAsset)
	}

	balance, err := exchangeClient.GetBalance(ctx, checkAsset)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to get balance for "+checkAsset, err)
		return
	}

	// Calculate required balance based on amount and price
	priceFloat := domain.CentsToUSDT(price)
	requiredBalance := req.Amount / priceFloat

	if req.Side == domain.OrderSideBuy && balance < req.Amount {
		response.Error(w, http.StatusBadRequest, fmt.Sprintf("Insufficient %s balance. Required: %.2f, Available: %.2f", checkAsset, req.Amount, balance), nil)
		return
	}
	if req.Side == domain.OrderSideSell && balance < requiredBalance {
		response.Error(w, http.StatusBadRequest, fmt.Sprintf("Insufficient %s balance. Required: %.8f, Available: %.8f", checkAsset, requiredBalance, balance), nil)
		return
	}

	// Place the order (market order)
	orderID, err := exchangeClient.PlaceOrder(ctx, req.Pair, req.Side, requiredBalance, 0)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to place order: "+err.Error(), err)
		return
	}

	// Create success notification
	notif := &domain.CryptoNotification{
		TenantID: tenantID,
		UserID:   userID,
		Title:    "Quick Trade Executed",
		Message:  fmt.Sprintf("Successfully placed %s order for %.8f %s at %s (Order ID: %s)", req.Side, requiredBalance, req.Pair, fmt.Sprintf("$%.2f", priceFloat), orderID),
		Type:     "success",
		IsRead:   false,
	}
	_ = h.repo.CreateNotification(ctx, notif)

	response.JSON(w, http.StatusOK, "Quick trade executed successfully", map[string]interface{}{
		"success":      true,
		"order_id":     orderID,
		"pair":         req.Pair,
		"side":         req.Side,
		"quantity":     requiredBalance,
		"price":        priceFloat,
		"total":        req.Amount,
		"quote_asset":  quoteAsset,
	})
}
