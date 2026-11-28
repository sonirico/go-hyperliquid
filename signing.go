package hyperliquid

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/vmihailenco/msgpack/v5"
)

func addressToBytes(address string) ([]byte, error) {
	address = strings.TrimPrefix(address, "0x")
	bytes, err := hex.DecodeString(address)
	if err != nil {
		return nil, fmt.Errorf("invalid address %q: %w", address, err)
	}
	return bytes, nil
}

func actionHash(action any, vaultAddress string, nonce int64, expiresAfter *int64) ([]byte, error) {
	var buf bytes.Buffer
	enc := msgpack.NewEncoder(&buf)
	// CRITICAL: Do NOT use SetSortMapKeys(true) - Python preserves insertion order
	// Structs in Go will serialize fields in the order they are defined
	enc.UseCompactInts(true)

	err := enc.Encode(action)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal action: %w", err)
	}
	data := buf.Bytes()

	// fmt.Printf("🔍 DEBUG actionHash msgpack: %s\n", hex.EncodeToString(data))

	// Add nonce as 8 bytes big endian
	if nonce < 0 {
		return nil, fmt.Errorf("nonce cannot be negative: %d", nonce)
	}
	nonceBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(nonceBytes, uint64(nonce))
	data = append(data, nonceBytes...)

	// Add vault address
	if vaultAddress == "" {
		data = append(data, 0x00)
	} else {
		data = append(data, 0x01)
		vaultBytes, err := addressToBytes(vaultAddress)
		if err != nil {
			return nil, err
		}
		data = append(data, vaultBytes...)
	}

	// Add expires_after if provided
	if expiresAfter != nil {
		if *expiresAfter < 0 {
			return nil, fmt.Errorf("expiresAfter cannot be negative: %d", *expiresAfter)
		}
		data = append(data, 0x00)
		expiresAfterBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(expiresAfterBytes, uint64(*expiresAfter))
		data = append(data, expiresAfterBytes...)
	}

	// Return keccak256 hash
	hash := crypto.Keccak256(data)
	// fmt.Printf("   Msgpack data: %s\n", hex.EncodeToString(data))
	// fmt.Printf("   Action hash: %s\n", hex.EncodeToString(hash))
	return hash, nil
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
	R string `json:"r" msgpack:"r"`
	S string `json:"s" msgpack:"s"`
	V int    `json:"v" msgpack:"v"`
}

// L1ActionSigner signs L1 actions (msgpack + phantom agent EIP-712).
// When nil on Exchange, the default ECDSA implementation is used.
type L1ActionSigner interface {
	SignL1Action(
		ctx context.Context,
		action any,
		vaultAddress string,
		timestamp int64,
		expiresAfter *int64,
		isMainnet bool,
	) (SignatureResult, error)
}

// UserSignedActionSigner signs direct EIP-712 user-signed actions.
// When nil on Exchange, the default ECDSA implementation is used.
type UserSignedActionSigner interface {
	SignUserSignedAction(
		ctx context.Context,
		action map[string]any,
		payloadTypes []apitypes.Type,
		primaryType string,
		isMainnet bool,
	) (SignatureResult, error)
}

// AgentSigner signs agent approval actions.
// When nil on Exchange, the default ECDSA implementation is used.
type AgentSigner interface {
	SignAgent(
		ctx context.Context,
		agentAddress, agentName string,
		nonce int64,
		isMainnet bool,
	) (SignatureResult, error)
}

// ECDSAL1Signer implements L1ActionSigner using an ECDSA private key.
func ECDSAL1Signer(pk *ecdsa.PrivateKey) L1ActionSigner {
	return &ecdsaL1Signer{pk: pk}
}

type ecdsaL1Signer struct{ pk *ecdsa.PrivateKey }

func (s *ecdsaL1Signer) SignL1Action(
	_ context.Context,
	action any,
	vaultAddress string,
	timestamp int64,
	expiresAfter *int64,
	isMainnet bool,
) (SignatureResult, error) {
	return SignL1Action(s.pk, action, vaultAddress, timestamp, expiresAfter, isMainnet)
}

