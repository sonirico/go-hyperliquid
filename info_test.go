package hyperliquid

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/require"
)

func TestMetaAndAssetCtxs(t *testing.T) {
	info := NewInfo(context.Background(), MainnetAPIURL, true, nil, nil, nil)

	initRecorder(t, false, "MetaAndAssetCtxs")

	res, err := info.MetaAndAssetCtxs(context.TODO(), MetaAndAssetCtxsParams{})
	t.Logf("res: %+v", res)
	t.Logf("err: %v", err)

	require.NoError(t, err)

	// Verify the response structure
	require.NotNil(t, res)
	require.NotNil(t, res.Meta.Universe)
	require.NotNil(t, res.Meta.MarginTables)
	require.NotNil(t, res.Ctxs)

	// Verify we have at least one asset in universe
	require.Greater(t, len(res.Meta.Universe), 0)
	require.NotEmpty(t, res.Meta.Universe[0].Name)

	// Test specific known assets from the cassette data
	var btcFound, ethFound bool
	for _, asset := range res.Meta.Universe {
		if asset.Name == "BTC" {
			btcFound = true
			require.Equal(t, 5, asset.SzDecimals)
			require.Equal(t, 40, asset.MaxLeverage)
			require.Equal(t, 56, asset.MarginTableId)
		}
		if asset.Name == "ETH" {
			ethFound = true
			require.Equal(t, 4, asset.SzDecimals)
			require.Equal(t, 25, asset.MaxLeverage)
			require.Equal(t, 55, asset.MarginTableId)
		}
	}
	require.True(t, btcFound, "BTC asset should be present in universe")
	require.True(t, ethFound, "ETH asset should be present in universe")

	// Verify we have at least one margin table
	require.Greater(t, len(res.Meta.MarginTables), 0)
	require.GreaterOrEqual(t, res.Meta.MarginTables[0].ID, 0)

	// Verify we have at least one margin tier
	require.Greater(t, len(res.Meta.MarginTables[0].MarginTiers), 0)

	// Test specific margin table structure
	for _, marginTable := range res.Meta.MarginTables {
		require.NotNil(t, marginTable)
		require.Greater(t, len(marginTable.MarginTiers), 0)
		for _, tier := range marginTable.MarginTiers {
			require.NotEmpty(t, tier.LowerBound)
			require.Greater(t, tier.MaxLeverage, 0)
		}
	}

	// Verify we have at least one context
	require.Greater(t, len(res.Ctxs), 0)
	require.NotEmpty(t, res.Ctxs[0].MarkPx)

	// Test with explicit empty dex (should be same as default)
	emptyDex := ""
	res2, err := info.MetaAndAssetCtxs(context.TODO(), MetaAndAssetCtxsParams{Dex: &emptyDex})
	require.NoError(t, err)
	require.NotNil(t, res2)
	require.Greater(t, len(res2.Meta.Universe), 0)
	require.Greater(t, len(res2.Ctxs), 0)

	// Test with explicit dex value
	xyzDex := "xyz"
	res3, err := info.MetaAndAssetCtxs(context.TODO(), MetaAndAssetCtxsParams{Dex: &xyzDex})
	require.NoError(t, err)
	require.NotNil(t, res3)
	require.Greater(t, len(res3.Meta.Universe), 0)
	require.Greater(t, len(res3.Ctxs), 0)

	var xyzTslaFound bool
	for _, asset := range res3.Meta.Universe {
		if asset.Name == "xyz:TSLA" {
			xyzTslaFound = true
			require.Equal(t, 3, asset.SzDecimals)
			require.Equal(t, 10, asset.MaxLeverage)
			require.Equal(t, 10, asset.MarginTableId)
			break
		}
	}
	require.True(t, xyzTslaFound, "xyz:TSLA asset should be present in universe")
}

