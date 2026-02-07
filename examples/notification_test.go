package examples

import (
	"context"
	"testing"
	"time"

	"github.com/sonirico/go-hyperliquid"
)

func TestNotificationWebSocket(t *testing.T) {
	ws := hyperliquid.NewWebsocketClient("")

	if err := ws.Connect(context.Background()); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer func() { _ = ws.Close() }()

	done := make(chan bool)

	// Use a test address (you would replace this with a real address in practice)
	testAddress := "0x1234567890123456789012345678901234567890"

	sub, err := ws.Notification(
		hyperliquid.NotificationSubscriptionParams{
			User: testAddress,
		},
		func(notification hyperliquid.Notification, err error) {
			if err != nil {
				t.Errorf("Error in notification callback: %v", err)
				return
			}

			t.Logf("Received Notification: %+v", notification)

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
		t.Log("No notification received within timeout (this is expected for test addresses)")
	}
}
