package hyperliquid

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/vmihailenco/msgpack/v5"
)

func addressToBytes(address string) []byte {
	address = strings.TrimPrefix(address, "0x")
	bytes, _ := hex.DecodeString(address)
	return bytes
}

func actionHash(action any, vaultAddress string, nonce int64, expiresAfter *int64) []byte {
	var buf bytes.Buffer
	enc := msgpack.NewEncoder(&buf)
	enc.SetSortMapKeys(true)
	enc.UseCompactInts(true)

	err := enc.Encode(action)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal action: %v", err))
	}
	data := buf.Bytes()

	// Add nonce as 8 bytes big endian
	if nonce < 0 {
		panic(fmt.Sprintf("nonce cannot be negative: %d", nonce))
	}
	nonceBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(nonceBytes, uint64(nonce))
	data = append(data, nonceBytes...)

	// Add vault address
	if vaultAddress == "" {
		data = append(data, 0x00)
	} else {
		data = append(data, 0x01)
		data = append(data, addressToBytes(vaultAddress)...)
	}

	// Add expires_after if provided
	if expiresAfter != nil {
		if *expiresAfter < 0 {
			panic(fmt.Sprintf("expiresAfter cannot be negative: %d", *expiresAfter))
		}
		data = append(data, 0x00)
		expiresAfterBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(expiresAfterBytes, uint64(*expiresAfter))
		data = append(data, expiresAfterBytes...)
	}

	// Return keccak256 hash
	hash := crypto.Keccak256(data)
	// fmt.Printf("go action hash: %s\n", hex.EncodeToString(hash))
	return hash
}

func constructPhantomAgent(hash []byte, isMainnet bool) map[string]any {
	source := "b" // testnet
	if isMainnet {
		source = "a" // mainnet
	}
	return map[string]any{
		"source":       source,
		"connectionId": hash,
	}
}