func TestSpotMetaAndAssetCtxs(t *testing.T) {
	info := NewInfo(context.TODO(), MainnetAPIURL, true, nil, nil, nil)

	initRecorder(t, false, "SpotMetaAndAssetCtxs")

	res, err := info.SpotMetaAndAssetCtxs(context.TODO())
	t.Logf("res: %+v", res)
	t.Logf("err: %v", err)

	require.NoError(t, err)

	// Verify the response structure
	require.NotNil(t, res)
	require.NotNil(t, res.Meta.Universe)
	require.NotNil(t, res.Meta.Tokens)
	require.NotNil(t, res.Ctxs)

	// Verify we have at least one asset in universe
	require.Greater(t, len(res.Meta.Universe), 0)
	require.NotEmpty(t, res.Meta.Universe[0].Name)

	// Test specific known assets from the cassette data
	var purrFound bool
	for _, asset := range res.Meta.Universe {
		if asset.Name == "PURR/USDC" {
			purrFound = true
			require.Equal(t, 0, asset.Index)
			require.True(t, asset.IsCanonical)
			require.Equal(t, []int{1, 0}, asset.Tokens)
		}
	}
	require.True(t, purrFound, "PURR/USDC asset should be present in universe")

	// Verify we have at least one token
	require.Greater(t, len(res.Meta.Tokens), 0)
	require.NotEmpty(t, res.Meta.Tokens[0].Name)

	// Test specific known tokens from the cassette data
	var usdcFound, purrTokenFound bool
	for _, token := range res.Meta.Tokens {
		if token.Name == "USDC" {
			usdcFound = true
			require.Equal(t, 8, token.SzDecimals)
			require.Equal(t, 8, token.WeiDecimals)
			require.Equal(t, 0, token.Index)
			require.True(t, token.IsCanonical)
		}
		if token.Name == "PURR" {
			purrTokenFound = true
			require.Equal(t, 0, token.SzDecimals)
			require.Equal(t, 5, token.WeiDecimals)
			require.Equal(t, 1, token.Index)
			require.True(t, token.IsCanonical)
		}
	}
	require.True(t, usdcFound, "USDC token should be present in tokens")
	require.True(t, purrTokenFound, "PURR token should be present in tokens")

	// Verify we have at least one context
	require.Greater(t, len(res.Ctxs), 0)
	require.NotEmpty(t, res.Ctxs[0].Coin)
}

func TestMeta(t *testing.T) {
	info := NewInfo(context.TODO(), MainnetAPIURL, true, nil, nil, nil)

	initRecorder(t, false, "Meta")

	res, err := info.Meta(context.TODO())
	t.Logf("res: %+v", res)
	t.Logf("err: %v", err)

	require.NoError(t, err)

	// Verify the response structure
	require.NotNil(t, res)
	require.NotNil(t, res.Universe)
	require.NotNil(t, res.MarginTables)

	// Verify we have at least one asset in universe
	require.Greater(t, len(res.Universe), 0)
	require.NotEmpty(t, res.Universe[0].Name)

	// Test specific known assets from the cassette data
	var btcFound, ethFound bool
	for _, asset := range res.Universe {
		if asset.Name == "BTC" {
			btcFound = true
			require.Equal(t, 5, asset.SzDecimals)
		}
		if asset.Name == "ETH" {
			ethFound = true
			require.Equal(t, 4, asset.SzDecimals)
		}
	}
	require.True(t, btcFound, "BTC asset should be present in universe")
	require.True(t, ethFound, "ETH asset should be present in universe")

	// Verify we have at least one margin table
	require.Greater(t, len(res.MarginTables), 0)
	require.GreaterOrEqual(t, res.MarginTables[0].ID, 0)

	// Test specific margin table structure
	for _, marginTable := range res.MarginTables {
		require.NotNil(t, marginTable)
		require.Greater(t, len(marginTable.MarginTiers), 0)
		for _, tier := range marginTable.MarginTiers {
			require.NotEmpty(t, tier.LowerBound)
			require.Greater(t, tier.MaxLeverage, 0)
		}
	}
}

