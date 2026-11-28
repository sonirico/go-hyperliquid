package hyperliquid

import (
	"encoding/json"
	"errors"

	"github.com/vmihailenco/msgpack/v5"
)

//go:generate easyjson -all

// Action structs with deterministic field ordering for consistent MessagePack serialization
// The order of fields in these structs is critical for signature generation

// CancelOrderWire represents cancel order item wire format
type CancelOrderWire struct {
	Asset   int   `json:"a" msgpack:"a"`
	OrderID int64 `json:"o" msgpack:"o"`
}

// CancelAction represents the cancel action
type CancelAction struct {
	Type    string            `json:"type"    msgpack:"type"`
	Cancels []CancelOrderWire `json:"cancels" msgpack:"cancels"`
}

// CancelByCloidWire represents cancel by cloid item wire format
// NB: the CancelByCloidWire MUST have `asset` and not `o` like
// CancelOrderWire has
// See: https://github.com/hyperliquid-dex/hyperliquid-python-sdk/blob/f19056ca1b65cc15a019d92dffa9ada887b3d808/hyperliquid/exchange.py#L305-L310
type CancelByCloidWire struct {
	Asset    int    `json:"asset" msgpack:"asset"`
	ClientID string `json:"cloid" msgpack:"cloid"`
}

// CancelByCloidAction represents the cancel by cloid action
type CancelByCloidAction struct {
	Type    string              `json:"type"    msgpack:"type"`
	Cancels []CancelByCloidWire `json:"cancels" msgpack:"cancels"`
}

// UsdClassTransferAction represents USD class transfer
type UsdClassTransferAction struct {
	Type   string `json:"type"   msgpack:"type"`
	Amount string `json:"amount" msgpack:"amount"`
	ToPerp bool   `json:"toPerp" msgpack:"toPerp"`
	Nonce  int64  `json:"nonce"  msgpack:"nonce"`
}

// SpotTransferAction represents spot transfer
type SpotTransferAction struct {
	Type        string `json:"type"        msgpack:"type"`
	Destination string `json:"destination" msgpack:"destination"`
	Amount      string `json:"amount"      msgpack:"amount"`
	Token       string `json:"token"       msgpack:"token"`
	Time        int64  `json:"time"        msgpack:"time"`
}

// UsdTransferAction represents USD transfer
type UsdTransferAction struct {
	Type        string `json:"type"        msgpack:"type"`
	Destination string `json:"destination" msgpack:"destination"`
	Amount      string `json:"amount"      msgpack:"amount"`
	Time        int64  `json:"time"        msgpack:"time"`
}

// SubAccountTransferAction represents sub-account transfer
type SubAccountTransferAction struct {
	Type           string `json:"type"           msgpack:"type"`
	SubAccountUser string `json:"subAccountUser" msgpack:"subAccountUser"`
	IsDeposit      bool   `json:"isDeposit"      msgpack:"isDeposit"`
	Usd            int    `json:"usd"            msgpack:"usd"`
}

// VaultUsdTransferAction represents vault USD transfer
type VaultUsdTransferAction struct {
	Type         string `json:"type"         msgpack:"type"`
	VaultAddress string `json:"vaultAddress" msgpack:"vaultAddress"`
	IsDeposit    bool   `json:"isDeposit"    msgpack:"isDeposit"`
	Usd          int    `json:"usd"          msgpack:"usd"`
}

// CreateVaultAction represents create vault action
type CreateVaultAction struct {
	Type        string `json:"type"        msgpack:"type"`
	Name        string `json:"name"        msgpack:"name"`
	Description string `json:"description" msgpack:"description"`
	InitialUsd  int    `json:"initialUsd"  msgpack:"initialUsd"`
	Nonce       int64  `json:"nonce"       msgpack:"nonce"`
}

// VaultModifyAction represents vault modify action
type VaultModifyAction struct {
	Type                  string `json:"type"                  msgpack:"type"`
	VaultAddress          string `json:"vaultAddress"          msgpack:"vaultAddress"`
	AllowDeposits         bool   `json:"allowDeposits"         msgpack:"allowDeposits"`
	AlwaysCloseOnWithdraw bool   `json:"alwaysCloseOnWithdraw" msgpack:"alwaysCloseOnWithdraw"`
}

// VaultDistributeAction represents vault distribute action
type VaultDistributeAction struct {
	Type         string `json:"type"         msgpack:"type"`
	VaultAddress string `json:"vaultAddress" msgpack:"vaultAddress"`
	Usd          int    `json:"usd"          msgpack:"usd"`
}

// UpdateLeverageAction represents leverage update
type UpdateLeverageAction struct {
	Type     string `json:"type"     msgpack:"type"`
	Asset    int    `json:"asset"    msgpack:"asset"`
	IsCross  bool   `json:"isCross"  msgpack:"isCross"`
	Leverage int    `json:"leverage" msgpack:"leverage"`
}

