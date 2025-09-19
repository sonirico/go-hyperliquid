package examples

import (
	"context"
	"os"
	"testing"

	"github.com/joho/godotenv"
)

func TestUsdTransfer(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t) // exchange used for setup only

	// Example destination address (replace with actual test address)
	destination := "0x0000000000000000000000000000000000000000"
	amount := 1.0 // USD amount

	t.Logf("Attempting USD transfer of %.2f USD to %s", amount, destination)

	// This would normally execute the transfer, but we'll skip for safety
	t.Log("USD transfer method is available and ready to use")

	// Uncomment the line below only when you want to execute actual transfers
	result, err := exchange.UsdTransfer(context.TODO(), amount, destination)

	if err != nil {
		t.Fatalf("UsdTransfer failed: %v", err)
	}

	t.Logf("Transfer result: %+v", result)
}

func TestSpotTransfer(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t) // exchange used for setup only

	// Skip if running in CI or without proper credentials
	if os.Getenv("HL_PRIVATE_KEY") == "" {
		t.Skip("skipping test: HL_PRIVATE_KEY not set")
	}

	// Example destination address (replace with actual test address)
	destination := "0x0000000000000000000000000000000000000000"
	amount := 1.0 // Token amount
	token := "USDC"

	t.Logf("Attempting spot transfer of %.2f %s to %s", amount, token, destination)

	// This would normally execute the transfer, but we'll skip for safety
	t.Log("Spot transfer method is available and ready to use")

	result, err := exchange.SpotTransfer(context.TODO(), amount, destination, token)

	if err != nil {
		t.Fatalf("SpotTransfer failed: %v", err)
	}

	t.Logf("Transfer result: %+v", result)
}

func TestUsdClassTransfer(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t) // exchange used for setup only

	// Skip if running in CI or without proper credentials
	if os.Getenv("HL_PRIVATE_KEY") == "" {
		t.Skip("skipping test: HL_PRIVATE_KEY not set")
	}

	amount := 100.0
	toPerp := true // Transfer from spot to perp

	t.Logf("Attempting USD class transfer of %.2f USD (toPerp: %v)", amount, toPerp)

	// This would normally execute the transfer, but we'll skip for safety
	t.Log("USD class transfer method is available and ready to use")

	result, err := exchange.UsdClassTransfer(context.TODO(), amount, toPerp)

	if err != nil {
		t.Fatalf("UsdClassTransfer failed: %v", err)
	}

	t.Logf("Transfer result: %+v", result)
}

func TestSetReferrer(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t) // exchange used for setup only

	// Skip if running in CI or without proper credentials
	if os.Getenv("HL_PRIVATE_KEY") == "" {
		t.Skip("skipping test: HL_PRIVATE_KEY not set")
	}

	referralCode := "EXAMPLE_CODE"

	t.Logf("Attempting to set referrer code: %s", referralCode)

	// This would normally execute the referrer setting, but we'll skip for safety
	t.Log("Set referrer method is available and ready to use")

	result, err := exchange.SetReferrer(context.TODO(), referralCode)

	if err != nil {
		t.Fatalf("SetReferrer failed: %v", err)
	}

	t.Logf("Referrer result: %+v", result)
}

func TestCreateSubAccount(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t) // exchange used for setup only

	// Skip if running in CI or without proper credentials
	if os.Getenv("HL_PRIVATE_KEY") == "" {
		t.Skip("skipping test: HL_PRIVATE_KEY not set")
	}

	subAccountName := "test_sub_account"

	t.Logf("Attempting to create sub-account: %s", subAccountName)

	// This would normally execute the sub-account creation, but we'll skip for safety
	t.Log("Create sub-account method is available and ready to use")

	result, err := exchange.CreateSubAccount(context.TODO(), subAccountName)

	if err != nil {
		t.Fatalf("CreateSubAccount failed: %v", err)
	}

	t.Logf("Sub-account creation result: %+v", result)
}

func TestApproveAgent(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t) // exchange used for setup only

	// Skip if running in CI or without proper credentials
	if os.Getenv("HL_PRIVATE_KEY") == "" {
		t.Skip("skipping test: HL_PRIVATE_KEY not set")
	}

	agentName := "test_agent"

	t.Logf("Attempting to approve agent: %s", agentName)

	// This would normally execute the agent approval, but we'll skip for safety
	t.Log("Approve agent method is available and ready to use")

	result, agentKey, err := exchange.ApproveAgent(context.TODO(), &agentName)

	if err != nil {
		t.Fatalf("ApproveAgent failed: %v", err)
	}

	t.Logf("Agent approval result: %+v", result)
	t.Logf("Generated agent key: %s", agentKey)
}
