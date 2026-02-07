package hyperliquid

import (
	"context"
	"testing"

	"github.com/sonirico/vago/ent"
	"github.com/stretchr/testify/require"
)

func setupExchange(t *testing.T) *Exchange {
	t.Helper()
	_ = loadEnvClean(".env.testnet")
	key := ent.Str("HL_PRIVATE_KEY", "")
	exchange, err := newExchange(key, TestnetAPIURL)
	require.NoError(t, err)
	return exchange
}

func TestPerpDeployHaltTrading(t *testing.T) {
	t.Run("halt trading success response", func(t *testing.T) {
		exchange := setupExchange(t)
		initRecorder(t, false, "PerpDeployHaltTrading_Success")

		res, err := exchange.PerpDeployHaltTrading(context.TODO(), "test:BTC", true)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, "ok", res.Status)
	})

	t.Run("halt trading error response", func(t *testing.T) {
		exchange := setupExchange(t)
		initRecorder(t, false, "PerpDeployHaltTrading_Error")

		res, err := exchange.PerpDeployHaltTrading(context.TODO(), "test:BTC", true)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, "err", res.Status)
		require.NotEmpty(t, res.Response)
	})
}

func TestPerpDeployRegisterAsset(t *testing.T) {
	oracleUpdater := "0xABCDEF1234567890ABCDEF1234567890ABCDEF12"

	t.Run("with schema", func(t *testing.T) {
		exchange := setupExchange(t)
		initRecorder(t, false, "PerpDeployRegisterAsset_WithSchema")

		maxGas := int(1000000000000)
		res, err := exchange.PerpDeployRegisterAsset(
			context.TODO(),
			"test",
			&maxGas,
			AssetRequest{
				Coin:          "test:TEST0",
				SzDecimals:    2,
				OraclePx:      "10.0",
				MarginTableID: 10,
				OnlyIsolated:  false,
			},
			&PerpDexSchemaInput{
				FullName:        "test dex",
				CollateralToken: 0,
				OracleUpdater:   &oracleUpdater,
			},
		)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, "ok", res.Status)
	})

	t.Run("without schema", func(t *testing.T) {
		exchange := setupExchange(t)
		initRecorder(t, false, "PerpDeployRegisterAsset_WithoutSchema")

		res, err := exchange.PerpDeployRegisterAsset(
			context.TODO(),
			"test",
			nil,
			AssetRequest{
				Coin:          "test:TEST0",
				SzDecimals:    2,
				OraclePx:      "10.0",
				MarginTableID: 10,
				OnlyIsolated:  false,
			},
			nil,
		)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, "ok", res.Status)
	})

	t.Run("error response", func(t *testing.T) {
		exchange := setupExchange(t)
		initRecorder(t, false, "PerpDeployRegisterAsset_Error")

		res, err := exchange.PerpDeployRegisterAsset(
			context.TODO(),
			"test",
			nil,
			AssetRequest{
				Coin:          "test:TEST0",
				SzDecimals:    2,
				OraclePx:      "10.0",
				MarginTableID: 10,
				OnlyIsolated:  false,
			},
			nil,
		)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, "err", res.Status)
		require.NotEmpty(t, res.Response)
	})
}

func TestPerpDeployRegisterAsset2(t *testing.T) {
	oracleUpdater := "0xABCDEF1234567890ABCDEF1234567890ABCDEF12"

	t.Run("with schema strictIsolated", func(t *testing.T) {
		exchange := setupExchange(t)
		initRecorder(t, false, "PerpDeployRegisterAsset2_WithSchema")

		maxGas := int(1000000000000)
		res, err := exchange.PerpDeployRegisterAsset2(
			context.TODO(),
			"test",
			&maxGas,
			AssetRequest2{
				Coin:          "test:TEST0",
				SzDecimals:    2,
				OraclePx:      "10.0",
				MarginTableID: 10,
				MarginMode:    "strictIsolated",
			},
			&PerpDexSchemaInput{
				FullName:        "test dex",
				CollateralToken: 0,
				OracleUpdater:   &oracleUpdater,
			},
		)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, "ok", res.Status)
	})

	t.Run("without schema noCross", func(t *testing.T) {
		exchange := setupExchange(t)
		initRecorder(t, false, "PerpDeployRegisterAsset2_WithoutSchema")

		res, err := exchange.PerpDeployRegisterAsset2(
			context.TODO(),
			"test",
			nil,
			AssetRequest2{
				Coin:          "test:TEST0",
				SzDecimals:    2,
				OraclePx:      "10.0",
				MarginTableID: 10,
				MarginMode:    "noCross",
			},
			nil,
		)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, "ok", res.Status)
	})
}
