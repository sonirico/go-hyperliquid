package examples

import (
	"context"
	"testing"

	"github.com/sonirico/go-hyperliquid"
)

func TestOrder(t *testing.T) {
	_ = loadEnvClean()
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
			resp, err := exchange.Order(context.TODO(), tt.req, nil)
			if err != nil {
				t.Fatalf("Order failed: %v", err)
			}
			t.Logf("Order response: %+v", resp)
		})
	}
}

func TestMarketOpen(t *testing.T) {
	_ = loadEnvClean()
	exchange := newTestExchange(t) // exchange used for setup only

	t.Log("Market open method is available and ready to use")

	// Example usage:
	name := "BTC"
	isBuy := true
	sz := 0.001
	slippage := 0.01 // 1%

	result, err := exchange.MarketOpen(context.TODO(), name, isBuy, sz, nil, slippage, nil, nil)
	if err != nil {
		t.Fatalf("MarketOpen failed: %v", err)
	}

	t.Logf("Market open result: %+v", result)
}

func TestMarketClose(t *testing.T) {
	_ = loadEnvClean()
	exchange := newTestExchange(t)
	t.Log("Market close method is available and ready to use")

	// Example usage:
	coin := "BTC"
	slippage := 0.01 // 1%

	result, err := exchange.MarketClose(context.TODO(), coin, nil, nil, slippage, nil, nil)
	if err != nil {
		t.Fatalf("MarketClose failed: %v", err)
	}

	t.Logf("Market close result: %+v", result)
}

func TestModifyOrder(t *testing.T) {
	_ = loadEnvClean()
	exchange := newTestExchange(t)

	t.Log("Modify order method is available and ready to use")

	// Example usage:
	oid := int64(12345)

	modifyReq := hyperliquid.ModifyOrderRequest{
		Oid: &oid,
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

	result, err := exchange.ModifyOrder(context.TODO(), modifyReq)
	if err != nil {
		t.Fatalf("ModifyOrder failed: %v", err)
	}

	t.Logf("Modify order result: %+v", result)
}

func TestBulkModifyOrders(t *testing.T) {
	_ = loadEnvClean()
	exchange := newTestExchange(t)

	t.Log("Bulk modify orders method is available and ready to use")

	// Example usage:
	oid := int64(12345)
	modifyRequests := []hyperliquid.ModifyOrderRequest{
		{
			Oid: &oid,
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

	result, err := exchange.BulkModifyOrders(context.TODO(), modifyRequests)
	if err != nil {
		t.Fatalf("BulkModifyOrders failed: %v", err)
	}

	t.Logf("Bulk modify orders result: %+v", result)
}

func Test_create_order_cancel(t *testing.T) {
	_ = loadEnvClean(".env.testnet")
	exchange := newTestExchange(t)

	// Place a limit order far from market price so it won't fill
	cloid := "0x06c60000000000000000000000003f5a"
	orderReq := hyperliquid.CreateOrderRequest{
		Coin:  "DOGE",
		IsBuy: true,
		Size:  1000,
		Price: 0.01, // Far below market, won't execute
		OrderType: hyperliquid.OrderType{
			Limit: &hyperliquid.LimitOrderType{
				Tif: hyperliquid.TifGtc,
			},
		},
		ClientOrderID: &cloid,
	}

	result, err := exchange.Order(context.TODO(), orderReq, nil)
	if err != nil {
		if hyperliquid.IsWalletDoesNotExistError(err) {
			t.Skip("wallet not registered on testnet")
		}
		t.Fatalf("failed to place order: %v", err)
	}

	if result.Resting == nil {
		t.Fatalf("expected resting order, got: %+v", result)
	}

	oid := result.Resting.Oid
	t.Logf("placed order: oid=%d cloid=%s", oid, *result.Resting.ClientID)

	// Cancel the order
	cancelResult, err := exchange.Cancel(context.TODO(), orderReq.Coin, oid)
	if err != nil {
		t.Fatalf("failed to cancel order: %v", err)
	}

	t.Logf("cancelled order: %+v", cancelResult)
}

func TestSLOrder(t *testing.T) {
	_ = loadEnvClean(".env.testnet")
	exchange := newTestExchange(t)

	tpOrderReq := hyperliquid.CreateOrderRequest{
		Coin:  "SOL",
		IsBuy: true,
		Price: 110_000,
		Size:  0.001,
		OrderType: hyperliquid.OrderType{
			Trigger: &hyperliquid.TriggerOrderType{
				TriggerPx: 100000,
				IsMarket:  true,
				Tpsl:      hyperliquid.StopLoss,
			},
		},
		ClientOrderID: func() *string { s := "0x06c60000000000000000000000003f5a"; return &s }(),
	}

	result, err := exchange.Order(context.TODO(), tpOrderReq, nil)
	if err != nil {
		// If wallet doesn't exist on Hyperliquid, skip the test
		// This is expected for test credentials that haven't been funded/registered
		if hyperliquid.IsWalletDoesNotExistError(err) {
			t.Skipf("Wallet not registered on Hyperliquid (expected for test credentials): %v", err)
		}
		t.Fatalf("SLOrder failed: %v", err)
	}

	t.Logf("SL order result: %+v", result)
}
