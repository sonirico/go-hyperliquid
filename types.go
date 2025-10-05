package hyperliquid

//go:generate easyjson -all

type Side string

const (
	SideAsk Side = "A"
	SideBid Side = "B"
)

type Grouping string

const (
	GroupingNA           Grouping = "na"
	GroupingNormalTpsl   Grouping = "normalTpsl"
	GroupingPositionTpls Grouping = "positionTpsl"
)

// Constants for default values
const (
	DefaultSlippage = 0.05 // 5%
)

type Tif string

// Order Time-in-Force constants
const (
	TifAlo Tif = "Alo" // Add Liquidity Only
	TifIoc Tif = "Ioc" // Immediate or Cancel
	TifGtc Tif = "Gtc" // Good Till Cancel
)

type Tpsl string // Advanced order type

const (
	TakeProfit Tpsl = "tp"
	StopLoss   Tpsl = "sl"
)

type AssetInfo struct {
	Name          string `json:"name"`
	SzDecimals    int    `json:"szDecimals"`
	MaxLeverage   int    `json:"maxLeverage"`
	MarginTableId int    `json:"marginTableId"`
	OnlyIsolated  bool   `json:"onlyIsolated"`
	IsDelisted    bool   `json:"isDelisted"`
}

type MarginTier struct {
	LowerBound  string `json:"lowerBound"`
	MaxLeverage int    `json:"maxLeverage"`
}

type MarginTable struct {
	ID          int
	Description string       `json:"description"`
	MarginTiers []MarginTier `json:"marginTiers"`
}

type Meta struct {
	Universe     []AssetInfo   `json:"universe"`
	MarginTables []MarginTable `json:"marginTables"`
}

type AssetCtx struct {
	Funding      string   `json:"funding"`
	OpenInterest string   `json:"openInterest"`
	PrevDayPx    string   `json:"prevDayPx"`
	DayNtlVlm    string   `json:"dayNtlVlm"`
	Premium      string   `json:"premium"`
	OraclePx     string   `json:"oraclePx"`
	MarkPx       string   `json:"markPx"`
	MidPx        string   `json:"midPx,omitempty"`
	ImpactPxs    []string `json:"impactPxs"`
	DayBaseVlm   string   `json:"dayBaseVlm,omitempty"`
}

// This type has no JSON annotation because it cannot be directly unmarshalled from the response
type MetaAndAssetCtxs struct {
	Meta
	Ctxs []AssetCtx
}

type SpotAssetInfo struct {
	Name        string `json:"name"`
	Tokens      []int  `json:"tokens"`
	Index       int    `json:"index"`
	IsCanonical bool   `json:"isCanonical"`
}

type EvmContract struct {
	Address             string `json:"address"`
	EvmExtraWeiDecimals int    `json:"evm_extra_wei_decimals"`
}

type SpotTokenInfo struct {
	Name        string       `json:"name"`
	SzDecimals  int          `json:"szDecimals"`
	WeiDecimals int          `json:"weiDecimals"`
	Index       int          `json:"index"`
	TokenID     string       `json:"tokenId"`
	IsCanonical bool         `json:"isCanonical"`
	EvmContract *EvmContract `json:"evmContract"`
	FullName    *string      `json:"fullName"`
}

type SpotMeta struct {
	Universe []SpotAssetInfo `json:"universe"`
	Tokens   []SpotTokenInfo `json:"tokens"`
}

type SpotAssetCtx struct {
	DayNtlVlm         string  `json:"dayNtlVlm"`
	MarkPx            string  `json:"markPx"`
	MidPx             *string `json:"midPx"`
	PrevDayPx         string  `json:"prevDayPx"`
	CirculatingSupply string  `json:"circulatingSupply"`
	Coin              string  `json:"coin"`
}

// This type has no JSON annotation because it cannot be directly unmarshalled from the response
type SpotMetaAndAssetCtxs struct {
	Meta SpotMeta
	Ctxs []SpotAssetCtx
}

