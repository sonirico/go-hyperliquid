package examples

import (
	"context"
	"testing"
	"time"

	"github.com/sonirico/go-hyperliquid"
)

func TestAllMidsWebSocket(t *testing.T) {
	ws := hyperliquid.NewWebsocketClient("")

	if err := ws.Connect(context.Background()); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer func() { _ = ws.Close() }()

	done := make(chan bool)

	sub, err := ws.AllMids(
		hyperliquid.AllMidsSubscriptionParams{
			Dex: nil, // Use default (first perp dex)
		},
		func(allmids hyperliquid.AllMids, err error) {
			if err != nil {
				t.Errorf("Error in allmids callback: %v", err)
				return
			}

			t.Logf("Received AllMids: %+v", allmids)

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
		t.Error("Timeout waiting for allmids update")
	}
}
