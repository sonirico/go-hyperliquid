package examples

import (
	"context"
	"testing"
	"time"

	"github.com/sonirico/go-hyperliquid"
)

func TestActiveAssetCtx(t *testing.T) {
	ws := hyperliquid.NewWebsocketClient("")

	if err := ws.Connect(context.Background()); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Close()

	done := make(chan bool)

	sub, err := ws.ActiveAssetCtx(
		hyperliquid.ActiveAssetCtxSubscriptionParams{
			Coin: "BTC",
		},
		func(activeAssetCtx hyperliquid.ActiveAssetCtx, err error) {
			if err != nil {
				t.Errorf("Error in active asset ctx callback: %v", err)
				return
			}

			t.Logf("Received active asset ctx: %+v", activeAssetCtx)

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
		t.Error("Timeout waiting for active asset ctx update")
	}
}
