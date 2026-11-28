//go:build live

package hyperliquid

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

type liveTransport struct {
	response string
}

func (r *liveTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	r.response = string(raw)
	resp.Body = io.NopCloser(bytes.NewReader(raw))
	return resp, nil
}

func newTestLiveExchange(t *testing.T, pk *ecdsa.PrivateKey) (*Exchange, *liveTransport) {
	t.Helper()
	transport := &liveTransport{}
	meta := &Meta{Universe: []AssetInfo{{Name: "BTC", SzDecimals: 5}, {Name: "ETH", SzDecimals: 4}}}
	ex := NewExchange(
		context.Background(), pk, TestnetAPIURL, meta, "", "", &SpotMeta{}, nil,
		ExchangeOptClientOptions(ClientOptHTTPClient(&http.Client{Transport: transport})),
	)
	return ex, transport
}

func liveRows() []goldenRow {
	return []goldenRow{
		{name: "spotDeploy_registerHyperliquidity_integerPrices", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.SpotDeployRegisterHyperliquidity(ctx, 0, 20, 4, 10, nil)
			return err
		}},
	}
}

// The key is generated per run and never funded: the testnet rejects the
// action with an error that names the address recovered from the signature,
// so finding our own address in the response proves the signature verifies.
func TestLive(t *testing.T) {
	pk, err := crypto.GenerateKey()
	require.NoError(t, err)
	address := strings.ToLower(crypto.PubkeyToAddress(pk.PublicKey).Hex())

	for _, row := range append(goldenRows(goldenOther), liveRows()...) {
		t.Run(row.name, func(t *testing.T) {
			if row.call == nil {
				t.Skip("nothing is sent")
			}

			ex, transport := newTestLiveExchange(t, pk)

			callErr := row.call(context.Background(), ex)
			t.Logf("method result: %v, response: %s", callErr, transport.response)

			require.NotEmpty(t, transport.response, "nothing was posted")
			want := address
			if row.live != "" {
				want = strings.ToLower(row.live)
			}
			require.Contains(t, strings.ToLower(transport.response), want)
		})
	}
}
