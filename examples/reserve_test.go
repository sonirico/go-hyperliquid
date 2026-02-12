package examples

import (
	"context"
	"fmt"
	"os"
	"testing"
)

func TestReserve(t *testing.T) {
	_ = loadEnvClean()
	exchange := newTestExchange(t)

	// Skip if running in CI or without proper credentials
	if os.Getenv("HL_PRIVATE_KEY") == "" {
		t.Skip("skipping test: HL_PRIVATE_KEY not set")
	}

	weight := 1000 // Adds 1000 extra limits, using 0.0005$ per weight from perps USDC balance

	result, err := exchange.Reserve(context.TODO(), weight)
	fmt.Println(result)
	if err != nil {
		t.Fatalf("Reserve failed: %v", err)
	}
	if !result.Ok() {
		t.Fatalf("Reserve returned non-ok status: %s, error: %s", result.Status, result.Error())
	}
}

func TestReserve_InvalidWeight(t *testing.T) {
	_ = loadEnvClean()
	exchange := newTestExchange(t)

	// Skip if running in CI or without proper credentials
	if os.Getenv("HL_PRIVATE_KEY") == "" {
		t.Skip("skipping test: HL_PRIVATE_KEY not set")
	}

	// Test with zero weight (should fail validation)
	_, err := exchange.Reserve(context.TODO(), 0)
	if err == nil {
		t.Error("Expected an error for weight = 0, but got none")
	}

	// Test with negative weight (should fail validation)
	_, err = exchange.Reserve(context.TODO(), -5)
	if err == nil {
		t.Error("Expected an error for weight = -5, but got none")
	}
}