// UpdateIsolatedMarginAction represents isolated margin update
type UpdateIsolatedMarginAction struct {
	Type  string `json:"type"  msgpack:"type"`
	Asset int    `json:"asset" msgpack:"asset"`
	IsBuy bool   `json:"isBuy" msgpack:"isBuy"`
	Ntli  int64  `json:"ntli"  msgpack:"ntli"`
}

// OrderWire represents the wire format for orders with deterministic field ordering
// CRITICAL: Field order MUST exactly match Python SDK insertion order: a, b, p, s, r, t, c
// See hyperliquid-python-sdk/hyperliquid/utils/signing.py:order_request_to_order_wire
type OrderWire struct {
	Asset      int           `json:"a"           msgpack:"a"`           // 1st
	IsBuy      bool          `json:"b"           msgpack:"b"`           // 2nd
	LimitPx    string        `json:"p"           msgpack:"p"`           // 3rd
	Size       string        `json:"s"           msgpack:"s"`           // 4th
	ReduceOnly bool          `json:"r"           msgpack:"r"`           // 5th
	OrderType  OrderWireType `json:"t"           msgpack:"t"`           // 6th
	Cloid      *string       `json:"c,omitempty" msgpack:"c,omitempty"` // 7th (optional)
}

// OrderWireType represents the type of order (limit or trigger).
type OrderWireType struct {
	Limit   *OrderWireTypeLimit   `json:"limit,omitempty"   msgpack:"limit,omitempty"`
	Trigger *OrderWireTypeTrigger `json:"trigger,omitempty" msgpack:"trigger,omitempty"`
}

// OrderWireTypeLimit represents a limit order with time-in-force.
type OrderWireTypeLimit struct {
	Tif Tif `json:"tif,string" msgpack:"tif"`
}

// OrderWireTypeTrigger represents a trigger order (stop-loss/take-profit).
type OrderWireTypeTrigger struct {
	// CRITICAL: Field order MUST match Python SDK insertion order: isMarket, triggerPx, tpsl
	// See hyperliquid-python-sdk/hyperliquid/utils/signing.py:order_type_to_wire (lines 151-154)
	IsMarket  bool   `json:"isMarket"  msgpack:"isMarket"`  // 1st
	TriggerPx string `json:"triggerPx" msgpack:"triggerPx"` // 2nd - Must be string for msgpack serialization
	Tpsl      Tpsl   `json:"tpsl"      msgpack:"tpsl"`      // 3rd - "tp" or "sl"
}

// OrderGrouping represents the grouping of an order action: a name, or a
// priority sent as {"p": n}
//
//easyjson:skip
type OrderGrouping struct {
	name     string
	priority *int
}

func NewOrderGroupingNA() OrderGrouping {
	return OrderGrouping{name: string(GroupingNA)}
}

func NewOrderGroupingNormalTpsl() OrderGrouping {
	return OrderGrouping{name: string(GroupingNormalTpsl)}
}

func NewOrderGroupingPositionTpsl() OrderGrouping {
	return OrderGrouping{name: string(GroupingPositionTpls)}
}

func NewOrderGroupingPriority(n int) OrderGrouping {
	return OrderGrouping{priority: &n}
}

func (g OrderGrouping) value() any {
	if g.priority != nil {
		return map[string]int{"p": *g.priority}
	}
	return g.name
}

func (g OrderGrouping) EncodeMsgpack(enc *msgpack.Encoder) error {
	return enc.Encode(g.value())
}

func (g OrderGrouping) MarshalJSON() ([]byte, error) {
	return json.Marshal(g.value())
}

func (g *OrderGrouping) UnmarshalJSON(data []byte) error {
	var priority struct {
		P *int `json:"p"`
	}
	if err := json.Unmarshal(data, &priority); err == nil && priority.P != nil {
		*g = NewOrderGroupingPriority(*priority.P)
		return nil
	}
	var name string
	if err := json.Unmarshal(data, &name); err != nil {
		return err
	}
	*g = OrderGrouping{name: name}
	return nil
}

// OrderAction represents the order action with deterministic field ordering
// CRITICAL: Field order MUST match Python SDK insertion order for msgpack hash consistency
type OrderAction struct {
	Type     string        `json:"type"              msgpack:"type"`
	Orders   []OrderWire   `json:"orders"            msgpack:"orders"`
	Grouping OrderGrouping `json:"grouping"          msgpack:"grouping"`
	Builder  *BuilderInfo  `json:"builder,omitempty" msgpack:"builder,omitempty"`
}

// ModifyAction represents a single order modification
type ModifyAction struct {
	Type  string    `json:"type,omitempty" msgpack:"type,omitempty"`
	Oid   any       `json:"oid"            msgpack:"oid"`
	Order OrderWire `json:"order"          msgpack:"order"`
}

