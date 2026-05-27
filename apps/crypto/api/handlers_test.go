package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"core_project/apps/crypto/domain"
	"core_project/shared/sdk/auth"
	"core_project/shared/sdk/config"
)

// MockRepository implements RepositoryInterface for testing
type MockRepository struct {
	Bots []domain.Bot
	Keys []domain.ExchangeAPIKey
}

func (m *MockRepository) CreateAPIKey(ctx context.Context, apiKey *domain.ExchangeAPIKey) error {
	apiKey.ID = "test-key-id"
	apiKey.CreatedAt = time.Now()
	m.Keys = append(m.Keys, *apiKey)
	return nil
}

func (m *MockRepository) ListAPIKeys(ctx context.Context, tenantID, userID string) ([]domain.ExchangeAPIKey, error) {
	return m.Keys, nil
}

func (m *MockRepository) DeleteAPIKey(ctx context.Context, id, tenantID, userID string) error {
	return nil
}

func (m *MockRepository) CreateBot(ctx context.Context, b *domain.Bot) error {
	b.ID = "test-bot-id"
	b.CreatedAt = time.Now()
	m.Bots = append(m.Bots, *b)
	return nil
}

func (m *MockRepository) ListBots(ctx context.Context, tenantID, userID string) ([]domain.Bot, error) {
	return m.Bots, nil
}

func (m *MockRepository) GetBot(ctx context.Context, id, tenantID, userID string) (*domain.Bot, error) {
	for _, b := range m.Bots {
		if b.ID == id {
			return &b, nil
		}
	}
	return nil, nil // Or error "bot not found"
}

func (m *MockRepository) UpdateBotStatus(ctx context.Context, id, tenantID, userID, status string) error {
	return nil
}

func (m *MockRepository) CreateOrder(ctx context.Context, o *domain.BotOrder) error {
	return nil
}

func (m *MockRepository) ListOrders(ctx context.Context, botID string) ([]domain.BotOrder, error) {
	return []domain.BotOrder{}, nil
}

func (m *MockRepository) CreatePnLSnapshot(ctx context.Context, pnl *domain.PnLSnapshot) error {
	return nil
}

func (m *MockRepository) GetPnLSnapshots(ctx context.Context, botID string) ([]domain.PnLSnapshot, error) {
	return []domain.PnLSnapshot{}, nil
}

func (m *MockRepository) GetDashboardStats(ctx context.Context, tenantID, userID string) (*domain.DashboardResponse, error) {
	return &domain.DashboardResponse{
		TotalPortfolioValue: 100.50,
		ActiveBots:          1,
		TotalBots:           2,
		TotalProfit:         10.50,
		TotalTrades:         5,
		WinRate:             0,
	}, nil
}

func TestCreateAPIKey(t *testing.T) {
	config.GlobalConfig = &config.Config{
		EncryptionKey: "12345678901234567890123456789012", // 32 bytes
	}

	mockRepo := &MockRepository{}
	handlers := NewHandlers(mockRepo)

	reqBody := domain.CreateAPIKeyRequest{
		Exchange:  domain.ExchangeBinance,
		Label:     "My Binance Key",
		APIKey:    "test-api-key",
		APISecret: "test-api-secret",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api-keys", bytes.NewReader(bodyBytes))
	req = req.WithContext(context.WithValue(req.Context(), auth.TenantIDKey, "tenant-1"))
	req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, "user-1"))

	rr := httptest.NewRecorder()
	handlers.CreateAPIKey(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d. Body: %s", rr.Code, rr.Body.String())
	}
}

func TestListBots(t *testing.T) {
	mockRepo := &MockRepository{
		Bots: []domain.Bot{
			{ID: "bot-1", Name: "Bot 1", BotType: domain.BotTypeDCA},
		},
	}
	handlers := NewHandlers(mockRepo)

	req := httptest.NewRequest(http.MethodGet, "/bots", nil)
	req = req.WithContext(context.WithValue(req.Context(), auth.TenantIDKey, "tenant-1"))
	req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, "user-1"))

	rr := httptest.NewRecorder()
	handlers.ListBots(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}