// WsMsg represents a WebSocket message with a channel and data payload.
type WsMsg struct {
	Channel string         `json:"channel"`
	Data    map[string]any `json:"data"`
}

type OrderType struct {
	Limit   *LimitOrderType   `json:"limit,omitempty"`
	Trigger *TriggerOrderType `json:"trigger,omitempty"`
}

type LimitOrderType struct {
	Tif Tif `json:"tif"` // TifAlo, TifIoc, TifGtc
}

type TriggerOrderType struct {
	TriggerPx float64 `json:"triggerPx"`
	IsMarket  bool    `json:"isMarket"`
	Tpsl      Tpsl    `json:"tpsl"` // "tp" or "sl"
}

type BuilderInfo struct {
	Builder string `json:"b"`
	Fee     int    `json:"f"`
}

type CancelRequest struct {
	Coin string `json:"coin"`
	Oid  int64  `json:"oid"`
}

type CancelByCloidRequest struct {
	Coin  string `json:"coin"`
	Cloid string `json:"cloid"`
}

type Cloid struct {
	Value string
}

func (c Cloid) ToRaw() string {
	return c.Value
}

type PerpDexSchemaInput struct {
	FullName        string  `json:"fullName"`
	CollateralToken int     `json:"collateralToken"`
	OracleUpdater   *string `json:"oracleUpdater"`
}

type AssetPosition struct {
	Position Position `json:"position"`
	Type     string   `json:"type"`
}

type Position struct {
	Coin           string   `json:"coin"`
	EntryPx        *string  `json:"entryPx"`
	Leverage       Leverage `json:"leverage"`
	LiquidationPx  *string  `json:"liquidationPx"`
	MarginUsed     string   `json:"marginUsed"`
	PositionValue  string   `json:"positionValue"`
	ReturnOnEquity string   `json:"returnOnEquity"`
	Szi            string   `json:"szi"`
	UnrealizedPnl  string   `json:"unrealizedPnl"`
}

type Leverage struct {
	Type   string  `json:"type"`
	Value  int     `json:"value"`
	RawUsd *string `json:"rawUsd,omitempty"`
}

type UserState struct {
	AssetPositions     []AssetPosition `json:"assetPositions"`
	CrossMarginSummary MarginSummary   `json:"crossMarginSummary"`
	MarginSummary      MarginSummary   `json:"marginSummary"`
	Withdrawable       string          `json:"withdrawable"`
}

type SpotBalance struct {
	Coin     string `json:"coin"`
	Token    int    `json:"token"`
	Hold     string `json:"hold"`
	Total    string `json:"total"`
	EntryNtl string `json:"entryNtl"`
}

type SpotUserState struct {
	Balances []SpotBalance `json:"balances"`
}

type MarginSummary struct {
	AccountValue    string `json:"accountValue"`
	TotalMarginUsed string `json:"totalMarginUsed"`
	TotalNtlPos     string `json:"totalNtlPos"`
	TotalRawUsd     string `json:"totalRawUsd"`
}

type OpenOrder struct {
	Coin      string  `json:"coin"`
	LimitPx   float64 `json:"limitPx,string"`
	Oid       int64   `json:"oid"`
	Side      string  `json:"side"`
	Size      float64 `json:"sz,string"`
	Timestamp int64   `json:"timestamp"`
}

type FrontendOpenOrder struct {
	Coin             string    `json:"coin"`
	IsPositionTpSl   bool      `json:"isPositionTpsl"`
	IsTrigger        bool      `json:"isTrigger"`
	LimitPx          float64   `json:"limitPx,string"`
	Oid              int64     `json:"oid"`
	OrderType        string    `json:"orderType"`
	OrigSz           float64   `json:"origSz,string"`
	ReduceOnly       bool      `json:"reduceOnly"`
	Side             OrderSide `json:"side"`
	Sz               float64   `json:"sz,string"`
	Timestamp        int64     `json:"timestamp"`
	TriggerCondition string    `json:"triggerCondition"`
	TriggerPx        string    `json:"triggerPx"`
}

type OrderSide string

