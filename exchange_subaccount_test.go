package hyperliquid

import (
	"context"
	"encoding/hex"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestSubAccountSpotTransfer_SendsAmountAsString(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	server, recorded := newUserSignedTestServer(t)
	exchange := NewExchange(
		context.Background(), key, server.URL, &Meta{}, "", "", &SpotMeta{}, &MixedArray{},
	)

	_, err = exchange.SubAccountSpotTransfer(
		context.Background(),
		"0x1d9470d4b963f552e6f671a81619d395877bf409",
		true,
		"PURR:0xc4bf3f870c0e9465323c0b6ed28096c2",
		1.5,
	)
	require.NoError(t, err)

	bodies := recorded()
	require.Len(t, bodies, 1)
	action := bodies[0]["action"].(map[string]any)
	require.Equal(t, "subAccountSpotTransfer", action["type"])
	require.Equal(t, "1.5", action["amount"])
}

// The expected hash comes from the Python SDK:
//
//	action_hash({"type": "subAccountSpotTransfer",
//	             "subAccountUser": "0x1d9470d4b963f552e6f671a81619d395877bf409",
//	             "isDeposit": True,
//	             "token": "PURR:0xc4bf3f870c0e9465323c0b6ed28096c2",
//	             "amount": str(1.5)}, None, 1700000000000, None)
func TestSubAccountSpotTransfer_ActionHashMatchesPythonSDK(t *testing.T) {
	action := SubAccountSpotTransferAction{
		Type:           "subAccountSpotTransfer",
		SubAccountUser: "0x1d9470d4b963f552e6f671a81619d395877bf409",
		IsDeposit:      true,
		Token:          "PURR:0xc4bf3f870c0e9465323c0b6ed28096c2",
		Amount:         "1.5",
	}

	hash := actionHash(action, "", 1700000000000, nil)
	require.Equal(
		t,
		"56e7f6cf4678cb307604a37ad02c187f33c67b0d4fc80e2d1b0e42784d18ce24",
		hex.EncodeToString(hash),
	)
}
