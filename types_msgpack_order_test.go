package hyperliquid

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/vmihailenco/msgpack/v5"
)

// TestMsgpackOrderSerialization verifies msgpack serialization matches Python SDK
func TestMsgpackOrderSerialization(t *testing.T) {
	// Test OrderWire serialization using real structs
	orderWire := OrderWire{
		Asset:      0,
		IsBuy:      true,
		LimitPx:    "40000",
		Size:       "0.001",
		ReduceOnly: false,
		OrderType: OrderWireType{
			Limit: &OrderWireTypeLimit{
				Tif: TifGtc,
			},
		},
		Cloid: nil, // No cloid for this test
	}

	action := OrderAction{
		Type:     "order",
		Orders:   []OrderWire{orderWire},
		Grouping: "na",
		Builder:  nil,
	}

	var buf bytes.Buffer
	enc := msgpack.NewEncoder(&buf)
	// CRITICAL: Do NOT use SetSortMapKeys - structs preserve field order
	enc.UseCompactInts(true)

	if err := enc.Encode(action); err != nil {
		t.Fatalf("ERROR encoding: %v", err)
	}

	goHex := hex.EncodeToString(buf.Bytes())
	t.Logf("=== GO MSGPACK HEX ===\n%s", goHex)
	t.Logf("=== GO MSGPACK BYTES LENGTH: %d ===", buf.Len())

	// Python expected output (from test_python_msgpack.py)
	pythonHex := "83a474797065a56f72646572a66f72646572739186a16100a162c3a170a53430303030a173a5302e303031a172c2a17481a56c696d697481a3746966a3477463a867726f7570696e67a26e61"

	if goHex != pythonHex {
		t.Errorf(
			"Msgpack serialization does NOT match Python SDK!\nGo:     %s\nPython: %s",
			goHex,
			pythonHex,
		)
	} else {
		t.Logf("✓ Msgpack serialization MATCHES Python SDK!")
	}
}