const (
	OrderSideAsk OrderSide = "A"
	OrderSideBid OrderSide = "B"
)

type QueriedOrder struct {
	Coin             string    `json:"coin"`
	Side             OrderSide `json:"side"`
	LimitPx          string    `json:"limitPx"`
	Sz               string    `json:"sz"`
	Oid              int64     `json:"oid"`
	Timestamp        int64     `json:"timestamp"`
	TriggerCondition string    `json:"triggerCondition"`
	IsTrigger        bool      `json:"isTrigger"`
	TriggerPx        string    `json:"triggerPx"`
	IsPositionTpsl   bool      `json:"isPositionTpsl"`
	ReduceOnly       bool      `json:"reduceOnly"`
	OrderType        string    `json:"orderType"`
	OrigSz           string    `json:"origSz"`
	Tif              Tif       `json:"tif"`
	Cloid            *string   `json:"cloid"`
}

type OrderQueryResponse struct {
	Order           QueriedOrder     `json:"order"`
	Status          OrderStatusValue `json:"status"`
	StatusTimestamp int64            `json:"statusTimestamp"`
}

type OrderStatusValue string

const (
	// Placed successfully
	OrderStatusValueOpen OrderStatusValue = "open"
	// Filled
	OrderStatusValueFilled OrderStatusValue = "filled"
	// Canceled by user
	OrderStatusValueCanceled OrderStatusValue = "canceled"
	// Trigger order triggered
	OrderStatusValueTriggered OrderStatusValue = "triggered"
	// Rejected at time of placement
	OrderStatusValueRejected OrderStatusValue = "rejected"
	// Canceled because insufficient margin to fill
	OrderStatusValueMarginCanceled OrderStatusValue = "marginCanceled"
	// Vaults only. Canceled due to a user's withdrawal from vault
	OrderStatusValueVaultWithdrawalCanceled OrderStatusValue = "vaultWithdrawalCanceled"
	// Canceled due to order being too aggressive when open interest was at cap
	OrderStatusValueOpenInterestCapCanceled OrderStatusValue = "openInterestCapCanceled"
	// Canceled due to self-trade prevention
	OrderStatusValueSelfTradeCanceled OrderStatusValue = "selfTradeCanceled"
	// Canceled reduced-only order that does not reduce position
	OrderStatusValueReduceOnlyCanceled OrderStatusValue = "reduceOnlyCanceled"
	// TP/SL only. Canceled due to sibling ordering being filled
	OrderStatusValueSiblingFilledCanceled OrderStatusValue = "siblingFilledCanceled"
	// Canceled due to asset delisting
	OrderStatusValueDelistedCanceled OrderStatusValue = "delistedCanceled"
	// Canceled due to liquidation
	OrderStatusValueLiquidatedCanceled OrderStatusValue = "liquidatedCanceled"
	// API only. Canceled due to exceeding scheduled cancel deadline (dead man's switch)
	OrderStatusValueScheduledCancel OrderStatusValue = "scheduledCancel"
	// Rejected due to invalid tick price
	OrderStatusValueTickRejected OrderStatusValue = "tickRejected"
	// Rejected due to order notional below minimum
	OrderStatusValueMinTradeNtlRejected OrderStatusValue = "minTradeNtlRejected"
	// Rejected due to insufficient margin
	OrderStatusValuePerpMarginRejected OrderStatusValue = "perpMarginRejected"
	// Rejected due to reduce only
	OrderStatusValueReduceOnlyRejected OrderStatusValue = "reduceOnlyRejected"
	// Rejected due to post-only immediate match
	OrderStatusValueBadAloPxRejected OrderStatusValue = "badAloPxRejected"
	// Rejected due to IOC not able to match
	OrderStatusValueIocCancelRejected OrderStatusValue = "iocCancelRejected"
	// Rejected due to invalid TP/SL price
	OrderStatusValueBadTriggerPxRejected OrderStatusValue = "badTriggerPxRejected"
	// Rejected due to lack of liquidity for market order
	OrderStatusValueMarketOrderNoLiquidityRejected OrderStatusValue = "marketOrderNoLiquidityRejected"
	// Rejected due to open interest cap
	OrderStatusValuePositionIncreaseAtOpenInterestCapRejected OrderStatusValue = "positionIncreaseAtOpenInterestCapRejected"
	// Rejected due to open interest cap
	OrderStatusValuePositionFlipAtOpenInterestCapRejected OrderStatusValue = "positionFlipAtOpenInterestCapRejected"
	// Rejected due to price too aggressive at open interest cap
	OrderStatusValueTooAggressiveAtOpenInterestCapRejected OrderStatusValue = "tooAggressiveAtOpenInterestCapRejected"
	// Rejected due to open interest cap
	OrderStatusValueOpenInterestIncreaseRejected OrderStatusValue = "openInterestIncreaseRejected"
	// Rejected due to insufficient spot balance
	OrderStatusValueInsufficientSpotBalanceRejected OrderStatusValue = "insufficientSpotBalanceRejected"
	// Rejected due to price too far from oracle
	OrderStatusValueOracleRejected OrderStatusValue = "oracleRejected"
	// Rejected due to exceeding margin tier limit at current leverage
	OrderStatusValuePerpMaxPositionRejected OrderStatusValue = "perpMaxPositionRejected"
)

