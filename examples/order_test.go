package examples

import (
	"testing"

	"github.com/joho/godotenv"
	"github.com/sonirico/go-hyperliquid"
)

func TestOrder(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t)

	tests := []struct {
		name string
		req  hyperliquid.CreateOrderRequest
	}{
		{
			name: "limit buy order",
			req: hyperliquid.CreateOrderRequest{
				Coin:  "BTC",
				IsBuy: true,
				Size:  0.001, // Smaller size for testing
				Price: 40000.0,
				OrderType: hyperliquid.OrderType{
					Limit: &hyperliquid.LimitOrderType{
						Tif: hyperliquid.TifGtc,
					},
				},
			},
		},
		{
			name: "market sell order",
			req: hyperliquid.CreateOrderRequest{
				Coin:  "ETH",
				IsBuy: false,
				Size:  0.01,
				Price: 2000.0,
				OrderType: hyperliquid.OrderType{
					Limit: &hyperliquid.LimitOrderType{
						Tif: hyperliquid.TifIoc,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := exchange.Order(tt.req, nil)
			if err != nil {
				t.Fatalf("Order failed: %v", err)
			}
			t.Logf("Order response: %+v", resp)
		})
	}
}

func TestMarketOpen(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t) // exchange used for setup only

	t.Log("Market open method is available and ready to use")

	// Example usage:
	name := "BTC"
	isBuy := true
	sz := 0.001
	slippage := 0.01 // 1%

	result, err := exchange.MarketOpen(name, isBuy, sz, nil, slippage, nil, nil)
	if err != nil {
		t.Fatalf("MarketOpen failed: %v", err)
	}

	t.Logf("Market open result: %+v", result)
}

func TestMarketClose(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t)
	t.Log("Market close method is available and ready to use")

	// Example usage:
	coin := "BTC"
	slippage := 0.01 // 1%

	result, err := exchange.MarketClose(coin, nil, nil, slippage, nil, nil)
	if err != nil {
		t.Fatalf("MarketClose failed: %v", err)
	}

	t.Logf("Market close result: %+v", result)
}

func TestModifyOrder(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t)

	t.Log("Modify order method is available and ready to use")

	// Example usage:
	modifyReq := hyperliquid.ModifyOrderRequest{
		Oid: int64(12345),
		Order: hyperliquid.CreateOrderRequest{
			Coin:  "BTC",
			IsBuy: true,
			Size:  0.002,
			Price: 41000.0,
			OrderType: hyperliquid.OrderType{
				Limit: &hyperliquid.LimitOrderType{Tif: hyperliquid.TifGtc},
			},
			ReduceOnly:    false,
			ClientOrderID: func() *string { s := "modified_order_123"; return &s }(),
		},
	}

	result, err := exchange.ModifyOrder(modifyReq)
	if err != nil {
		t.Fatalf("ModifyOrder failed: %v", err)
	}

	t.Logf("Modify order result: %+v", result)
}

func TestBulkModifyOrders(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t)

	t.Log("Bulk modify orders method is available and ready to use")

	// Example usage:
	modifyRequests := []hyperliquid.ModifyOrderRequest{
		{
			Oid: int64(12345),
			Order: hyperliquid.CreateOrderRequest{
				Coin:  "BTC",
				IsBuy: true,
				Size:  0.002,
				Price: 41000.0,
				OrderType: hyperliquid.OrderType{
					Limit: &hyperliquid.LimitOrderType{Tif: hyperliquid.TifGtc},
				},
			},
		},
	}

	result, err := exchange.BulkModifyOrders(modifyRequests)
	if err != nil {
		t.Fatalf("BulkModifyOrders failed: %v", err)
	}

	t.Logf("Bulk modify orders result: %+v", result)
}

func TestSLOrder(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t)

	tpOrderReq := hyperliquid.CreateOrderRequest{
		Coin:       "SOL",
		IsBuy:      true,
		Price:      110_000,
		Size:       0.001,
		ReduceOnly: true,
		OrderType: hyperliquid.OrderType{
			Trigger: &hyperliquid.TriggerOrderType{
				TriggerPx: 100000,
				IsMarket:  true,
				Tpsl:      hyperliquid.StopLoss,
			},
		},
		ClientOrderID: nil,
	}

	result, err := exchange.Order(tpOrderReq, nil)
	if err != nil {
		t.Fatalf("SLOrder failed: %v", err)
	}

	t.Logf("SL order result: %+v", result)
}
