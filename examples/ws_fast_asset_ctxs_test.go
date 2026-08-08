package examples

import (
	"context"
	"testing"
	"time"

	"github.com/sonirico/go-hyperliquid"
)

func TestFastAssetCtxs(t *testing.T) {
	ws := hyperliquid.NewWebsocketClient("")

	if err := ws.Connect(context.Background()); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer func() { _ = ws.Close() }()

	done := make(chan bool)

	sub, err := ws.FastAssetCtxs(
		func(ctxs hyperliquid.WsFastAssetCtxs, err error) {
			if err != nil {
				t.Errorf("Error in fast asset ctxs callback: %v", err)
				return
			}

			t.Logf("Received fast asset ctxs for %d coins", len(ctxs))

			done <- true
		},
	)

	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	defer sub.Close()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for fast asset ctxs update")
	}
}