// BatchModifyAction represents multiple order modifications
type BatchModifyAction struct {
	Type     string         `json:"type"     msgpack:"type"`
	Modifies []ModifyAction `json:"modifies" msgpack:"modifies"`
}

// SubAccountSpotTransferAction represents sub-account spot transfer
type SubAccountSpotTransferAction struct {
	Type           string `json:"type"           msgpack:"type"`
	SubAccountUser string `json:"subAccountUser" msgpack:"subAccountUser"`
	IsDeposit      bool   `json:"isDeposit"      msgpack:"isDeposit"`
	Token          string `json:"token"          msgpack:"token"`
	Amount         string `json:"amount"         msgpack:"amount"`
}

// BorrowLendAction represents borrow/lend action
type BorrowLendAction struct {
	Type      string `json:"type"      msgpack:"type"`
	Operation string `json:"operation" msgpack:"operation"`
	Token     int64  `json:"token"     msgpack:"token"`
	Amount    string `json:"amount"    msgpack:"amount"`
}

// SpotDeployTokenSpec represents the token spec of a spot deploy registerToken2
type SpotDeployTokenSpec struct {
	Name        string `json:"name"        msgpack:"name"`
	SzDecimals  int    `json:"szDecimals"  msgpack:"szDecimals"`
	WeiDecimals int    `json:"weiDecimals" msgpack:"weiDecimals"`
}

// SpotDeployRegisterToken2 represents the registerToken2 variant of a spot deploy action
type SpotDeployRegisterToken2 struct {
	Spec     SpotDeployTokenSpec `json:"spec"     msgpack:"spec"`
	MaxGas   int                 `json:"maxGas"   msgpack:"maxGas"`
	FullName string              `json:"fullName" msgpack:"fullName"`
}

// SpotDeployUserGenesis represents the userGenesis variant of a spot deploy action
type SpotDeployUserGenesis struct {
	Token               int         `json:"token"               msgpack:"token"`
	UserAndWei          [][2]string `json:"userAndWei"          msgpack:"userAndWei"`
	ExistingTokenAndWei [][2]any    `json:"existingTokenAndWei" msgpack:"existingTokenAndWei"`
}

// SpotDeployEnableFreezePrivilege represents the enableFreezePrivilege variant of a spot deploy action
type SpotDeployEnableFreezePrivilege struct {
	Token int `json:"token" msgpack:"token"`
}

// SpotDeployFreezeUser represents the freezeUser variant of a spot deploy action
type SpotDeployFreezeUser struct {
	Token  int    `json:"token"  msgpack:"token"`
	User   string `json:"user"   msgpack:"user"`
	Freeze bool   `json:"freeze" msgpack:"freeze"`
}

// SpotDeployRevokeFreezePrivilege represents the revokeFreezePrivilege variant of a spot deploy action
type SpotDeployRevokeFreezePrivilege struct {
	Token int `json:"token" msgpack:"token"`
}

// SpotDeployGenesis represents the genesis variant of a spot deploy action
type SpotDeployGenesis struct {
	Token            int    `json:"token"                      msgpack:"token"`
	MaxSupply        string `json:"maxSupply"                  msgpack:"maxSupply"`
	NoHyperliquidity bool   `json:"noHyperliquidity,omitempty" msgpack:"noHyperliquidity,omitempty"`
}

// SpotDeployRegisterSpot represents the registerSpot variant of a spot deploy action
type SpotDeployRegisterSpot struct {
	Tokens [2]int `json:"tokens" msgpack:"tokens"`
}

// SpotDeployRegisterHyperliquidity represents the registerHyperliquidity variant of a spot deploy action
type SpotDeployRegisterHyperliquidity struct {
	Spot          int    `json:"spot"                    msgpack:"spot"`
	StartPx       string `json:"startPx"                 msgpack:"startPx"`
	OrderSz       string `json:"orderSz"                 msgpack:"orderSz"`
	NOrders       int    `json:"nOrders"                 msgpack:"nOrders"`
	NSeededLevels *int   `json:"nSeededLevels,omitempty" msgpack:"nSeededLevels,omitempty"`
}

// SpotDeploySetDeployerTradingFeeShare represents the setDeployerTradingFeeShare variant of a spot deploy action
type SpotDeploySetDeployerTradingFeeShare struct {
	Token int    `json:"token" msgpack:"token"`
	Share string `json:"share" msgpack:"share"`
}

// SpotDeployEnableQuoteToken represents the enableQuoteToken variant of a spot deploy action
type SpotDeployEnableQuoteToken struct {
	Token int `json:"token" msgpack:"token"`
}

