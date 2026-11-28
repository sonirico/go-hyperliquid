package hyperliquid

import (
	"context"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/stretchr/testify/require"
)

// newUserSignedTestServer fakes /exchange and records every request body it gets.
func newUserSignedTestServer(t *testing.T) (*httptest.Server, func() []map[string]any) {
	t.Helper()

	var mu sync.Mutex
	var bodies []map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/exchange", r.URL.Path)
		raw, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var body map[string]any
		require.NoError(t, json.Unmarshal(raw, &body))
		mu.Lock()
		bodies = append(bodies, body)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":"ok"}`)
	}))
	t.Cleanup(server.Close)

	return server, func() []map[string]any {
		mu.Lock()
		defer mu.Unlock()
		return append([]map[string]any(nil), bodies...)
	}
}

// recoverUserSignedSigner rebuilds the EIP-712 digest that Hyperliquid checks for a
// user-signed action and returns the address that produced the posted signature.
func recoverUserSignedSigner(
	t *testing.T,
	body map[string]any,
	primaryType string,
	payloadTypes []apitypes.Type,
) string {
	t.Helper()

	action := body["action"].(map[string]any)
	chainID := math.HexOrDecimal256(*big.NewInt(421614))
	typedData := apitypes.TypedData{
		Domain: apitypes.TypedDataDomain{
			ChainId:           &chainID,
			Name:              "HyperliquidSignTransaction",
			Version:           "1",
			VerifyingContract: "0x0000000000000000000000000000000000000000",
		},
		Types: apitypes.Types{
			primaryType: payloadTypes,
			"EIP712Domain": []apitypes.Type{
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
		},
		PrimaryType: primaryType,
		Message:     action,
	}
	domainSeparator, err := typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
	require.NoError(t, err)
	messageHash, err := hashStructLenient(typedData, primaryType, action)
	require.NoError(t, err)
	digest := crypto.Keccak256(
		append(append([]byte{0x19, 0x01}, domainSeparator...), messageHash...),
	)

	sig := body["signature"].(map[string]any)
	raw := make([]byte, 65)
	hexutil.MustDecodeBig(sig["r"].(string)).FillBytes(raw[:32])
	hexutil.MustDecodeBig(sig["s"].(string)).FillBytes(raw[32:64])
	raw[64] = byte(int(sig["v"].(float64)) - 27)
	pub, err := crypto.SigToPub(digest, raw)
	require.NoError(t, err)
	return crypto.PubkeyToAddress(*pub).Hex()
}

func TestUserSignedActions_SignedByTheWallet(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	signer := crypto.PubkeyToAddress(key.PublicKey).Hex()

	tests := []struct {
		name         string
		call         func(context.Context, *Exchange) error
		primaryType  string
		payloadTypes []apitypes.Type
	}{
		{
			name: "tokenDelegate",
			call: func(ctx context.Context, e *Exchange) error {
				_, err := e.TokenDelegate(
					ctx,
					"0x5ac99df645f3414876c816caa18b2d234024b487",
					100000000,
					false,
				)
				return err
			},
			primaryType: "HyperliquidTransaction:TokenDelegate",
			payloadTypes: []apitypes.Type{
				{Name: "hyperliquidChain", Type: "string"},
				{Name: "validator", Type: "address"},
				{Name: "wei", Type: "uint64"},
				{Name: "isUndelegate", Type: "bool"},
				{Name: "nonce", Type: "uint64"},
			},
		},
		{
			name: "convertToMultiSigUser",
			call: func(ctx context.Context, e *Exchange) error {
				_, err := e.ConvertToMultiSigUser(
					ctx,
					[]string{"0x0000000000000000000000000000000000000001"},
					1,
				)
				return err
			},
			primaryType: "HyperliquidTransaction:ConvertToMultiSigUser",
			payloadTypes: []apitypes.Type{
				{Name: "hyperliquidChain", Type: "string"},
				{Name: "signers", Type: "string"},
				{Name: "nonce", Type: "uint64"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, recorded := newUserSignedTestServer(t)
			exchange := NewExchange(
				context.Background(), key, server.URL, &Meta{}, "", "", &SpotMeta{}, &MixedArray{},
			)

			require.NoError(t, tt.call(context.Background(), exchange))

			bodies := recorded()
			require.Len(t, bodies, 1)
			action := bodies[0]["action"].(map[string]any)
			require.Equal(t, "0x66eee", action["signatureChainId"])
			require.Equal(t, "Testnet", action["hyperliquidChain"])
			require.Equal(
				t,
				signer,
				recoverUserSignedSigner(t, bodies[0], tt.primaryType, tt.payloadTypes),
			)
		})
	}
}

func TestUserSignedAmounts(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	type testCase struct {
		name string
		call func(ctx context.Context, e *Exchange, amount float64) error
	}

	type amountCase struct {
		amount float64
		wire   string
	}

	testCases := []testCase{
		{"UsdClassTransfer", func(ctx context.Context, e *Exchange, amount float64) error {
			_, err := e.UsdClassTransfer(ctx, amount, true)
			return err
		}},
		{"UsdTransfer", func(ctx context.Context, e *Exchange, amount float64) error {
			_, err := e.UsdTransfer(ctx, amount, goldenOther)
			return err
		}},
		{"SpotTransfer", func(ctx context.Context, e *Exchange, amount float64) error {
			_, err := e.SpotTransfer(ctx, amount, goldenOther, goldenToken)
			return err
		}},
		{"WithdrawFromBridge", func(ctx context.Context, e *Exchange, amount float64) error {
			_, err := e.WithdrawFromBridge(ctx, amount, goldenOther)
			return err
		}},
		{"SendAsset", func(ctx context.Context, e *Exchange, amount float64) error {
			_, err := e.SendAsset(ctx, goldenOther, "", "spot", goldenToken, amount)
			return err
		}},
		{"BorrowLend", func(ctx context.Context, e *Exchange, amount float64) error {
			_, err := e.BorrowLend(ctx, amount, "supply", 0)
			return err
		}},
	}

	amounts := []amountCase{
		{0.12345678, "0.12345678"},
		{0.0000005, "0.0000005"},
	}

	for _, tc := range testCases {
		for _, ac := range amounts {
			t.Run(tc.name+"/"+ac.wire, func(t *testing.T) {
				server, recorded := newUserSignedTestServer(t)
				exchange := NewExchange(
					context.Background(),
					key,
					server.URL,
					&Meta{},
					"",
					"",
					&SpotMeta{},
					&MixedArray{},
				)

				callErr := tc.call(context.Background(), exchange, ac.amount)

				require.NoError(t, callErr)
				bodies := recorded()
				require.Len(t, bodies, 1)
				action := bodies[0]["action"].(map[string]any)
				require.Equal(t, ac.wire, action["amount"])
			})
		}
	}
}

func TestSendToEVMWithData_JSON(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	server, recorded := newUserSignedTestServer(t)
	exchange := NewExchange(
		context.Background(), key, server.URL, &Meta{}, "", "", &SpotMeta{}, &MixedArray{},
	)

	_, callErr := exchange.SendToEVMWithData(
		context.Background(),
		goldenToken,
		"1.5",
		"spot",
		goldenOther,
		"hex",
		1337,
		100000,
		"0xdeadbeef",
	)

	require.NoError(t, callErr)
	bodies := recorded()
	require.Len(t, bodies, 1)
	action := bodies[0]["action"].(map[string]any)
	require.Equal(t, "0xdeadbeef", action["data"])
}
