package hyperliquid

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
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

	nonce := e.nextNonce()

	sig, err := e.signL1Action(
		ctx,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce, e.vault)
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
		return nil, err
	}

	var result UserState
	if err := json.Unmarshal(resp, &result); err != nil {
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

	nonce := e.nextNonce()

	sig, err := e.signL1Action(
		ctx,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce, e.vault)
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
		return nil, err
	}

	var result UserState
	if err := json.Unmarshal(resp, &result); err != nil {
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
		mids, err := e.info.AllMids(ctx, e.dex)
		if err != nil {
			return 0, err
		}
		if midPriceStr, exists := mids[name]; exists {
			price, err = parseFloatStrict(midPriceStr)
			if err != nil {
				return 0, fmt.Errorf("invalid mid price %q for coin %s: %w", midPriceStr, name, err)
			}
		} else {
			return 0, fmt.Errorf("could not get mid price for coin: %s", name)
		}
	}

	asset := e.info.coinToAsset[name]
	isSpot := asset >= spotAssetIndexOffset && asset < builderPerpAssetBase

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

	sig, err := e.signL1Action(
		ctx,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce, e.vault)
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
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

	// Sign the action
	sig, err := e.signL1Action(
		ctx,
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
	resp, err := e.postAction(ctx, action, sig, nonce, e.vault)
	if err != nil {
		return nil, fmt.Errorf("failed to post reserve action: %w", err)
	}

	if err := actionError(resp); err != nil {
		return nil, err
	}

	// Parse response — the "response" field is polymorphic:
	// on success it is {"type":"..."}, on error it is a plain string.
	var raw struct {
		Status   string          `json:"status"`
		Response json.RawMessage `json:"response,omitempty"`
	}
	if err := json.Unmarshal(resp, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse reserve response: %w", err)
	}
	result := ReserveRequestWeightResponse{Status: raw.Status}
	if len(raw.Response) > 0 {
		if raw.Status == "ok" {
			var data ReserveResponseData
			if err := json.Unmarshal(raw.Response, &data); err == nil {
				result.Response = &data
			}
		} else {
			var errMsg string
			if err := json.Unmarshal(raw.Response, &errMsg); err == nil {
				result.Error = errMsg
			}
		}
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

	sig, err := e.signL1Action(
		ctx,
		action,
		"",
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce, e.vault)
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
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

	sig, err := e.signL1Action(
		ctx,
		action,
		"",
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce, e.vault)
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
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
	strAmount, err := floatToWire(amount)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	nonce := e.nextNonce()

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

	sig, err := e.signUserSignedAction(
		ctx,
		action,
		payloadTypes,
		"HyperliquidTransaction:UsdClassTransfer",
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce, "")
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
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

	sig, err := e.signL1Action(
		ctx,
		action,
		"",
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce, e.vault)
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
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

	sig, err := e.signL1Action(
		ctx,
		action,
		"",
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce, e.vault)
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
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

	sig, err := e.signL1Action(
		ctx,
		action,
		"",
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce, e.vault)
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
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

	sig, err := e.signL1Action(
		ctx,
		action,
		"",
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce, e.vault)
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
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

	sig, err := e.signL1Action(
		ctx,
		action,
		"",
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce, e.vault)
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
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
	amountWire, err := floatToWire(amount)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	nonce := e.nextNonce()

	action := map[string]any{
		"destination": destination,
		"amount":      amountWire,
		"time":        big.NewInt(nonce),
		"type":        "usdSend",
	}

	payloadTypes := []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "destination", Type: "string"},
		{Name: "amount", Type: "string"},
		{Name: "time", Type: "uint64"},
	}

	sig, err := e.signUserSignedAction(
		ctx,
		action,
		payloadTypes,
		"HyperliquidTransaction:UsdSend",
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce, e.vault)
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
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
	amountWire, err := floatToWire(amount)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	nonce := e.nextNonce()

	action := map[string]any{
		"destination": destination,
		"amount":      amountWire,
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

	sig, err := e.signUserSignedAction(
		ctx,
		action,
		payloadTypes,
		"HyperliquidTransaction:SpotSend",
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce, e.vault)
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
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

	sig, err := e.signL1Action(
		ctx,
		action,
		"",
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce, e.vault)
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
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

	sig, err := e.signL1Action(
		ctx,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce, e.vault)
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
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
	// The API takes the amount as a decimal string, like the Python SDK's
	// str(amount). A float64 here is hashed as a msgpack float, so the
	// signature would not match the action the exchange decodes.
	amountWire, err := floatToWire(amount)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	nonce := e.nextNonce()

	action := SubAccountSpotTransferAction{
		Type:           "subAccountSpotTransfer",
		SubAccountUser: subAccountUser,
		IsDeposit:      isDeposit,
		Token:          token,
		Amount:         amountWire,
	}

	sig, err := e.signL1Action(
		ctx,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce, e.vault)
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
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

	sig, err := e.signUserSignedAction(
		ctx,
		action,
		payloadTypes,
		"HyperliquidTransaction:TokenDelegate",
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce, e.vault)
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
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
	amountWire, err := floatToWire(amount)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	nonce := e.nextNonce()

	action := map[string]any{
		"destination": destination,
		"amount":      amountWire,
		"time":        big.NewInt(nonce),
		"type":        "withdraw3",
	}

	payloadTypes := []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "destination", Type: "string"},
		{Name: "amount", Type: "string"},
		{Name: "time", Type: "uint64"},
	}

	sig, err := e.signUserSignedAction(
		ctx,
		action,
		payloadTypes,
		"HyperliquidTransaction:Withdraw",
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce, e.vault)
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
		return nil, err
	}

	var result TransferResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SendToEVMWithData sends tokens to EVM with additional data