func TestSpotMeta(t *testing.T) {
	info := NewInfo(context.TODO(), MainnetAPIURL, true, nil, nil, nil)

	initRecorder(t, false, "SpotMeta")

	res, err := info.SpotMeta(context.TODO())
	t.Logf("res: %+v", res)
	t.Logf("err: %v", err)

	require.NoError(t, err)

	// Verify the response structure
	require.NotNil(t, res)
	require.NotNil(t, res.Universe)
	require.NotNil(t, res.Tokens)

	// Verify we have at least one asset in universe
	require.Greater(t, len(res.Universe), 0)
	require.NotEmpty(t, res.Universe[0].Name)

	// Test specific known assets from the cassette data
	var purrFound bool
	for _, asset := range res.Universe {
		if asset.Name == "PURR/USDC" {
			purrFound = true
			require.Equal(t, 0, asset.Index)
			require.True(t, asset.IsCanonical)
			require.Equal(t, []int{1, 0}, asset.Tokens)
		}
	}
	require.True(t, purrFound, "PURR/USDC asset should be present in universe")

	// Verify we have at least one token
	require.Greater(t, len(res.Tokens), 0)
	require.NotEmpty(t, res.Tokens[0].Name)

	// Test specific known tokens from the cassette data
	var usdcFound, purrTokenFound bool
	for _, token := range res.Tokens {
		if token.Name == "USDC" {
			usdcFound = true
			require.Equal(t, 8, token.SzDecimals)
			require.Equal(t, 8, token.WeiDecimals)
			require.Equal(t, 0, token.Index)
			require.True(t, token.IsCanonical)
		}
		if token.Name == "PURR" {
			purrTokenFound = true
			require.Equal(t, 0, token.SzDecimals)
			require.Equal(t, 5, token.WeiDecimals)
			require.Equal(t, 1, token.Index)
			require.True(t, token.IsCanonical)
		}
	}
	require.True(t, usdcFound, "USDC token should be present in tokens")
	require.True(t, purrTokenFound, "PURR token should be present in tokens")
}

func TestQueryOrderByOid(t *testing.T) {
	type tc struct {
		name         string
		cassetteName string
		user         string
		oid          int64
		expected     *OrderQueryResult
		wantErr      string
		record       bool
		useTestnet   bool
	}

	info := NewInfo(context.TODO(), MainnetAPIURL, true, nil, nil, nil)

	cases := []tc{
		{
			name:         "TRX unknown order",
			cassetteName: "QueryOrderByOid",
			user:         "0x31ca8395cf837de08b24da3f660e77761dfb974b",
			oid:          141622259364,
			expected: &OrderQueryResult{
				Status: OrderQueryStatusError,
				Order: OrderQueryResponse{
					Order: QueriedOrder{
						Children: []QueriedOrder{},
					},
				},
			},
			record: false,
		},
		{
			name:         "SAND unknown order",
			cassetteName: "QueryOrderByOid",
			user:         "0x31ca8395cf837de08b24da3f660e77761dfb974b",
			oid:          141623226620,
			expected: &OrderQueryResult{
				Status: OrderQueryStatusError,
				Order:  OrderQueryResponse{},
			},
			record: false,
		},
		{
			name:         "User 0x8e0C473fed9630906779f982Cd0F80Cb7011812D order 37907159219",
			cassetteName: "QueryOrderByOid",
			user:         "0x8e0C473fed9630906779f982Cd0F80Cb7011812D",
			oid:          37907159219,
			expected: &OrderQueryResult{
				Status: OrderQueryStatusSuccess,
				Order: OrderQueryResponse{
					Order: QueriedOrder{
						Coin:             "ETH",
						Side:             OrderSideBid,
						LimitPx:          "4650.4",
						Sz:               "0.0",
						Oid:              37907159219,
						Timestamp:        1755857898644,
						TriggerCondition: "N/A",
						IsTrigger:        false,
						TriggerPx:        "0.0",
						IsPositionTpsl:   false,
						ReduceOnly:       false,
						OrderType:        "Market",
						OrigSz:           "0.0025",
						Tif:              "FrontendMarket",
						Cloid:            nil,
					},
					Status:          OrderStatusValueFilled,
					StatusTimestamp: 1755857898644,
				},
			},
			record:     false,
			useTestnet: true,
		},
		{
			name:         "User 0x8e0C473fed9630906779f982Cd0F80Cb7011812D order 37907165748",
			cassetteName: "QueryOrderByOid",
			user:         "0x8e0C473fed9630906779f982Cd0F80Cb7011812D",
			oid:          37907165748,
			expected: &OrderQueryResult{
				Status: OrderQueryStatusSuccess,
				Order: OrderQueryResponse{
					Order: QueriedOrder{
						Coin:             "ETH",
						Side:             OrderSideAsk,
						LimitPx:          "3960.7",
						Sz:               "0.0",
						Oid:              37907165748,
						Timestamp:        1755857910772,
						TriggerCondition: "N/A",
						IsTrigger:        false,
						TriggerPx:        "0.0",
						IsPositionTpsl:   false,
						ReduceOnly:       true,
						OrderType:        "Market",
						OrigSz:           "0.0025",
						Tif:              "FrontendMarket",
						Cloid:            nil,
					},
					Status:          OrderStatusValueFilled,
					StatusTimestamp: 1755857910772,
				},
			},
			record:     false,
			useTestnet: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(tt *testing.T) {
			initRecorder(tt, tc.record, tc.cassetteName)

			var infoInstance *Info
			if tc.useTestnet {
				infoInstance = NewInfo(context.TODO(), TestnetAPIURL, true, nil, nil, nil)
			} else {
				infoInstance = info
			}

			res, err := infoInstance.QueryOrderByOid(context.TODO(), tc.user, tc.oid)
			tt.Logf("res: %+v", res)
			tt.Logf("err: %v", err)

			if tc.wantErr != "" {
				require.Error(tt, err)
				require.Contains(tt, err.Error(), tc.wantErr)
				return
			} else {
				require.NoError(tt, err)
			}

			if err == nil {
				require.NotNil(tt, res)
				require.Equal(tt, tc.expected.Status, res.Status)

				// If order is found, compare order details
				if res.Status == OrderQueryStatusSuccess {
					require.Equal(tt, tc.expected.Order.Status, res.Order.Status)
					require.Equal(tt, tc.expected.Order.StatusTimestamp, res.Order.StatusTimestamp)

					// Compare order details - use cmp.Diff to treat nil and empty slices as equal
					expectedOrder := tc.expected.Order.Order
					actualOrder := res.Order.Order
					diff := cmp.Diff(expectedOrder, actualOrder, cmpopts.EquateEmpty())
					if diff != "" {
						tt.Errorf("Order mismatch (-want +got):\n%s", diff)
					}
				}
			}
		})
	}
}

