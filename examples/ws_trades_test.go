package examples

import (
	"context"
	"testing"
	"time"

	"github.com/sonirico/go-hyperliquid"
)

func TestWebsocket(t *testing.T) {
	ws := hyperliquid.NewWebsocketClient(hyperliquid.MainnetAPIURL)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Log("Connecting to websocket")
	if err := ws.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Close()

	sub1, err := ws.Trades(
		hyperliquid.TradesSubscriptionParams{Coin: "SOL"},
		func(trades []hyperliquid.Trade, err error) {
			if err != nil {
				t.Fatalf("Failed to receive trades1: %v", err)
			}
			t.Logf("Received a set of trades1: %+v", len(trades))
		},
	)

	if err != nil {
		t.Fatalf("Failed to subscribe to trades1: %v", err)
	}

	t.Log("Subscribed to trades1")

	defer sub1.Close()

	sub2, err := ws.Trades(
		hyperliquid.TradesSubscriptionParams{Coin: "SOL"},
		func(trades []hyperliquid.Trade, err error) {
			if err != nil {
				t.Fatalf("Failed to receive trades2: %v", err)
			}
			t.Logf("Received a set of trades2: %+v", len(trades))
		},
	)

	if err != nil {
		t.Fatalf("Failed to subscribe to trades2: %v", err)
	}

	t.Log("Subscribed to trades2")
	defer sub2.Close()

	<-time.After(time.Second * 5)
	t.Log("Unsubscribing from trades1")
	sub1.Close()

	<-time.After(time.Second * 5)
	t.Log("Unsubscribing from trades2")
	sub2.Close()
}
