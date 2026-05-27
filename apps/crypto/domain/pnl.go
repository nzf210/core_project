package domain

// CalculatePnL calculates the realized PnL based on a list of orders.
// Simplified calculation: value of sold - cost of bought for realized.
// For unrealized: (current quantity * current price) - cost of remaining bought.
func CalculatePnL(orders []BotOrder, currentPriceCents int64) (realizedPnL int64, unrealizedPnL int64, currentValue int64) {
	var totalBoughtQty int64
	var totalSoldQty int64
	var totalCost int64
	var totalRevenue int64

	for _, order := range orders {
		if order.Status != OrderStatusFilled {
			continue
		}

		if order.Side == OrderSideBuy {
			totalBoughtQty += order.Quantity
			totalCost += order.Total + order.Fee
		} else if order.Side == OrderSideSell {
			totalSoldQty += order.Quantity
			totalRevenue += order.Total - order.Fee
		}
	}

	// Very simplified realized PnL: Total Revenue - Total Cost
	// Note: In a real system, you'd match specific buys to sells (FIFO/LIFO).
	// This is a naive calculation for the MVP.
	
	remainingQty := totalBoughtQty - totalSoldQty
	
	// Realized PnL is revenue from sales minus the proportional cost of those sales
	var costOfSold int64
	if totalBoughtQty > 0 {
		// (totalSold / totalBought) * totalCost
		costOfSold = int64((float64(totalSoldQty) / float64(totalBoughtQty)) * float64(totalCost))
	}
	realizedPnL = totalRevenue - costOfSold

	// Current value of holdings
	currentValueFloat := (float64(remainingQty) / 1e8) * float64(currentPriceCents)
	currentValue = int64(currentValueFloat)

	// Unrealized PnL: current value of holdings - cost of remaining holdings
	costOfRemaining := totalCost - costOfSold
	unrealizedPnL = currentValue - costOfRemaining

	return realizedPnL, unrealizedPnL, currentValue
}

// CalculateWinRate calculates the percentage of profitable sell orders.
func CalculateWinRate(orders []BotOrder) float64 {
	var sellOrders int
	var profitableSells int

	// Naive win rate calculation: this requires linking buy and sell orders in reality.
	// For MVP, we just return a placeholder or calculate based on overall PnL.
	// Since we don't have order linking, we'll iterate through sells and assume 
	// profit if sell price > average buy price.
	
	var totalBuyQty int64
	var totalBuyCost int64
	for _, o := range orders {
		if o.Status == OrderStatusFilled && o.Side == OrderSideBuy {
			totalBuyQty += o.Quantity
			totalBuyCost += o.Total
		}
	}

	var avgBuyPrice int64
	if totalBuyQty > 0 {
		avgBuyPrice = int64(float64(totalBuyCost) / (float64(totalBuyQty) / 1e8))
	}

	for _, o := range orders {
		if o.Status == OrderStatusFilled && o.Side == OrderSideSell {
			sellOrders++
			if avgBuyPrice > 0 && o.Price > avgBuyPrice {
				profitableSells++
			}
		}
	}

	if sellOrders == 0 {
		return 0
	}

	return float64(profitableSells) / float64(sellOrders) * 100
}