func (e *Exchange) SendToEVMWithData(
	ctx context.Context,
	token string,
	amount string,
	sourceDex string,
	destinationRecipient string,
	addressEncoding string,
	destinationChainID int64,
	gasLimit int64,
	data string,
) (*TransferResponse, error) {
	if destinationChainID < 0 || destinationChainID > math.MaxUint32 {
		return nil, fmt.Errorf("destinationChainID must be in range [0, %d]", math.MaxUint32)
	}
	if gasLimit < 0 {
		return nil, fmt.Errorf("gasLimit cannot be negative: %d", gasLimit)
	}

	dbyte, err := hexutil.Decode(data)
	if err != nil {
		return nil, fmt.Errorf("invalid data: %w", err)
	}

	nonce := e.nextNonce()

	action := map[string]any{
		"token":                token,
		"amount":               amount,
		"sourceDex":            sourceDex,
		"destinationRecipient": destinationRecipient,
		"addressEncoding":      addressEncoding,
		"destinationChainId":   new(big.Int).SetInt64(destinationChainID),
		"gasLimit":             new(big.Int).SetInt64(gasLimit),
		"data":                 dbyte,
		"nonce":                nonce,
		"type":                 "sendToEvmWithData",
	}

	payloadTypes := []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "token", Type: "string"},
		{Name: "amount", Type: "string"},
		{Name: "sourceDex", Type: "string"},
		{Name: "destinationRecipient", Type: "string"},
		{Name: "addressEncoding", Type: "string"},
		{Name: "destinationChainId", Type: "uint32"},
		{Name: "gasLimit", Type: "uint64"},
		{Name: "data", Type: "bytes"},
		{Name: "nonce", Type: "uint64"},
	}

	sig, err := e.signUserSignedAction(
		ctx,
		action,
		payloadTypes,
		"HyperliquidTransaction:SendToEvmWithData",
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce, e.vault)
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
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

	sig, err := e.signAgent(ctx, agentAddress, agentName, nonce, e.client.baseURL == MainnetAPIURL)
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

	resp, err := e.postAction(ctx, action, sig, nonce, e.vault)
	if err != nil {
		return nil, "", err
	}

	if err := actionError(resp); err != nil {
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

	sig, err := e.signUserSignedAction(
		ctx,
		action,
		payloadTypes,
		"HyperliquidTransaction:ApproveBuilderFee",
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce, e.vault)
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
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

	action := map[string]any{
		"type":    "convertToMultiSigUser",
		"signers": string(signersJSON),
		"nonce":   big.NewInt(nonce),
	}

	payloadTypes := []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "signers", Type: "string"},
		{Name: "nonce", Type: "uint64"},
	}

	sig, err := e.signUserSignedAction(
		ctx,
		action,
		payloadTypes,
		"HyperliquidTransaction:ConvertToMultiSigUser",
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce, e.vault)
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
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

	action := SpotDeployAction{
		Type: "spotDeploy",
		RegisterToken2: &SpotDeployRegisterToken2{
			Spec: SpotDeployTokenSpec{
				Name:        tokenName,
				SzDecimals:  szDecimals,
				WeiDecimals: weiDecimals,
			},
			MaxGas:   maxGas,
			FullName: fullName,
		},
	}

	sig, err := e.signL1Action(
		ctx,
		action,
		"",
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce, e.vault)
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
		return nil, err
	}

	var result SpotDeployResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SpotDeployExistingTokenWei is the amount of an existing token granted by userGenesis