// SpotDeployAction represents spot deploy action, exactly one variant is set
type SpotDeployAction struct {
	Type                       string                                `json:"type"                                 msgpack:"type"`
	RegisterToken2             *SpotDeployRegisterToken2             `json:"registerToken2,omitempty"             msgpack:"registerToken2,omitempty"`
	UserGenesis                *SpotDeployUserGenesis                `json:"userGenesis,omitempty"                msgpack:"userGenesis,omitempty"`
	EnableFreezePrivilege      *SpotDeployEnableFreezePrivilege      `json:"enableFreezePrivilege,omitempty"      msgpack:"enableFreezePrivilege,omitempty"`
	FreezeUser                 *SpotDeployFreezeUser                 `json:"freezeUser,omitempty"                 msgpack:"freezeUser,omitempty"`
	RevokeFreezePrivilege      *SpotDeployRevokeFreezePrivilege      `json:"revokeFreezePrivilege,omitempty"      msgpack:"revokeFreezePrivilege,omitempty"`
	Genesis                    *SpotDeployGenesis                    `json:"genesis,omitempty"                    msgpack:"genesis,omitempty"`
	RegisterSpot               *SpotDeployRegisterSpot               `json:"registerSpot,omitempty"               msgpack:"registerSpot,omitempty"`
	RegisterHyperliquidity     *SpotDeployRegisterHyperliquidity     `json:"registerHyperliquidity,omitempty"     msgpack:"registerHyperliquidity,omitempty"`
	SetDeployerTradingFeeShare *SpotDeploySetDeployerTradingFeeShare `json:"setDeployerTradingFeeShare,omitempty" msgpack:"setDeployerTradingFeeShare,omitempty"`
	EnableQuoteToken           *SpotDeployEnableQuoteToken           `json:"enableQuoteToken,omitempty"           msgpack:"enableQuoteToken,omitempty"`
}

// ScheduleCancelAction represents schedule cancel action
type ScheduleCancelAction struct {
	Type string `json:"type"           msgpack:"type"`
	Time *int64 `json:"time,omitempty" msgpack:"time,omitempty"`
}

// SetReferrerAction represents set referrer action
type SetReferrerAction struct {
	Type string `json:"type" msgpack:"type"`
	Code string `json:"code" msgpack:"code"`
}

// ReserveRequestWeightAction reserves request weight capacity
// Weight reservation costs 0.0005 USDC per weight unit
type ReserveRequestWeightAction struct {
	Type   string `json:"type"   msgpack:"type"`
	Weight int    `json:"weight" msgpack:"weight"`
}

// CreateSubAccountAction represents create sub-account action
type CreateSubAccountAction struct {
	Type string `json:"type" msgpack:"type"`
	Name string `json:"name" msgpack:"name"`
}

// UseBigBlocksAction represents use big blocks action
type UseBigBlocksAction struct {
	Type           string `json:"type"           msgpack:"type"`
	UsingBigBlocks bool   `json:"usingBigBlocks" msgpack:"usingBigBlocks"`
}

// TokenDelegateAction represents token delegate action
type TokenDelegateAction struct {
	Type         string `json:"type"         msgpack:"type"`
	Validator    string `json:"validator"    msgpack:"validator"`
	Wei          int    `json:"wei"          msgpack:"wei"`
	IsUndelegate bool   `json:"isUndelegate" msgpack:"isUndelegate"`
	Nonce        int64  `json:"nonce"        msgpack:"nonce"`
}

// WithdrawFromBridgeAction represents withdraw from bridge action
type WithdrawFromBridgeAction struct {
	Type        string `json:"type"        msgpack:"type"`
	Destination string `json:"destination" msgpack:"destination"`
	Amount      string `json:"amount"      msgpack:"amount"`
	Time        int64  `json:"time"        msgpack:"time"`
}

// ApproveAgentAction represents approve agent action
type ApproveAgentAction struct {
	Type             string  `json:"type"                msgpack:"type"`
	SignatureChainId string  `json:"signatureChainId"    msgpack:"signatureChainId"`
	HyperliquidChain string  `json:"hyperliquidChain"    msgpack:"hyperliquidChain"`
	AgentAddress     string  `json:"agentAddress"        msgpack:"agentAddress"`
	AgentName        *string `json:"agentName,omitempty" msgpack:"agentName,omitempty"`
	Nonce            int64   `json:"nonce"               msgpack:"nonce"`
}

// ApproveBuilderFeeAction represents approve builder fee action
type ApproveBuilderFeeAction struct {
	Type       string `json:"type"       msgpack:"type"`
	Builder    string `json:"builder"    msgpack:"builder"`
	MaxFeeRate string `json:"maxFeeRate" msgpack:"maxFeeRate"`
	Nonce      int64  `json:"nonce"      msgpack:"nonce"`
}

// ConvertToMultiSigUserAction represents convert to multi-sig user action
type ConvertToMultiSigUserAction struct {
	Type    string `json:"type"    msgpack:"type"`
	Signers string `json:"signers" msgpack:"signers"`
	Nonce   int64  `json:"nonce"   msgpack:"nonce"`
}

