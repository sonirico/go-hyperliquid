package hyperliquid

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
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

// convertStr16ToStr8 converts msgpack str16 (0xda + 2 byte length) to str8 (0xd9 + 1 byte length)
// for strings <256 bytes to match Python msgpack behavior
func convertStr16ToStr8(data []byte) []byte {
	result := make([]byte, 0, len(data))
	i := 0

	for i < len(data) {
		b := data[i]

		// Check if it's str16 (0xda)
		if b == 0xda && i+2 < len(data) {
			// Read 2-byte big-endian length
			length := (int(data[i+1]) << 8) | int(data[i+2])

			// If length fits in 1 byte, convert to str8 (0xd9)
			if length < 256 {
				result = append(result, 0xd9)
				result = append(result, byte(length))
				i += 3
				// Copy the string data
				if i+length <= len(data) {
					result = append(result, data[i:i+length]...)
					i += length
				}
				continue
			}
		}

		result = append(result, b)
		i++
	}

	return result
}

func actionHash(action any, vaultAddress string, nonce int64, expiresAfter *int64) []byte {
	var buf bytes.Buffer
	enc := msgpack.NewEncoder(&buf)
	// CRITICAL: Do NOT use SetSortMapKeys(true) - Python preserves insertion order
	// Structs in Go will serialize fields in the order they are defined
	enc.UseCompactInts(true)

	err := enc.Encode(action)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal action: %v", err))
	}
	data := buf.Bytes()

	// Convert fixstr to str8 for Python compatibility
	data = convertStr16ToStr8(data)

	// fmt.Printf("🔍 DEBUG actionHash msgpack: %s\n", hex.EncodeToString(data))

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
	//fmt.Printf("   Msgpack data: %s\n", hex.EncodeToString(data))
	//fmt.Printf("   Action hash: %s\n", hex.EncodeToString(hash))
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

func l1Payload(phantomAgent map[string]any, isMainnet bool) apitypes.TypedData {
	// Note: chainId is 1337 for both mainnet and testnet - it's just a signing domain identifier
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

// hashStructLenient is like HashStruct but ignores fields in message that are not in types
// This matches Python's eth_account behavior where extra fields in message are silently ignored
func hashStructLenient(
	typedData apitypes.TypedData,
	primaryType string,
	message map[string]any,
) ([]byte, error) {
	types := typedData.Types[primaryType]

	// Filter message to only include fields that exist in type definition
	filteredMessage := make(map[string]any)
	for _, t := range types {
		if val, ok := message[t.Name]; ok {
			filteredMessage[t.Name] = val
		}
	}

	// Now use standard HashStruct with filtered message
	return typedData.HashStruct(primaryType, filteredMessage)
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

	// Use lenient hashing to allow extra fields in message (Python compatibility)
	typedDataHash, err := hashStructLenient(typedData, typedData.PrimaryType, typedData.Message)
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

	// DEBUG: Verify signature recovery
	//pubKey, err := crypto.SigToPub(msgHash.Bytes(), signature)
	//if err == nil {
	//	recoveredAddr := crypto.PubkeyToAddress(*pubKey)
	//	expectedAddr := crypto.PubkeyToAddress(privateKey.PublicKey)
	//	fmt.Printf("   DEBUG SIGNATURE:\n")
	//	fmt.Printf("   Expected address: %s\n", expectedAddr.Hex())
	//	fmt.Printf("   Recovered address: %s\n", recoveredAddr.Hex())
	//	fmt.Printf("   Match: %v\n", recoveredAddr.Hex() == expectedAddr.Hex())
	//	fmt.Printf("   msgHash: %s\n", msgHash.Hex())
	//}

	return SignatureResult{
		R: hexutil.EncodeBig(r),
		S: hexutil.EncodeBig(s),
		V: v,
	}, nil
}

// structToOrderedMap converts a struct to a map preserving JSON tag order
// This is needed for EIP-712 signing where field order matters
func structToOrderedMap(v any) (map[string]any, error) {
	// First marshal to JSON (preserves struct field order)
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal struct: %w", err)
	}

	// Then unmarshal to map
	var result map[string]any
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to map: %w", err)
	}

	return result, nil
}

// SignUserSignedAction signs actions that require direct EIP-712 signing
// (e.g., approveAgent, approveBuilderFee, convertToMultiSigUser)
//
// IMPORTANT: The message will contain MORE fields than declared in payloadTypes to avoid the error
// "422 Failed to deserialize the JSON body" and "User or API Wallet 0x123... does not exist".
// This matches Python SDK behavior where the field order doesn't matter and extra fields (type, signatureChainId)
// are present in the message but ignored during EIP-712 hashing via hashStructLenient.
func SignUserSignedAction(
	privateKey *ecdsa.PrivateKey,
	action map[string]any,
	payloadTypes []apitypes.Type,
	primaryType string,
	isMainnet bool,
) (SignatureResult, error) {
	// Add signatureChainId based on environment
	// signatureChainId is the chain used by the wallet to sign.
	// hyperliquidChain determines the environment and prevents replay attacks.
	action["signatureChainId"] = "0x66eee"
	action["hyperliquidChain"] = "Mainnet"
	if !isMainnet {
		action["hyperliquidChain"] = "Testnet"
	}

	// Create typed data
	// Note: chainId is hardcoded to 421614 just like the Python SDK
	chainId := math.HexOrDecimal256(*big.NewInt(421614))
	typedData := apitypes.TypedData{
		Domain: apitypes.TypedDataDomain{
			ChainId:           &chainId,
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

	// signInner uses hashStructLenient which filters message to only include
	// fields declared in payloadTypes, matching Python eth_account behavior
	return signInner(privateKey, typedData)
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
	//fmt.Printf("[DEBUG] SignL1Action - ActionHash: %x\n", hash)

	// Step 2: Construct phantom agent
	phantomAgent := constructPhantomAgent(hash, isMainnet)

	// Step 3: Create l1 payload
	typedData := l1Payload(phantomAgent, isMainnet)

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

// signApproveAgentAction is the struct used for signing agent approval
// It has both JSON tags (for EIP-712) and msgpack tags (for request body)
type signApproveAgentAction struct {
	Type             string `json:"type"             msgpack:"type"`
	HyperliquidChain string `json:"hyperliquidChain" msgpack:"hyperliquidChain"`
	AgentAddress     string `json:"agentAddress"     msgpack:"agentAddress"`
	AgentName        string `json:"agentName"        msgpack:"agentName"`
	Nonce            int64  `json:"nonce"            msgpack:"nonce"`
}

// SignAgent signs agent approval action using EIP-712 direct signing
func SignAgent(
	privateKey *ecdsa.PrivateKey,
	agentAddress, agentName string,
	nonce int64,
	isMainnet bool,
) (SignatureResult, error) {
	action := map[string]any{
		"type":         "approveAgent",
		"agentAddress": agentAddress,
		"agentName":    agentName,
		"nonce":        nonce,
	}

	// payload_types from Python: only declares fields that are in the original action
	// signatureChainId and hyperliquidChain are added by SignUserSignedAction
	// but they're NOT declared in payloadTypes (they're added to message dynamically)
	payloadTypes := []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "agentAddress", Type: "address"},
		{Name: "agentName", Type: "string"},
		{Name: "nonce", Type: "uint64"},
	}

	return SignUserSignedAction(
		privateKey,
		action,
		payloadTypes,
		"HyperliquidTransaction:ApproveAgent",
		isMainnet,
	)
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