// ECDSAUserSignedSigner implements UserSignedActionSigner using an ECDSA private key.
func ECDSAUserSignedSigner(pk *ecdsa.PrivateKey) UserSignedActionSigner {
	return &ecdsaUserSignedSigner{pk: pk}
}

type ecdsaUserSignedSigner struct{ pk *ecdsa.PrivateKey }

func (s *ecdsaUserSignedSigner) SignUserSignedAction(
	_ context.Context,
	action map[string]any,
	payloadTypes []apitypes.Type,
	primaryType string,
	isMainnet bool,
) (SignatureResult, error) {
	return SignUserSignedAction(s.pk, action, payloadTypes, primaryType, isMainnet)
}

// ECDSAAgentSigner implements AgentSigner using an ECDSA private key.
func ECDSAAgentSigner(pk *ecdsa.PrivateKey) AgentSigner {
	return &ecdsaAgentSigner{pk: pk}
}

type ecdsaAgentSigner struct{ pk *ecdsa.PrivateKey }

func (s *ecdsaAgentSigner) SignAgent(
	_ context.Context,
	agentAddress, agentName string,
	nonce int64,
	isMainnet bool,
) (SignatureResult, error) {
	return SignAgent(s.pk, agentAddress, agentName, nonce, isMainnet)
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
	// Also convert numeric types to ensure proper type handling for EIP-712
	filteredMessage := make(map[string]any)
	for _, t := range types {
		if val, ok := message[t.Name]; ok {
			// Convert numeric types to ensure proper type handling for EIP-712
			// apitypes.HashStruct expects specific types based on the type declaration
			switch t.Type {
			case "uint64":
				var uintVal uint64
				switch v := val.(type) {
				case uint64:
					uintVal = v
				case int64:
					if v < 0 {
						return nil, fmt.Errorf("cannot convert negative int64 %d to uint64", v)
					}
					uintVal = uint64(v)
				case float64:
					// JSON unmarshaling can convert numbers to float64
					if v < 0 || v > float64(^uint64(0)) || v != float64(uint64(v)) {
						return nil, fmt.Errorf("invalid float64 value %f for uint64", v)
					}
					uintVal = uint64(v)
				case int:
					if v < 0 {
						return nil, fmt.Errorf("cannot convert negative int %d to uint64", v)
					}
					uintVal = uint64(v)
				case json.Number:
					// Handle json.Number type
					parsed, err := strconv.ParseUint(string(v), 10, 64)
					if err != nil {
						return nil, fmt.Errorf(
							"failed to parse json.Number %s to uint64 for %s: %w",
							v,
							t.Name,
							err,
						)
					}
					uintVal = parsed
				case string:
					// Try to parse as string representation of uint64
					parsed, err := strconv.ParseUint(v, 10, 64)
					if err != nil {
						return nil, fmt.Errorf(
							"failed to parse string %s to uint64 for %s: %w",
							v,
							t.Name,
							err,
						)
					}
					uintVal = parsed
				default:
					// Try to convert via json marshal/unmarshal to handle edge cases
					jsonBytes, err := json.Marshal(v)
					if err != nil {
						return nil, fmt.Errorf("failed to marshal value for %s: %w", t.Name, err)
					}
					if err := json.Unmarshal(jsonBytes, &uintVal); err != nil {
						return nil, fmt.Errorf(
							"failed to convert value to uint64 for %s: %w",
							t.Name,
							err,
						)
					}
				}
				// apitypes.HashStruct may not handle uint64 directly from map[string]any
				// Convert to *big.Int which is commonly used for EIP-712 uint types
				filteredMessage[t.Name] = new(big.Int).SetUint64(uintVal)
			default:
				filteredMessage[t.Name] = val
			}
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
	// pubKey, err := crypto.SigToPub(msgHash.Bytes(), signature)
	// if err == nil {
	// 	recoveredAddr := crypto.PubkeyToAddress(*pubKey)
	// 	expectedAddr := crypto.PubkeyToAddress(privateKey.PublicKey)
	// 	fmt.Printf("   DEBUG SIGNATURE:\n")
	// 	fmt.Printf("   Expected address: %s\n", expectedAddr.Hex())
	// 	fmt.Printf("   Recovered address: %s\n", recoveredAddr.Hex())
	// 	fmt.Printf("   Match: %v\n", recoveredAddr.Hex() == expectedAddr.Hex())
	// 	fmt.Printf("   msgHash: %s\n", msgHash.Hex())
	//}

	return SignatureResult{
		R: hexutil.EncodeBig(r),
		S: hexutil.EncodeBig(s),
		V: v,
	}, nil
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
	hash, err := actionHash(action, vaultAddress, timestamp, expiresAfter)
	if err != nil {
		return SignatureResult{}, err
	}
	// fmt.Printf("[DEBUG] SignL1Action - ActionHash: %x\n", hash)

	// Step 2: Construct phantom agent
	phantomAgent := constructPhantomAgent(hash, isMainnet)

	// Step 3: Create l1 payload
	typedData := l1Payload(phantomAgent, isMainnet)

	// Step 4: Sign using EIP-712
	return signInner(privateKey, typedData)
}

// SignUsdClassTransferAction signs a usdClassTransfer action.
//
// usdClassTransfer is a user-signed action: it is signed as EIP-712
// HyperliquidTransaction:UsdClassTransfer, not as an L1 action. timestamp is
// the action nonce. The action posted with this signature must carry the
// amount formatted by floatToWire (e.g. "1", "12.5").
func SignUsdClassTransferAction(
	privateKey *ecdsa.PrivateKey,
	amount float64,
	toPerp bool,
	timestamp int64,
	isMainnet bool,
) (SignatureResult, error) {
	amountWire, err := floatToWire(amount)
	if err != nil {
		return SignatureResult{}, err
	}

	action := map[string]any{
		"type":   "usdClassTransfer",
		"amount": amountWire,
		"toPerp": toPerp,
		"nonce":  big.NewInt(timestamp),
	}

	payloadTypes := []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "amount", Type: "string"},
		{Name: "toPerp", Type: "bool"},
		{Name: "nonce", Type: "uint64"},
	}

	return SignUserSignedAction(
		privateKey,
		action,
		payloadTypes,
		"HyperliquidTransaction:UsdClassTransfer",
		isMainnet,
	)
}