// MultiSigPayload represents the payload of a multi-signature action
type MultiSigPayload struct {
	MultiSigUser string `json:"multiSigUser" msgpack:"multiSigUser"`
	OuterSigner  string `json:"outerSigner"  msgpack:"outerSigner"`
	Action       any    `json:"action"       msgpack:"action"`
}

// MultiSigAction represents multi-signature action. The type is not part of the
// msgpack encoding because the signature covers the action without it
type MultiSigAction struct {
	Type             string            `json:"type"             msgpack:"-"`
	SignatureChainId string            `json:"signatureChainId" msgpack:"signatureChainId"`
	Signatures       []SignatureResult `json:"signatures"       msgpack:"signatures"`
	Payload          MultiSigPayload   `json:"payload"          msgpack:"payload"`
}

type RegisterAssetSchema struct {
	FullName        string  `json:"fullName"        msgpack:"fullName"`
	CollateralToken int     `json:"collateralToken" msgpack:"collateralToken"`
	OracleUpdater   *string `json:"oracleUpdater"   msgpack:"oracleUpdater"`
}

type AssetRequest struct {
	Coin          string `json:"coin"          msgpack:"coin"`
	SzDecimals    int    `json:"szDecimals"    msgpack:"szDecimals"`
	OraclePx      string `json:"oraclePx"      msgpack:"oraclePx"`
	MarginTableID int    `json:"marginTableId" msgpack:"marginTableId"`
	OnlyIsolated  bool   `json:"onlyIsolated"  msgpack:"onlyIsolated"`
}

type AssetRequest2 struct {
	Coin          string `json:"coin"          msgpack:"coin"`
	SzDecimals    int    `json:"szDecimals"    msgpack:"szDecimals"`
	OraclePx      string `json:"oraclePx"      msgpack:"oraclePx"`
	MarginTableID int    `json:"marginTableId" msgpack:"marginTableId"`
	MarginMode    string `json:"marginMode"    msgpack:"marginMode"`
}

type RegisterAsset struct {
	MaxGas       *int                 `json:"maxGas"       msgpack:"maxGas"`
	AssetRequest AssetRequest         `json:"assetRequest" msgpack:"assetRequest"`
	Dex          string               `json:"dex"          msgpack:"dex"`
	Schema       *RegisterAssetSchema `json:"schema"       msgpack:"schema"`
}

type RegisterAsset2 struct {
	MaxGas       *int                 `json:"maxGas"       msgpack:"maxGas"`
	AssetRequest AssetRequest2        `json:"assetRequest" msgpack:"assetRequest"`
	Dex          string               `json:"dex"          msgpack:"dex"`
	Schema       *RegisterAssetSchema `json:"schema"       msgpack:"schema"`
}

type PerpDeployRegisterAssetAction struct {
	Type          string        `json:"type"          msgpack:"type"`
	RegisterAsset RegisterAsset `json:"registerAsset" msgpack:"registerAsset"`
}

type PerpDeployRegisterAsset2Action struct {
	Type           string         `json:"type"           msgpack:"type"`
	RegisterAsset2 RegisterAsset2 `json:"registerAsset2" msgpack:"registerAsset2"`
}

type HaltTrading struct {
	Coin     string `json:"coin"     msgpack:"coin"`
	IsHalted bool   `json:"isHalted" msgpack:"isHalted"`
}

type PerpDeployHaltTradingAction struct {
	Type        string      `json:"type"        msgpack:"type"`
	HaltTrading HaltTrading `json:"haltTrading" msgpack:"haltTrading"`
}

type SetOracle struct {
	Dex             string       `json:"dex"             msgpack:"dex"`
	OraclePxs       [][]string   `json:"oraclePxs"       msgpack:"oraclePxs"`
	MarkPxs         [][][]string `json:"markPxs"         msgpack:"markPxs"`
	ExternalPerpPxs [][]string   `json:"externalPerpPxs" msgpack:"externalPerpPxs"`
}

type PerpDeploySetOracleAction struct {
	Type      string    `json:"type"      msgpack:"type"`
	SetOracle SetOracle `json:"setOracle" msgpack:"setOracle"`
}

type SetFeeRecipient struct {
	Dex          string `json:"dex"          msgpack:"dex"`
	FeeRecipient string `json:"feeRecipient" msgpack:"feeRecipient"`
}

type PerpDeploySetFeeRecipientAction struct {
	Type            string          `json:"type"            msgpack:"type"`
	SetFeeRecipient SetFeeRecipient `json:"setFeeRecipient" msgpack:"setFeeRecipient"`
}

type PerpDeployDisableDexAction struct {
	Type       string `json:"type"       msgpack:"type"`
	DisableDex string `json:"disableDex" msgpack:"disableDex"`
}

