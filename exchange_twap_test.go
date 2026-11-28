package hyperliquid

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

// Expected hashes come from hyperliquid-python-sdk 0.24.1:
// hyperliquid.utils.signing.action_hash(action, None, nonce, None)
func TestTwapActionHash_MatchesPythonSDK(t *testing.T) {
	cases := []struct {
		name   string
		action any
		nonce  int64
		want   string
	}{
		{
			name: "twapOrder buy",
			action: TwapOrderAction{Type: "twapOrder", Twap: TwapWire{
				Asset: 0, IsBuy: true, Size: "0.1", ReduceOnly: false, Minutes: 30, Randomize: false,
			}},
			nonce: 1700000000000,
			want:  "1220289874c2dcf7607e5f62232c14f5bd0d62328082e53cc87aab921865d638",
		},
		{
			name: "twapOrder reduce-only randomized sell",
			action: TwapOrderAction{Type: "twapOrder", Twap: TwapWire{
				Asset: 10107, IsBuy: false, Size: "12.5", ReduceOnly: true, Minutes: 1440, Randomize: true,
			}},
			nonce: 1700000000001,
			want:  "b8ab93d1de7293b28f19b1f6bdfaa6bf0b4266e65d718a53213f10f72ccf8904",
		},
		{
			name:   "twapCancel",
			action: TwapCancelAction{Type: "twapCancel", Asset: 0, TwapID: 77738308},
			nonce:  1700000000002,
			want:   "bdc766810e5166d546921ab0185d4664780b1d683dac6b472384efea71ca5d23",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := actionHash(tc.action, "", tc.nonce, nil)
			require.Equal(t, tc.want, hex.EncodeToString(got))
		})
	}
}

func newTwapTestExchange(t *testing.T, response string) (*Exchange, *map[string]any) {
	t.Helper()
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(raw, &body))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(response))
	}))
	t.Cleanup(server.Close)

	key, err := crypto.HexToECDSA(
		"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	)
	require.NoError(t, err)
	meta := &Meta{Universe: []AssetInfo{{Name: "BTC", SzDecimals: 5}, {Name: "ETH", SzDecimals: 4}}}
	ex := NewExchange(context.TODO(), key, server.URL, meta, "", "", &SpotMeta{}, nil)
	return ex, &body
}

func TestTwapOrder(t *testing.T) {
	t.Run("running", func(t *testing.T) {
		ex, body := newTwapTestExchange(t,
			`{"status":"ok","response":{"type":"twapOrder","data":{"status":{"running":{"twapId":77738308}}}}}`)

		res, err := ex.TwapOrder(context.TODO(), TwapOrderRequest{
			Coin: "ETH", IsBuy: true, Size: 0.5, Minutes: 30, Randomize: true,
		})
		require.NoError(t, err)
		require.NotNil(t, res.Data.Status.Running)
		require.Equal(t, int64(77738308), res.Data.Status.Running.TwapID)

		action := (*body)["action"].(map[string]any)
		require.Equal(t, "twapOrder", action["type"])
		require.Equal(t, map[string]any{
			"a": float64(1), "b": true, "s": "0.5", "r": false, "m": float64(30), "t": true,
		}, action["twap"])
		require.Contains(t, *body, "signature")
	})

	t.Run("error status", func(t *testing.T) {
		ex, _ := newTwapTestExchange(t,
			`{"status":"ok","response":{"type":"twapOrder","data":{"status":{"error":"Invalid TWAP duration: 1 min(s)"}}}}`)

		res, err := ex.TwapOrder(context.TODO(), TwapOrderRequest{Coin: "BTC", Size: 0.001, Minutes: 1})
		require.EqualError(t, err, "Invalid TWAP duration: 1 min(s)")
		require.NotNil(t, res)
	})

	t.Run("unknown coin", func(t *testing.T) {
		ex, _ := newTwapTestExchange(t, `{}`)
		_, err := ex.TwapOrder(context.TODO(), TwapOrderRequest{Coin: "NOPE", Size: 1, Minutes: 5})
		require.EqualError(t, err, "coin NOPE not found in info")
	})
}

func TestTwapCancel(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ex, body := newTwapTestExchange(t,
			`{"status":"ok","response":{"type":"twapCancel","data":{"status":"success"}}}`)

		res, err := ex.TwapCancel(context.TODO(), "BTC", 77738308)
		require.NoError(t, err)
		require.True(t, res.Data.Status.Success)

		action := (*body)["action"].(map[string]any)
		require.Equal(t, map[string]any{
			"type": "twapCancel", "a": float64(0), "t": float64(77738308),
		}, action)
	})

	t.Run("error status", func(t *testing.T) {
		ex, _ := newTwapTestExchange(t,
			`{"status":"ok","response":{"type":"twapCancel","data":{"status":{"error":"TWAP was never placed, already canceled, or filled."}}}}`)

		res, err := ex.TwapCancel(context.TODO(), "BTC", 1)
		require.EqualError(t, err, "TWAP was never placed, already canceled, or filled.")
		require.False(t, res.Data.Status.Success)
	})
}
