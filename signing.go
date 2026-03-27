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

func addressToBytes(address string) []byte {
	address = strings.TrimPrefix(address, "0x")
	bytes, _ := hex.DecodeString(address)
	return bytes
}

// convertStr16ToStr8 converts msgpack str16 (0xda + 2 byte length) to str8 (0xd9 + 1 byte length)
// for strings <256 bytes to match Python msgpack behavior.
// Uses a structure-aware msgpack walker to avoid corrupting non-string data
// that happens to contain 0xda as a data byte (e.g. inside uint64 values).
func convertStr16ToStr8(data []byte) []byte {
	result := make([]byte, 0, len(data))
	pos := 0
	for pos < len(data) {
		consumed := walkMsgpackValue(data, pos, &result)
		if consumed <= 0 {
			// Malformed data: copy remaining bytes as-is (fail-safe)
			result = append(result, data[pos:]...)
			break
		}
		pos += consumed
	}
	return result
}

// walkMsgpackValue parses one msgpack value at data[pos], appends the
// (possibly converted) bytes to *result, and returns the number of bytes
// consumed from data. Returns 0 if the data is truncated/malformed.
func walkMsgpackValue(data []byte, pos int, result *[]byte) int {
	if pos >= len(data) {
		return 0
	}
	b := data[pos]
	remaining := len(data) - pos

	// --- Fixed-length single-byte types ---
	// positive fixint 0x00-0x7f, negative fixint 0xe0-0xff, nil 0xc0, never used 0xc1, bool 0xc2-0xc3
	if b <= 0x7f || b >= 0xe0 || (b >= 0xc0 && b <= 0xc3) {
		*result = append(*result, b)
		return 1
	}

	// --- fixstr (0xa0-0xbf): 1 header + N data bytes ---
	if b >= 0xa0 && b <= 0xbf {
		n := int(b & 0x1f)
		total := 1 + n
		if remaining < total {
			return 0
		}
		*result = append(*result, data[pos:pos+total]...)
		return total
	}

	// --- fixmap (0x80-0x8f): N key-value pairs ---
	if b >= 0x80 && b <= 0x8f {
		count := int(b & 0x0f)
		*result = append(*result, b)
		consumed := 1
		for i := 0; i < count*2; i++ {
			c := walkMsgpackValue(data, pos+consumed, result)
			if c <= 0 {
				return 0
			}
			consumed += c
		}
		return consumed
	}

	// --- fixarray (0x90-0x9f): N elements ---
	if b >= 0x90 && b <= 0x9f {
		count := int(b & 0x0f)
		*result = append(*result, b)
		consumed := 1
		for i := 0; i < count; i++ {
			c := walkMsgpackValue(data, pos+consumed, result)
			if c <= 0 {
				return 0
			}
			consumed += c
		}
		return consumed
	}

	switch b {
	// --- float32, float64 ---
	case 0xca: // float32: 1+4
		return copyFixed(data, pos, 5, result)
	case 0xcb: // float64: 1+8
		return copyFixed(data, pos, 9, result)

	// --- unsigned integers ---
	case 0xcc: // uint8: 1+1
		return copyFixed(data, pos, 2, result)
	case 0xcd: // uint16: 1+2
		return copyFixed(data, pos, 3, result)
	case 0xce: // uint32: 1+4
		return copyFixed(data, pos, 5, result)
	case 0xcf: // uint64: 1+8
		return copyFixed(data, pos, 9, result)

	// --- signed integers ---
	case 0xd0: // int8: 1+1
		return copyFixed(data, pos, 2, result)
	case 0xd1: // int16: 1+2
		return copyFixed(data, pos, 3, result)
	case 0xd2: // int32: 1+4
		return copyFixed(data, pos, 5, result)
	case 0xd3: // int64: 1+8
		return copyFixed(data, pos, 9, result)

	// --- fixext 1/2/4/8/16 ---
	case 0xd4: // fixext1: 1+1+1
		return copyFixed(data, pos, 3, result)
	case 0xd5: // fixext2: 1+1+2
		return copyFixed(data, pos, 4, result)
	case 0xd6: // fixext4: 1+1+4
		return copyFixed(data, pos, 6, result)
	case 0xd7: // fixext8: 1+1+8
		return copyFixed(data, pos, 10, result)
	case 0xd8: // fixext16: 1+1+16
		return copyFixed(data, pos, 18, result)

	// --- bin 8/16/32 ---
	case 0xc4: // bin8: 1 + 1-byte len + data
		return copyVarLen(data, pos, 1, result)
	case 0xc5: // bin16: 1 + 2-byte len + data
		return copyVarLen(data, pos, 2, result)
	case 0xc6: // bin32: 1 + 4-byte len + data
		return copyVarLen(data, pos, 4, result)

	// --- ext 8/16/32 ---
	case 0xc7: // ext8: 1 + 1-byte len + 1 type + data
		return copyExtVarLen(data, pos, 1, result)
	case 0xc8: // ext16: 1 + 2-byte len + 1 type + data
		return copyExtVarLen(data, pos, 2, result)
	case 0xc9: // ext32: 1 + 4-byte len + 1 type + data
		return copyExtVarLen(data, pos, 4, result)

	// --- str8 (0xd9): already compact, just copy ---
	case 0xd9: // str8: 1 + 1-byte len + data
		return copyVarLen(data, pos, 1, result)

	// --- str16 (0xda): THE conversion target ---
	case 0xda: // str16: 1 + 2-byte len + data
		if remaining < 3 {
			return 0
		}
		length := (int(data[pos+1]) << 8) | int(data[pos+2])
		total := 3 + length
		if remaining < total {
			return 0
		}
		if length < 256 {
			*result = append(*result, 0xd9)
			*result = append(*result, byte(length)) // #nosec G115 -- length is guaranteed < 256 by the if-guard above
			*result = append(*result, data[pos+3:pos+total]...)
		} else {
			*result = append(*result, data[pos:pos+total]...)
		}
		return total

	// --- str32 (0xdb): just copy ---
	case 0xdb: // str32: 1 + 4-byte len + data
		return copyVarLen(data, pos, 4, result)

	// --- array16/32 ---
	case 0xdc: // array16: 1 + 2-byte count
		if remaining < 3 {
			return 0
		}
		count := (int(data[pos+1]) << 8) | int(data[pos+2])
		*result = append(*result, data[pos:pos+3]...)
		consumed := 3
		for i := 0; i < count; i++ {
			c := walkMsgpackValue(data, pos+consumed, result)
			if c <= 0 {
				return 0
			}
			consumed += c
		}
		return consumed
	case 0xdd: // array32: 1 + 4-byte count
		if remaining < 5 {
			return 0
		}
		count := (int(data[pos+1]) << 24) | (int(data[pos+2]) << 16) | (int(data[pos+3]) << 8) | int(data[pos+4])
		*result = append(*result, data[pos:pos+5]...)
		consumed := 5
		for i := 0; i < count; i++ {
			c := walkMsgpackValue(data, pos+consumed, result)
			if c <= 0 {
				return 0
			}
			consumed += c
		}
		return consumed

	// --- map16/32 ---
	case 0xde: // map16: 1 + 2-byte count
		if remaining < 3 {
			return 0
		}
		count := (int(data[pos+1]) << 8) | int(data[pos+2])
		*result = append(*result, data[pos:pos+3]...)
		consumed := 3
		for i := 0; i < count*2; i++ {
			c := walkMsgpackValue(data, pos+consumed, result)
			if c <= 0 {
				return 0
			}
			consumed += c
		}
		return consumed
	case 0xdf: // map32: 1 + 4-byte count
		if remaining < 5 {
			return 0
		}
		count := (int(data[pos+1]) << 24) | (int(data[pos+2]) << 16) | (int(data[pos+3]) << 8) | int(data[pos+4])
		*result = append(*result, data[pos:pos+5]...)
		consumed := 5
		for i := 0; i < count*2; i++ {
			c := walkMsgpackValue(data, pos+consumed, result)
			if c <= 0 {
				return 0
			}
			consumed += c
		}
		return consumed

	default:
		// Unknown type: copy single byte as fail-safe
		*result = append(*result, b)
		return 1
	}
}

