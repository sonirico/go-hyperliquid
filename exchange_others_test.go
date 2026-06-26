package hyperliquid

import (
	"context"
	"encoding/json"
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

func TestUpdateIsolatedMarginNtli(t *testing.T) {
	tests := []struct {
		name   string
		amount float64
		want   int64
	}{
		{name: "whole dollar", amount: 64, want: 64_000_000},
		{name: "fractional dollars", amount: 64.71885, want: 64_718_850},
		{name: "rounds up sub micro dollar", amount: 0.0000001, want: 1},
		{name: "negative withdraw", amount: -1.25, want: 1_250_000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, updateIsolatedMarginNtli(tt.amount))
		})
	}
}

func TestUpdateIsolatedMarginActionJSONUsesIntegerNtli(t *testing.T) {
	action := UpdateIsolatedMarginAction{
		Type:  "updateIsolatedMargin",
		Asset: 110066,
		IsBuy: true,
		Ntli:  64_718_850,
	}

	body, err := json.Marshal(action)
	require.NoError(t, err)
	require.NotContains(t, string(body), "64718850.0")
	require.JSONEq(t, `{"type":"updateIsolatedMargin","asset":110066,"isBuy":true,"ntli":64718850}`, string(body))

	var decoded UpdateIsolatedMarginAction
	require.NoError(t, json.Unmarshal(body, &decoded))
	require.Equal(t, action, decoded)
}

func TestExchangeActionError(t *testing.T) {
	err := exchangeActionError([]byte(`{"status":"err","response":"Cannot switch leverage type with open position."}`))
	require.EqualError(t, err, "Cannot switch leverage type with open position.")

	require.NoError(t, exchangeActionError([]byte(`{"status":"ok","response":{"type":"default"}}`)))
	require.NoError(t, exchangeActionError([]byte(`not json`)))
}

func TestIsAPIResponseTarget(t *testing.T) {
	var userState UserState
	require.False(t, isAPIResponseTarget(&userState))

	var resp APIResponse[OrderResponse]
	require.True(t, isAPIResponseTarget(&resp))

	var respPtr *APIResponse[OrderResponse]
	require.True(t, isAPIResponseTarget(&respPtr))
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
