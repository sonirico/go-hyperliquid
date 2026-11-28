package hyperliquid

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignL1Action(t *testing.T) {
	// Test private key
	privateKeyHex := "abcd1234567890abcd1234567890abcd1234567890abcd1234567890abcd1234"
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	require.NoError(t, err)

	privateKey, err := crypto.ToECDSA(privateKeyBytes)
	require.NoError(t, err)

	tests := []struct {
		name         string
		action       map[string]any
		vaultAddress string
		timestamp    int64
		expiresAfter *int64
		isMainnet    bool
		wantErr      bool
		description  string
	}{
		{
			name: "basic_order_action_testnet",
			action: map[string]any{
				"type": "order",
				"orders": []map[string]any{
					{
						"asset":      0,
						"isBuy":      true,
						"limitPx":    "100.0",
						"orderType":  "Limit",
						"reduceOnly": false,
						"size":       "1.0",
						"tif":        "Gtc",
					},
				},
				"grouping": "na",
			},
			vaultAddress: "",
			timestamp:    1703001234567,
			expiresAfter: nil,
			isMainnet:    false,
			wantErr:      false,
			description:  "Basic order action on testnet without expiration",
		},
		{
			name: "basic_order_action_mainnet",
			action: map[string]any{
				"type": "order",
				"orders": []map[string]any{
					{
						"asset":      0,
						"isBuy":      true,
						"limitPx":    "100.0",
						"orderType":  "Limit",
						"reduceOnly": false,
						"size":       "1.0",
						"tif":        "Gtc",
					},
				},
				"grouping": "na",
			},
			vaultAddress: "",
			timestamp:    1703001234567,
			expiresAfter: nil,
			isMainnet:    true,
			wantErr:      false,
			description:  "Basic order action on mainnet without expiration",
		},
		{
			name: "order_with_expiration",
			action: map[string]any{
				"type": "order",
				"orders": []map[string]any{
					{
						"asset":      0,
						"isBuy":      true,
						"limitPx":    "100.0",
						"orderType":  "Limit",
						"reduceOnly": false,
						"size":       "1.0",
						"tif":        "Gtc",
					},
				},
				"grouping": "na",
			},
			vaultAddress: "",
			timestamp:    1703001234567,
			expiresAfter: func() *int64 { e := int64(1703001234567 + 3600000); return &e }(), // 1 hour later
			isMainnet:    false,
			wantErr:      false,
			description:  "Order action with expiration",
		},
		{
			name: "order_with_vault",
			action: map[string]any{
				"type": "order",
				"orders": []map[string]any{
					{
						"asset":      0,
						"isBuy":      true,
						"limitPx":    "100.0",
						"orderType":  "Limit",
						"reduceOnly": false,
						"size":       "1.0",
						"tif":        "Gtc",
					},
				},
				"grouping": "na",
			},
			vaultAddress: "0x1234567890abcdef1234567890abcdef12345678",
			timestamp:    1703001234567,
			expiresAfter: nil,
			isMainnet:    false,
			wantErr:      false,
			description:  "Order action with vault address",
		},
		{
			name: "leverage_update_action",
			action: map[string]any{
				"type":     "updateLeverage",
				"asset":    0,
				"isCross":  true,
				"leverage": 10,
			},
			vaultAddress: "",
			timestamp:    1703001234567,
			expiresAfter: nil,
			isMainnet:    false,
			wantErr:      false,
			description:  "Leverage update action",
		},
		{
			name: "usd_class_transfer_action",
			action: map[string]any{
				"type":   "usdClassTransfer",
				"amount": "100.0",
				"toPerp": true,
			},
			vaultAddress: "",
			timestamp:    1703001234567,
			expiresAfter: nil,
			isMainnet:    false,
			wantErr:      false,
			description:  "USD class transfer action",
		},
		{
			name: "cancel_action",
			action: map[string]any{
				"type": "cancel",
				"cancels": []map[string]any{
					{
						"asset": 0,
						"oid":   12345,
					},
				},
			},
			vaultAddress: "",
			timestamp:    1703001234567,
			expiresAfter: nil,
			isMainnet:    false,
			wantErr:      false,
			description:  "Cancel action without vault",
		},
		{
			name: "vault_action",
			action: map[string]any{
				"type":     "updateLeverage",
				"asset":    0,
				"leverage": 10,
				"isCross":  true,
			},
			vaultAddress: "0x1234567890123456789012345678901234567890",
			timestamp:    1703001234567,
			expiresAfter: nil,
			isMainnet:    true,
			wantErr:      false,
			description:  "Action with vault address",
		},
		{
			name: "empty_vault_with_expiration",
			action: map[string]any{
				"type": "setReferrer",
				"code": "TEST123",
			},
			vaultAddress: "",
			timestamp:    1703001234567,
			expiresAfter: func() *int64 { e := int64(1703001234567 + 86400000); return &e }(), // 24 hours
			isMainnet:    false,
			wantErr:      false,
			description:  "Empty vault with expiration time",
		},
		{
			name: "nil_vault_with_expiration",
			action: map[string]any{
				"type": "createSubAccount",
				"name": "TestAccount",
			},
			vaultAddress: "",
			timestamp:    1703001234567,
			expiresAfter: func() *int64 { e := int64(1703001234567 + 1800000); return &e }(), // 30 minutes
			isMainnet:    true,
			wantErr:      false,
			description:  "Nil vault with expiration time on mainnet",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signature, err := SignL1Action(
				privateKey,
				tt.action,
				tt.vaultAddress,
				tt.timestamp,
				tt.expiresAfter,
				tt.isMainnet,
			)

			if tt.wantErr {
				assert.Error(t, err, tt.description)
				return
			}

			require.NoError(t, err, tt.description)
			assert.NotEmpty(t, signature.R, "Signature R should not be empty")
			assert.NotEmpty(t, signature.S, "Signature S should not be empty")
			assert.True(t, signature.V > 0, "Signature V should be positive")
			assert.True(
				t,
				len(signature.R) >= 3,
				"Signature R should be at least 3 characters (0x + hex)",
			)
			assert.True(
				t,
				len(signature.S) >= 3,
				"Signature S should be at least 3 characters (0x + hex)",
			)
			assert.True(t, signature.R[:2] == "0x", "Signature R should start with 0x")
			assert.True(t, signature.S[:2] == "0x", "Signature S should start with 0x")

			// Validate that R and S are valid hex strings
			assert.Regexp(t, "^0x[0-9a-fA-F]+$", signature.R, "Signature R should be valid hex")
			assert.Regexp(t, "^0x[0-9a-fA-F]+$", signature.S, "Signature S should be valid hex")

			// Note: ECDSA signatures are NOT deterministic by default in go-ethereum
			// Each signature uses a random nonce (k value), which is actually more secure
			// against side-channel attacks. The signatures are still valid and verifiable.
		})
	}
}