func l1Payload(phantomAgent map[string]any) apitypes.TypedData {
	chainId := math.HexOrDecimal256(*big.NewInt(1337))
	return apitypes.TypedData{
		Domain: apitypes.TypedDataDomain{
			ChainId:           &chainId,
			Name:              "Exchange",
			Version:           "1",
			VerifyingContract: "0x0000000000000000000000000000000000000000",
		},
		Types: apitypes.Types{
			"Agent": []apitypes.Type{
				{Name: "source", Type: "string"},
				{Name: "connectionId", Type: "bytes32"},
			},
			"EIP712Domain": []apitypes.Type{
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
		},
		PrimaryType: "Agent",
		Message:     phantomAgent,
	}
}

// SignatureResult represents the structured signature result
type SignatureResult struct {
	R string `json:"r"`
	S string `json:"s"`
	V int    `json:"v"`
}

func signInner(
	privateKey *ecdsa.PrivateKey,
	typedData apitypes.TypedData,
) (SignatureResult, error) {
	// Create EIP-712 hash
	domainSeparator, err := typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
	if err != nil {
		return SignatureResult{}, fmt.Errorf("failed to hash domain: %w", err)
	}

	typedDataHash, err := typedData.HashStruct(typedData.PrimaryType, typedData.Message)
	if err != nil {
		return SignatureResult{}, fmt.Errorf("failed to hash typed data: %w", err)
	}

	rawData := []byte{0x19, 0x01}
	rawData = append(rawData, domainSeparator...)
	rawData = append(rawData, typedDataHash...)
	msgHash := crypto.Keccak256Hash(rawData)

	signature, err := crypto.Sign(msgHash.Bytes(), privateKey)
	if err != nil {
		return SignatureResult{}, fmt.Errorf("failed to sign message: %w", err)
	}

	// Extract r, s, v components
	r := new(big.Int).SetBytes(signature[:32])
	s := new(big.Int).SetBytes(signature[32:64])
	v := int(signature[64]) + 27

	return SignatureResult{
		R: hexutil.EncodeBig(r),
		S: hexutil.EncodeBig(s),
		V: v,
	}, nil
}

func SignL1Action(
	privateKey *ecdsa.PrivateKey,
	action any,
	vaultAddress string,
	timestamp int64,
	expiresAfter *int64,
	isMainnet bool,
) (SignatureResult, error) {
	// Step 1: Create action hash
	hash := actionHash(action, vaultAddress, timestamp, expiresAfter)

	// Step 2: Construct phantom agent
	phantomAgent := constructPhantomAgent(hash, isMainnet)

	// Step 3: Create l1 payload
	typedData := l1Payload(phantomAgent)

	// Step 4: Sign using EIP-712
	return signInner(privateKey, typedData)
}

type signUsdClassTransferAction struct {
	Type   string  `msgpack:"type"`
	Amount float64 `msgpack:"amount"`
	ToPerp bool    `msgpack:"toPerp"`
}

// SignUsdClassTransferAction signs USD class transfer action
func SignUsdClassTransferAction(
	privateKey *ecdsa.PrivateKey,
	amount float64,
	toPerp bool,
	timestamp int64,
	isMainnet bool,
) (SignatureResult, error) {
	action := signUsdClassTransferAction{
		Type:   "usdClassTransfer",
		Amount: amount,
		ToPerp: toPerp,
	}

	return SignL1Action(privateKey, action, "", timestamp, nil, isMainnet)
}

type signSpotTransferAction struct {
	Type        string  `msgpack:"type"`
	Amount      float64 `msgpack:"amount"`
	Destination string  `msgpack:"destination"`
	Token       string  `msgpack:"token"`
}

// SignSpotTransferAction signs spot transfer action
func SignSpotTransferAction(
	privateKey *ecdsa.PrivateKey,
	amount float64,
	destination, token string,
	timestamp int64,
	isMainnet bool,
) (SignatureResult, error) {
	action := signSpotTransferAction{
		Type:        "spotTransfer",
		Amount:      amount,
		Destination: destination,
		Token:       token,
	}

	return SignL1Action(privateKey, action, "", timestamp, nil, isMainnet)
}

type signUsdTransferAction struct {
	Type        string  `msgpack:"type"`
	Amount      float64 `msgpack:"amount"`
	Destination string  `msgpack:"destination"`
}

// SignUsdTransferAction signs USD transfer action
func SignUsdTransferAction(
	privateKey *ecdsa.PrivateKey,
	amount float64,
	destination string,
	timestamp int64,
	isMainnet bool,
) (SignatureResult, error) {
	action := signUsdTransferAction{
		Type:        "usdTransfer",
		Amount:      amount,
		Destination: destination,
	}

	return SignL1Action(privateKey, action, "", timestamp, nil, isMainnet)
}

type signPerpDexClassTransferAction struct {
	Type   string  `msgpack:"type"`
	Dex    string  `msgpack:"dex"`
	Token  string  `msgpack:"token"`
	Amount float64 `msgpack:"amount"`
	ToPerp bool    `msgpack:"toPerp"`
}

// SignPerpDexClassTransferAction signs perp dex class transfer action
func SignPerpDexClassTransferAction(
	privateKey *ecdsa.PrivateKey,
	dex, token string,
	amount float64,
	toPerp bool,
	timestamp int64,
	isMainnet bool,
) (SignatureResult, error) {
	action := signPerpDexClassTransferAction{
		Type:   "perpDexClassTransfer",
		Dex:    dex,
		Token:  token,
		Amount: amount,
		ToPerp: toPerp,
	}

	return SignL1Action(privateKey, action, "", timestamp, nil, isMainnet)
}

type signTokenDelegateAction struct {
	Type             string  `msgpack:"type"`
	Token            string  `msgpack:"token"`
	Amount           float64 `msgpack:"amount"`
	ValidatorAddress string  `msgpack:"validatorAddress"`
}

// SignTokenDelegateAction signs token delegate action
func SignTokenDelegateAction(
	privateKey *ecdsa.PrivateKey,
	token string,
	amount float64,
	validatorAddress string,
	timestamp int64,
	isMainnet bool,
) (SignatureResult, error) {
	action := signTokenDelegateAction{
		Type:             "tokenDelegate",
		Token:            token,
		Amount:           amount,
		ValidatorAddress: validatorAddress,
	}

	return SignL1Action(privateKey, action, "", timestamp, nil, isMainnet)
}

type signWithdrawFromBridgeAction struct {
	Type        string  `msgpack:"type"`
	Destination string  `msgpack:"destination"`
	Amount      float64 `msgpack:"amount"`
	Fee         float64 `msgpack:"fee"`
}

// SignWithdrawFromBridgeAction signs withdraw from bridge action
func SignWithdrawFromBridgeAction(
	privateKey *ecdsa.PrivateKey,
	destination string,
	amount, fee float64,
	timestamp int64,
	isMainnet bool,
) (SignatureResult, error) {
	action := signWithdrawFromBridgeAction{
		Type:        "withdrawFromBridge",
		Destination: destination,
		Amount:      amount,
		Fee:         fee,
	}

	return SignL1Action(privateKey, action, "", timestamp, nil, isMainnet)
}

type signAgent struct {
	Type         string `msgpack:"type"`
	AgentAddress string `msgpack:"agentAddress"`
	AgentName    string `msgpack:"agentName"`
}

// SignAgent signs agent approval action
func SignAgent(
	privateKey *ecdsa.PrivateKey,
	agentAddress, agentName string,
	timestamp int64,
	isMainnet bool,
) (SignatureResult, error) {
	action := signAgent{
		Type:         "approveAgent",
		AgentAddress: agentAddress,
		AgentName:    agentName,
	}

	return SignL1Action(privateKey, action, "", timestamp, nil, isMainnet)
}

type signApproveBuilderFee struct {
	Type string `msgpack:"type"`
	// BuilderAddress is the address of the builder
	BuilderAddress string `msgpack:"builderAddress"`
	// MaxFeeRate is the maximum fee rate the user is willing to pay
	MaxFeeRate float64 `msgpack:"maxFeeRate"`
}

// SignApproveBuilderFee signs approve builder fee action
func SignApproveBuilderFee(
	privateKey *ecdsa.PrivateKey,
	builderAddress string,
	maxFeeRate float64,
	timestamp int64,
	isMainnet bool,
) (SignatureResult, error) {
	action := signApproveBuilderFee{
		Type:           "approveBuilderFee",
		BuilderAddress: builderAddress,
		MaxFeeRate:     maxFeeRate,
	}

	return SignL1Action(privateKey, action, "", timestamp, nil, isMainnet)
}

type signConvertToMultiSigUserAction struct {
	Type      string   `msgpack:"type"`
	Signers   []string `msgpack:"signers"`
	Threshold int      `msgpack:"threshold"`
}

// SignConvertToMultiSigUserAction signs convert to multi-sig user action
func SignConvertToMultiSigUserAction(
	privateKey *ecdsa.PrivateKey,
	signers []string,
	threshold int,
	timestamp int64,
	isMainnet bool,
) (SignatureResult, error) {
	action := signConvertToMultiSigUserAction{
		Type:      "convertToMultiSigUser",
		Signers:   signers,
		Threshold: threshold,
	}

	return SignL1Action(privateKey, action, "", timestamp, nil, isMainnet)
}

type signMultiSigAction struct {
	Type       string         `msgpack:"type"`
	Action     map[string]any `msgpack:"action"`
	Signers    []string       `msgpack:"signers"`
	Signatures []string       `msgpack:"signatures"`
}

// SignMultiSigAction signs multi-signature action
func SignMultiSigAction(
	privateKey *ecdsa.PrivateKey,
	innerAction map[string]any,
	signers []string,
	signatures []string,
	timestamp int64,
	isMainnet bool,
) (SignatureResult, error) {
	action := signMultiSigAction{
		Type:       "multiSig",
		Action:     innerAction,
		Signers:    signers,
		Signatures: signatures,
	}

	return SignL1Action(privateKey, action, "", timestamp, nil, isMainnet)
}

// FloatToUsdInt converts float to USD integer representation
func FloatToUsdInt(value float64) int {
	// Convert float USD to integer representation (assuming 6 decimals for USDC)
	return int(value * 1e6)
}

// GetTimestampMs returns current timestamp in milliseconds
func GetTimestampMs() int64 {
	return time.Now().UnixMilli()
}
