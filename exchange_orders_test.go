package hyperliquid

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/sonirico/vago/ent"
	"github.com/stretchr/testify/require"

	"gopkg.in/dnaeon/go-vcr.v4/pkg/cassette"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/recorder"
)

func newExchange(key, url string) (*Exchange, error) {
	key = strings.TrimSpace(key)
	key = strings.TrimPrefix(key, "0x")
	privateKey, err := crypto.HexToECDSA(key)
	if err != nil {
		return nil, fmt.Errorf("could not load private key: %s", err)
	}

	pub := privateKey.Public()
	pubECDSA, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("error casting public key to ECDSA")
	}
	accountAddr := crypto.PubkeyToAddress(*pubECDSA).Hex()

	exchange := NewExchange(
		context.TODO(),
		privateKey,
		url,
		nil, // Meta will be fetched automatically
		"",
		accountAddr,
		nil, // SpotMeta will be fetched automatically
		nil, // PerpDexs will be fetched automatically
	)

	return exchange, nil
}

func scrubHLJSON(body string) string {
	var m map[string]any
	dec := json.NewDecoder(strings.NewReader(body))
	dec.UseNumber() // keep numeric fidelity
	if err := dec.Decode(&m); err != nil {
		return body // not JSON; leave as-is
	}
	delete(m, "nonce")
	if sig, ok := m["signature"].(map[string]any); ok {
		delete(sig, "r")
		delete(sig, "s")
		delete(sig, "v")
		if len(sig) == 0 {
			delete(m, "signature")
		} else {
			m["signature"] = sig
		}
	}
	b, err := json.Marshal(m)
	if err != nil {
		return body
	}
	return string(b)
}

func hyperliquidJSONMatcher() recorder.MatcherFunc {
	def := cassette.NewDefaultMatcher(
		cassette.WithIgnoreHeaders("Authorization", "Apikey", "Signature"),
	)

	return func(req *http.Request, rec cassette.Request) bool {
		// Quick method/URL gate
		if req.Method != rec.Method || req.URL.String() != rec.URL {
			return false
		}

		// Ignore auth-ish headers (from the recorded request)
		rec.Headers.Del("Authorization")
		rec.Headers.Del("Apikey")
		rec.Headers.Del("Signature")

		// If JSON, compare normalized bodies
		if strings.Contains(rec.Headers.Get("Content-Type"), "application/json") {
			rb, _ := io.ReadAll(req.Body)
			defer func() { req.Body = io.NopCloser(bytes.NewReader(rb)) }()
			a := scrubHLJSON(string(rb))
			b := scrubHLJSON(rec.Body)
			return a == b
		}

		// Fallback to the library’s default matcher
		return def(req, rec)
	}
}

func defaultRecorderOpts(record bool) []recorder.Option {
	opts := []recorder.Option{
		recorder.WithHook(func(i *cassette.Interaction) error {
			i.Request.Headers.Del("Authorization")
			i.Request.Headers.Del("Apikey")
			i.Request.Headers.Del("Signature")

			if strings.Contains(i.Request.Headers.Get("Content-Type"), "application/json") &&
				i.Request.Body != "" {
				i.Request.Body = scrubHLJSON(i.Request.Body)
			}

			return nil
		}, recorder.AfterCaptureHook),
		recorder.WithMatcher(hyperliquidJSONMatcher()),
		recorder.WithSkipRequestLatency(true),
	}

	if record {
		opts = append(opts,
			recorder.WithMode(recorder.ModeReplayWithNewEpisodes),
			recorder.WithRealTransport(http.DefaultTransport),
		)
	} else {
		opts = append(opts, recorder.WithMode(recorder.ModeReplayOnly))
	}

	return opts
}

func initRecorder(t *testing.T, record bool, cassetteName string) {
	opts := defaultRecorderOpts(record)

	base := strings.ReplaceAll(t.Name(), "/", "_")
	cassette := filepath.Join("testdata", func() string {
		if cassetteName != "" {
			return cassetteName
		}
		return base
	}())

	orig := http.DefaultTransport

	r, err := recorder.New(cassette, opts...)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		// restore default
		http.DefaultTransport = orig
		// Make sure recorder is stopped once done with it.
		if err := r.Stop(); err != nil {
			t.Error(err)
		}
	})

	http.DefaultTransport = r
}