type SubDeployer struct {
	Variant string `json:"variant" msgpack:"variant"`
	User    string `json:"user"    msgpack:"user"`
	Allowed bool   `json:"allowed" msgpack:"allowed"`
}

type SetSubDeployers struct {
	Dex          string        `json:"dex"          msgpack:"dex"`
	SubDeployers []SubDeployer `json:"subDeployers" msgpack:"subDeployers"`
}

type PerpDeploySetSubDeployersAction struct {
	Type            string          `json:"type"            msgpack:"type"`
	SetSubDeployers SetSubDeployers `json:"setSubDeployers" msgpack:"setSubDeployers"`
}

type PerpDeploySetFundingMultipliersAction struct {
	Type                  string  `json:"type"                  msgpack:"type"`
	SetFundingMultipliers [][]any `json:"setFundingMultipliers" msgpack:"setFundingMultipliers"`
}

type PerpDeploySetFundingInterestRatesAction struct {
	Type                    string  `json:"type"                    msgpack:"type"`
	SetFundingInterestRates [][]any `json:"setFundingInterestRates" msgpack:"setFundingInterestRates"`
}

type PerpDeploySetFundingClampsAction struct {
	Type             string  `json:"type"             msgpack:"type"`
	SetFundingClamps [][]any `json:"setFundingClamps" msgpack:"setFundingClamps"`
}

type PerpDeployMarginTier struct {
	LowerBound  int `json:"lowerBound"  msgpack:"lowerBound"`
	MaxLeverage int `json:"maxLeverage" msgpack:"maxLeverage"`
}

type PerpDeployMarginTable struct {
	Description string                 `json:"description" msgpack:"description"`
	MarginTiers []PerpDeployMarginTier `json:"marginTiers" msgpack:"marginTiers"`
}

type InsertMarginTable struct {
	Dex         string                `json:"dex"         msgpack:"dex"`
	MarginTable PerpDeployMarginTable `json:"marginTable" msgpack:"marginTable"`
}

type PerpDeployInsertMarginTableAction struct {
	Type              string            `json:"type"              msgpack:"type"`
	InsertMarginTable InsertMarginTable `json:"insertMarginTable" msgpack:"insertMarginTable"`
}

type PerpDeploySetMarginTableIdsAction struct {
	Type              string  `json:"type"              msgpack:"type"`
	SetMarginTableIds [][]any `json:"setMarginTableIds" msgpack:"setMarginTableIds"`
}

type PerpDeploySetOpenInterestCapsAction struct {
	Type                string  `json:"type"                msgpack:"type"`
	SetOpenInterestCaps [][]any `json:"setOpenInterestCaps" msgpack:"setOpenInterestCaps"`
}

type PerpDeploySetMarginModesAction struct {
	Type           string  `json:"type"           msgpack:"type"`
	SetMarginModes [][]any `json:"setMarginModes" msgpack:"setMarginModes"`
}

type PerpDeployDeployerFee struct {
	Scale      string `json:"scale"      msgpack:"scale"`
	GrowthMode bool   `json:"growthMode" msgpack:"growthMode"`
}

type PerpDeploySetDeployerFeesAction struct {
	Type            string  `json:"type"            msgpack:"type"`
	SetDeployerFees [][]any `json:"setDeployerFees" msgpack:"setDeployerFees"`
}

type SetPerpAnnotation struct {
	Coin        string   `json:"coin"        msgpack:"coin"`
	Category    string   `json:"category"    msgpack:"category"`
	Description string   `json:"description" msgpack:"description"`
	DisplayName *string  `json:"displayName" msgpack:"displayName"`
	Keywords    []string `json:"keywords"    msgpack:"keywords"`
}

type PerpDeploySetPerpAnnotationAction struct {
	Type              string            `json:"type"              msgpack:"type"`
	SetPerpAnnotation SetPerpAnnotation `json:"setPerpAnnotation" msgpack:"setPerpAnnotation"`
}

func encodeVariantMsgpack(enc *msgpack.Encoder, actionType, variant string, value any) error {
	if err := enc.EncodeMapLen(2); err != nil {
		return err
	}
	if err := enc.EncodeString("type"); err != nil {
		return err
	}
	if err := enc.EncodeString(actionType); err != nil {
		return err
	}
	if err := enc.EncodeString(variant); err != nil {
		return err
	}
	return enc.Encode(value)
}

func marshalVariantJSON(actionType, variant string, value any) ([]byte, error) {
	typeJSON, err := json.Marshal(actionType)
	if err != nil {
		return nil, err
	}
	variantJSON, err := json.Marshal(variant)
	if err != nil {
		return nil, err
	}
	valueJSON, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return []byte(
		`{"type":` + string(typeJSON) + `,` + string(variantJSON) + `:` + string(valueJSON) + `}`,
	), nil
}

