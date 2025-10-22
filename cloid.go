package hyperliquid

import (
	"encoding/hex"
	"fmt"
	"strings"
)

// normalizeCloid normalizes a client order ID to match Python SDK format.
// Python SDK uses cloid.to_raw() which returns hex WITH 0x prefix.
// The cloid is serialized in msgpack and JSON with the 0x prefix.
// This function:
// - Adds 0x prefix if not present
// - Validates it's exactly 32 hex characters (16 bytes) excluding prefix
// - Returns error if format is invalid
func normalizeCloid(cloid *string) (*string, error) {
	if cloid == nil || *cloid == "" {
		return nil, nil
	}

	cloidValue := *cloid
	// Add 0x prefix if not present
	if !strings.HasPrefix(cloidValue, "0x") {
		cloidValue = "0x" + cloidValue
	}

	// Validate it's exactly 34 characters (0x + 32 hex chars = 16 bytes)
	if len(cloidValue) != 34 {
		return nil, fmt.Errorf(
			"cloid must be exactly 32 hex characters (got %d excluding 0x prefix): %s",
			len(cloidValue)-2,
			cloidValue,
		)
	}

	// Verify the hex part (excluding 0x) is valid hex
	if _, err := hex.DecodeString(cloidValue[2:]); err != nil {
		return nil, fmt.Errorf("cloid must be valid hex string: %w", err)
	}

	// Return normalized value WITH 0x prefix to match Python SDK
	return &cloidValue, nil
}