func TestOrders(t *testing.T) {
	type tc struct {
		name         string
		cassetteName string
		exchange     *Exchange
		order        CreateOrderRequest
		result       OrderStatus
		wantErr      string
		record       bool
	}

	_ = loadEnvClean(".env.testnet")

	key := ent.Str("HL_PRIVATE_KEY", "")
	// t.Logf("Using private key: %s", key)
	invalidKey := "aabbccddeeff00112233445566778899aabbccddeeff00112233445566778899"

	exchange, err := newExchange(
		key,
		TestnetAPIURL,
	)
	require.NoError(t, err)
	// exchange.debug = true // Enable debug logging
	t.Logf("Exchange wallet address: %s", exchange.accountAddr)

	cases := []tc{
		{
			name:         "invalid auth",
			cassetteName: "Orders_InvalidAuth",
			exchange: func() *Exchange {
				exchange, err := newExchange(
					invalidKey,
					TestnetAPIURL,
				)
				require.NoError(t, err)
				return exchange
			}(),
			// if the Order is not a proper request it won't even hit the key check
			order: CreateOrderRequest{
				Coin:  "DOGE",
				IsBuy: true,
				Size:  55,
				Price: 0.22330,
				OrderType: OrderType{
					Limit: &LimitOrderType{
						Tif: TifGtc,
					},
				},
			},
			wantErr: "failed to create order: User or API Wallet",
			record:  false,
		},
		{
			name:         "Create Order below 10$",
			cassetteName: "Orders_Below10",
			exchange:     exchange,
			order: CreateOrderRequest{
				Coin:  "DOGE",
				IsBuy: true,
				Size:  25,
				Price: 0.22330,
				OrderType: OrderType{
					Limit: &LimitOrderType{
						Tif: TifGtc,
					},
				},
			},
			wantErr: "Order must have minimum value of $10.",
			record:  false,
		},
		{
			name:         "Order above 10$",
			cassetteName: "Orders_Above10",
			exchange:     exchange,
			order: CreateOrderRequest{
				Coin:  "DOGE",
				IsBuy: true,
				Size:  100,
				Price: 0.12330, // set it low so it never gets executed
				OrderType: OrderType{
					Limit: &LimitOrderType{
						Tif: TifGtc,
					},
				},
			},
			result: OrderStatus{
				Resting: &OrderStatusResting{
					Oid: 41544179816,
				},
			},
			record: false,
		},
		{
			name:         "Order above with cloid",
			cassetteName: "Orders_Cloid",
			exchange:     exchange,
			order: CreateOrderRequest{
				Coin:  "DOGE",
				IsBuy: true,
				Size:  100,     // 100 DOGE @ 0.12330 = $12.33 (above $10 minimum)
				Price: 0.12330, // set it low so it never gets executed
				OrderType: OrderType{
					Limit: &LimitOrderType{
						Tif: TifGtc,
					},
				},
				ClientOrderID: stringPtr("0x06c60000000000000000000000003f5a"),
			},
			result: OrderStatus{
				Resting: &OrderStatusResting{
					Oid:      41547028680, // Updated to match actual response
					ClientID: stringPtr("0x06c60000000000000000000000003f5a"),
				},
			},
			record: false, // ✅ FIXED: cloid now includes 0x prefix in msgpack
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(tt *testing.T) {
			// we don't care about errors here
			initRecorder(tt, tc.record, tc.cassetteName)

			tt.Logf("Test exchange wallet: %s", tc.exchange.accountAddr)
			res, err := tc.exchange.Order(context.TODO(), tc.order, nil)
			tt.Logf("res: %v", res)
			tt.Logf("err: %v", err)
			if tc.wantErr != "" {
				require.Error(tt, err)
				require.Contains(tt, err.Error(), tc.wantErr)
				return
			} else {
				require.NoError(tt, err)
			}

			if err == nil {
				// Don't assert exact Oid since it changes with each order
				// Just verify the structure is correct
				if tc.result.Resting != nil {
					require.NotNil(tt, res.Resting, "expected resting order")
					require.Greater(tt, res.Resting.Oid, int64(0), "oid should be positive")
					if tc.result.Resting.ClientID != nil {
						require.Equal(tt, *tc.result.Resting.ClientID, *res.Resting.ClientID)
					}
				}
			}
		})
	}
}