type OrderQueryStatus string

const (
	OrderQueryStatusSuccess OrderQueryStatus = "order"
	OrderQueryStatusError   OrderQueryStatus = "unknownOid"
)

type OrderQueryResult struct {
	Status OrderQueryStatus   `json:"status"`
	Order  OrderQueryResponse `json:"order,omitempty"`
}

type Fill struct {
	ClosedPnl     string `json:"closedPnl"`
	Coin          string `json:"coin"`
	Crossed       bool   `json:"crossed"`
	Dir           string `json:"dir"`
	Hash          string `json:"hash"`
	Oid           int64  `json:"oid"`
	Price         string `json:"px"`
	Side          string `json:"side"`
	StartPosition string `json:"startPosition"`
	Size          string `json:"sz"`
	Time          int64  `json:"time"`
	Fee           string `json:"fee"`
	FeeToken      string `json:"feeToken"`
	BuilderFee    string `json:"builderFee,omitempty"`
	Tid           int64  `json:"tid"`
}

type FundingHistory struct {
	Coin        string `json:"coin"`
	FundingRate string `json:"fundingRate"`
	Premium     string `json:"premium"`
	Time        int64  `json:"time"`
}

type UserFundingHistory struct {
	User      string `json:"user"`
	Type      string `json:"type"`
	StartTime int64  `json:"startTime"`
	EndTime   int64  `json:"endTime"`
}

type UserFees struct {
	ActiveReferralDiscount string       `json:"activeReferralDiscount"`
	DailyUserVolume        []UserVolume `json:"dailyUserVlm"`
	FeeSchedule            FeeSchedule  `json:"feeSchedule"`
	UserAddRate            string       `json:"userAddRate"`
	UserCrossRate          string       `json:"userCrossRate"`
	UserSpotCrossRate      string       `json:"userSpotCrossRate"`
	UserSpotAddRate        string       `json:"userSpotAddRate"`
}

type UserActiveAssetData struct {
	User             string   `json:"user"`
	Coin             string   `json:"coin"`
	Leverage         Leverage `json:"leverage"`
	MaxTradeSzs      []string `json:"maxTradeSzs"`
	AvailableToTrade []string `json:"availableToTrade"`
	MarkPx           string   `json:"markPx"`
}

type UserVolume struct {
	Date      string `json:"date"`
	Exchange  string `json:"exchange"`
	UserAdd   string `json:"userAdd"`
	UserCross string `json:"userCross"`
}

type FeeSchedule struct {
	Add              string `json:"add"`
	Cross            string `json:"cross"`
	ReferralDiscount string `json:"referralDiscount"`
	Tiers            Tiers  `json:"tiers"`
}