// SignSpotTransferAction signs a spotSend action.
//
// spotSend is a user-signed action: it is signed as EIP-712
// HyperliquidTransaction:SpotSend, not as an L1 action. timestamp is the
// action's time field. The action posted with this signature must carry the
// amount formatted by floatToWire.
func SignSpotTransferAction(
	privateKey *ecdsa.PrivateKey,
	amount float64,
	destination, token string,
	timestamp int64,
	isMainnet bool,
) (SignatureResult, error) {
	amountWire, err := floatToWire(amount)
	if err != nil {
		return SignatureResult{}, err
	}

	action := map[string]any{
		"type":        "spotSend",
		"destination": destination,
		"token":       token,
		"amount":      amountWire,
		"time":        big.NewInt(timestamp),
	}

	payloadTypes := []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "destination", Type: "string"},
		{Name: "token", Type: "string"},
		{Name: "amount", Type: "string"},
		{Name: "time", Type: "uint64"},
	}

	return SignUserSignedAction(
		privateKey,
		action,
		payloadTypes,
		"HyperliquidTransaction:SpotSend",
		isMainnet,
	)
}

// SignUsdTransferAction signs a usdSend action.
//
// usdSend is a user-signed action: it is signed as EIP-712
// HyperliquidTransaction:UsdSend, not as an L1 action. timestamp is the
// action's time field. The action posted with this signature must carry the
// amount formatted by floatToWire.
func SignUsdTransferAction(
	privateKey *ecdsa.PrivateKey,
	amount float64,
	destination string,
	timestamp int64,
	isMainnet bool,
) (SignatureResult, error) {
	amountWire, err := floatToWire(amount)
	if err != nil {
		return SignatureResult{}, err
	}

	action := map[string]any{
		"type":        "usdSend",
		"destination": destination,
		"amount":      amountWire,
		"time":        big.NewInt(timestamp),
	}

	payloadTypes := []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "destination", Type: "string"},
		{Name: "amount", Type: "string"},
		{Name: "time", Type: "uint64"},
	}

	return SignUserSignedAction(
		privateKey,
		action,
		payloadTypes,
		"HyperliquidTransaction:UsdSend",
		isMainnet,
	)
}

