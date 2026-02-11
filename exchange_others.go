package hyperliquid

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

func (e *Exchange) UpdateLeverage(
	ctx context.Context,
	leverage int,
	name string,
	isCross bool,
) (*UserState, error) {
	asset, ok := e.info.CoinToAsset(name)
	if !ok {
		return nil, fmt.Errorf("coin %s not found in info", name)
	}

	action := UpdateLeverageAction{
		Type:     "updateLeverage",
		Asset:    asset,
		IsCross:  isCross,
		Leverage: leverage,
	}

	var result UserState
	if err := e.executeAction(ctx, action, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (e *Exchange) UpdateIsolatedMargin(
	ctx context.Context,
	amount float64,
	name string,
) (*UserState, error) {
	asset, ok := e.info.CoinToAsset(name)
	if !ok {
		return nil, fmt.Errorf("coin %s not found in info", name)
	}

	action := UpdateIsolatedMarginAction{
		Type:  "updateIsolatedMargin",
		Asset: asset,
		IsBuy: amount > 0,
		Ntli:  abs(amount),
	}

	var result UserState
	if err := e.executeAction(ctx, action, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SlippagePrice calculates the slippage price for market orders
func (e *Exchange) SlippagePrice(
	ctx context.Context,
	name string,
	isBuy bool,
	slippage float64,
	px *float64,
) (float64, error) {
	var price float64

	if px != nil {
		price = *px
	} else {
		// Get midprice
		mids, err := e.info.AllMids(ctx)
		if err != nil {
			return 0, err
		}
		if midPriceStr, exists := mids[name]; exists {
			price = parseFloat(midPriceStr)
		} else {
			return 0, fmt.Errorf("could not get mid price for coin: %s", name)
		}
	}

	asset := e.info.coinToAsset[name]
	isSpot := asset >= 10000

	// Calculate slippage
	if isBuy {
		price *= (1 + slippage)
	} else {
		price *= (1 - slippage)
	}

	// First we need to round the price to Hyperliquid's max 5 significant figures (see: https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api/tick-and-lot-size)
	price = roundToSignificantFigures(price, 5)

	// Round to appropriate decimals
	decimals := 6
	if isSpot {
		decimals = 8
	}
	szDecimals := e.info.assetToDecimal[asset]

	return roundToDecimals(price, decimals-szDecimals), nil
}

// ScheduleCancel schedules cancellation of all open orders
func (e *Exchange) ScheduleCancel(
	ctx context.Context,
	scheduleTime *int64,
) (*ScheduleCancelResponse, error) {
	nonce := e.nextNonce()

	action := ScheduleCancelAction{
		Type: "scheduleCancel",
		Time: scheduleTime,
	}

	sig, err := SignL1Action(
		e.privateKey,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce)
	if err != nil {
		return nil, err
	}

	var result ScheduleCancelResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Reserve reserves request weight capacity on the exchange.
// Each weight unit costs 0.0005 USDC from Perps balance.
// This increases address-based rate limits without requiring trading volume.
//
// Parameters:
//   - ctx: Context for the request
//   - weight: Number of weight units to reserve (must be > 0)
//
// Returns:
//   - *ReserveRequestWeightResponse: Response from the API
//   - error: nil on success, or error describing what went wrong
//
// Example:
//   resp, err := exchange.Reserve(ctx, 100) // Reserve 100 units (costs 0.05 USDC)
func (e *Exchange) Reserve(ctx context.Context, weight int) (*ReserveRequestWeightResponse, error) {
	// Validation
	if weight <= 0 {
		return nil, fmt.Errorf("weight must be greater than 0, got: %d", weight)
	}

	// Get nonce
	nonce := e.nextNonce()

	// Create action
	action := ReserveRequestWeightAction{
		Type:   "reserveRequestWeight",
		Weight: weight,
	}

	// DEBUG: Print action details
	fmt.Printf("\n=== DEBUG Reserve Action ===\n")
	fmt.Printf("Action Type: %s\n", action.Type)
	fmt.Printf("Weight: %d\n", action.Weight)
	fmt.Printf("Nonce: %d\n", nonce)
	fmt.Printf("Vault (for signing): %q\n", "")
	fmt.Printf("ExpiresAfter: %v\n", e.expiresAfter)
	fmt.Printf("IsMainnet: %v\n", e.client.baseURL == MainnetAPIURL)
	fmt.Printf("===========================\n\n")

	// Sign the action
	sig, err := SignL1Action(
		e.privateKey,
		action,
		"", // No vault address - reserve must be performed by main wallet
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign reserve action: %w", err)
	}

	// Post to exchange
	resp, err := e.postAction(ctx, action, sig, nonce)
	if err != nil {
		return nil, fmt.Errorf("failed to post reserve action: %w", err)
	}

	// Parse response
	var result ReserveRequestWeightResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse reserve response: %w", err)
	}

	return &result, nil
}

// SetReferrer sets a referral code
func (e *Exchange) SetReferrer(ctx context.Context, code string) (*SetReferrerResponse, error) {
	nonce := e.nextNonce()

	action := SetReferrerAction{
		Type: "setReferrer",
		Code: code,
	}

	sig, err := SignL1Action(
		e.privateKey,
		action,
		"", // No vault address for referrer
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce)
	if err != nil {
		return nil, err
	}

	var result SetReferrerResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateSubAccount creates a new sub-account
func (e *Exchange) CreateSubAccount(
	ctx context.Context,
	name string,
) (*CreateSubAccountResponse, error) {
	nonce := e.nextNonce()

	action := CreateSubAccountAction{
		Type: "createSubAccount",
		Name: name,
	}

	sig, err := SignL1Action(
		e.privateKey,
		action,
		"", // No vault address for sub-account creation
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce)
	if err != nil {
		return nil, err
	}

	var result CreateSubAccountResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UsdClassTransfer transfers between USD classes
func (e *Exchange) UsdClassTransfer(
	ctx context.Context,
	amount float64,
	toPerp bool,
) (*TransferResponse, error) {
	nonce := e.nextNonce()

	strAmount := formatFloat(amount)
	if e.vault != "" {
		strAmount += " subaccount:" + e.vault
	}

	action := map[string]any{
		"amount": strAmount,
		"toPerp": toPerp,
		"nonce":  big.NewInt(nonce),
		"type":   "usdClassTransfer",
	}

	payloadTypes := []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "amount", Type: "string"},
		{Name: "toPerp", Type: "bool"},
		{Name: "nonce", Type: "uint64"},
	}

	sig, err := SignUserSignedAction(
		e.privateKey,
		action,
		payloadTypes,
		"HyperliquidTransaction:UsdClassTransfer",
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce)
	if err != nil {
		return nil, err
	}

	var result TransferResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SubAccountTransfer transfers funds to/from sub-account
func (e *Exchange) SubAccountTransfer(
	ctx context.Context,
	subAccountUser string,
	isDeposit bool,
	usd int,
) (*TransferResponse, error) {
	nonce := e.nextNonce()

	action := SubAccountTransferAction{
		Type:           "subAccountTransfer",
		SubAccountUser: subAccountUser,
		IsDeposit:      isDeposit,
		Usd:            usd,
	}

	sig, err := SignL1Action(
		e.privateKey,
		action,
		"", // No vault address
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce)
	if err != nil {
		return nil, err
	}

	var result TransferResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// VaultUsdTransfer transfers to/from vault
func (e *Exchange) VaultUsdTransfer(
	ctx context.Context,
	vaultAddress string,
	isDeposit bool,
	usd int,
) (*TransferResponse, error) {
	nonce := e.nextNonce()

	action := VaultUsdTransferAction{
		Type:         "vaultTransfer",
		VaultAddress: vaultAddress,
		IsDeposit:    isDeposit,
		Usd:          usd,
	}

	sig, err := SignL1Action(
		e.privateKey,
		action,
		"", // No vault address
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce)
	if err != nil {
		return nil, err
	}

	var result TransferResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateVault creates a new vault
func (e *Exchange) CreateVault(
	ctx context.Context,
	name string,
	description string,
	initialUsd int,
) (*CreateVaultResponse, error) {
	nonce := e.nextNonce()

	action := CreateVaultAction{
		Type:        "createVault",
		Name:        name,
		Description: description,
		InitialUsd:  initialUsd,
	}

	sig, err := SignL1Action(
		e.privateKey,
		action,
		"", // No vault address
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce)
	if err != nil {
		return nil, err
	}

	var result CreateVaultResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (e *Exchange) VaultModify(
	ctx context.Context,
	vaultAddress string,
	allowDeposits bool,
	alwaysCloseOnWithdraw bool,
) (*TransferResponse, error) {
	nonce := e.nextNonce()

	action := VaultModifyAction{
		Type:                  "vaultModify",
		VaultAddress:          vaultAddress,
		AllowDeposits:         allowDeposits,
		AlwaysCloseOnWithdraw: alwaysCloseOnWithdraw,
	}

	sig, err := SignL1Action(
		e.privateKey,
		action,
		"", // No vault address
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce)
	if err != nil {
		return nil, err
	}

	var result TransferResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (e *Exchange) VaultDistribute(
	ctx context.Context,
	vaultAddress string,
	usd int,
) (*TransferResponse, error) {
	nonce := e.nextNonce()

	action := VaultDistributeAction{
		Type:         "vaultDistribute",
		VaultAddress: vaultAddress,
		Usd:          usd,
	}

	sig, err := SignL1Action(
		e.privateKey,
		action,
		"", // No vault address
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce)
	if err != nil {
		return nil, err
	}

	var result TransferResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UsdTransfer transfers USD to another address
func (e *Exchange) UsdTransfer(
	ctx context.Context,
	amount float64,
	destination string,
) (*TransferResponse, error) {
	nonce := e.nextNonce()

	action := map[string]any{
		"destination": destination,
		"amount":      formatFloat(amount),
		"time":        big.NewInt(nonce),
		"type":        "usdSend",
	}

	payloadTypes := []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "destination", Type: "string"},
		{Name: "amount", Type: "string"},
		{Name: "time", Type: "uint64"},
	}

	sig, err := SignUserSignedAction(
		e.privateKey,
		action,
		payloadTypes,
		"HyperliquidTransaction:UsdSend",
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce)
	if err != nil {
		return nil, err
	}

	var result TransferResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SpotTransfer transfers spot tokens to another address
func (e *Exchange) SpotTransfer(
	ctx context.Context,
	amount float64,
	destination, token string,
) (*TransferResponse, error) {
	nonce := e.nextNonce()

	action := map[string]any{
		"destination": destination,
		"amount":      formatFloat(amount),
		"token":       token,
		"time":        big.NewInt(nonce),
		"type":        "spotSend",
	}

	payloadTypes := []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "destination", Type: "string"},
		{Name: "token", Type: "string"},
		{Name: "amount", Type: "string"},
		{Name: "time", Type: "uint64"},
	}

	sig, err := SignUserSignedAction(
		e.privateKey,
		action,
		payloadTypes,
		"HyperliquidTransaction:SpotSend",
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce)
	if err != nil {
		return nil, err
	}

	var result TransferResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UseBigBlocks enables or disables big blocks
func (e *Exchange) UseBigBlocks(ctx context.Context, enable bool) (*ApprovalResponse, error) {
	nonce := e.nextNonce()

	action := UseBigBlocksAction{
		Type:           "evmUserModify",
		UsingBigBlocks: enable,
	}

	sig, err := SignL1Action(
		e.privateKey,
		action,
		"", // No vault address
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce)
	if err != nil {
		return nil, err
	}

	var result ApprovalResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// PerpDexClassTransfer transfers tokens between perp dex classes
func (e *Exchange) PerpDexClassTransfer(
	ctx context.Context,
	dex, token string,
	amount float64,
	toPerp bool,
) (*TransferResponse, error) {
	nonce := e.nextNonce()

	action := PerpDexClassTransferAction{
		Type:   "perpDexClassTransfer",
		Dex:    dex,
		Token:  token,
		Amount: amount,
		ToPerp: toPerp,
	}

	sig, err := SignL1Action(
		e.privateKey,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce)
	if err != nil {
		return nil, err
	}

	var result TransferResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SubAccountSpotTransfer transfers spot tokens to/from sub-account
func (e *Exchange) SubAccountSpotTransfer(
	ctx context.Context,
	subAccountUser string,
	isDeposit bool,
	token string,
	amount float64,
) (*TransferResponse, error) {
	nonce := e.nextNonce()

	action := SubAccountSpotTransferAction{
		Type:           "subAccountSpotTransfer",
		SubAccountUser: subAccountUser,
		IsDeposit:      isDeposit,
		Token:          token,
		Amount:         amount,
	}

	sig, err := SignL1Action(
		e.privateKey,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce)
	if err != nil {
		return nil, err
	}

	var result TransferResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// TokenDelegate delegates tokens for staking
func (e *Exchange) TokenDelegate(
	ctx context.Context,
	validator string,
	wei int,
	isUndelegate bool,
) (*TransferResponse, error) {
	nonce := e.nextNonce()

	action := TokenDelegateAction{
		Type:         "tokenDelegate",
		Validator:    validator,
		Wei:          wei,
		IsUndelegate: isUndelegate,
		Nonce:        nonce,
	}

	sig, err := SignL1Action(
		e.privateKey,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce)
	if err != nil {
		return nil, err
	}

	var result TransferResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// WithdrawFromBridge withdraws tokens from bridge
func (e *Exchange) WithdrawFromBridge(
	ctx context.Context,
	amount float64,
	destination string,
) (*TransferResponse, error) {
	nonce := e.nextNonce()

	action := map[string]any{
		"destination": destination,
		"amount":      formatFloat(amount),
		"time":        big.NewInt(nonce),
		"type":        "withdraw3",
	}

	payloadTypes := []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "destination", Type: "string"},
		{Name: "amount", Type: "string"},
		{Name: "time", Type: "uint64"},
	}

	sig, err := SignUserSignedAction(
		e.privateKey,
		action,
		payloadTypes,
		"HyperliquidTransaction:Withdraw",
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce)
	if err != nil {
		return nil, err
	}

	var result TransferResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ApproveAgent approves an agent to trade on behalf of the user
// Returns the result and the generated agent private key
func (e *Exchange) ApproveAgent(
	ctx context.Context,
	name *string,
) (*AgentApprovalResponse, string, error) {
	// Generate agent key
	agentBytes := make([]byte, 32)
	if _, err := rand.Read(agentBytes); err != nil {
		return nil, "", fmt.Errorf("failed to generate agent key: %w", err)
	}
	agentKey := "0x" + hex.EncodeToString(agentBytes)

	privateKey, err := crypto.HexToECDSA(agentKey[2:])
	if err != nil {
		return nil, "", fmt.Errorf("failed to create private key: %w", err)
	}

	agentAddress := crypto.PubkeyToAddress(privateKey.PublicKey).Hex()
	nonce := e.nextNonce()

	agentName := ""
	if name != nil {
		agentName = *name
	}

	// Use SignAgent which does EIP-712 signing (not L1Action)
	sig, err := SignAgent(
		e.privateKey,
		agentAddress,
		agentName,
		nonce,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, "", err
	}

	// Build action for submission
	// Note: We send the pointer version for omitempty behavior in JSON/msgpack
	hyperliquidChain := "Testnet"
	if e.client.baseURL == MainnetAPIURL {
		hyperliquidChain = "Mainnet"
	}

	action := ApproveAgentAction{
		Type:             "approveAgent",
		SignatureChainId: "0x66eee",
		HyperliquidChain: hyperliquidChain,
		AgentAddress:     agentAddress,
		AgentName:        name,
		Nonce:            nonce,
	}

	resp, err := e.postAction(ctx, action, sig, nonce)
	if err != nil {
		return nil, "", err
	}

	var result AgentApprovalResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, "", err
	}
	return &result, agentKey, nil
}

// ApproveBuilderFee approves builder fee payment
func (e *Exchange) ApproveBuilderFee(
	ctx context.Context,
	builder string,
	maxFeeRate string,
) (*ApprovalResponse, error) {
	nonce := e.nextNonce()

	action := map[string]any{
		"maxFeeRate": maxFeeRate,
		"builder":    strings.ToLower(builder),
		"nonce":      big.NewInt(nonce),
		"type":       "approveBuilderFee",
	}

	payloadTypes := []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "maxFeeRate", Type: "string"},
		{Name: "builder", Type: "address"},
		{Name: "nonce", Type: "uint64"},
	}

	sig, err := SignUserSignedAction(
		e.privateKey,
		action,
		payloadTypes,
		"HyperliquidTransaction:ApproveBuilderFee",
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce)
	if err != nil {
		return nil, err
	}

	var result ApprovalResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ConvertToMultiSigUser converts account to multi-signature user
func (e *Exchange) ConvertToMultiSigUser(
	ctx context.Context,
	authorizedUsers []string,
	threshold int,
) (*MultiSigConversionResponse, error) {
	nonce := e.nextNonce()

	// Sort users as done in Python
	sort.Strings(authorizedUsers)

	signers := map[string]any{
		"authorizedUsers": authorizedUsers,
		"threshold":       threshold,
	}

	signersJSON, err := json.Marshal(signers)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal signers: %w", err)
	}

	action := ConvertToMultiSigUserAction{
		Type:    "convertToMultiSigUser",
		Signers: string(signersJSON),
		Nonce:   nonce,
	}

	sig, err := SignL1Action(
		e.privateKey,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce)
	if err != nil {
		return nil, err
	}

	var result MultiSigConversionResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Spot Deploy Methods

// SpotDeployRegisterToken registers a new spot token
func (e *Exchange) SpotDeployRegisterToken(
	ctx context.Context,
	tokenName string,
	szDecimals int,
	weiDecimals int,
	maxGas int,
	fullName string,
) (*SpotDeployResponse, error) {
	nonce := e.nextNonce()

	action := map[string]any{
		"type": "spotDeploy",
		"registerToken2": map[string]any{
			"spec": map[string]any{
				"name":        tokenName,
				"szDecimals":  szDecimals,
				"weiDecimals": weiDecimals,
			},
			"maxGas":   maxGas,
			"fullName": fullName,
		},
	}

	sig, err := SignL1Action(
		e.privateKey,
		action,
		"", // No vault address for spot deploy
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce)
	if err != nil {
		return nil, err
	}

	var result SpotDeployResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SpotDeployUserGenesis initializes user genesis for spot trading
func (e *Exchange) SpotDeployUserGenesis(
	ctx context.Context,
	balances map[string]float64,
) (*SpotDeployResponse, error) {
	nonce := e.nextNonce()

	action := map[string]any{
		"type":     "spotDeployUserGenesis",
		"balances": balances,
	}

	sig, err := SignL1Action(
		e.privateKey,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce)
	if err != nil {
		return nil, err
	}

	var result SpotDeployResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SpotDeployEnableFreezePrivilege enables freeze privilege for spot deployer
func (e *Exchange) SpotDeployEnableFreezePrivilege(
	ctx context.Context,
) (*SpotDeployResponse, error) {
	nonce := e.nextNonce()

	action := map[string]any{
		"type": "spotDeployEnableFreezePrivilege",
	}

	sig, err := SignL1Action(
		e.privateKey,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce)
	if err != nil {
		return nil, err
	}

	var result SpotDeployResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SpotDeployFreezeUser freezes a user in spot trading
func (e *Exchange) SpotDeployFreezeUser(
	ctx context.Context,
	userAddress string,
) (*SpotDeployResponse, error) {
	nonce := e.nextNonce()

	action := map[string]any{
		"type":        "spotDeployFreezeUser",
		"userAddress": userAddress,
	}

	sig, err := SignL1Action(
		e.privateKey,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce)
	if err != nil {
		return nil, err
	}

	var result SpotDeployResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SpotDeployRevokeFreezePrivilege revokes freeze privilege for spot deployer
func (e *Exchange) SpotDeployRevokeFreezePrivilege(
	ctx context.Context,
) (*SpotDeployResponse, error) {
	nonce := e.nextNonce()

	action := map[string]any{
		"type": "spotDeployRevokeFreezePrivilege",
	}

	sig, err := SignL1Action(
		e.privateKey,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce)
	if err != nil {
		return nil, err
	}

	var result SpotDeployResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SpotDeployGenesis initializes spot genesis
func (e *Exchange) SpotDeployGenesis(
	ctx context.Context,
	deployer string,
	dexName string,
) (*SpotDeployResponse, error) {
	nonce := e.nextNonce()

	action := map[string]any{
		"type":     "spotDeployGenesis",
		"deployer": deployer,
		"dexName":  dexName,
	}

	sig, err := SignL1Action(
		e.privateKey,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce)
	if err != nil {
		return nil, err
	}

	var result SpotDeployResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SpotDeployRegisterSpot registers spot market
func (e *Exchange) SpotDeployRegisterSpot(
	ctx context.Context,
	baseToken string,
	quoteToken string,
) (*SpotDeployResponse, error) {
	nonce := e.nextNonce()

	action := map[string]any{
		"type":       "spotDeployRegisterSpot",
		"baseToken":  baseToken,
		"quoteToken": quoteToken,
	}

	sig, err := SignL1Action(
		e.privateKey,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce)
	if err != nil {
		return nil, err
	}

	var result SpotDeployResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SpotDeployRegisterHyperliquidity registers hyperliquidity spot
func (e *Exchange) SpotDeployRegisterHyperliquidity(
	ctx context.Context,
	name string,
	tokens []string,
) (*SpotDeployResponse, error) {
	nonce := e.nextNonce()

	action := map[string]any{
		"type":   "spotDeployRegisterHyperliquidity",
		"name":   name,
		"tokens": tokens,
	}

	sig, err := SignL1Action(
		e.privateKey,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce)
	if err != nil {
		return nil, err
	}

	var result SpotDeployResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SpotDeploySetDeployerTradingFeeShare sets deployer trading fee share
func (e *Exchange) SpotDeploySetDeployerTradingFeeShare(
	ctx context.Context,
	feeShare float64,
) (*SpotDeployResponse, error) {
	nonce := e.nextNonce()

	action := map[string]any{
		"type":     "spotDeploySetDeployerTradingFeeShare",
		"feeShare": feeShare,
	}

	sig, err := SignL1Action(
		e.privateKey,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce)
	if err != nil {
		return nil, err
	}

	var result SpotDeployResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Perp Deploy Methods

// PerpDeployRegisterAsset registers a new perpetual asset on a builder-deployed DEX.
// Provide schema to also initialize a new dex alongside the asset registration.
func (e *Exchange) PerpDeployRegisterAsset(
	ctx context.Context,
	dex string,
	maxGas *int,
	assetRequest AssetRequest,
	schema *PerpDexSchemaInput,
) (*PerpDeployResponse, error) {
	nonce := e.nextNonce()

	action := PerpDeployRegisterAssetAction{
		Type: "perpDeploy",
		RegisterAsset: RegisterAsset{
			MaxGas:       maxGas,
			AssetRequest: assetRequest,
			Dex:          dex,
		},
	}

	if schema != nil {
		action.RegisterAsset.Schema = buildSchemaWire(schema)
	}

	sig, err := SignL1Action(
		e.privateKey,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce)
	if err != nil {
		return nil, err
	}

	var result PerpDeployResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// PerpDeployRegisterAsset2 registers a new perpetual asset on a builder-deployed DEX
// using marginMode instead of onlyIsolated. Provide schema to also initialize a new dex.
func (e *Exchange) PerpDeployRegisterAsset2(
	ctx context.Context,
	dex string,
	maxGas *int,
	assetRequest AssetRequest2,
	schema *PerpDexSchemaInput,
) (*PerpDeployResponse, error) {
	nonce := e.nextNonce()

	action := PerpDeployRegisterAsset2Action{
		Type: "perpDeploy",
		RegisterAsset2: RegisterAsset2{
			MaxGas:       maxGas,
			AssetRequest: assetRequest,
			Dex:          dex,
		},
	}

	if schema != nil {
		action.RegisterAsset2.Schema = buildSchemaWire(schema)
	}

	sig, err := SignL1Action(
		e.privateKey,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce)
	if err != nil {
		return nil, err
	}

	var result PerpDeployResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func buildSchemaWire(schema *PerpDexSchemaInput) *RegisterAssetSchema {
	wire := &RegisterAssetSchema{
		FullName:        schema.FullName,
		CollateralToken: schema.CollateralToken,
	}
	if schema.OracleUpdater != nil {
		oracleUpdater := strings.ToLower(*schema.OracleUpdater)
		wire.OracleUpdater = &oracleUpdater
	}
	return wire
}

// PerpHaltTrading halts or unhalts trading for a builder-deployed DEX
func (e *Exchange) PerpDeployHaltTrading(
	ctx context.Context,
	coin string,
	isHalted bool,
) (*PerpDeployResponse, error) {
	nonce := e.nextNonce()

	action := PerpDeployHaltTradingAction{
		Type: "perpDeploy",
		HaltTrading: HaltTrading{
			Coin:     coin,
			IsHalted: isHalted,
		},
	}

	sig, err := SignL1Action(
		e.privateKey,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce)
	if err != nil {
		return nil, err
	}

	var result PerpDeployResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// PerpDeploySetOracle sets oracle prices for a builder-deployed DEX
// This matches the Python SDK's perp_deploy_set_oracle method
// oraclePxs: map of coin to oracle price string
// allMarkPxs: list of maps, each map contains coin to mark price string
// externalPerpPxs: map of coin to external perp price string
func (e *Exchange) PerpDeploySetOracle(
	ctx context.Context,
	dex string,
	oraclePxs map[string]string,
	allMarkPxs []map[string]string,
	externalPerpPxs map[string]string,
) (*PerpDeployResponse, error) {
	nonce := e.nextNonce()

	// Sort oracle prices for deterministic ordering
	oraclePxsWire := make([][]string, 0, len(oraclePxs))
	for coin, px := range oraclePxs {
		oraclePxsWire = append(oraclePxsWire, []string{coin, px})
	}
	sort.Slice(oraclePxsWire, func(i, j int) bool {
		return oraclePxsWire[i][0] < oraclePxsWire[j][0]
	})

	// Sort mark prices - each element is a list of [coin, px] pairs
	markPxsWire := make([][][]string, 0, len(allMarkPxs))
	for _, markPxs := range allMarkPxs {
		markPxList := make([][]string, 0, len(markPxs))
		for coin, px := range markPxs {
			markPxList = append(markPxList, []string{coin, px})
		}
		sort.Slice(markPxList, func(i, j int) bool {
			return markPxList[i][0] < markPxList[j][0]
		})
		markPxsWire = append(markPxsWire, markPxList)
	}

	// Sort external perp prices
	externalPerpPxsWire := make([][]string, 0, len(externalPerpPxs))
	for coin, px := range externalPerpPxs {
		externalPerpPxsWire = append(externalPerpPxsWire, []string{coin, px})
	}
	sort.Slice(externalPerpPxsWire, func(i, j int) bool {
		return externalPerpPxsWire[i][0] < externalPerpPxsWire[j][0]
	})

	action := PerpDeploySetOracleAction{
		Type: "perpDeploy",
		SetOracle: SetOracle{
			Dex:             dex,
			OraclePxs:       oraclePxsWire,
			MarkPxs:         markPxsWire,
			ExternalPerpPxs: externalPerpPxsWire,
		},
	}

	sig, err := SignL1Action(
		e.privateKey,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce)
	if err != nil {
		return nil, err
	}

	var result PerpDeployResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CSigner Methods

// CSignerUnjailSelf unjails self as consensus signer
func (e *Exchange) CSignerUnjailSelf(ctx context.Context) (*ValidatorResponse, error) {
	nonce := e.nextNonce()

	action := map[string]any{
		"type": "cSignerUnjailSelf",
	}

	sig, err := SignL1Action(
		e.privateKey,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce)
	if err != nil {
		return nil, err
	}

	var result ValidatorResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CSignerJailSelf jails self as consensus signer
func (e *Exchange) CSignerJailSelf(ctx context.Context) (*ValidatorResponse, error) {
	nonce := e.nextNonce()

	action := map[string]any{
		"type": "cSignerJailSelf",
	}

	sig, err := SignL1Action(
		e.privateKey,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce)
	if err != nil {
		return nil, err
	}

	var result ValidatorResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CSignerInner executes inner consensus signer action
func (e *Exchange) CSignerInner(
	ctx context.Context,
	innerAction map[string]any,
) (*ValidatorResponse, error) {
	nonce := e.nextNonce()

	action := map[string]any{
		"type":        "cSignerInner",
		"innerAction": innerAction,
	}

	sig, err := SignL1Action(
		e.privateKey,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce)
	if err != nil {
		return nil, err
	}

	var result ValidatorResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CValidator Methods

// CValidatorRegister registers as consensus validator
func (e *Exchange) CValidatorRegister(
	ctx context.Context,
	validatorProfile map[string]any,
) (*ValidatorResponse, error) {
	nonce := e.nextNonce()

	action := map[string]any{
		"type":             "cValidatorRegister",
		"validatorProfile": validatorProfile,
	}

	sig, err := SignL1Action(
		e.privateKey,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce)
	if err != nil {
		return nil, err
	}

	var result ValidatorResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CValidatorChangeProfile changes validator profile
func (e *Exchange) CValidatorChangeProfile(
	ctx context.Context,
	newProfile map[string]any,
) (*ValidatorResponse, error) {
	nonce := e.nextNonce()

	action := map[string]any{
		"type":       "cValidatorChangeProfile",
		"newProfile": newProfile,
	}

	sig, err := SignL1Action(
		e.privateKey,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce)
	if err != nil {
		return nil, err
	}

	var result ValidatorResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CValidatorUnregister unregisters as consensus validator
func (e *Exchange) CValidatorUnregister(ctx context.Context) (*ValidatorResponse, error) {
	nonce := e.nextNonce()

	action := map[string]any{
		"type": "cValidatorUnregister",
	}

	sig, err := SignL1Action(
		e.privateKey,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce)
	if err != nil {
		return nil, err
	}

	var result ValidatorResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (e *Exchange) MultiSig(
	ctx context.Context,
	action map[string]any,
	signers []string,
	signatures []string,
) (*MultiSigResponse, error) {
	nonce := e.nextNonce()

	multiSigAction := map[string]any{
		"type":       "multiSig",
		"action":     action,
		"signers":    signers,
		"signatures": signatures,
	}

	sig, err := SignL1Action(
		e.privateKey,
		multiSigAction,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, multiSigAction, sig, nonce)
	if err != nil {
		return nil, err
	}

	var result MultiSigResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
