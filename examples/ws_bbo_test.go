package examples

import (
	"context"
	"testing"
	"time"

	"github.com/sonirico/go-hyperliquid"
)

func TestBboWebSocket(t *testing.T) {
	ws := hyperliquid.NewWebsocketClient("")

	if err := ws.Connect(context.Background()); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Close()

	done := make(chan bool)

	sub, err := ws.Bbo(
		hyperliquid.BboSubscriptionParams{
			Coin: "BTC",
		},
		func(bbo hyperliquid.Bbo, err error) {
			if err != nil {
				t.Errorf("Error in bbo callback: %v", err)
				return
			}

			t.Logf("Received bbo: %+v", bbo)

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
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for bbo update")
	}
}