// SignTokenDelegateAction signs a tokenDelegate action.
//
// tokenDelegate is a user-signed action: it is signed as EIP-712
// HyperliquidTransaction:TokenDelegate, not as an L1 action. nonce is the
// action nonce.
func SignTokenDelegateAction(
	privateKey *ecdsa.PrivateKey,
	validator string,
	wei uint64,
	isUndelegate bool,
	nonce int64,
	isMainnet bool,
) (SignatureResult, error) {
	action := map[string]any{
		"type":         "tokenDelegate",
		"validator":    validator,
		"wei":          wei,
		"isUndelegate": isUndelegate,
		"nonce":        big.NewInt(nonce),
	}

	payloadTypes := []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "validator", Type: "address"},
		{Name: "wei", Type: "uint64"},
		{Name: "isUndelegate", Type: "bool"},
		{Name: "nonce", Type: "uint64"},
	}

	return SignUserSignedAction(
		privateKey,
		action,
		payloadTypes,
		"HyperliquidTransaction:TokenDelegate",
		isMainnet,
	)
}

// SignWithdrawFromBridgeAction signs a withdraw3 action.
//
// withdraw3 is a user-signed action: it is signed as EIP-712
// HyperliquidTransaction:Withdraw, not as an L1 action. timestamp is the
// action's time field. The action posted with this signature must carry the
// amount formatted by floatToWire.
//
// fee is ignored because withdraw3 has no fee field. The parameter is kept
// for compatibility.
func SignWithdrawFromBridgeAction(
	privateKey *ecdsa.PrivateKey,
	destination string,
	amount, fee float64,
	timestamp int64,
	isMainnet bool,
) (SignatureResult, error) {
	amountWire, err := floatToWire(amount)
	if err != nil {
		return SignatureResult{}, err
	}

	action := map[string]any{
		"type":        "withdraw3",
		"destination": destination,
		"amount":      amountWire,
		"time":        big.NewInt(timestamp),
	}

	payloadTypes := []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "destination", Type: "string"},
		{Name: "amount", Type: "string"},
		{Name: "time", Type: "uint64"},
	}

	return SignUserSignedAction(
		privateKey,
		action,
		payloadTypes,
		"HyperliquidTransaction:Withdraw",
		isMainnet,
	)
}

// SignAgent signs agent approval action using EIP-712 direct signing
func SignAgent(
	privateKey *ecdsa.PrivateKey,
	agentAddress, agentName string,
	nonce int64,
	isMainnet bool,
) (SignatureResult, error) {
	// The nonce must be non-negative
	if nonce < 0 {
		return SignatureResult{}, fmt.Errorf("nonce cannot be negative: %d", nonce)
	}

	// Use int64 in the action map - apitypes will handle the conversion to uint64
	// based on the type declaration in payloadTypes
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

// SignApproveBuilderFee signs an approveBuilderFee action.
//
// approveBuilderFee is a user-signed action: it is signed as EIP-712
// HyperliquidTransaction:ApproveBuilderFee, not as an L1 action. nonce is the
// action nonce.
func SignApproveBuilderFee(
	privateKey *ecdsa.PrivateKey,
	builder string,
	maxFeeRate string,
	nonce int64,
	isMainnet bool,
) (SignatureResult, error) {
	action := map[string]any{
		"type":       "approveBuilderFee",
		"maxFeeRate": maxFeeRate,
		"builder":    builder,
		"nonce":      big.NewInt(nonce),
	}

	payloadTypes := []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "maxFeeRate", Type: "string"},
		{Name: "builder", Type: "address"},
		{Name: "nonce", Type: "uint64"},
	}

	return SignUserSignedAction(
		privateKey,
		action,
		payloadTypes,
		"HyperliquidTransaction:ApproveBuilderFee",
		isMainnet,
	)
}