// copyFixed copies exactly `size` bytes from data[pos:] to result.
// Returns 0 if data is truncated.
func copyFixed(data []byte, pos, size int, result *[]byte) int {
	if len(data)-pos < size {
		return 0
	}
	*result = append(*result, data[pos:pos+size]...)
	return size
}

// copyVarLen handles msgpack types with a variable-length data payload:
// header(1) + length(lenBytes) + data(length). Copies as-is.
func copyVarLen(data []byte, pos, lenBytes int, result *[]byte) int {
	headerSize := 1 + lenBytes
	if len(data)-pos < headerSize {
		return 0
	}
	length := readLen(data, pos+1, lenBytes)
	total := headerSize + length
	if len(data)-pos < total {
		return 0
	}
	*result = append(*result, data[pos:pos+total]...)
	return total
}

// copyExtVarLen handles ext types: header(1) + length(lenBytes) + type(1) + data(length).
func copyExtVarLen(data []byte, pos, lenBytes int, result *[]byte) int {
	headerSize := 1 + lenBytes + 1 // format + len + type byte
	if len(data)-pos < headerSize {
		return 0
	}
	length := readLen(data, pos+1, lenBytes)
	total := headerSize + length
	if len(data)-pos < total {
		return 0
	}
	*result = append(*result, data[pos:pos+total]...)
	return total
}

// readLen reads a big-endian unsigned integer of 1, 2, or 4 bytes.
func readLen(data []byte, pos, size int) int {
	switch size {
	case 1:
		return int(data[pos])
	case 2:
		return (int(data[pos]) << 8) | int(data[pos+1])
	case 4:
		return (int(data[pos]) << 24) | (int(data[pos+1]) << 16) | (int(data[pos+2]) << 8) | int(data[pos+3])
	default:
		return 0
	}
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
	// fmt.Printf("   Msgpack data: %s\n", hex.EncodeToString(data))
	// fmt.Printf("   Action hash: %s\n", hex.EncodeToString(hash))
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
	hash := actionHash(action, vaultAddress, timestamp, expiresAfter)
	// fmt.Printf("[DEBUG] SignL1Action - ActionHash: %x\n", hash)

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
