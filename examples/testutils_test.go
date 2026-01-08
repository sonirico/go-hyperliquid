package examples

import (
	"context"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/sonirico/go-hyperliquid"
)

func newTestInfo(t *testing.T) *hyperliquid.Info {
	t.Helper()

	apiURL := os.Getenv("HL_API_URL")

	// Default to testnet if not specified
	if apiURL == "" {
		apiURL = hyperliquid.TestnetAPIURL
	}

	t.Logf("API URL: %s", apiURL)

	// Initialize test info
	return hyperliquid.NewInfo(
		context.TODO(),
		apiURL,
		true,
		nil,
		nil,
		nil,
		hyperliquid.InfoOptDebugMode(),
	)
}

func newTestExchange(t *testing.T) *hyperliquid.Exchange {
	t.Helper()

	privKeyHex := os.Getenv("HL_PRIVATE_KEY")
	vaultAddr := os.Getenv("HL_VAULT_ADDRESS")
	walletAddr := os.Getenv("HL_WALLET_ADDRESS")
	apiURL := os.Getenv("HL_API_URL")

	// Default to testnet if not specified
	if apiURL == "" {
		apiURL = hyperliquid.TestnetAPIURL
	}

	// t.Logf("private key: %s", privKeyHex)
	// t.Logf("vault address: %s", vaultAddr)
	// t.Logf("wallet address: %s", walletAddr)
	t.Logf("API URL: %s", apiURL)

	testPrivateKey, err := crypto.HexToECDSA(privKeyHex)

	//if err == nil {
	//	// Verify what address this private key generates
	//	pubKey := testPrivateKey.Public()
	//	publicKeyECDSA, ok := pubKey.(*ecdsa.PublicKey)
	//	if ok {
	//		address := crypto.PubkeyToAddress(*publicKeyECDSA)
	//		t.Logf("🔑 PRIVATE KEY GENERATES ADDRESS: %s", address.Hex())
	//		t.Logf("📝 EXPECTED WALLET ADDRESS:       %s", walletAddr)
	//		if address.Hex() != walletAddr {
	//			t.Errorf(
	//				"⚠️  ADDRESS MISMATCH! Private key generates %s but expected %s",
	//				address.Hex(),
	//				walletAddr,
	//			)
	//		}
	//	}
	//}

	if err != nil {
		t.Fatalf("Failed to create test private key: %v", err)
	}

	// Initialize test exchange
	return hyperliquid.NewExchange(
		context.TODO(),
		testPrivateKey,
		apiURL,
		nil,
		vaultAddr,
		walletAddr,
		nil,
		nil,
		hyperliquid.ExchangeOptDebugMode(),
	)
}