func TestUserFillsByTime(t *testing.T) {
	type tc struct {
		name         string
		cassetteName string
		user         string
		startTime    int64
		endTime      *int64
		expected     []Fill
		wantErr      string
		record       bool
		useTestnet   bool
	}

	info := NewInfo(context.TODO(), MainnetAPIURL, true, nil, nil, nil)

	cases := []tc{
		{
			name:         "User 0x8e0C473fed9630906779f982Cd0F80Cb7011812D fills in time range",
			cassetteName: "UserFillsByTime",
			user:         "0x8e0C473fed9630906779f982Cd0F80Cb7011812D",
			startTime:    1755857880000,
			endTime:      func() *int64 { t := int64(1755857940000); return &t }(),
			expected: []Fill{
				{
					ClosedPnl:     "0.0",
					Coin:          "ETH",
					Crossed:       true,
					Dir:           "Open Long",
					Hash:          "0x7d6e6ad7ce8fdfdf7ee8041907273d010f0082bd6982feb12137162a8d83b9ca",
					Oid:           37907159219,
					Price:         "4307.4",
					Side:          "B",
					StartPosition: "0.0",
					Size:          "0.0025",
					Time:          1755857898644,
					Fee:           "0.004845",
					FeeToken:      "USDC",
					BuilderFee:    "",
					Tid:           1070455675927460,
				},
				{
					ClosedPnl:     "-0.00925",
					Coin:          "ETH",
					Crossed:       true,
					Dir:           "Close Long",
					Hash:          "0x93ebdf1acc4dd95a9565041907278a010a00f7006740f82c37b48a6d8b41b345",
					Oid:           37907165748,
					Price:         "4303.7",
					Side:          "A",
					StartPosition: "0.0025",
					Size:          "0.0025",
					Time:          1755857910772,
					Fee:           "0.004841",
					FeeToken:      "USDC",
					BuilderFee:    "",
					Tid:           912424546441675,
				},
			},
			record:     false,
			useTestnet: true,
		},
		{
			name:         "User with no fills in time range",
			cassetteName: "UserFillsByTimeEmpty",
			user:         "0x31ca8395cf837de08b24da3f660e77761dfb974b",
			startTime:    1755857880000,
			endTime:      func() *int64 { t := int64(1755857940000); return &t }(),
			expected:     []Fill{},
			record:       false,
			useTestnet:   true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(tt *testing.T) {
			initRecorder(tt, tc.record, tc.cassetteName)

			var infoInstance *Info
			if tc.useTestnet {
				infoInstance = NewInfo(context.TODO(), TestnetAPIURL, true, nil, nil, nil)
			} else {
				infoInstance = info
			}

			res, err := infoInstance.UserFillsByTime(
				context.TODO(),
				tc.user,
				tc.startTime,
				tc.endTime,
				nil,
			)
			tt.Logf("res: %+v", res)
			tt.Logf("err: %v", err)

			if tc.wantErr != "" {
				require.Error(tt, err)
				require.Contains(tt, err.Error(), tc.wantErr)
				return
			} else {
				require.NoError(tt, err)
			}

			if err == nil {
				require.NotNil(tt, res)
				require.Equal(tt, len(tc.expected), len(res))

				// Compare each fill in the response
				for i, expectedFill := range tc.expected {
					if i < len(res) {
						actualFill := res[i]
						require.Equal(tt, expectedFill, actualFill)
					}
				}
			}
		})
	}
}

