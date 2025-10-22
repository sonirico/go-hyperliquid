package examples

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
)

func TestInfo_UserState(t *testing.T) {
	_ = godotenv.Overload()
	// Use private key from environment
	exchange := newTestExchange(t)
	address := os.Getenv("HL_VAULT_ADDRESS")
	log.Printf("Using vault address: %s", address)
	log.Printf("using private key: %s", os.Getenv("HL_PRIVATE_KEY"))

	resp, err := exchange.Info().UserState(context.TODO(), address)
	require.NoError(t, err)
	log.Printf("info response: %+v", resp)
}
