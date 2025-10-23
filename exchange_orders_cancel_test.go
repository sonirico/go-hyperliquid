package hyperliquid

import (
	"context"
	"testing"

	"github.com/sonirico/vago/ent"
	"github.com/stretchr/testify/require"
)

func TestCancelByCloid(t *testing.T) {
	type tc struct {
		name         string
		cassetteName string
		exchange     *Exchange
		// If placeFirst is true, we first place a resting order and use its OID.
		placeFirst bool
		order      CreateOrderRequest
		coin       string
		// used for cancelling a non existent
		cloid   *string
		wantErr string
		record  bool
	}

	loadEnvClean(".env.testnet")

	key := ent.Str("HL_PRIVATE_KEY", "")
	// t.Logf("Using private key: %s", key)

	exchange, err := newExchange(key, TestnetAPIURL)
	require.NoError(t, err)
	t.Logf("Exchange wallet address: %s", exchange.accountAddr)

	cases := []tc{
		{
			name:         "cancel resting order by cloid",
			cassetteName: "CancelByCloid_Success",
			exchange:     exchange,
			placeFirst:   true,
			order: CreateOrderRequest{
				Coin:  "DOGE",
				IsBuy: true,
				Size:  100, // 100 DOGE @ 0.12330 = $12.33 (above $10 minimum)
				Price: 0.12330,
				OrderType: OrderType{
					Limit: &LimitOrderType{Tif: TifGtc},
				},
				ClientOrderID: stringPtr("0x285ad26a251f390c83d065af51e3f8d9"),
			},
			coin:   "DOGE",
			record: false, // Using recorded cassette
		},
		{
			name:         "cancel non-existent cloid",
			cassetteName: "CancelByCloid_NonExistent",
			exchange:     exchange,
			placeFirst:   false,
			coin:         "DOGE",
			cloid:        stringPtr("0x0000000000000000000000000000fe54"),
			wantErr:      "Order was never placed, already canceled, or filled.",
			record:       false, // Using recorded cassette
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(tt *testing.T) {
			initRecorder(tt, tc.record, tc.cassetteName)

			cloid := tc.cloid
			if tc.placeFirst {
				placed, err := tc.exchange.Order(context.TODO(), tc.order, nil)
				require.NoError(tt, err)
				require.NotNil(tt, placed.Resting, "expected resting order so it can be canceled")
				cloid = placed.Resting.ClientID
			}

			// Cancel by cloid
			resp, err := tc.exchange.CancelByCloid(context.TODO(), tc.coin, *cloid)
			if tc.wantErr != "" {
				require.Error(tt, err)
				require.Contains(tt, err.Error(), tc.wantErr)
				return
			}
			require.NoError(tt, err)
			tt.Logf("cancel response: %+v", resp)
		})
	}
}

func TestCancel(t *testing.T) {
	type tc struct {
		name         string
		cassetteName string
		exchange     *Exchange
		// If placeFirst is true, we first place a resting order and use its OID.
		placeFirst bool
		order      CreateOrderRequest
		coin       string
		oid        int64 // used only when placeFirst == false
		// If doubleCancel is true, we attempt to cancel the same OID twice to exercise the error path.
		doubleCancel bool
		wantErr      string
		record       bool
	}

	loadEnvClean(".env.testnet")

	key := ent.Str("HL_PRIVATE_KEY", "")
	// t.Logf("Using private key: %s", key)
	exchange, err := newExchange(key, TestnetAPIURL)
	require.NoError(t, err)
	// t.Logf("Exchange wallet address: %s", exchange.accountAddr)

	cases := []tc{
		{
			name:         "cancel resting order by oid",
			cassetteName: "Cancel_Success",
			exchange:     exchange,
			placeFirst:   true,
			order: CreateOrderRequest{
				Coin:  "DOGE",
				IsBuy: true,
				Size:  100, // 100 DOGE @ 0.12330 = $12.33 (above $10 minimum)
				Price: 0.12330,
				OrderType: OrderType{
					Limit: &LimitOrderType{Tif: TifGtc},
				},
			},
			coin:   "DOGE",
			record: false, // Using recorded cassette
		},
		{
			name:         "double cancel returns error on second attempt",
			cassetteName: "Cancel_DoubleCancel",
			exchange:     exchange,
			placeFirst:   true,
			order: CreateOrderRequest{
				Coin:  "DOGE",
				IsBuy: true,
				Size:  100,
				Price: 0.12330,
				OrderType: OrderType{
					Limit: &LimitOrderType{Tif: TifGtc},
				},
			},
			coin:         "DOGE",
			doubleCancel: true,
			wantErr:      "Order was never placed, already canceled, or filled",
			record:       false, // Using recorded cassette
		},
		{
			name:         "cancel non-existent oid",
			cassetteName: "Cancel_NonExistent",
			exchange:     exchange,
			placeFirst:   false,
			coin:         "DOGE",
			oid:          1,
			wantErr:      "Order was never placed, already canceled, or filled.",
			record:       false, // Using recorded cassette
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(tt *testing.T) {
			initRecorder(tt, tc.record, tc.cassetteName)

			oid := tc.oid
			if tc.placeFirst {
				placed, err := tc.exchange.Order(context.TODO(), tc.order, nil)
				require.NoError(tt, err)
				require.NotNil(tt, placed.Resting, "expected resting order so it can be canceled")
				oid = placed.Resting.Oid
			}

			// First cancel
			resp, err := tc.exchange.Cancel(context.TODO(), tc.coin, oid)
			if tc.wantErr != "" && !tc.doubleCancel {
				require.Error(tt, err)
				require.Contains(tt, err.Error(), tc.wantErr)
				return
			}
			require.NoError(tt, err)
			tt.Logf("cancel response: %+v", resp)

			// Optional second cancel to test error path
			if tc.doubleCancel {
				resp2, err2 := tc.exchange.Cancel(context.TODO(), tc.coin, oid)
				if tc.wantErr != "" {
					require.Error(tt, err2, "expected error on second cancel")
					require.Contains(tt, err2.Error(), tc.wantErr)
				} else {
					require.NoError(tt, err2)
				}
				tt.Logf("second cancel response: %+v, err: %v", resp2, err2)
			}
		})
	}
}
