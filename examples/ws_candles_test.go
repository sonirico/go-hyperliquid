package examples

import (
	"context"
	"testing"
	"time"

	hl "github.com/corverroos/go-hyperliquid"
)

func TestCandleWebSocket(t *testing.T) {
	ws := hl.NewWebsocketClient(hl.MainnetAPIURL)

	if err := ws.Connect(context.Background()); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Close()

	done := make(chan bool)

	sub, err := ws.Candles(
		hl.CandlesSubscriptionParams{
			Coin:     "BTC",
			Interval: "1m",
		},
		func(candle hl.Candle, err error) {
			if err != nil {
				t.Errorf("Error in candle callback: %v", err)
				return
			}

			t.Logf("Received candle: %+v", candle)

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