type SpotDeployExistingTokenWei struct {
	Token int
	Wei   string
}

// SpotDeployUserGenesis initializes user genesis for spot trading
func (e *Exchange) SpotDeployUserGenesis(
	ctx context.Context,
	token int,
	userAndWei [][2]string,
	existingTokenAndWei []SpotDeployExistingTokenWei,
) (*SpotDeployResponse, error) {
	nonce := e.nextNonce()

	users := make([][2]string, len(userAndWei))
	for i, item := range userAndWei {
		users[i] = [2]string{strings.ToLower(item[0]), item[1]}
	}

	existing := make([][2]any, len(existingTokenAndWei))
	for i, item := range existingTokenAndWei {
		existing[i] = [2]any{item.Token, item.Wei}
	}

	action := SpotDeployAction{
		Type: "spotDeploy",
		UserGenesis: &SpotDeployUserGenesis{
			Token:               token,
			UserAndWei:          users,
			ExistingTokenAndWei: existing,
		},
	}

	sig, err := e.signL1Action(
		ctx,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce, e.vault)
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
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
	token int,
) (*SpotDeployResponse, error) {
	nonce := e.nextNonce()

	action := SpotDeployAction{
		Type:                  "spotDeploy",
		EnableFreezePrivilege: &SpotDeployEnableFreezePrivilege{Token: token},
	}

	sig, err := e.signL1Action(
		ctx,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce, e.vault)
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
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
	token int,
	userAddress string,
	freeze bool,
) (*SpotDeployResponse, error) {
	nonce := e.nextNonce()

	action := SpotDeployAction{
		Type:       "spotDeploy",
		FreezeUser: &SpotDeployFreezeUser{Token: token, User: userAddress, Freeze: freeze},
	}

	sig, err := e.signL1Action(
		ctx,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce, e.vault)
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
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
	token int,
) (*SpotDeployResponse, error) {
	nonce := e.nextNonce()

	action := SpotDeployAction{
		Type:                  "spotDeploy",
		RevokeFreezePrivilege: &SpotDeployRevokeFreezePrivilege{Token: token},
	}

	sig, err := e.signL1Action(
		ctx,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce, e.vault)
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
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
	token int,
	maxSupply string,
	noHyperliquidity bool,
) (*SpotDeployResponse, error) {
	nonce := e.nextNonce()

	action := SpotDeployAction{
		Type: "spotDeploy",
		Genesis: &SpotDeployGenesis{
			Token:            token,
			MaxSupply:        maxSupply,
			NoHyperliquidity: noHyperliquidity,
		},
	}

	sig, err := e.signL1Action(
		ctx,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce, e.vault)
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
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
	baseToken int,
	quoteToken int,
) (*SpotDeployResponse, error) {
	nonce := e.nextNonce()

	action := SpotDeployAction{
		Type:         "spotDeploy",
		RegisterSpot: &SpotDeployRegisterSpot{Tokens: [2]int{baseToken, quoteToken}},
	}

	sig, err := e.signL1Action(
		ctx,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce, e.vault)
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
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
	spot int,
	startPx float64,
	orderSz float64,
	nOrders int,
	nSeededLevels *int,
) (*SpotDeployResponse, error) {
	startPxWire, err := floatToWire(startPx)
	if err != nil {
		return nil, fmt.Errorf("invalid start price: %w", err)
	}

	orderSzWire, err := floatToWire(orderSz)
	if err != nil {
		return nil, fmt.Errorf("invalid order size: %w", err)
	}

	nonce := e.nextNonce()

	action := SpotDeployAction{
		Type: "spotDeploy",
		RegisterHyperliquidity: &SpotDeployRegisterHyperliquidity{
			Spot:          spot,
			StartPx:       startPxWire,
			OrderSz:       orderSzWire,
			NOrders:       nOrders,
			NSeededLevels: nSeededLevels,
		},
	}

	sig, err := e.signL1Action(
		ctx,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce, e.vault)
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
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
	token int,
	share string,
) (*SpotDeployResponse, error) {
	nonce := e.nextNonce()

	action := SpotDeployAction{
		Type:                       "spotDeploy",
		SetDeployerTradingFeeShare: &SpotDeploySetDeployerTradingFeeShare{Token: token, Share: share},
	}

	sig, err := e.signL1Action(
		ctx,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce, e.vault)
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
		return nil, err
	}

	var result SpotDeployResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SpotDeployEnableQuoteToken enables a token as a quote token
