package examples

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInfo_UserState(t *testing.T) {
	_ = loadEnvClean()
	// Use private key from environment
	exchange := newTestExchange(t)
	address := os.Getenv("HL_VAULT_ADDRESS")
	// log.Printf("Using vault address: %s", address)
	// log.Printf("using private key: %s", os.Getenv("HL_PRIVATE_KEY"))

	resp, err := exchange.Info().UserState(context.TODO(), address)
	require.NoError(t, err)
	log.Printf("info response: %+v", resp)
}

func TestInfo_HistoricalOrders(t *testing.T) {
	_ = loadEnvClean()
	info := newTestInfo(t)
	address := os.Getenv("HL_WALLET_ADDRESS")

	resp, err := info.HistoricalOrders(context.TODO(), address)
	require.NoError(t, err)
	log.Printf("info response: %+v", resp)
}
