package examples

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"testing"

	"github.com/joho/godotenv"

	"github.com/sonirico/go-hyperliquid"
)

func TestCancelOrder(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t)

	// First place an order to cancel
	orderReq := hyperliquid.CreateOrderRequest{
		Coin:  "BTC",
		IsBuy: true,
		Size:  0.1,
		Price: 40000.0,
		OrderType: hyperliquid.OrderType{
			Limit: &hyperliquid.LimitOrderType{
				Tif: hyperliquid.TifGtc,
			},
		},
	}

	resp, err := exchange.Order(context.TODO(), orderReq, nil)
	if err != nil {
		t.Fatalf("Failed to place order: %v", err)
	}

	// Extract order ID from response
	var orderID int64
	if resp.Resting != nil {
		orderID = resp.Resting.Oid
	} else {
		t.Skip("Order was filled immediately, cannot test cancel")
	}

	// Cancel the order
	cancelResp, err := exchange.Cancel(context.TODO(), "BTC", orderID)
	if err != nil {
		t.Fatalf("Failed to cancel order: %v", err)
	}

	t.Logf("Cancel response: %+v", cancelResp)
}

func TestCancelByCloid(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t)

	// Generate a random cloid
	cloidBytes := make([]byte, 16)
	if _, err := rand.Read(cloidBytes); err != nil {
		t.Fatalf("Failed to generate random cloid: %v", err)
	}
	cloid := "0x" + hex.EncodeToString(cloidBytes)

	// Place an order with cloid
	orderReq := hyperliquid.CreateOrderRequest{
		Coin:  "BTC",
		IsBuy: true,
		Size:  0.1,
		Price: 40000.0,
		OrderType: hyperliquid.OrderType{
			Limit: &hyperliquid.LimitOrderType{
				Tif: hyperliquid.TifGtc,
			},
		},
		ClientOrderID: &cloid,
	}

	_, err := exchange.Order(context.TODO(), orderReq, nil)
	if err != nil {
		t.Fatalf("Failed to place order: %v", err)
	}

	// Cancel by cloid
	cancelResp, err := exchange.CancelByCloid(context.TODO(), "BTC", cloid)
	if err != nil {
		t.Fatalf("Failed to cancel order by cloid: %v", err)
	}

	t.Logf("Cancel by cloid response: %+v", cancelResp)
}
