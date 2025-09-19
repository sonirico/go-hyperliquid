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
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"

	"gopkg.in/dnaeon/go-vcr.v4/pkg/cassette"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/recorder"
)

func newExchange(key string, url string) (*Exchange, error) {
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

	exchange, err := newExchange(
		"0x38d55ff1195c57b9dbc8a72c93119500f1fcd47a33f98149faa18d2fc37932fa",
		TestnetAPIURL)
	require.NoError(t, err)

	cases := []tc{
		{
			name:         "invalid auth",
			cassetteName: "Orders",
			exchange: func() *Exchange {
				exchange, err := newExchange(
					"0x38d55ff1195c57b9dbc8a72c93119500f1fcd47a33f98149faa18d2fc37932fa",
					TestnetAPIURL)
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
			cassetteName: "Orders",
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
			cassetteName: "Orders",
			exchange:     exchange,
			order: CreateOrderRequest{
				Coin:  "DOGE",
				IsBuy: true,
				Size:  45,
				Price: 0.12330, // set it low so it never gets executed
				OrderType: OrderType{
					Limit: &LimitOrderType{
						Tif: TifGtc,
					},
				},
			},
			result: OrderStatus{
				Resting: &OrderStatusResting{
					Oid: 37543129873,
				},
			},
			record: false,
		},
		{
			name:         "Order above with cloid",
			cassetteName: "Orders",
			exchange:     exchange,
			order: CreateOrderRequest{
				Coin:  "DOGE",
				IsBuy: true,
				Size:  45,
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
					Oid:      37543130760,
					ClientID: stringPtr("0x06c60000000000000000000000003f5a"),
				},
			},
			record: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(tt *testing.T) {
			// we don't care about errors here
			initRecorder(tt, tc.record, tc.cassetteName)

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
				if diff := cmp.Diff(res, tc.result); diff != "" {
					tt.Errorf("not equal\nwant: %v\ngot:  %v", tc.result, res)
				}
			}
		})
	}
}
