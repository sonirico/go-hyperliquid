package hyperliquid

import (
	"encoding/json"
	"testing"
)

// The exchange validates the action hash over the msgpack-encoded action, and
// the reference Python SDK encodes ntli as a signed integer in micro-USD with
// isBuy always true (hyperliquid-python-sdk signing.float_to_usd_int). This
// pins the wire shape so a regression back to absolute-value floats fails CI.
func TestUpdateIsolatedMarginActionWireShape(t *testing.T) {
	action := UpdateIsolatedMarginAction{
		Type:  "updateIsolatedMargin",
		Asset: 7,
		IsBuy: true,
		Ntli:  -2_500_000, // remove $2.50
	}
	raw, err := json.Marshal(action)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"type":"updateIsolatedMargin","asset":7,"isBuy":true,"ntli":-2500000}`
	if string(raw) != want {
		t.Fatalf("wire mismatch:\n got: %s\nwant: %s", raw, want)
	}
}