// SignConvertToMultiSigUserAction signs a convertToMultiSigUser action.
//
// convertToMultiSigUser is a user-signed action: it is signed as EIP-712
// HyperliquidTransaction:ConvertToMultiSigUser, not as an L1 action. nonce is
// the action nonce and signersJSON the compact JSON the action carries.
func SignConvertToMultiSigUserAction(
	privateKey *ecdsa.PrivateKey,
	signersJSON string,
	nonce int64,
	isMainnet bool,
) (SignatureResult, error) {
	action := map[string]any{
		"type":    "convertToMultiSigUser",
		"signers": signersJSON,
		"nonce":   big.NewInt(nonce),
	}

	payloadTypes := []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "signers", Type: "string"},
		{Name: "nonce", Type: "uint64"},
	}

	return SignUserSignedAction(
		privateKey,
		action,
		payloadTypes,
		"HyperliquidTransaction:ConvertToMultiSigUser",
		isMainnet,
	)
}

var multiSigEnvelopeTypes = []apitypes.Type{
	{Name: "hyperliquidChain", Type: "string"},
	{Name: "multiSigActionHash", Type: "bytes32"},
	{Name: "nonce", Type: "uint64"},
}

func newMultiSigEnvelope(
	action MultiSigAction,
	vaultAddress string,
	nonce int64,
	expiresAfter *int64,
) (map[string]any, error) {
	hash, err := actionHash(action, vaultAddress, nonce, expiresAfter)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"multiSigActionHash": hash,
		"nonce":              big.NewInt(nonce),
	}, nil
}

// SignMultiSigAction signs multi-signature action as the outer signer
func SignMultiSigAction(
	privateKey *ecdsa.PrivateKey,
	action MultiSigAction,
	vaultAddress string,
	nonce int64,
	expiresAfter *int64,
	isMainnet bool,
) (SignatureResult, error) {
	envelope, err := newMultiSigEnvelope(action, vaultAddress, nonce, expiresAfter)
	if err != nil {
		return SignatureResult{}, err
	}

	return SignUserSignedAction(
		privateKey,
		envelope,
		multiSigEnvelopeTypes,
		"HyperliquidTransaction:SendMultiSig",
		isMainnet,
	)
}

// SignMultiSigL1ActionPayload signs an L1 action as one of the signers of a multi-sig user
func SignMultiSigL1ActionPayload(
	privateKey *ecdsa.PrivateKey,
	action any,
	vaultAddress string,
	nonce int64,
	expiresAfter *int64,
	multiSigUser string,
	outerSigner string,
	isMainnet bool,
) (SignatureResult, error) {
	envelope := []any{strings.ToLower(multiSigUser), strings.ToLower(outerSigner), action}

	return SignL1Action(privateKey, envelope, vaultAddress, nonce, expiresAfter, isMainnet)
}

// SignMultiSigUserSignedActionPayload signs a user-signed action as one of the signers of a multi-sig user
func SignMultiSigUserSignedActionPayload(
	privateKey *ecdsa.PrivateKey,
	action map[string]any,
	payloadTypes []apitypes.Type,
	primaryType string,
	multiSigUser string,
	outerSigner string,
	isMainnet bool,
) (SignatureResult, error) {
	envelope := make(map[string]any, len(action)+2)
	for key, value := range action {
		envelope[key] = value
	}
	envelope["payloadMultiSigUser"] = strings.ToLower(multiSigUser)
	envelope["outerSigner"] = strings.ToLower(outerSigner)

	envelopeTypes := make([]apitypes.Type, 0, len(payloadTypes)+2)
	for _, payloadType := range payloadTypes {
		envelopeTypes = append(envelopeTypes, payloadType)
		if payloadType.Name == "hyperliquidChain" {
			envelopeTypes = append(
				envelopeTypes,
				apitypes.Type{Name: "payloadMultiSigUser", Type: "address"},
				apitypes.Type{Name: "outerSigner", Type: "address"},
			)
		}
	}

	return SignUserSignedAction(privateKey, envelope, envelopeTypes, primaryType, isMainnet)
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