// CSignerAction represents consensus signer action, exactly one variant is set
// and it is sent with a null value
//
//easyjson:skip
type CSignerAction struct {
	Type       string
	UnjailSelf *struct{}
	JailSelf   *struct{}
}

func (a CSignerAction) variant() (string, error) {
	switch {
	case a.UnjailSelf != nil && a.JailSelf == nil:
		return "unjailSelf", nil
	case a.JailSelf != nil && a.UnjailSelf == nil:
		return "jailSelf", nil
	}
	return "", errors.New("CSignerAction needs exactly one variant")
}

func (a CSignerAction) EncodeMsgpack(enc *msgpack.Encoder) error {
	variant, err := a.variant()
	if err != nil {
		return err
	}
	return encodeVariantMsgpack(enc, a.Type, variant, nil)
}

func (a CSignerAction) MarshalJSON() ([]byte, error) {
	variant, err := a.variant()
	if err != nil {
		return nil, err
	}
	return marshalVariantJSON(a.Type, variant, nil)
}

// CValidatorNodeIp represents the node ip of a validator profile
type CValidatorNodeIp struct {
	Ip string `json:"Ip" msgpack:"Ip"`
}

// CValidatorProfile represents the profile of a validator register
type CValidatorProfile struct {
	NodeIp              CValidatorNodeIp `json:"node_ip"              msgpack:"node_ip"`
	Name                string           `json:"name"                 msgpack:"name"`
	Description         string           `json:"description"          msgpack:"description"`
	DelegationsDisabled bool             `json:"delegations_disabled" msgpack:"delegations_disabled"`
	CommissionBps       int              `json:"commission_bps"       msgpack:"commission_bps"`
	Signer              string           `json:"signer"               msgpack:"signer"`
}

// CValidatorRegister represents the register variant of a validator action
type CValidatorRegister struct {
	Profile    CValidatorProfile `json:"profile"     msgpack:"profile"`
	Unjailed   bool              `json:"unjailed"    msgpack:"unjailed"`
	InitialWei int               `json:"initial_wei" msgpack:"initial_wei"`
}

// CValidatorChangeProfile represents the changeProfile variant of a validator action,
// unset fields are sent as null
type CValidatorChangeProfile struct {
	NodeIp             *CValidatorNodeIp `json:"node_ip"             msgpack:"node_ip"`
	Name               *string           `json:"name"                msgpack:"name"`
	Description        *string           `json:"description"         msgpack:"description"`
	Unjailed           bool              `json:"unjailed"            msgpack:"unjailed"`
	DisableDelegations *bool             `json:"disable_delegations" msgpack:"disable_delegations"`
	CommissionBps      *int              `json:"commission_bps"      msgpack:"commission_bps"`
	Signer             *string           `json:"signer"              msgpack:"signer"`
}

// CValidatorAction represents consensus validator action, exactly one variant is set
//
//easyjson:skip
type CValidatorAction struct {
	Type          string
	Register      *CValidatorRegister
	ChangeProfile *CValidatorChangeProfile
	Unregister    *struct{}
}

func (a CValidatorAction) variant() (string, any, error) {
	switch {
	case a.Register != nil && a.ChangeProfile == nil && a.Unregister == nil:
		return "register", a.Register, nil
	case a.ChangeProfile != nil && a.Register == nil && a.Unregister == nil:
		return "changeProfile", a.ChangeProfile, nil
	case a.Unregister != nil && a.Register == nil && a.ChangeProfile == nil:
		return "unregister", nil, nil
	}
	return "", nil, errors.New("CValidatorAction needs exactly one variant")
}

func (a CValidatorAction) EncodeMsgpack(enc *msgpack.Encoder) error {
	variant, value, err := a.variant()
	if err != nil {
		return err
	}
	return encodeVariantMsgpack(enc, a.Type, variant, value)
}

func (a CValidatorAction) MarshalJSON() ([]byte, error) {
	variant, value, err := a.variant()
	if err != nil {
		return nil, err
	}
	return marshalVariantJSON(a.Type, variant, value)
}

// TrailingStopRetracement represents the retracement of a trailing stop, exactly one of
// Pct or Px is set
type TrailingStopRetracement struct {
	Pct string `json:"pct,omitempty" msgpack:"pct,omitempty"`
	Px  string `json:"px,omitempty"  msgpack:"px,omitempty"`
}

// TrailingStopAction represents trailing stop action, unset activationPx is sent as null
type TrailingStopAction struct {
	Type         string                  `json:"type"         msgpack:"type"`
	Asset        int                     `json:"asset"        msgpack:"asset"`
	IsBuy        bool                    `json:"isBuy"        msgpack:"isBuy"`
	Sz           string                  `json:"sz"           msgpack:"sz"`
	ReduceOnly   bool                    `json:"reduceOnly"   msgpack:"reduceOnly"`
	Retracement  TrailingStopRetracement `json:"retracement"  msgpack:"retracement"`
	ActivationPx *string                 `json:"activationPx" msgpack:"activationPx"`
}

