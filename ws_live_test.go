//go:build live

package hyperliquid

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLiveWsPost(t *testing.T) {
	t.Run("posts an allMids info request", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		client := NewWebsocketClient(MainnetAPIURL)
		require.NoError(t, client.Connect(ctx))
		defer func() { require.NoError(t, client.Close()) }()

		got, err := client.Post(ctx, "info", map[string]string{"type": "allMids"})

		require.NoError(t, err)
		require.Contains(t, string(got), `"BTC"`)
	})
}
