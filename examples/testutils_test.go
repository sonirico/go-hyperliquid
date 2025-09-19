package examples

import (
	"context"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/sonirico/go-hyperliquid"
)

func newTestExchange(t *testing.T) *hyperliquid.Exchange {
	t.Helper()

	privKeyHex := os.Getenv("HL_PRIVATE_KEY")
	vaultAddr := os.Getenv("HL_VAULT_ADDRESS")
	testPrivateKey, err := crypto.HexToECDSA(privKeyHex)

	if err != nil {
		t.Fatalf("Failed to create test private key: %v", err)
	}

	// Initialize test exchange
	return hyperliquid.NewExchange(
		context.TODO(),
		testPrivateKey,
		hyperliquid.MainnetAPIURL,
		nil,
		vaultAddr,
		crypto.PubkeyToAddress(testPrivateKey.PublicKey).Hex(),
		nil,
		hyperliquid.ExchangeOptDebugMode(),
	)
}
