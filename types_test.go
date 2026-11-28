package hyperliquid

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpotTokenInfo_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		token    SpotTokenInfo
		expected string
	}{
		{
			name: "nil_evm_contract_and_fullname",
			token: SpotTokenInfo{
				Name:        "USDC",
				SzDecimals:  6,
				WeiDecimals: 6,
				Index:       0,
				TokenID:     "0x1",
				IsCanonical: true,
				EvmContract: nil,
				FullName:    nil,
			},
			expected: `{"name":"USDC","szDecimals":6,"weiDecimals":6,"index":0,"tokenId":"0x1","isCanonical":true,"evmContract":null,"fullName":null,"deployerTradingFeeShare":""}`,
		},
		{
			name: "with_evm_contract_and_fullname",
			token: SpotTokenInfo{
				Name:        "USDC",
				SzDecimals:  6,
				WeiDecimals: 6,
				Index:       1,
				TokenID:     "0x2",
				IsCanonical: false,
				EvmContract: &EvmContract{
					Address:             "0x1234567890abcdef1234567890abcdef12345678",
					EvmExtraWeiDecimals: 12,
				},
				FullName: stringPtr("USD Coin"),
			},
			expected: `{"name":"USDC","szDecimals":6,"weiDecimals":6,"index":1,"tokenId":"0x2","isCanonical":false,"evmContract":{"address":"0x1234567890abcdef1234567890abcdef12345678","evm_extra_wei_decimals":12},"fullName":"USD Coin","deployerTradingFeeShare":""}`,
		},
		{
			name: "empty_evm_contract",
			token: SpotTokenInfo{
				Name:        "ETH",
				SzDecimals:  18,
				WeiDecimals: 18,
				Index:       2,
				TokenID:     "0x3",
				IsCanonical: true,
				EvmContract: &EvmContract{
					Address:             "",
					EvmExtraWeiDecimals: 0,
				},
				FullName: stringPtr("Ethereum"),
			},
			expected: `{"name":"ETH","szDecimals":18,"weiDecimals":18,"index":2,"tokenId":"0x3","isCanonical":true,"evmContract":{"address":"","evm_extra_wei_decimals":0},"fullName":"Ethereum","deployerTradingFeeShare":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			jsonData, err := json.Marshal(tt.token)
			require.NoError(t, err, "marshaling should not fail")
			assert.JSONEq(t, tt.expected, string(jsonData), "marshaled JSON should match expected")

			// Test round-trip: unmarshal the marshaled data
			var unmarshaled SpotTokenInfo
			err = json.Unmarshal(jsonData, &unmarshaled)
			require.NoError(t, err, "unmarshaling should not fail")
			assert.Equal(t, tt.token, unmarshaled, "round-trip should preserve data")
		})
	}
}

func TestSpotTokenInfo_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		expected SpotTokenInfo
		wantErr  bool
	}{
		{
			name:     "complete_token_info",
			jsonData: `{"name":"USDC","szDecimals":6,"weiDecimals":6,"index":0,"tokenId":"0x1","isCanonical":true,"evmContract":{"address":"0x1234567890abcdef","evm_extra_wei_decimals":12},"fullName":"USD Coin"}`,
			expected: SpotTokenInfo{
				Name:        "USDC",
				SzDecimals:  6,
				WeiDecimals: 6,
				Index:       0,
				TokenID:     "0x1",
				IsCanonical: true,
				EvmContract: &EvmContract{
					Address:             "0x1234567890abcdef",
					EvmExtraWeiDecimals: 12,
				},
				FullName: stringPtr("USD Coin"),
			},
			wantErr: false,
		},
		{
			name:     "null_optional_fields",
			jsonData: `{"name":"ETH","szDecimals":18,"weiDecimals":18,"index":1,"tokenId":"0x2","isCanonical":false,"evmContract":null,"fullName":null}`,
			expected: SpotTokenInfo{
				Name:        "ETH",
				SzDecimals:  18,
				WeiDecimals: 18,
				Index:       1,
				TokenID:     "0x2",
				IsCanonical: false,
				EvmContract: nil,
				FullName:    nil,
			},
			wantErr: false,
		},
		{
			name:     "missing_optional_fields",
			jsonData: `{"name":"BTC","szDecimals":8,"weiDecimals":8,"index":2,"tokenId":"0x3","isCanonical":true}`,
			expected: SpotTokenInfo{
				Name:        "BTC",
				SzDecimals:  8,
				WeiDecimals: 8,
				Index:       2,
				TokenID:     "0x3",
				IsCanonical: true,
				EvmContract: nil,
				FullName:    nil,
			},
			wantErr: false,
		},
		{
			name:     "zero_decimals_in_evm_contract",
			jsonData: `{"name":"WETH","szDecimals":18,"weiDecimals":18,"index":3,"tokenId":"0x4","isCanonical":false,"evmContract":{"address":"0xabcdef1234567890abcdef1234567890abcdef12","evm_extra_wei_decimals":0},"fullName":"Wrapped Ethereum"}`,
			expected: SpotTokenInfo{
				Name:        "WETH",
				SzDecimals:  18,
				WeiDecimals: 18,
				Index:       3,
				TokenID:     "0x4",
				IsCanonical: false,
				EvmContract: &EvmContract{
					Address:             "0xabcdef1234567890abcdef1234567890abcdef12",
					EvmExtraWeiDecimals: 0,
				},
				FullName: stringPtr("Wrapped Ethereum"),
			},
			wantErr: false,
		},
		{
			name:     "malformed_json",
			jsonData: `{"name":"INVALID"`,
			expected: SpotTokenInfo{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var token SpotTokenInfo
			err := json.Unmarshal([]byte(tt.jsonData), &token)

			if tt.wantErr {
				assert.Error(t, err, "unmarshaling should fail for malformed JSON")
			} else {
				require.NoError(t, err, "unmarshaling should not fail")
				assert.Equal(t, tt.expected, token, "unmarshaled token should match expected")
			}
		})
	}
}

func TestSpotMeta_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		meta     SpotMeta
		expected string
	}{
		{
			name: "empty_meta",
			meta: SpotMeta{
				Universe: []SpotAssetInfo{},
				Tokens:   []SpotTokenInfo{},
			},
			expected: `{"universe":[],"tokens":[]}`,
		},
		{
			name: "meta_with_assets_and_tokens",
			meta: SpotMeta{
				Universe: []SpotAssetInfo{
					{
						Name:        "USDC/USDT",
						Tokens:      []int{0, 1},
						Index:       0,
						IsCanonical: true,
					},
				},
				Tokens: []SpotTokenInfo{
					{
						Name:        "USDC",
						SzDecimals:  6,
						WeiDecimals: 6,
						Index:       0,
						TokenID:     "0x1",
						IsCanonical: true,
						EvmContract: &EvmContract{
							Address:             "0x1234567890abcdef",
							EvmExtraWeiDecimals: 12,
						},
						FullName: stringPtr("USD Coin"),
					},
				},
			},
			expected: `{"universe":[{"name":"USDC/USDT","tokens":[0,1],"index":0,"isCanonical":true}],"tokens":[{"name":"USDC","szDecimals":6,"weiDecimals":6,"index":0,"tokenId":"0x1","isCanonical":true,"evmContract":{"address":"0x1234567890abcdef","evm_extra_wei_decimals":12},"fullName":"USD Coin","deployerTradingFeeShare":""}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			jsonData, err := json.Marshal(tt.meta)
			require.NoError(t, err, "marshaling should not fail")
			assert.JSONEq(t, tt.expected, string(jsonData), "marshaled JSON should match expected")

			// Test round-trip
			var unmarshaled SpotMeta
			err = json.Unmarshal(jsonData, &unmarshaled)
			require.NoError(t, err, "unmarshaling should not fail")
			assert.Equal(t, tt.meta, unmarshaled, "round-trip should preserve data")
		})
	}
}

func TestSpotAssetCtx_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		ctx      SpotAssetCtx
		expected string
	}{
		{
			name: "complete_asset_context",
			ctx: SpotAssetCtx{
				DayNtlVlm:         "1000000.50",
				MarkPx:            "1.0001",
				MidPx:             stringPtr("1.0002"),
				PrevDayPx:         "1.0000",
				CirculatingSupply: "1000000000",
				Coin:              "USDC",
			},
			expected: `{"dayNtlVlm":"1000000.50","markPx":"1.0001","midPx":"1.0002","prevDayPx":"1.0000","circulatingSupply":"1000000000","coin":"USDC","totalSupply":"","dayBaseVlm":""}`,
		},
		{
			name: "null_mid_price",
			ctx: SpotAssetCtx{
				DayNtlVlm:         "500000.25",
				MarkPx:            "50000.00",
				MidPx:             nil,
				PrevDayPx:         "49950.00",
				CirculatingSupply: "21000000",
				Coin:              "BTC",
			},
			expected: `{"dayNtlVlm":"500000.25","markPx":"50000.00","midPx":null,"prevDayPx":"49950.00","circulatingSupply":"21000000","coin":"BTC","totalSupply":"","dayBaseVlm":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			jsonData, err := json.Marshal(tt.ctx)
			require.NoError(t, err, "marshaling should not fail")
			assert.JSONEq(t, tt.expected, string(jsonData), "marshaled JSON should match expected")

			// Test round-trip
			var unmarshaled SpotAssetCtx
			err = json.Unmarshal(jsonData, &unmarshaled)
			require.NoError(t, err, "unmarshaling should not fail")
			assert.Equal(t, tt.ctx, unmarshaled, "round-trip should preserve data")
		})
	}
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