func TestSpotUserState(t *testing.T) {
	type tc struct {
		name         string
		cassetteName string
		user         string
		expected     *SpotUserState
		wantErr      string
		record       bool
		useTestnet   bool
	}

	info := NewInfo(context.TODO(), MainnetAPIURL, true, nil, nil, nil)

	cases := []tc{
		{
			name:         "User 0x8e0C473fed9630906779f982Cd0F80Cb7011812D spot state",
			cassetteName: "SpotUserState",
			user:         "0x8e0C473fed9630906779f982Cd0F80Cb7011812D",
			expected: &SpotUserState{
				Balances: []SpotBalance{
					{
						Coin:     "USDC",
						Token:    0,
						Hold:     "0.0",
						Total:    "19.9969993",
						EntryNtl: "0.0",
					},
					{
						Coin:     "HYPE",
						Token:    1105,
						Hold:     "0.2",
						Total:    "0.24965",
						EntryNtl: "24.982487",
					},
					{
						Coin:     "USOL",
						Token:    1279,
						Hold:     "0.0",
						Total:    "0.9993",
						EntryNtl: "249.99",
					},
				},
			},
			record:     false,
			useTestnet: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(tt *testing.T) {
			initRecorder(tt, tc.record, tc.cassetteName)

			var infoInstance *Info
			if tc.useTestnet {
				infoInstance = NewInfo(context.TODO(), TestnetAPIURL, true, nil, nil, nil)
			} else {
				infoInstance = info
			}

			res, err := infoInstance.SpotUserState(context.TODO(), tc.user)
			tt.Logf("res: %+v", res)
			tt.Logf("err: %v", err)

			if tc.wantErr != "" {
				require.Error(tt, err)
				require.Contains(tt, err.Error(), tc.wantErr)
				return
			} else {
				require.NoError(tt, err)
			}

			if err == nil {
				require.NotNil(tt, res)
				require.NotNil(tt, tc.expected)
				require.Equal(tt, len(tc.expected.Balances), len(res.Balances))

				// Compare each balance in the response
				for i, expectedBalance := range tc.expected.Balances {
					if i < len(res.Balances) {
						actualBalance := res.Balances[i]
						require.Equal(tt, expectedBalance, actualBalance)
					}
				}
			}
		})
	}
}

func TestUserActiveAssetData(t *testing.T) {
	type tc struct {
		name         string
		cassetteName string
		user         string
		coin         string
		expected     *UserActiveAssetData
		wantErr      string
		record       bool
		useTestnet   bool
	}

	info := NewInfo(context.TODO(), MainnetAPIURL, true, nil, nil, nil)

	cases := []tc{
		{
			name:         "User 0x8e0C473fed9630906779f982Cd0F80Cb7011812D active asset data for HYPE",
			cassetteName: "UserActiveAssetData",
			user:         "0x8e0C473fed9630906779f982Cd0F80Cb7011812D",
			coin:         "HYPE",
			expected: &UserActiveAssetData{
				User: "0x8e0c473fed9630906779f982cd0f80cb7011812d",
				Coin: "HYPE",
				Leverage: Leverage{
					Type:  "cross",
					Value: 10,
				},
				MaxTradeSzs:      []string{"72.42", "72.42"},
				AvailableToTrade: []string{"680.955673", "680.955673"},
				MarkPx:           "94.017",
			},
			record:     false, // Set to false after recording
			useTestnet: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(tt *testing.T) {
			initRecorder(tt, tc.record, tc.cassetteName)

			var infoInstance *Info
			if tc.useTestnet {
				infoInstance = NewInfo(context.TODO(), TestnetAPIURL, true, nil, nil, nil)
			} else {
				infoInstance = info
			}

			res, err := infoInstance.UserActiveAssetData(context.TODO(), tc.user, tc.coin)
			tt.Logf("res: %+v", res)
			tt.Logf("err: %v", err)

			if tc.wantErr != "" {
				require.Error(tt, err)
				require.Contains(tt, err.Error(), tc.wantErr)
				return
			} else {
				require.NoError(tt, err)
			}

			if err == nil {
				require.NotNil(tt, res)

				// Verify the response structure
				require.Equal(tt, tc.expected, res)
			}
		})
	}
}