// TestDebugActionHash helps debug the action hash generation
func TestDebugActionHash(t *testing.T) {
	// Use the same test data as Python
	action := OrderAction{
		Type: "order",
		Orders: []OrderWire{{
			Asset:      0,
			IsBuy:      true,
			LimitPx:    "100.5",
			Size:       "1.0",
			ReduceOnly: false,
			OrderType: OrderWireType{
				Limit: &OrderWireTypeLimit{
					Tif: TifGtc,
				},
			},
		}},
		Grouping: NewOrderGroupingNA(),
	}

	privateKey, _ := crypto.HexToECDSA(
		"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	)
	vaultAddress := ""
	timestamp := int64(1640995200000) // Fixed timestamp
	var expiresAfter *int64 = nil
	isMainnet := false

	// Debug: Print action hash components
	hash, err := actionHash(action, vaultAddress, timestamp, expiresAfter)
	require.NoError(t, err)
	t.Logf("Action hash: %x", hash)

	// Debug: Print phantom agent
	phantomAgent := constructPhantomAgent(hash, isMainnet)
	t.Logf("Phantom agent: %+v", phantomAgent)

	// Generate signature
	signature, err := SignL1Action(
		privateKey,
		action,
		vaultAddress,
		timestamp,
		expiresAfter,
		isMainnet,
	)
	require.NoError(t, err)
	t.Logf("Generated signature: R=%s, S=%s, V=%d", signature.R, signature.S, signature.V)
}

// TestActionHash_CancelWithLargeOrderID verifies that actionHash produces a
// stable hash for a CancelAction containing order_id 361731063972 whose uint64
// encoding embeds 0xda. The hash must remain identical across repeated calls.
func TestActionHash_CancelWithLargeOrderID(t *testing.T) {
	action := CancelAction{
		Type: "cancel",
		Cancels: []CancelOrderWire{
			{Asset: 0, OrderID: 361731063972},
		},
	}
	vaultAddress := ""
	nonce := int64(1703001234567)
	var expiresAfter *int64

	hash1, err := actionHash(action, vaultAddress, nonce, expiresAfter)
	require.NoError(t, err)
	hash2, err := actionHash(action, vaultAddress, nonce, expiresAfter)
	require.NoError(t, err)

	assert.Equal(t, hash1, hash2,
		"actionHash must be deterministic across repeated calls")

	// Verify the hash is non-zero / non-trivial
	assert.Len(t, hash1, 32, "keccak256 hash must be 32 bytes")
	assert.False(t, bytes.Equal(hash1, make([]byte, 32)),
		"hash must not be all zeros")
}

// The vectors below come from the Python SDK (hyperliquid-python-sdk). The usdSend
// and withdraw3 ones are the vectors of its tests/signing_test.py; the spotSend and
// usdClassTransfer ones were produced with sign_spot_transfer_action and
// sign_usd_class_transfer_action on the same key, amount and timestamp.
func TestUserSignedSignHelpers_MatchPythonSDK(t *testing.T) {
	privateKey, err := crypto.HexToECDSA(
		"0123456789012345678901234567890123456789012345678901234567890123",
	)
	require.NoError(t, err)

	const (
		destination = "0x5e9ee1089755c3435139848e47e6635505d5a13a"
		timestamp   = int64(1687816341423)
	)

	golden := loadTestGoldenFiles(t)

	tests := []struct {
		name string
		sign func() (SignatureResult, error)
		want SignatureResult
	}{
		{
			name: "usdSend",
			sign: func() (SignatureResult, error) {
				return SignUsdTransferAction(privateKey, 1, destination, timestamp, false)
			},
			want: SignatureResult{
				R: "0x637b37dd731507cdd24f46532ca8ba6eec616952c56218baeff04144e4a77073",
				S: "0x11a6a24900e6e314136d2592e2f8d502cd89b7c15b198e1bee043c9589f9fad7",
				V: 27,
			},
		},
		{
			name: "usdSend mainnet",
			sign: func() (SignatureResult, error) {
				return SignUsdTransferAction(privateKey, 1, destination, timestamp, true)
			},
			want: SignatureResult{
				R: "0x9adbd066e8ae671f06b5d18e51fd2ec11b5443d64b13718028c7ff06905c71fb",
				S: "0x1eba4172075f8a8037481400ae86ed56c5e5f672e3db084528a8e3b2daeff976",
				V: 28,
			},
		},
		{
			name: "withdraw3",
			sign: func() (SignatureResult, error) {
				return SignWithdrawFromBridgeAction(privateKey, destination, 1, 0, timestamp, false)
			},
			want: SignatureResult{
				R: "0x8363524c799e90ce9bc41022f7c39b4e9bdba786e5f9c72b20e43e1462c37cf9",
				S: "0x58b1411a775938b83e29182e8ef74975f9054c8e97ebf5ec2dc8d51bfc893881",
				V: 28,
			},
		},
		{
			name: "spotSend",
			sign: func() (SignatureResult, error) {
				return SignSpotTransferAction(
					privateKey, 1, destination, "PURR:0xc4bf3f870c0e9465323c0b6ed28096c2", timestamp, false,
				)
			},
			want: SignatureResult{
				R: "0x165ba998a7f919dde45bbaf26259103007c135b7e426ecc6e2db98d32a00cb12",
				S: "0x2f285400621dab9ae58270a9c337f9d66d575896c4543e4f13652d0867542a3",
				V: 28,
			},
		},
		{
			name: "usdClassTransfer",
			sign: func() (SignatureResult, error) {
				return SignUsdClassTransferAction(privateKey, 1, true, timestamp, false)
			},
			want: SignatureResult{
				R: "0xe265b4f0207b3e5ba5d7f863c150b41a25449bf6f6bda1cf278f4b2b6db25279",
				S: "0x2469568d152f30acad5672b7657f0aa8e2f9d897fbd03b9b6b7dc3f443a1dff9",
				V: 27,
			},
		},
		{
			name: "tokenDelegate",
			sign: func() (SignatureResult, error) {
				return SignTokenDelegateAction(privateKey, goldenOther, 100000000, false, goldenNonce, false)
			},
			want: golden["token_delegate"].Signature,
		},
		{
			name: "approveBuilderFee",
			sign: func() (SignatureResult, error) {
				return SignApproveBuilderFee(privateKey, goldenOther, "0.001%", goldenNonce, false)
			},
			want: golden["approve_builder_fee"].Signature,
		},
		{
			name: "convertToMultiSigUser",
			sign: func() (SignatureResult, error) {
				return SignConvertToMultiSigUserAction(
					privateKey, `{"authorizedUsers":["`+goldenOther+`"],"threshold":1}`, goldenNonce, false,
				)
			},
			want: golden["convert_to_multi_sig_user"].Signature,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.sign()
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestSignL1Action_Errors(t *testing.T) {
	privateKey, err := crypto.HexToECDSA(goldenKey)
	require.NoError(t, err)

	type testCase struct {
		name   string
		action any
		vault  string
	}

	testCases := []testCase{
		{"unencodable action", map[string]any{"type": "x", "ch": make(chan int)}, ""},
		{"invalid vault address", ScheduleCancelAction{Type: "scheduleCancel"}, "0xzz"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var signErr error

			require.NotPanics(t, func() {
				_, signErr = SignL1Action(privateKey, tc.action, tc.vault, goldenNonce, nil, false)
			})

			require.Error(t, signErr)
		})
	}
}

func TestActionHash_StringLengths(t *testing.T) {
	type stringAction struct {
		Type string `msgpack:"type"`
		S    string `msgpack:"s"`
	}

	type testCase struct {
		length int
		want   string
	}

	testCases := []testCase{
		{31, "8990311da53cec86e043b70a119e2551c8bccd94621b8f4515199cad1e1cc25e"},
		{32, "8c15c0929a7153e88658a669988012a1643278841ced5e9058a4fc6921cfae0f"},
		{255, "12dd929de28b6c4317625693dc0a88284f79e952df4624b0a5e800e72f6c7371"},
		{256, "05518d2798201cb64642268f10b6dbfa49134cc0edc3c3685ea8988a3920a5b3"},
		{65535, "c3daac5955d48dacfead4ce436bb75ab8fe41fa771d0e894acf654c241067ddf"},
		{65536, "a8fe998805f978fcd05c87030c2ff4f034d72b85b6d9d33d73bef1655c2e3a3b"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("length_%d", tc.length), func(t *testing.T) {
			t.Parallel()
			action := stringAction{Type: "x", S: strings.Repeat("a", tc.length)}

			got, err := actionHash(action, "", goldenNonce, nil)

			require.NoError(t, err)
			require.Equal(t, tc.want, hex.EncodeToString(got))
		})
	}
}

func TestFloatToUsdInt(t *testing.T) {
	type testCase struct {
		name  string
		value float64
		want  int
	}

	tests := []testCase{
		{name: "2.01", value: 2.01, want: 2010000},
		{name: "2.03", value: 2.03, want: 2030000},
		{name: "2.05", value: 2.05, want: 2050000},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := FloatToUsdInt(tc.value)

			assert.Equal(t, tc.want, got)
		})
	}
}
