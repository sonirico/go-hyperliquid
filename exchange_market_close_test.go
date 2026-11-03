package hyperliquid

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

type marketCloseOrder struct {
	IsBuy bool   `json:"b"`
	Size  string `json:"s"`
}

// newMarketCloseTestServer fakes /info and /exchange with a different BTC
// position for each user, and records whose state was read and which close
// orders were sent.
func newMarketCloseTestServer(
	t *testing.T,
	positions map[string]string,
) (*httptest.Server, func() ([]string, []marketCloseOrder)) {
	t.Helper()

	var mu sync.Mutex
	var users []string
	var orders []marketCloseOrder

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/exchange" {
			var req struct {
				Action struct {
					Orders []marketCloseOrder `json:"orders"`
				} `json:"action"`
			}
			require.NoError(t, json.Unmarshal(body, &req))
			mu.Lock()
			orders = append(orders, req.Action.Orders...)
			mu.Unlock()
			_, _ = io.WriteString(w, `{"status":"ok","response":{"type":"order","data":{"statuses":[{"filled":{"totalSz":"1","avgPx":"100000","oid":1}}]}}}`)
			return
		}

		var req struct {
			Type string `json:"type"`
			User string `json:"user"`
		}
		require.NoError(t, json.Unmarshal(body, &req))

		switch req.Type {
		case "perpDexs":
			_, _ = io.WriteString(w, `[null]`)
		case "spotMeta":
			_, _ = io.WriteString(w, `{"universe":[],"tokens":[]}`)
		case "meta":
			_, _ = io.WriteString(w, `{"universe":[{"name":"BTC","szDecimals":5}],"marginTables":[]}`)
		case "allMids":
			_, _ = io.WriteString(w, `{"BTC":"100000"}`)
		case "clearinghouseState":
			mu.Lock()
			users = append(users, req.User)
			mu.Unlock()
			positionsJSON := `[]`
			if szi, ok := positions[strings.ToLower(req.User)]; ok {
				positionsJSON = `[{"type":"oneWay","position":{"coin":"BTC","szi":"` + szi + `"}}]`
			}
			_, _ = io.WriteString(w, `{"assetPositions":`+positionsJSON+`,"marginSummary":{},"crossMarginSummary":{},"withdrawable":"0"}`)
		default:
			t.Fatalf("unexpected info request type %q", req.Type)
		}
	}))
	t.Cleanup(server.Close)

	return server, func() ([]string, []marketCloseOrder) {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), users...), append([]marketCloseOrder(nil), orders...)
	}
}

func TestMarketClose_ReadsThePositionOfTheAccountItTrades(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	signer := strings.ToLower(crypto.PubkeyToAddress(key.PublicKey).Hex())

	const (
		vault   = "0x1111111111111111111111111111111111111111"
		account = "0x2222222222222222222222222222222222222222"
	)
	// The vault is short 3 BTC, the account long 1 and the signer long 0.5.
	positions := map[string]string{vault: "-3", account: "1", signer: "0.5"}

	tests := []struct {
		name          string
		vault         string
		account       string
		wantUser      string
		wantOrderBuy  bool
		wantOrderSize string
	}{
		// Orders on a vault are signed for the vault, so its position is the one to close.
		{"vault and account", vault, account, vault, true, "3"},
		{"vault only", vault, "", vault, true, "3"},
		{"account only", "", account, account, false, "1"},
		{"neither", "", "", signer, false, "0.5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, recorded := newMarketCloseTestServer(t, positions)
			exchange := NewExchange(
				context.Background(), key, server.URL, nil, tt.vault, tt.account, nil, nil,
			)

			_, err := exchange.MarketClose(context.Background(), "BTC", nil, nil, 0.05, nil, nil)
			require.NoError(t, err)

			users, orders := recorded()
			require.Equal(t, []string{tt.wantUser}, lowerAll(users))
			require.Len(t, orders, 1)
			require.Equal(t, tt.wantOrderBuy, orders[0].IsBuy)
			require.Equal(t, tt.wantOrderSize, orders[0].Size)
		})
	}
}

func lowerAll(values []string) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = strings.ToLower(v)
	}
	return out
}