func TestTokenDetails(t *testing.T) {
	info := NewInfo(context.TODO(), MainnetAPIURL, true, nil, nil, nil)

	type tc struct {
		name    string
		address string
		coin    string
	}

	cases := []tc{
		{
			name:    "fetch USDC token details",
			address: "0x6d1e7cde53ba9467b783cb7c530ce054",
			coin:    "USDC",
		},
		{
			name:    "fetch MAGA token details",
			address: "0x9b91002773083d9292b8cb02dacb7e79",
			coin:    "MAGA",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(tt *testing.T) {
			resp, err := info.TokenDetails(context.TODO(), tc.address)
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.Equal(t, tc.coin, resp.Name)
		})
	}
}

func TestPerpDexs(t *testing.T) {
	info := NewInfo(context.TODO(), MainnetAPIURL, true, nil, nil, nil)

	initRecorder(t, false, "PerpDexs")

	res, err := info.PerpDexs(context.TODO())
	t.Logf("res: %+v", res)
	t.Logf("err: %v", err)

	require.NoError(t, err)
	require.NotNil(t, res)

	// PerpDexs returns a MixedArray where first element is null (default dex)
	// and subsequent elements are PerpDex objects
	require.Greater(t, len(res), 0)

	// First element should be null (default dex)
	if len(res) > 0 {
		firstType := res[0].Type()
		// First element can be null or an object
		require.Contains(t, []string{"null", "object"}, firstType)
	}
}

func TestMetaWithDex(t *testing.T) {
	info := NewInfo(context.TODO(), MainnetAPIURL, true, nil, nil, nil)

	initRecorder(t, false, "Meta_WithDex")

	// Test with default dex (empty string)
	res1, err := info.Meta(context.TODO())
	require.NoError(t, err)
	require.NotNil(t, res1)
	require.Greater(t, len(res1.Universe), 0)

	// Test with explicit empty dex (should be same as default)
	res2, err := info.Meta(context.TODO(), "")
	require.NoError(t, err)
	require.NotNil(t, res2)
	require.Equal(t, len(res1.Universe), len(res2.Universe))
}

func TestUserStateWithDex(t *testing.T) {
	// Use a known test address
	testAddress := "0xcd5051944f780a621ee62e39e493c489668acf4d"
	info := NewInfo(context.TODO(), MainnetAPIURL, true, nil, nil, nil)

	initRecorder(t, false, "UserState_WithDex")

	// Test with default dex (empty string)
	res1, err := info.UserState(context.TODO(), testAddress)
	require.NoError(t, err)
	require.NotNil(t, res1)

	// Test with explicit empty dex (should be same as default)
	res2, err := info.UserState(context.TODO(), testAddress, "")
	require.NoError(t, err)
	require.NotNil(t, res2)
	require.Equal(t, res1.MarginSummary.AccountValue, res2.MarginSummary.AccountValue)
}

func TestAllMidsWithDex(t *testing.T) {
	info := NewInfo(context.TODO(), MainnetAPIURL, true, nil, nil, nil)

	initRecorder(t, false, "AllMids_WithDex")

	// Test with default dex (empty string)
	res1, err := info.AllMids(context.TODO())
	require.NoError(t, err)
	require.NotNil(t, res1)
	require.Greater(t, len(res1), 0)

	// Test with explicit empty dex (should be same as default)
	res2, err := info.AllMids(context.TODO(), "")
	require.NoError(t, err)
	require.NotNil(t, res2)
	require.Equal(t, len(res1), len(res2))
}

func TestOpenOrdersWithDex(t *testing.T) {
	// Use a known test address
	testAddress := "0xcd5051944f780a621ee62e39e493c489668acf4d"
	info := NewInfo(context.TODO(), MainnetAPIURL, true, nil, nil, nil)

	initRecorder(t, false, "OpenOrders_WithDex")

	// Test with default dex (empty string)
	res1, err := info.OpenOrders(context.TODO(), testAddress)
	require.NoError(t, err)
	require.NotNil(t, res1)

	// Test with explicit empty dex (should be same as default)
	res2, err := info.OpenOrders(context.TODO(), testAddress, "")
	require.NoError(t, err)
	require.NotNil(t, res2)
	require.Equal(t, len(res1), len(res2))
}

