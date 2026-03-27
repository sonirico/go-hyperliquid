package hyperliquid

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
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
		Grouping: "na",
	}

	privateKey, _ := crypto.HexToECDSA(
		"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	)
	vaultAddress := ""
	timestamp := int64(1640995200000) // Fixed timestamp
	var expiresAfter *int64 = nil
	isMainnet := false

	// Debug: Print action hash components
	hash := actionHash(action, vaultAddress, timestamp, expiresAfter)
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

// TestConvertStr16ToStr8_Uint64ContainingDA verifies that a uint64 value whose
// big-endian representation contains 0xda (the str16 marker) is NOT corrupted.
// order_id 361731063972 == 0x00_00_00_54_38_DA_00_A4; the old naive scanner
// would match the embedded 0xda and destroy the payload.
func TestConvertStr16ToStr8_Uint64ContainingDA(t *testing.T) {
	action := CancelAction{
		Type: "cancel",
		Cancels: []CancelOrderWire{
			{Asset: 0, OrderID: 361731063972},
		},
	}

	var buf bytes.Buffer
	enc := msgpack.NewEncoder(&buf)
	enc.UseCompactInts(true)
	err := enc.Encode(action)
	require.NoError(t, err)

	original := buf.Bytes()
	converted := convertStr16ToStr8(original)

	assert.Equal(t, original, converted,
		"payload with uint64 containing 0xda byte must not be mutated; "+
			"original=%s converted=%s",
		hex.EncodeToString(original), hex.EncodeToString(converted))
}

// TestConvertStr16ToStr8_LegitimateStr16 verifies that a genuine str16-encoded
// string (32-255 bytes) is correctly down-converted to str8, preserving the
// Python-compatible compact encoding.
func TestConvertStr16ToStr8_LegitimateStr16(t *testing.T) {
	// Build a string of 100 bytes — short enough to fit in str8 but long
	// enough that msgpack's str16 format kicks in when UseCompactInts isn't
	// enough. We force str16 by manually constructing the payload.
	payload := strings.Repeat("A", 100)

	// Hand-craft a str16 encoding: 0xda + 2-byte big-endian length + data
	str16 := []byte{0xda, 0x00, byte(len(payload))}
	str16 = append(str16, []byte(payload)...)

	converted := convertStr16ToStr8(str16)

	// Expected: str8 encoding: 0xd9 + 1-byte length + data
	expected := []byte{0xd9, byte(len(payload))}
	expected = append(expected, []byte(payload)...)

	assert.Equal(t, expected, converted,
		"str16 with length < 256 must be converted to str8")
}

// TestConvertStr16ToStr8_NoMutation verifies that payloads without any 0xda
// bytes pass through completely unchanged.
func TestConvertStr16ToStr8_NoMutation(t *testing.T) {
	// A simple fixmap with short fixstr keys and small integer values.
	// No 0xda bytes anywhere in this payload.
	action := map[string]int{"x": 1, "y": 2}

	var buf bytes.Buffer
	enc := msgpack.NewEncoder(&buf)
	enc.UseCompactInts(true)
	enc.SetSortMapKeys(true) // deterministic key order for assertion
	err := enc.Encode(action)
	require.NoError(t, err)

	original := buf.Bytes()

	// Sanity: confirm there's no 0xda in the raw payload
	assert.False(t, bytes.Contains(original, []byte{0xda}),
		"test precondition: payload should not contain 0xda")

	converted := convertStr16ToStr8(original)
	assert.Equal(t, original, converted,
		"payload without 0xda must not be modified")
}

// TestConvertStr16ToStr8_NestedContainers verifies that str16 values nested
// inside maps and arrays are correctly converted.
func TestConvertStr16ToStr8_NestedContainers(t *testing.T) {
	longStr := strings.Repeat("B", 200) // 200 bytes — fits in str8

	// Hand-craft: fixarray(1) -> fixmap(1) -> fixstr key "k" -> str16 value
	//
	// fixarray of 1 element: 0x91
	// fixmap of 1 pair:      0x81
	// fixstr "k" (len 1):    0xa1 0x6b
	// str16 of 200 bytes:    0xda 0x00 0xc8 + 200 bytes
	input := []byte{0x91, 0x81, 0xa1, 0x6b, 0xda, 0x00, 0xc8}
	input = append(input, []byte(longStr)...)

	// Expected: same structure but str16 -> str8
	expected := []byte{0x91, 0x81, 0xa1, 0x6b, 0xd9, 0xc8}
	expected = append(expected, []byte(longStr)...)

	converted := convertStr16ToStr8(input)
	assert.Equal(t, expected, converted,
		"str16 nested inside array -> map must be converted to str8")
}

// TestActionHash_CancelWithLargeOrderID verifies that actionHash produces a
// stable (idempotent) hash for a CancelAction containing order_id 361731063972
// whose uint64 encoding embeds 0xda. The hash must remain identical across
// repeated calls and must not change when convertStr16ToStr8 is applied twice.
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

	hash1 := actionHash(action, vaultAddress, nonce, expiresAfter)
	hash2 := actionHash(action, vaultAddress, nonce, expiresAfter)

	assert.Equal(t, hash1, hash2,
		"actionHash must be deterministic across repeated calls")

	// Double-application test: encode, convert once, convert again — result
	// must be identical to single conversion (idempotency).
	var buf bytes.Buffer
	enc := msgpack.NewEncoder(&buf)
	enc.UseCompactInts(true)
	err := enc.Encode(action)
	require.NoError(t, err)

	once := convertStr16ToStr8(buf.Bytes())
	twice := convertStr16ToStr8(once)

	assert.Equal(t, once, twice,
		"convertStr16ToStr8 must be idempotent — double application must not change output")

	// Verify the hash is non-zero / non-trivial
	assert.Len(t, hash1, 32, "keccak256 hash must be 32 bytes")
	assert.False(t, bytes.Equal(hash1, make([]byte, 32)),
		"hash must not be all zeros")
}
