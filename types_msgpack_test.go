package hyperliquid

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/vmihailenco/msgpack/v5"
)

func Test_Msgpack_Field_Ordering(t *testing.T) {
	// CRITICAL: This test verifies that OrderWire struct serializes in the correct order
	// to match Python SDK's msgpack output. Python preserves dict insertion order.

	// Python order for order_wire is: a, b, p, s, r, t (and optionally c)
	// See: hyperliquid-python-sdk/hyperliquid/utils/signing.py:order_request_to_order_wire

	orderTypeNew := OrderWireType{
		Limit: &OrderWireTypeLimit{
			Tif: TifGtc,
		},
	}

	// Test with current struct (must match Python order)
	newOrder := OrderWire{
		Asset:      0,
		IsBuy:      true,
		LimitPx:    "40000",
		Size:       "0.001",
		ReduceOnly: false,
		OrderType:  orderTypeNew,
		Cloid:      nil, // No cloid for this test
	}

	// Serialize with msgpack
	var bufNew bytes.Buffer
	encNew := msgpack.NewEncoder(&bufNew)
	// CRITICAL: Do NOT use SetSortMapKeys(true) - we need field order from struct
	encNew.UseCompactInts(true)

	err := encNew.Encode(newOrder)
	if err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}

	newBytes := bufNew.Bytes()
	goHex := hex.EncodeToString(newBytes)

	// Expected output from Python SDK for equivalent order_wire
	// Python code: {"a": 0, "b": True, "p": "40000", "s": "0.001", "r": False, "t": {"limit": {"tif": "Gtc"}}}
	// Verified with: python3 test_order_wire.py
	pythonExpectedHex := "86a16100a162c3a170a53430303030a173a5302e303031a172c2a17481a56c696d697481a3746966a3477463"

	t.Logf("Go msgpack output:     %s", goHex)
	t.Logf("Python expected:       %s", pythonExpectedHex)

	if goHex != pythonExpectedHex {
		t.Errorf(
			"Msgpack output does NOT match Python SDK!\nGot:      %s\nExpected: %s",
			goHex,
			pythonExpectedHex,
		)
	} else {
		t.Logf("✓ Msgpack field ordering matches Python SDK!")
	}
}
