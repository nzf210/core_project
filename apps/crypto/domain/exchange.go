package domain

import (
	"context"
	"errors"
	"strconv"

	"github.com/adshao/go-binance/v2"
)

var (
	ErrExchangeRequired    = errors.New("exchange is required")
	ErrUnsupportedExchange = errors.New("unsupported exchange")
	ErrLabelRequired       = errors.New("label is required")
	ErrAPIKeyRequired      = errors.New("api key and secret are required")
	ErrBotNameRequired     = errors.New("bot name is required")
	ErrInvalidBotType      = errors.New("invalid bot type")
	ErrPairRequired        = errors.New("trading pair is required")
	ErrDCAIntervalRequired = errors.New("dca interval is required")
	ErrDCAAmountRequired   = errors.New("dca amount must be greater than 0")
	ErrGridPriceRequired   = errors.New("grid lower and upper prices are required")
	ErrGridPriceInvalid    = errors.New("grid lower price must be less than upper price")
	ErrGridCountInvalid    = errors.New("grid count must be at least 2")
)

// ExchangeClient defines the interface for interacting with crypto exchanges
type ExchangeClient interface {
	GetPrice(ctx context.Context, symbol string) (int64, error)
	GetBalance(ctx context.Context, asset string) (float64, error)
	GetOrderStatus(ctx context.Context, symbol, orderID string) (string, error)
	PlaceOrder(ctx context.Context, symbol string, side string, quantity float64, price float64) (string, error)
}

// BinanceClient implements ExchangeClient for Binance
type BinanceClient struct {
	client *binance.Client
}

// NewExchangeClient creates a new ExchangeClient based on the exchange name
func NewExchangeClient(exchange, apiKey, apiSecret string) (ExchangeClient, error) {
	switch exchange {
	case ExchangeBinance:
		return &BinanceClient{
			client: binance.NewClient(apiKey, apiSecret),
		}, nil
	default:
		return nil, ErrUnsupportedExchange
	}
}

// GetPrice fetches the current price for a symbol and returns it in cents
func (b *BinanceClient) GetPrice(ctx context.Context, symbol string) (int64, error) {
	prices, err := b.client.NewListPricesService().Symbol(symbol).Do(ctx)
	if err != nil {
		return 0, err
	}
	if len(prices) == 0 {
		return 0, errors.New("no price found for symbol")
	}

	priceFloat, err := strconv.ParseFloat(prices[0].Price, 64)
	if err != nil {
		return 0, err
	}

	return USDTToCents(priceFloat), nil
}

// GetBalance fetches the available balance for a specific asset
func (b *BinanceClient) GetBalance(ctx context.Context, asset string) (float64, error) {
	res, err := b.client.NewGetAccountService().Do(ctx)
	if err != nil {
		return 0, err
	}
	
	for _, b := range res.Balances {
		if b.Asset == asset {
			freeFloat, err := strconv.ParseFloat(b.Free, 64)
			if err != nil {
				return 0, err
			}
			return freeFloat, nil
		}
	}
	
	return 0, nil
}

// GetOrderStatus checks the status of an order
func (b *BinanceClient) GetOrderStatus(ctx context.Context, symbol, orderID string) (string, error) {
	orderIDInt, err := strconv.ParseInt(orderID, 10, 64)
	if err != nil {
		return "", err
	}
	order, err := b.client.NewGetOrderService().Symbol(symbol).OrderID(orderIDInt).Do(ctx)
	if err != nil {
		return "", err
	}
	
	switch order.Status {
	case binance.OrderStatusTypeNew:
		return OrderStatusPending, nil
	case binance.OrderStatusTypePartiallyFilled, binance.OrderStatusTypeFilled:
		return OrderStatusFilled, nil
	case binance.OrderStatusTypeCanceled, binance.OrderStatusTypeRejected, binance.OrderStatusTypeExpired:
		return OrderStatusCancelled, nil
	default:
		return OrderStatusPending, nil
	}
}

// PlaceOrder places an order on Binance
func (b *BinanceClient) PlaceOrder(ctx context.Context, symbol string, side string, quantity float64, price float64) (string, error) {
	orderSide := binance.SideTypeBuy
	if side == OrderSideSell {
		orderSide = binance.SideTypeSell
	}

	// For simplicity in MVP, we use market orders if price is 0
	if price == 0 {
		order, err := b.client.NewCreateOrderService().
			Symbol(symbol).
			Side(orderSide).
			Type(binance.OrderTypeMarket).
			Quantity(strconv.FormatFloat(quantity, 'f', 8, 64)).
			Do(ctx)
		if err != nil {
			return "", err
		}
		return strconv.FormatInt(order.OrderID, 10), nil
	}

	// Limit order
	order, err := b.client.NewCreateOrderService().
		Symbol(symbol).
		Side(orderSide).
		Type(binance.OrderTypeLimit).
		TimeInForce(binance.TimeInForceTypeGTC).
		Quantity(strconv.FormatFloat(quantity, 'f', 8, 64)).
		Price(strconv.FormatFloat(price, 'f', 8, 64)).
		Do(ctx)
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(order.OrderID, 10), nil
}
