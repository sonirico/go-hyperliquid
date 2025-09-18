package hyperliquid

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/ethereum/go-ethereum/crypto"
)

func (e *Exchange) UpdateLeverage(leverage int, name string, isCross bool) (*UserState, error) {
	return e.UpdateLeverageWithContext(context.Background(), leverage, name, isCross)
}
func (e *Exchange) UpdateLeverageWithContext(ctx context.Context, leverage int, name string, isCross bool) (*UserState, error) {
	leverageType := "isolated"
	if isCross {
		leverageType = "cross"
	}

	action := UpdateLeverageAction{
		Type:  "updateLeverage",
		Asset: e.info.NameToAsset(name),
		Leverage: map[string]any{
			"type":  leverageType,
			"value": leverage,
		},
	}

	var result UserState
	if err := e.executeAction(ctx, action, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
func (e *Exchange) UpdateIsolatedMargin(amount float64, name string) (*UserState, error) {
	return e.UpdateIsolatedMarginWithContext(context.Background(), amount, name)
}
func (e *Exchange) UpdateIsolatedMarginWithContext(ctx context.Context, amount float64, name string) (*UserState, error) {
	action := UpdateIsolatedMarginAction{
		Type:  "updateIsolatedMargin",
		Asset: e.info.NameToAsset(name),
		IsBuy: amount > 0,
		Ntli:  abs(amount),
	}

	var result UserState
	if err := e.executeAction(ctx, action, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SetExpiresAfter sets the expiration time for actions
// If expiresAfter is nil, actions will not have an expiration time
// If expiresAfter is set, actions will include this expiration nonce
func (e *Exchange) SetExpiresAfter(expiresAfter *int64) {
	e.expiresAfter = expiresAfter
}

// SlippagePrice calculates the slippage price for market orders
func (e *Exchange) SlippagePrice(
	name string,
	isBuy bool,
	slippage float64,
	px *float64,
) (float64, error) {
	coin := e.info.nameToCoin[name]
	var price float64

	if px != nil {
		price = *px
	} else {
		// Get midprice
		mids, err := e.info.AllMids()
		if err != nil {
			return 0, err
		}
		if midPriceStr, exists := mids[coin]; exists {
			price = parseFloat(midPriceStr)
		} else {
			return 0, fmt.Errorf("could not get mid price for coin: %s", coin)
		}
	}

	asset := e.info.coinToAsset[coin]
	isSpot := asset >= 10000

	// Calculate slippage
	if isBuy {
		price *= (1 + slippage)
	} else {
		price *= (1 - slippage)
	}

	// First we need to round the price to Hyperliquid's max 5 significant figures (see: https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api/tick-and-lot-size)
	price, err := roundToSignificantFigures(price, 5)
	if err != nil {
		return 0, err
	}

	// Round to appropriate decimals
	decimals := 6
	if isSpot {
		decimals = 8
	}
	szDecimals := e.info.assetToDecimal[asset]

	return roundToDecimals(price, decimals-szDecimals), nil
}

// ScheduleCancel schedules cancellation of all open orders
func (e *Exchange) ScheduleCancel(scheduleTime *int64) (*ScheduleCancelResponse, error) {
	return e.ScheduleCancelWithContext(context.Background(), scheduleTime)
}
func (e *Exchange) ScheduleCancelWithContext(ctx context.Context, scheduleTime *int64) (*ScheduleCancelResponse, error) {
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

// SetReferrer sets a referral code
func (e *Exchange) SetReferrer(code string) (*SetReferrerResponse, error) {
	return e.SetReferrerWithContext(context.Background(), code)
}
func (e *Exchange) SetReferrerWithContext(ctx context.Context, code string) (*SetReferrerResponse, error) {
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
func (e *Exchange) CreateSubAccount(name string) (*CreateSubAccountResponse, error) {
	return e.CreateSubAccountWithContext(context.Background(), name)
}
func (e *Exchange) CreateSubAccountWithContext(ctx context.Context, name string) (*CreateSubAccountResponse, error) {
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
func (e *Exchange) UsdClassTransfer(amount float64, toPerp bool) (*TransferResponse, error) {
	return e.UsdClassTransferWithContext(context.Background(), amount, toPerp)
}
func (e *Exchange) UsdClassTransferWithContext(ctx context.Context, amount float64, toPerp bool) (*TransferResponse, error) {
	nonce := e.nextNonce()

	strAmount := formatFloat(amount)
	if e.vault != "" {
		strAmount += " subaccount:" + e.vault
	}

	action := UsdClassTransferAction{
		Type:   "usdClassTransfer",
		Amount: strAmount,
		ToPerp: toPerp,
		Nonce:  nonce,
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

// SubAccountTransfer transfers funds to/from sub-account
func (e *Exchange) SubAccountTransfer(subAccountUser string, isDeposit bool, usd int) (*TransferResponse, error) {
	return e.SubAccountTransferWithContext(context.Background(), subAccountUser, isDeposit, usd)
}
func (e *Exchange) SubAccountTransferWithContext(ctx context.Context,
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
func (e *Exchange) VaultUsdTransfer(vaultAddress string, isDeposit bool, usd int) (*TransferResponse, error) {
	return e.VaultUsdTransferWithContext(context.Background(), vaultAddress, isDeposit, usd)
}
func (e *Exchange) VaultUsdTransferWithContext(ctx context.Context,
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
func (e *Exchange) CreateVault(name string, description string, initialUsd int) (*CreateVaultResponse, error) {
	return e.CreateVaultWithContext(context.Background(), name, description, initialUsd)
}
func (e *Exchange) CreateVaultWithContext(ctx context.Context,
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
func (e *Exchange) VaultModify(vaultAddress string, allowDeposits bool, alwaysCloseOnWithdraw bool) (*TransferResponse, error) {
	return e.VaultModifyWithContext(context.Background(), vaultAddress, allowDeposits, alwaysCloseOnWithdraw)
}
func (e *Exchange) VaultModifyWithContext(ctx context.Context,
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
func (e *Exchange) VaultDistribute(vaultAddress string, usd int) (*TransferResponse, error) {
	return e.VaultDistributeWithContext(context.Background(), vaultAddress, usd)
}
func (e *Exchange) VaultDistributeWithContext(ctx context.Context, vaultAddress string, usd int) (*TransferResponse, error) {
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
func (e *Exchange) UsdTransfer(amount float64, destination string) (*TransferResponse, error) {
	return e.UsdTransferWithContext(context.Background(), amount, destination)
}
func (e *Exchange) UsdTransferWithContext(ctx context.Context, amount float64, destination string) (*TransferResponse, error) {
	nonce := e.nextNonce()

	action := UsdTransferAction{
		Type:        "usdSend",
		Destination: destination,
		Amount:      formatFloat(amount),
		Time:        nonce,
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

// SpotTransfer transfers spot tokens to another address
func (e *Exchange) SpotTransfer(amount float64, destination, token string) (*TransferResponse, error) {
	return e.SpotTransferWithContext(context.Background(), amount, destination, token)
}
func (e *Exchange) SpotTransferWithContext(ctx context.Context,
	amount float64,
	destination, token string,
) (*TransferResponse, error) {
	nonce := e.nextNonce()

	action := SpotTransferAction{
		Type:        "spotSend",
		Destination: destination,
		Amount:      formatFloat(amount),
		Token:       token,
		Time:        nonce,
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

// UseBigBlocks enables or disables big blocks
func (e *Exchange) UseBigBlocks(enable bool) (*ApprovalResponse, error) {
	return e.UseBigBlocksWithContext(context.Background(), enable)
}

func (e *Exchange) UseBigBlocksWithContext(ctx context.Context, enable bool) (*ApprovalResponse, error) {
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
func (e *Exchange) PerpDexClassTransfer(dex, token string, amount float64, toPerp bool) (*TransferResponse, error) {
	return e.PerpDexClassTransferWithContext(context.Background(), dex, token, amount, toPerp)
}

func (e *Exchange) PerpDexClassTransferWithContext(ctx context.Context,
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
func (e *Exchange) SubAccountSpotTransfer(subAccountUser string, isDeposit bool, token string, amount float64) (*TransferResponse, error) {
	return e.SubAccountSpotTransferWithContext(context.Background(), subAccountUser, isDeposit, token, amount)
}

func (e *Exchange) SubAccountSpotTransferWithContext(ctx context.Context,
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
func (e *Exchange) TokenDelegate(validator string, wei int, isUndelegate bool) (*TransferResponse, error) {
	return e.TokenDelegateWithContext(context.Background(), validator, wei, isUndelegate)
}

func (e *Exchange) TokenDelegateWithContext(ctx context.Context,
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
func (e *Exchange) WithdrawFromBridge(amount float64, destination string) (*TransferResponse, error) {
	return e.WithdrawFromBridgeWithContext(context.Background(), amount, destination)
}

func (e *Exchange) WithdrawFromBridgeWithContext(ctx context.Context,
	amount float64,
	destination string,
) (*TransferResponse, error) {
	nonce := e.nextNonce()

	action := WithdrawFromBridgeAction{
		Type:        "withdraw3",
		Destination: destination,
		Amount:      fmt.Sprintf("%.6f", amount),
		Time:        nonce,
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

// ApproveAgent approves an agent to trade on behalf of the user
// Returns the result and the generated agent private key
func (e *Exchange) ApproveAgent(name *string) (*AgentApprovalResponse, string, error) {
	return e.ApproveAgentWithContext(context.Background(), name)
}

func (e *Exchange) ApproveAgentWithContext(ctx context.Context, name *string) (*AgentApprovalResponse, string, error) {
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

	action := ApproveAgentAction{
		Type:         "approveAgent",
		AgentAddress: agentAddress,
		AgentName:    name,
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
		return nil, "", err
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
func (e *Exchange) ApproveBuilderFee(builder string, maxFeeRate string) (*ApprovalResponse, error) {
	return e.ApproveBuilderFeeWithContext(context.Background(), builder, maxFeeRate)
}

func (e *Exchange) ApproveBuilderFeeWithContext(ctx context.Context, builder string, maxFeeRate string) (*ApprovalResponse, error) {
	nonce := e.nextNonce()

	action := ApproveBuilderFeeAction{
		Type:       "approveBuilderFee",
		Builder:    builder,
		MaxFeeRate: maxFeeRate,
		Nonce:      nonce,
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

	var result ApprovalResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ConvertToMultiSigUser converts account to multi-signature user
func (e *Exchange) ConvertToMultiSigUser(authorizedUsers []string, threshold int) (*MultiSigConversionResponse, error) {
	return e.ConvertToMultiSigUserWithContext(context.Background(), authorizedUsers, threshold)
}

func (e *Exchange) ConvertToMultiSigUserWithContext(ctx context.Context,
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
func (e *Exchange) SpotDeployRegisterToken(tokenName string, szDecimals int, weiDecimals int, maxGas int, fullName string) (*SpotDeployResponse, error) {
	return e.SpotDeployRegisterTokenWithContext(context.Background(), tokenName, szDecimals, weiDecimals, maxGas, fullName)
}

func (e *Exchange) SpotDeployRegisterTokenWithContext(ctx context.Context,
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
func (e *Exchange) SpotDeployUserGenesis(balances map[string]float64) (*SpotDeployResponse, error) {
	return e.SpotDeployUserGenesisWithContext(context.Background(), balances)
}

func (e *Exchange) SpotDeployUserGenesisWithContext(ctx context.Context, balances map[string]float64) (*SpotDeployResponse, error) {
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
func (e *Exchange) SpotDeployEnableFreezePrivilege() (*SpotDeployResponse, error) {
	return e.SpotDeployEnableFreezePrivilegeWithContext(context.Background())
}

func (e *Exchange) SpotDeployEnableFreezePrivilegeWithContext(ctx context.Context) (*SpotDeployResponse, error) {
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
func (e *Exchange) SpotDeployFreezeUser(userAddress string) (*SpotDeployResponse, error) {
	return e.SpotDeployFreezeUserWithContext(context.Background(), userAddress)
}

func (e *Exchange) SpotDeployFreezeUserWithContext(ctx context.Context, userAddress string) (*SpotDeployResponse, error) {
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
func (e *Exchange) SpotDeployRevokeFreezePrivilege() (*SpotDeployResponse, error) {
	return e.SpotDeployRevokeFreezePrivilegeWithContext(context.Background())
}

func (e *Exchange) SpotDeployRevokeFreezePrivilegeWithContext(ctx context.Context) (*SpotDeployResponse, error) {
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
func (e *Exchange) SpotDeployGenesis(deployer string, dexName string) (*SpotDeployResponse, error) {
	return e.SpotDeployGenesisWithContext(context.Background(), deployer, dexName)
}

func (e *Exchange) SpotDeployGenesisWithContext(ctx context.Context, deployer string, dexName string) (*SpotDeployResponse, error) {
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
func (e *Exchange) SpotDeployRegisterSpot(baseToken string, quoteToken string) (*SpotDeployResponse, error) {
	return e.SpotDeployRegisterSpotWithContext(context.Background(), baseToken, quoteToken)
}

func (e *Exchange) SpotDeployRegisterSpotWithContext(ctx context.Context,
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
func (e *Exchange) SpotDeployRegisterHyperliquidity(name string, tokens []string) (*SpotDeployResponse, error) {
	return e.SpotDeployRegisterHyperliquidityWithContext(context.Background(), name, tokens)
}

func (e *Exchange) SpotDeployRegisterHyperliquidityWithContext(ctx context.Context,
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
func (e *Exchange) SpotDeploySetDeployerTradingFeeShare(feeShare float64) (*SpotDeployResponse, error) {
	return e.SpotDeploySetDeployerTradingFeeShareWithContext(context.Background(), feeShare)
}

func (e *Exchange) SpotDeploySetDeployerTradingFeeShareWithContext(ctx context.Context,
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

// PerpDeployRegisterAsset registers a new perpetual asset
func (e *Exchange) PerpDeployRegisterAsset(asset string, perpDexInput PerpDexSchemaInput) (*PerpDeployResponse, error) {
	return e.PerpDeployRegisterAssetWithContext(context.Background(), asset, perpDexInput)
}

func (e *Exchange) PerpDeployRegisterAssetWithContext(ctx context.Context,
	asset string,
	perpDexInput PerpDexSchemaInput,
) (*PerpDeployResponse, error) {
	nonce := e.nextNonce()

	action := map[string]any{
		"type":         "perpDeployRegisterAsset",
		"asset":        asset,
		"perpDexInput": perpDexInput,
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

// PerpDeploySetOracle sets oracle for perpetual asset
func (e *Exchange) PerpDeploySetOracle(asset string, oracleAddress string) (*SpotDeployResponse, error) {
	return e.PerpDeploySetOracleWithContext(context.Background(), asset, oracleAddress)
}

func (e *Exchange) PerpDeploySetOracleWithContext(ctx context.Context,
	asset string,
	oracleAddress string,
) (*SpotDeployResponse, error) {
	nonce := e.nextNonce()

	action := map[string]any{
		"type":          "perpDeploySetOracle",
		"asset":         asset,
		"oracleAddress": oracleAddress,
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

// CSigner Methods

// CSignerUnjailSelf unjails self as consensus signer
func (e *Exchange) CSignerUnjailSelf() (*ValidatorResponse, error) {
	return e.CSignerUnjailSelfWithContext(context.Background())
}

func (e *Exchange) CSignerUnjailSelfWithContext(ctx context.Context) (*ValidatorResponse, error) {
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
func (e *Exchange) CSignerJailSelf() (*ValidatorResponse, error) {
	return e.CSignerJailSelfWithContext(context.Background())
}

func (e *Exchange) CSignerJailSelfWithContext(ctx context.Context) (*ValidatorResponse, error) {
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
func (e *Exchange) CSignerInner(innerAction map[string]any) (*ValidatorResponse, error) {
	return e.CSignerInnerWithContext(context.Background(), innerAction)
}

func (e *Exchange) CSignerInnerWithContext(ctx context.Context, innerAction map[string]any) (*ValidatorResponse, error) {
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
func (e *Exchange) CValidatorRegister(validatorProfile map[string]any) (*ValidatorResponse, error) {
	return e.CValidatorRegisterWithContext(context.Background(), validatorProfile)
}

func (e *Exchange) CValidatorRegisterWithContext(ctx context.Context, validatorProfile map[string]any) (*ValidatorResponse, error) {
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
func (e *Exchange) CValidatorChangeProfile(newProfile map[string]any) (*ValidatorResponse, error) {
	return e.CValidatorChangeProfileWithContext(context.Background(), newProfile)
}

func (e *Exchange) CValidatorChangeProfileWithContext(ctx context.Context, newProfile map[string]any) (*ValidatorResponse, error) {
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
func (e *Exchange) CValidatorUnregister() (*ValidatorResponse, error) {
	return e.CValidatorUnregisterWithContext(context.Background())
}

func (e *Exchange) CValidatorUnregisterWithContext(ctx context.Context) (*ValidatorResponse, error) {
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
func (e *Exchange) MultiSig(action map[string]any, signers []string, signatures []string) (*MultiSigResponse, error) {
	return e.MultiSigWithContext(context.Background(), action, signers, signatures)
}

func (e *Exchange) MultiSigWithContext(ctx context.Context,
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