func (e *Exchange) SpotDeployEnableQuoteToken(
	ctx context.Context,
	token int,
) (*SpotDeployResponse, error) {
	nonce := e.nextNonce()

	action := SpotDeployAction{
		Type:             "spotDeploy",
		EnableQuoteToken: &SpotDeployEnableQuoteToken{Token: token},
	}

	sig, err := e.signL1Action(
		ctx,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce, e.vault)
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
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

	sig, err := e.signL1Action(
		ctx,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce, e.vault)
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
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

	sig, err := e.signL1Action(
		ctx,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce, e.vault)
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
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

	sig, err := e.signL1Action(
		ctx,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce, e.vault)
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
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

	sig, err := e.signL1Action(
		ctx,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce, e.vault)
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
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

	action := CSignerAction{Type: "CSignerAction", UnjailSelf: &struct{}{}}

	sig, err := e.signL1Action(
		ctx,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce, e.vault)
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
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

	action := CSignerAction{Type: "CSignerAction", JailSelf: &struct{}{}}

	sig, err := e.signL1Action(
		ctx,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce, e.vault)
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
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
	nodeIp string,
	name string,
	description string,
	delegationsDisabled bool,
	commissionBps int,
	signer string,
	unjailed bool,
	initialWei int,
) (*ValidatorResponse, error) {
	nonce := e.nextNonce()

	action := CValidatorAction{
		Type: "CValidatorAction",
		Register: &CValidatorRegister{
			Profile: CValidatorProfile{
				NodeIp:              CValidatorNodeIp{Ip: nodeIp},
				Name:                name,
				Description:         description,
				DelegationsDisabled: delegationsDisabled,
				CommissionBps:       commissionBps,
				Signer:              signer,
			},
			Unjailed:   unjailed,
			InitialWei: initialWei,
		},
	}

	sig, err := e.signL1Action(
		ctx,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce, e.vault)
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
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
	nodeIp *string,
	name *string,
	description *string,
	unjailed bool,
	disableDelegations *bool,
	commissionBps *int,
	signer *string,
) (*ValidatorResponse, error) {
	nonce := e.nextNonce()

	action := CValidatorAction{
		Type: "CValidatorAction",
		ChangeProfile: &CValidatorChangeProfile{
			Name:               name,
			Description:        description,
			Unjailed:           unjailed,
			DisableDelegations: disableDelegations,
			CommissionBps:      commissionBps,
			Signer:             signer,
		},
	}
	if nodeIp != nil {
		action.ChangeProfile.NodeIp = &CValidatorNodeIp{Ip: *nodeIp}
	}

	sig, err := e.signL1Action(
		ctx,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce, e.vault)
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
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

	sig, err := e.signL1Action(
		ctx,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce, e.vault)
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
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

	sig, err := e.signL1Action(
		ctx,
		multiSigAction,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, multiSigAction, sig, nonce, e.vault)
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
		return nil, err
	}

	var result MultiSigResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UserSetAbstraction Methods set abstraction