type Tiers struct {
	MM  []MMTier  `json:"mm"`
	VIP []VIPTier `json:"vip"`
}

type MMTier struct {
	Add                 string `json:"add"`
	MakerFractionCutoff string `json:"makerFractionCutoff"`
}

type VIPTier struct {
	Add       string `json:"add"`
	Cross     string `json:"cross"`
	NtlCutoff string `json:"ntlCutoff"`
}

type StakingSummary struct {
	Delegated              string `json:"delegated"`
	Undelegated            string `json:"undelegated"`
	TotalPendingWithdrawal string `json:"totalPendingWithdrawal"`
	NPendingWithdrawals    int    `json:"nPendingWithdrawals"`
}

type StakingDelegation struct {
	Validator            string `json:"validator"`
	Amount               string `json:"amount"`
	LockedUntilTimestamp int64  `json:"lockedUntilTimestamp"`
}

type StakingReward struct {
	Time        int64  `json:"time"`
	Source      string `json:"source"`
	TotalAmount string `json:"totalAmount"`
}

type ReferralState struct {
	ReferralCode string   `json:"referralCode"`
	Referrer     string   `json:"referrer"`
	Referred     []string `json:"referred"`
}

type SubAccount struct {
	Name        string   `json:"name"`
	User        string   `json:"user"`
	Permissions []string `json:"permissions"`
}

type MultiSigSigner struct {
	User      string `json:"user"`
	Threshold int    `json:"threshold"`
}

type BulkOrderResponse struct {
	Status string        `json:"status"`
	Data   []OrderStatus `json:"data,omitempty"`
	Error  string        `json:"error,omitempty"`
}

type CancelResponse struct {
	Status string     `json:"status"`
	Data   *OpenOrder `json:"data,omitempty"`
	Error  string     `json:"error,omitempty"`
}

type BulkCancelResponse struct {
	Status string      `json:"status"`
	Data   []OpenOrder `json:"data,omitempty"`
	Error  string      `json:"error,omitempty"`
}

type ModifyResponse struct {
	Status string        `json:"status"`
	Data   []OrderStatus `json:"data,omitempty"`
	Error  string        `json:"error,omitempty"`
}

type TransferResponse struct {
	Status string `json:"status"`
	TxHash string `json:"txHash,omitempty"`
	Error  string `json:"error,omitempty"`
}

type ApprovalResponse struct {
	Status string `json:"status"`
	TxHash string `json:"txHash,omitempty"`
	Error  string `json:"error,omitempty"`
}

type CreateVaultResponse struct {
	Status string `json:"status"`
	Data   string `json:"data,omitempty"`
	Error  string `json:"error,omitempty"`
}

type CreateSubAccountResponse struct {
	Status string      `json:"status"`
	Data   *SubAccount `json:"data,omitempty"`
	Error  string      `json:"error,omitempty"`
}

type SetReferrerResponse struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type ScheduleCancelResponse struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type AgentApprovalResponse struct {
	Status string `json:"status"`
	TxHash string `json:"txHash,omitempty"`
	Error  string `json:"error,omitempty"`
}

type MultiSigConversionResponse struct {
	Status string `json:"status"`
	TxHash string `json:"txHash,omitempty"`
	Error  string `json:"error,omitempty"`
}

type SpotDeployResponse struct {
	Status string `json:"status"`
	TxHash string `json:"txHash,omitempty"`
	Error  string `json:"error,omitempty"`
}

type ValidatorResponse struct {
	Status string `json:"status"`
	TxHash string `json:"txHash,omitempty"`
	Error  string `json:"error,omitempty"`
}

type MultiSigResponse struct {
	Status string `json:"status"`
	TxHash string `json:"txHash,omitempty"`
	Error  string `json:"error,omitempty"`
}

type PerpDeployResponse struct {
	Status string `json:"status"`
	Data   struct {
		Statuses []TxStatus `json:"statuses"`
	} `json:"data"`
}

type TxStatus struct {
	Coin   string `json:"coin"`
	Status string `json:"status"`
}
