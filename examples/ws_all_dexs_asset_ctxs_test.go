package examples

import (
	"context"
	"testing"
	"time"

	"github.com/sonirico/go-hyperliquid"
)

func TestAllDexsAssetCtxs(t *testing.T) {
	ws := hyperliquid.NewWebsocketClient("")

	if err := ws.Connect(context.Background()); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer func() { _ = ws.Close() }()

	done := make(chan bool)

	sub, err := ws.AllDexsAssetCtxs(
		func(ctxs hyperliquid.WsAllDexsAssetCtxs, err error) {
			if err != nil {
				t.Errorf("Error in all dexs asset ctxs callback: %v", err)
				return
			}

			t.Logf("Received asset ctxs for %d dexs", len(ctxs.Ctxs))
			if len(ctxs.Ctxs) > 0 {
				first := ctxs.Ctxs[0]
				t.Logf("First dex %q has %d asset ctxs", first.First, len(first.Second))
			}

			select {
			case done <- true:
			default:
			}
		},
	)

	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	defer sub.Close()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for all dexs asset ctxs update")
	}
}
