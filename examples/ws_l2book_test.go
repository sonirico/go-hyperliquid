package examples

import (
	"context"
	"testing"
	"time"

	"github.com/sonirico/go-hyperliquid"
)

func TestL2BookWebSocket(t *testing.T) {
	ws := hyperliquid.NewWebsocketClient("")

	if err := ws.Connect(context.Background()); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer func() { _ = ws.Close() }()

	done := make(chan bool)

	sub, err := ws.L2Book(
		hyperliquid.L2BookSubscriptionParams{
			Coin:     "BTC",
			Mantissa: 2,
			NSigFigs: 5,
		},
		func(orderbook hyperliquid.L2Book, err error) {
			if err != nil {
				t.Errorf("Error in l2book callback: %v", err)
				return
			}

			t.Logf("Received L2Book: %+v", orderbook)

			done <- true
		},
	)

	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	defer sub.Close()

	select {
	case <-done:
		// Test passed
	case <-time.After(10 * time.Second):
		t.Error("Timeout waiting for candle update")
	}
}
