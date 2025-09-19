package hyperliquid

import (
	"context"
	"log"
	"testing"

	"github.com/stretchr/testify/require"
)

var recordForDebug = true

func TestCancelByCloid(t *testing.T) {
	type tc struct {
		name         string
		cassetteName string
		// If placeFirst is true, we first place a resting order and use its OID.
		placeFirst bool
		order      CreateOrderRequest
		coin       string
		// used for cancelling a non existent
		cloid *string
		// If doubleCancel is true, we attempt to cancel the same OID twice to exercise the error path.
		doubleCancel bool
		wantErr      string
		record       bool
	}

	cases := []tc{
		{
			name:         "cancel resting order by cloid",
			cassetteName: "CancelByCloid",
			placeFirst:   true,
			order: CreateOrderRequest{
				Coin:  "DOGE",
				IsBuy: true,
				Size:  50,
				Price: 0.12330, // low so it stays resting
				OrderType: OrderType{
					Limit: &LimitOrderType{Tif: TifGtc},
				},
				ClientOrderID: stringPtr("0x285ad26a251f390c83d065af51e3f8d9"),
			},
			coin:   "DOGE",
			record: false,
		},
		{
			name:         "cancel non-existent cloid",
			cassetteName: "CancelByCloid",
			placeFirst:   false,
			coin:         "BTC",
			cloid:        stringPtr("0x0000000000000000000000000000fe54"),
			wantErr:      "Order was never placed, already canceled, or filled.",
			record:       false,
		},
		// {
		// 	name:         "double cancel DOES NOT return error on second attempt",
		// 	cassetteName: "CancelByCloid",
		// 	placeFirst:   true,
		// 	order: CreateOrderRequest{
		// 		Coin:  "KAS",
		// 		IsBuy: true,
		// 		Size:  170,
		// 		Price: 0.060793,
		// 		OrderType: OrderType{
		// 			Limit: &LimitOrderType{Tif: TifGtc},
		// 		},
		// 		ClientOrderID: stringPtr("0x185ad26a251f390c83d065af51e3f8d8"),
		// 	},
		// 	coin:         "KAS",
		// 	doubleCancel: true,
		// 	// we would expect an error, but it actually passes
		// 	// even with a timeout, we leave this in in case the API ever changes
		// 	// wantErr: "already canceled",
		// 	record: false,
		// },
	}

	for _, tc := range cases {
		t.Run(tc.name, func(tt *testing.T) {
			log.Printf("name: %s", tc.name)
			initRecorder(tt, tc.record, tc.cassetteName)

			exchange, err := newExchange(
				"0x38d55ff1195c57b9dbc8a72c93119500f1fcd47a33f98149faa18d2fc37932fa",
				TestnetAPIURL)
			require.NoError(t, err)

			cloid := tc.cloid
			if tc.placeFirst {
				placed, err := exchange.Order(context.TODO(), tc.order, nil)
				require.NoError(tt, err)
				require.NotNil(tt, placed.Resting, "expected resting order so it can be canceled")
				cloid = placed.Resting.ClientID
			}

			// First cancel
			resp, err := exchange.CancelByCloid(context.TODO(), tc.coin, *cloid)
			if tc.wantErr != "" && !tc.doubleCancel {
				require.Error(tt, err)
				require.Contains(tt, err.Error(), tc.wantErr)
				return
			}
			require.NoError(tt, err)
			tt.Logf("cancel response: %+v", resp)

			// // Optional second cancel to test error path
			// if tc.doubleCancel {
			// 	resp2, err2 := exchange.CancelByCloid(tc.coin, *cloid)
			// 	if tc.wantErr != "" {
			// 		require.Error(tt, err2, "expected error on second cancel")
			// 		require.Contains(tt, err2.Error(), tc.wantErr)
			// 	} else {
			// 		require.NoError(tt, err2)
			// 	}
			// 	tt.Logf("second cancel response: %+v, err: %v", resp2, err2)
			// }
		})
	}
}

func TestCancel(t *testing.T) {
	type tc struct {
		name         string
		cassetteName string
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

	cases := []tc{
		{
			name:         "cancel resting order by oid",
			cassetteName: "Cancel",
			placeFirst:   true,
			order: CreateOrderRequest{
				Coin:  "DOGE",
				IsBuy: true,
				Size:  45,
				Price: 0.12330, // low so it stays resting
				OrderType: OrderType{
					Limit: &LimitOrderType{Tif: TifGtc},
				},
			},
			coin:   "DOGE",
			record: false,
		},
		{
			name:         "double cancel returns error on second attempt",
			cassetteName: "Cancel",
			placeFirst:   true,
			order: CreateOrderRequest{
				Coin:  "DOGE",
				IsBuy: true,
				Size:  45,
				Price: 0.12330,
				OrderType: OrderType{
					Limit: &LimitOrderType{Tif: TifGtc},
				},
			},
			coin:         "DOGE",
			doubleCancel: true,
			wantErr:      "already canceled",
			record:       false,
		},
		{
			name:         "cancel non-existent oid",
			cassetteName: "Cancel",
			placeFirst:   false,
			coin:         "DOGE",
			oid:          1,
			wantErr:      "Order was never placed, already canceled, or filled.",
			record:       false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(tt *testing.T) {
			initRecorder(tt, tc.record, tc.cassetteName)

			exchange, err := newExchange(
				"0x38d55ff1195c57b9dbc8a72c93119500f1fcd47a33f98149faa18d2fc37932fa",
				TestnetAPIURL)
			require.NoError(t, err)

			oid := tc.oid
			if tc.placeFirst {
				placed, err := exchange.Order(context.TODO(), tc.order, nil)
				require.NoError(tt, err)
				require.NotNil(tt, placed.Resting, "expected resting order so it can be canceled")
				oid = placed.Resting.Oid
			}

			// First cancel
			resp, err := exchange.Cancel(context.TODO(), tc.coin, oid)
			if tc.wantErr != "" && !tc.doubleCancel {
				require.Error(tt, err)
				require.Contains(tt, err.Error(), tc.wantErr)
				return
			}
			require.NoError(tt, err)
			tt.Logf("cancel response: %+v", resp)

			// Optional second cancel to test error path
			if tc.doubleCancel {
				resp2, err2 := exchange.Cancel(context.TODO(), tc.coin, oid)
				require.Error(tt, err2, "expected error on second cancel")
				if tc.wantErr != "" {
					require.Contains(tt, err2.Error(), tc.wantErr)
				}
				tt.Logf("second cancel response: %+v, err: %v", resp2, err2)
			}
		})
	}
}