func (e *Exchange) UserSetAbstraction(
	ctx context.Context,
	user string,
	abstraction string,
) (*ReserveRequestWeightResponse, error) {
	nonce := e.nextNonce()

	action := map[string]any{
		"user":        user,
		"abstraction": abstraction,
		"nonce":       big.NewInt(nonce),
		"type":        "userSetAbstraction",
	}

	payloadTypes := []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "user", Type: "address"},
		{Name: "abstraction", Type: "string"},
		{Name: "nonce", Type: "uint64"},
	}

	sig, err := e.signUserSignedAction(
		ctx,
		action,
		payloadTypes,
		"HyperliquidTransaction:UserSetAbstraction",
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce, e.vault)
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
		return nil, err
	}

	var result ReserveRequestWeightResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UserDexAbstraction Methods set dex abstraction
func (e *Exchange) UserDexAbstraction(
	ctx context.Context,
	user string,
	enabled bool,
) (*ReserveRequestWeightResponse, error) {
	nonce := e.nextNonce()

	action := map[string]any{
		"user":    user,
		"enabled": enabled,
		"nonce":   big.NewInt(nonce),
		"type":    "userDexAbstraction",
	}

	payloadTypes := []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "user", Type: "address"},
		{Name: "enabled", Type: "bool"},
		{Name: "nonce", Type: "uint64"},
	}

	sig, err := e.signUserSignedAction(
		ctx,
		action,
		payloadTypes,
		"HyperliquidTransaction:UserDexAbstraction",
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce, e.vault)
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
		return nil, err
	}

	var result ReserveRequestWeightResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// For the default perp dex use the empty string "" as name. For spot use "spot".
// Token must match the collateral token if transferring to or from a perp dex.
func (e *Exchange) SendAsset(
	ctx context.Context,
	destination string,
	sourceDex string,
	destinationDex string,
	token string,
	amount float64,
) (*ReserveRequestWeightResponse, error) {
	amountWire, err := floatToWire(amount)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	nonce := e.nextNonce()

	action := map[string]any{
		"destination":    destination,
		"sourceDex":      sourceDex,
		"destinationDex": destinationDex,
		"token":          token,
		"amount":         amountWire,
		"fromSubAccount": e.vault,
		"nonce":          big.NewInt(nonce),
		"type":           "sendAsset",
	}

	payloadTypes := []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "destination", Type: "string"},
		{Name: "sourceDex", Type: "string"},
		{Name: "destinationDex", Type: "string"},
		{Name: "token", Type: "string"},
		{Name: "amount", Type: "string"},
		{Name: "fromSubAccount", Type: "string"},
		{Name: "nonce", Type: "uint64"},
	}

	sig, err := e.signUserSignedAction(
		ctx,
		action,
		payloadTypes,
		"HyperliquidTransaction:SendAsset",
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce, "")
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
		return nil, err
	}

	var result ReserveRequestWeightResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Borrow, repay, supply, or withdraw HIP-2 borrow/lend assets
func (e *Exchange) BorrowLend(
	ctx context.Context,
	amount float64,
	operation string,
	token int64,
) (*ReserveRequestWeightResponse, error) {
	amountWire, err := floatToWire(amount)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	nonce := e.nextNonce()

	action := BorrowLendAction{
		Type:      "borrowLend",
		Operation: operation,
		Token:     token,
		Amount:    amountWire,
	}

	sig, err := e.signL1Action(
		ctx,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(ctx, action, sig, nonce, e.vault)
	if err != nil {
		return nil, err
	}

	if err := actionError(resp); err != nil {
		return nil, err
	}
	var result ReserveRequestWeightResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