func TestFrontendOpenOrdersWithDex(t *testing.T) {
	// Use a known test address
	testAddress := "0xcd5051944f780a621ee62e39e493c489668acf4d"
	info := NewInfo(context.TODO(), MainnetAPIURL, true, nil, nil, nil)

	initRecorder(t, false, "FrontendOpenOrders_WithDex")

	// Test with default dex (empty string)
	res1, err := info.FrontendOpenOrders(context.TODO(), testAddress)
	require.NoError(t, err)
	require.NotNil(t, res1)

	// Test with explicit empty dex (should be same as default)
	res2, err := info.FrontendOpenOrders(context.TODO(), testAddress, "")
	require.NoError(t, err)
	require.NotNil(t, res2)
	require.Equal(t, len(res1), len(res2))
}

func TestPerpDexLimits(t *testing.T) {
	info := NewInfo(context.TODO(), MainnetAPIURL, true, nil, nil, nil)

	// First get available DEXs
	initRecorder(t, false, "PerpDexs_ForLimits")
	dexs, err := info.PerpDexs(context.TODO())
	require.NoError(t, err)

	// Find a non-null DEX (skip the first one which is null/default)
	var testDex string
	for i, dex := range dexs {
		if i == 0 {
			continue // Skip first (default dex)
		}
		if dex.Type() == "object" {
			var perpDex PerpDex
			if err := dex.Parse(&perpDex); err == nil && perpDex.Name != "" {
				testDex = perpDex.Name
				break
			}
		}
	}

	// Only test if we have a builder-deployed DEX
	if testDex != "" {
		initRecorder(t, false, "PerpDexLimits")

		res, err := info.PerpDexLimits(context.TODO(), testDex)
		t.Logf("res: %+v", res)
		t.Logf("err: %v", err)

		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotEmpty(t, res.TotalOiCap)
		require.NotEmpty(t, res.OiSzCapPerPerp)
		require.NotEmpty(t, res.MaxTransferNtl)
	} else {
		t.Skip("No builder-deployed DEX available for testing")
	}
}

func TestPerpDexStatus(t *testing.T) {
	info := NewInfo(context.TODO(), MainnetAPIURL, true, nil, nil, nil)

	// First get available DEXs to test with a specific DEX
	initRecorder(t, false, "PerpDexs_ForStatus")
	dexs, err := info.PerpDexs(context.TODO())
	require.NoError(t, err)

	// Find a non-null DEX
	var testDex string
	for i, dex := range dexs {
		if i == 0 {
			continue // Skip first (default dex)
		}
		if dex.Type() == "object" {
			var perpDex PerpDex
			if err := dex.Parse(&perpDex); err == nil && perpDex.Name != "" {
				testDex = perpDex.Name
				break
			}
		}
	}

	// Test with specific DEX if available
	if testDex != "" {
		initRecorder(t, false, "PerpDexStatus_WithDex")

		res2, err := info.PerpDexStatus(context.TODO(), testDex)
		require.NoError(t, err)
		require.NotNil(t, res2)
		require.NotEmpty(t, res2.TotalNetDeposit)
	}
}

func TestPerpDeployAuctionStatus(t *testing.T) {
	info := NewInfo(context.TODO(), MainnetAPIURL, true, nil, nil, nil)

	initRecorder(t, false, "PerpDeployAuctionStatus")

	res, err := info.PerpDeployAuctionStatus(context.TODO())
	t.Logf("res: %+v", res)
	t.Logf("err: %v", err)

	require.NoError(t, err)
	require.NotNil(t, res)
	require.Greater(t, res.StartTimeSeconds, int64(0))
	require.Greater(t, res.DurationSeconds, int64(0))
	require.NotEmpty(t, res.StartGas)
	require.NotEmpty(t, res.CurrentGas)
}

func TestPerpDexLimits_RequiresNonEmptyDex(t *testing.T) {
	info := NewInfo(context.TODO(), MainnetAPIURL, true, nil, nil, nil)

	// PerpDexLimits should fail with empty dex
	_, err := info.PerpDexLimits(context.TODO(), "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "dex parameter is required")
}
