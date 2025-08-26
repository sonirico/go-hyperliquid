package examples

import (
	"context"
	"testing"
	"time"

	"github.com/sonirico/go-hyperliquid"
)

func TestOrderUpdatesWebSocket(t *testing.T) {
	ws := hyperliquid.NewWebsocketClient("")

	if err := ws.Connect(context.Background()); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Close()

	done := make(chan bool)

	// Use a test address (you would replace this with a real address in practice)
	testAddress := "0x1234567890123456789012345678901234567890"

	sub, err := ws.OrderUpdates(
		hyperliquid.OrderUpdatesSubscriptionParams{
			User: testAddress,
		},
		func(orders []hyperliquid.WsOrder, err error) {
			if err != nil {
				t.Errorf("Error in order updates callback: %v", err)
				return
			}

			t.Logf("Received Order Updates: %+v", orders)

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
	case <-time.After(30 * time.Second):
		t.Log("No order updates received within timeout (this is expected for test addresses)")
	}
}
