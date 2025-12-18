package hyperliquid

import (
	"context"
	"testing"

	"github.com/sonirico/vago/ent"
	"github.com/stretchr/testify/require"
)

func TestPerpDeployHaltTrading(t *testing.T) {
	t.Run("halt trading success response", func(t *testing.T) {
		loadEnvClean(".env.testnet")

		key := ent.Str("HL_PRIVATE_KEY", "")

		exchange, err := newExchange(key, TestnetAPIURL)

		require.NoError(t, err)

		initRecorder(t, false, "PerpDeployHaltTrading_Success")

		res, err := exchange.PerpDeployHaltTrading(context.TODO(), "test:BTC", true)
		t.Logf("res: %+v", res)
		t.Logf("err: %v", err)

		require.NoError(t, err)
		require.NotNil(t, res)

		require.Equal(t, "ok", res.Status)
	})

	t.Run("halt trading error response", func(t *testing.T) {
		loadEnvClean(".env.testnet")

		key := ent.Str("HL_PRIVATE_KEY", "")

		exchange, err := newExchange(key, TestnetAPIURL)

		require.NoError(t, err)

		initRecorder(t, false, "PerpDeployHaltTrading_Error")

		res, err := exchange.PerpDeployHaltTrading(context.TODO(), "test:BTC", true)
		t.Logf("res: %+v", res)
		t.Logf("err: %v", err)

		require.NoError(t, err)
		require.NotNil(t, res)

		require.Equal(t, "err", res.Status)
		require.NotEmpty(t, res.Response)
	})
}
