package hyperliquid

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

// newPerpDexTestServer fakes /info and /exchange for a builder-deployed perp dex
// named "xyz". Like the real API, meta, allMids and clearinghouseState only
// describe xyz when the request carries dex "xyz"; without it they describe the
// default perp dex.
func newPerpDexTestServer(t *testing.T) (*httptest.Server, func() []int) {
	t.Helper()

	var mu sync.Mutex
	var orderAssets []int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/exchange" {
			var req struct {
				Action struct {
					Orders []struct {
						Asset int `json:"a"`
					} `json:"orders"`
				} `json:"action"`
			}
			require.NoError(t, json.Unmarshal(body, &req))
			mu.Lock()
			for _, o := range req.Action.Orders {
				orderAssets = append(orderAssets, o.Asset)
			}
			mu.Unlock()
			_, _ = io.WriteString(w, `{"status":"ok","response":{"type":"order","data":{"statuses":[{"resting":{"oid":1}}]}}}`)
			return
		}

		var req struct {
			Type string `json:"type"`
			Dex  string `json:"dex"`
		}
		require.NoError(t, json.Unmarshal(body, &req))
		xyz := req.Dex == "xyz"

		switch req.Type {
		case "perpDexs":
			_, _ = io.WriteString(w, `[null,{"name":"xyz","fullName":"XYZ"}]`)
		case "spotMeta":
			_, _ = io.WriteString(w, `{"universe":[],"tokens":[]}`)
		case "meta":
			if xyz {
				_, _ = io.WriteString(w, `{"universe":[{"name":"xyz:XYZ100","szDecimals":4},{"name":"xyz:TSLA","szDecimals":3}],"marginTables":[]}`)
			} else {
				_, _ = io.WriteString(w, `{"universe":[{"name":"BTC","szDecimals":5},{"name":"ETH","szDecimals":4}],"marginTables":[]}`)
			}
		case "allMids":
			if xyz {
				_, _ = io.WriteString(w, `{"xyz:XYZ100":"25000","xyz:TSLA":"1.23456"}`)
			} else {
				_, _ = io.WriteString(w, `{"BTC":"100000","ETH":"4000"}`)
			}
		case "clearinghouseState":
			if xyz {
				_, _ = io.WriteString(w, `{"assetPositions":[{"type":"oneWay","position":{"coin":"xyz:TSLA","szi":"2.5"}}],"marginSummary":{},"crossMarginSummary":{},"withdrawable":"0"}`)
			} else {
				_, _ = io.WriteString(w, `{"assetPositions":[],"marginSummary":{},"crossMarginSummary":{},"withdrawable":"0"}`)
			}
		default:
			t.Fatalf("unexpected info request type %q", req.Type)
		}
	}))
	t.Cleanup(server.Close)

	return server, func() []int {
		mu.Lock()
		defer mu.Unlock()
		return append([]int(nil), orderAssets...)
	}
}

func TestNewInfo_FetchesMetaForConfiguredPerpDex(t *testing.T) {
	server, _ := newPerpDexTestServer(t)

	info := NewInfo(context.Background(), server.URL, true, nil, nil, nil, InfoOptPerpDexName("xyz"))

	asset, ok := info.CoinToAsset("xyz:TSLA")
	require.True(t, ok, "xyz:TSLA should be mapped")
	require.Equal(t, 110001, asset)

	_, ok = info.CoinToAsset("BTC")
	require.False(t, ok, "default dex coins must not be mapped onto builder dex asset ids")
}

func TestSlippagePrice_UsesConfiguredPerpDexMids(t *testing.T) {
	server, _ := newPerpDexTestServer(t)

	exchange := NewExchange(context.Background(), nil, server.URL, nil, "", "", nil, nil, ExchangeOptPerpDex("xyz"))

	price, err := exchange.SlippagePrice(context.Background(), "xyz:XYZ100", true, 0, nil)
	require.NoError(t, err)
	require.Equal(t, 25000.0, price)
}

func TestSlippagePrice_BuilderPerpUsesPerpDecimals(t *testing.T) {
	server, _ := newPerpDexTestServer(t)

	exchange := NewExchange(context.Background(), nil, server.URL, nil, "", "", nil, nil, ExchangeOptPerpDex("xyz"))

	// szDecimals is 3, so a perp price may have at most 6 - 3 = 3 decimals.
	px := 1.23456
	price, err := exchange.SlippagePrice(context.Background(), "xyz:TSLA", true, 0, &px)
	require.NoError(t, err)
	require.Equal(t, 1.235, price)
}

func TestMarketClose_FindsConfiguredPerpDexPosition(t *testing.T) {
	server, orderAssets := newPerpDexTestServer(t)

	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	address := crypto.PubkeyToAddress(privateKey.PublicKey).Hex()

	exchange := NewExchange(context.Background(), privateKey, server.URL, nil, "", address, nil, nil, ExchangeOptPerpDex("xyz"))

	px := 1.2
	_, err = exchange.MarketClose(context.Background(), "xyz:TSLA", nil, &px, 0, nil, nil)
	require.NoError(t, err)
	require.Equal(t, []int{110001}, orderAssets())
}