// TopUpIsolatedOnlyMarginAction represents top up isolated only margin action
type TopUpIsolatedOnlyMarginAction struct {
	Type     string `json:"type"     msgpack:"type"`
	Asset    int    `json:"asset"    msgpack:"asset"`
	Leverage string `json:"leverage" msgpack:"leverage"`
}

// NoopAction represents noop action
type NoopAction struct {
	Type string `json:"type" msgpack:"type"`
}

// AgentSetAbstractionAction represents agent set abstraction action, abstraction is i, u or p
type AgentSetAbstractionAction struct {
	Type        string `json:"type"        msgpack:"type"`
	Abstraction string `json:"abstraction" msgpack:"abstraction"`
}

// GossipPriorityBidAction represents gossip priority bid action
type GossipPriorityBidAction struct {
	Type   string `json:"type"   msgpack:"type"`
	SlotID int    `json:"slotId" msgpack:"slotId"`
	Ip     string `json:"ip"     msgpack:"ip"`
	MaxGas int    `json:"maxGas" msgpack:"maxGas"`
}

// AgentSendAssetAction represents agent send asset action
type AgentSendAssetAction struct {
	Type           string `json:"type"           msgpack:"type"`
	Destination    string `json:"destination"    msgpack:"destination"`
	SourceDex      string `json:"sourceDex"      msgpack:"sourceDex"`
	DestinationDex string `json:"destinationDex" msgpack:"destinationDex"`
	Token          string `json:"token"          msgpack:"token"`
	Amount         string `json:"amount"         msgpack:"amount"`
	FromSubAccount string `json:"fromSubAccount" msgpack:"fromSubAccount"`
	Nonce          int64  `json:"nonce"          msgpack:"nonce"`
}

// CDeposit2Action represents c deposit 2 action
type CDeposit2Action struct {
	Type string `json:"type" msgpack:"type"`
	Wei  int    `json:"wei"  msgpack:"wei"`
}

// ClaimRewardsAction represents claim rewards action
type ClaimRewardsAction struct {
	Type string `json:"type" msgpack:"type"`
}

// Hip3LiquidatorTransferAction represents hip3 liquidator transfer action
type Hip3LiquidatorTransferAction struct {
	Type      string `json:"type"      msgpack:"type"`
	Dex       string `json:"dex"       msgpack:"dex"`
	Ntl       int    `json:"ntl"       msgpack:"ntl"`
	IsDeposit bool   `json:"isDeposit" msgpack:"isDeposit"`
}

// UserOutcomeAction represents user outcome action, with one variant set
type UserOutcomeAction struct {
	Type          string               `json:"type"                    msgpack:"type"`
	SplitOutcome  *SplitOutcomeAction  `json:"splitOutcome,omitempty"  msgpack:"splitOutcome,omitempty"`
	MergeOutcome  *MergeOutcomeAction  `json:"mergeOutcome,omitempty"  msgpack:"mergeOutcome,omitempty"`
	MergeQuestion *MergeQuestionAction `json:"mergeQuestion,omitempty" msgpack:"mergeQuestion,omitempty"`
	NegateOutcome *NegateOutcomeAction `json:"negateOutcome,omitempty" msgpack:"negateOutcome,omitempty"`
}

// SplitOutcomeAction represents the split outcome variant of user outcome action
type SplitOutcomeAction struct {
	Outcome int    `json:"outcome" msgpack:"outcome"`
	Amount  string `json:"amount"  msgpack:"amount"`
}

// MergeOutcomeAction represents the merge outcome variant of user outcome action
type MergeOutcomeAction struct {
	Outcome int     `json:"outcome" msgpack:"outcome"`
	Amount  *string `json:"amount"  msgpack:"amount"`
}

// MergeQuestionAction represents the merge question variant of user outcome action
type MergeQuestionAction struct {
	Question int     `json:"question" msgpack:"question"`
	Amount   *string `json:"amount"   msgpack:"amount"`
}

// NegateOutcomeAction represents the negate outcome variant of user outcome action
type NegateOutcomeAction struct {
	Question int    `json:"question" msgpack:"question"`
	Outcome  int    `json:"outcome"  msgpack:"outcome"`
	Amount   string `json:"amount"   msgpack:"amount"`
}

// AuthorizeAqav2RoleAction represents authorize aqav2 role action
type AuthorizeAqav2RoleAction struct {
	Type  string `json:"type"  msgpack:"type"`
	Token int    `json:"token" msgpack:"token"`
	Role  string `json:"role"  msgpack:"role"`
}

// ValidatorL1StreamAction represents validator L1 stream action
type ValidatorL1StreamAction struct {
	Type         string `json:"type"         msgpack:"type"`
	RiskFreeRate string `json:"riskFreeRate" msgpack:"riskFreeRate"`
}
