package hyperliquid

import (
	"encoding/json"
)

//go:generate easyjson -all

const (
	ChannelPong                        string = "pong"
	ChannelTrades                      string = "trades"
	ChannelActiveAssetCtx              string = "activeAssetCtx"
	ChannelActiveSpotAssetCtx          string = "activeSpotAssetCtx"
	ChannelFastAssetCtxs               string = "fastAssetCtxs"
	ChannelAllDexsAssetCtxs            string = "allDexsAssetCtxs"
	ChannelL2Book                      string = "l2Book"
	ChannelCandle                      string = "candle"
	ChannelAllMids                     string = "allMids"
	ChannelNotification                string = "notification"
	ChannelOrderUpdates                string = "orderUpdates"
	ChannelUser                        string = "user"
	ChannelUserFills                   string = "userFills"
	ChannelBbo                         string = "bbo"
	ChannelSubResponse                 string = "subscriptionResponse"
	ChannelClearinghouseState          string = "clearinghouseState"
	ChannelOpenOrders                  string = "openOrders"
	ChannelTwapStates                  string = "twapStates"
	ChannelWebData3                    string = "webData3"
	ChannelUserFundings                string = "userFundings"
	ChannelUserNonFundingLedgerUpdates string = "userNonFundingLedgerUpdates"
	ChannelActiveAssetData             string = "activeAssetData"
	ChannelUserTwapSliceFills          string = "userTwapSliceFills"
	ChannelUserTwapHistory             string = "userTwapHistory"
	ChannelSpotState                   string = "spotState"
	ChannelAllDexsClearinghouseState   string = "allDexsClearinghouseState"
	ChannelError                       string = "error"
)

type wsMessage struct {
	Channel string          `json:"channel"`
	Data    json.RawMessage `json:"data"`
}

type wsCommand struct {
	Method       string `json:"method"`
	Subscription any    `json:"subscription,omitempty"`
}

type (
	Trade struct {
		Coin  string   `json:"coin"`
		Side  string   `json:"side"`
		Px    string   `json:"px"`
		Sz    string   `json:"sz"`
		Time  int64    `json:"time"`
		Hash  string   `json:"hash"`
		Tid   int64    `json:"tid"`
		Users []string `json:"users"`
	}

	ActiveAssetCtx struct {
		Coin string         `json:"coin"`
		Ctx  SharedAssetCtx `json:"ctx"`
	}

	FastAssetCtx struct {
		MarkPx *float64 `json:"markPx,string,omitempty"`
		MidPx  *float64 `json:"midPx,string,omitempty"`
	}

	//easyjson:skip
	WsAllDexsAssetCtxs struct {
		Ctxs []Tuple2[string, AssetCtxs] `json:"ctxs"`
	}

	SharedAssetCtx struct {
		DayNtlVlm float64 `json:"dayNtlVlm,string"`
		PrevDayPx float64 `json:"prevDayPx,string"`
		MarkPx    float64 `json:"markPx,string"`
		MidPx     float64 `json:"midPx,string"`

		DayBaseVlm float64 `json:"dayBaseVlm,string"`

		// PerpsAssetCtx
		Funding      float64  `json:"funding,string,omitempty"`
		OpenInterest float64  `json:"openInterest,string,omitempty"`
		OraclePx     float64  `json:"oraclePx,string,omitempty"`
		Premium      float64  `json:"premium,string,omitempty"`
		ImpactPxs    []string `json:"impactPxs,omitempty"`

		// SpotAssetCtx
		CirculatingSupply float64 `json:"circulatingSupply,string,omitempty"`
		Coin              string  `json:"coin,omitempty"`
		TotalSupply       float64 `json:"totalSupply,string,omitempty"`
	}

	AllMids struct {
		Mids map[string]string `json:"mids"`
	}

	Notification struct {
		Notification string `json:"notification"`
	}

	ClearinghouseState struct {
		MarginSummary              *MarginSummary  `json:"marginSummary,omitempty"`
		CrossMarginSummary         *MarginSummary  `json:"crossMarginSummary,omitempty"`
		CrossMaintenanceMarginUsed string          `json:"crossMaintenanceMarginUsed,omitempty"`
		Withdrawable               string          `json:"withdrawable,omitempty"`
		AssetPositions             []AssetPosition `json:"assetPositions,omitempty"`
		Time                       int64           `json:"time,omitempty"`
	}

	ClearinghouseStateMessage struct {
		Dex                string             `json:"dex"`
		User               string             `json:"user,omitempty"`
		ClearinghouseState ClearinghouseState `json:"clearinghouseState"`
	}

	SpotState struct {
		Balances []SpotBalance `json:"balances,omitempty"`
	}

	OpenOrders struct {
		Dex    string              `json:"dex"`
		User   string              `json:"user"`
		Orders []FrontendOpenOrder `json:"orders"`
	}

	WsOrder struct {
		Order           WsBasicOrder     `json:"order"`
		Status          OrderStatusValue `json:"status"`
		StatusTimestamp int64            `json:"statusTimestamp"`
	}

	WsBasicOrder struct {
		Coin      string  `json:"coin"`
		Side      string  `json:"side"`
		LimitPx   string  `json:"limitPx"`
		Sz        string  `json:"sz"`
		Oid       int64   `json:"oid"`
		Timestamp int64   `json:"timestamp"`
		OrigSz    string  `json:"origSz"`
		Cloid     *string `json:"cloid"`
	}

	WsOrderFills struct {
		IsSnapshot bool          `json:"isSnapshot"`
		User       string        `json:"user"`
		Fills      []WsOrderFill `json:"fills"`
	}

	WsOrderFill struct {
		Coin          string           `json:"coin"`
		Px            string           `json:"px"` // price
		Sz            string           `json:"sz"` // size
		Side          string           `json:"side"`
		Time          int64            `json:"time"`
		StartPosition string           `json:"startPosition"`
		Dir           string           `json:"dir"` // used for frontend display
		ClosedPnl     string           `json:"closedPnl"`
		Hash          string           `json:"hash"`    // L1 transaction hash
		Oid           int64            `json:"oid"`     // order id
		Crossed       bool             `json:"crossed"` // whether order crossed the spread (was taker)
		Fee           string           `json:"fee"`     // negative means rebate
		Tid           int64            `json:"tid"`     // unique trade id
		Liquidation   *FillLiquidation `json:"liquidation,omitempty"`
		FeeToken      string           `json:"feeToken"`             // the token the fee was paid in
		BuilderFee    *string          `json:"builderFee,omitempty"` // amount paid to builder, also included in fee
		Cloid         *string          `json:"cloid,omitempty"`
		TwapId        *int64           `json:"twapId"`
	}

	UserEvents struct {
		Fills         []WsOrderFill     `json:"fills,omitempty"`
		Funding       *WsUserFunding    `json:"funding,omitempty"`
		Liquidation   *WsLiquidation    `json:"liquidation,omitempty"`
		NonUserCancel []WsNonUserCancel `json:"nonUserCancel,omitempty"`
	}

	WsUserFunding struct {
		Time        int64  `json:"time"`
		Coin        string `json:"coin"`
		Usdc        string `json:"usdc"`
		Szi         string `json:"szi"`
		FundingRate string `json:"fundingRate"`
		NSamples    *int   `json:"nSamples"`
	}

	WsLiquidation struct {
		Lid                    int64  `json:"lid"`
		Liquidator             string `json:"liquidator"`
		LiquidatedUser         string `json:"liquidated_user"`
		LiquidatedNtlPos       string `json:"liquidated_ntl_pos"`
		LiquidatedAccountValue string `json:"liquidated_account_value"`
	}

	WsNonUserCancel struct {
		Coin string `json:"coin"`
		Oid  int64  `json:"oid"`
	}

	UserFundings struct {
		IsSnapshot bool            `json:"isSnapshot"`
		User       string          `json:"user"`
		Fundings   []WsUserFunding `json:"fundings"`
	}

	WsUserNonFundingLedgerUpdates struct {
		IsSnapshot              bool                          `json:"isSnapshot"`
		User                    string                        `json:"user"`
		NonFundingLedgerUpdates []UserNonFundingLedgerUpdates `json:"nonFundingLedgerUpdates"`
	}

	WsUserTwapSliceFills struct {
		IsSnapshot     bool            `json:"isSnapshot"`
		User           string          `json:"user"`
		TwapSliceFills []TwapSliceFill `json:"twapSliceFills"`
	}

	WsUserTwapHistory struct {
		IsSnapshot bool            `json:"isSnapshot"`
		User       string          `json:"user"`
		History    []WsTwapHistory `json:"history"`
	}

	WsTwapHistory struct {
		State  TwapState    `json:"state"`
		Status WsTwapStatus `json:"status"`
		Time   int64        `json:"time"`
		TwapId int64        `json:"twapId"`
	}

	WsTwapStatus struct {
		Status      string `json:"status"`
		Description string `json:"description"`
	}

	WsSpotState struct {
		User      string        `json:"user"`
		SpotState SpotUserState `json:"spotState"`
	}

	//easyjson:skip
	WsAllDexsClearinghouseState struct {
		User                string                      `json:"user"`
		ClearinghouseStates []Tuple2[string, UserState] `json:"clearinghouseStates"`
	}

	FillLiquidation struct {
		LiquidatedUser *string `json:"liquidatedUser,omitempty"`
		MarkPx         string  `json:"markPx"`
		Method         string  `json:"method"`
	}

	L2Book struct {
		Coin   string    `json:"coin"`
		Levels [][]Level `json:"levels"`
		Time   int64     `json:"time"`
	}

	Bbo struct {
		Coin string  `json:"coin"`
		Time int64   `json:"time"`
		Bbo  []Level `json:"bbo"`
	}

	Level struct {
		N  int     `json:"n"`
		Px float64 `json:"px,string"`
		Sz float64 `json:"sz,string"`
	}

	Candle struct {
		TimeOpen    int64  `json:"t"` // open millis
		TimeClose   int64  `json:"T"` // close millis
		Interval    string `json:"i"` // interval
		TradesCount int    `json:"n"` // number of trades
		Open        string `json:"o"` // open price
		High        string `json:"h"` // high price
		Low         string `json:"l"` // low price
		Close       string `json:"c"` // close price
		Symbol      string `json:"s"` // coin
		Volume      string `json:"v"` // volume (base unit)
	}

	//easyjson:skip
	TwapStates struct {
		Dex    string                   `json:"dex"`
		User   string                   `json:"user"`
		States []Tuple2[int, TwapState] `json:"states"`
	}

	TwapState struct {
		Coin        string          `json:"coin"`
		User        string          `json:"user"`
		Side        string          `json:"side"`
		Sz          float64         `json:"sz,string"`
		ExecutedSz  float64         `json:"executedSz,string"`
		ExecutedNtl float64         `json:"executedNtl,string"`
		Minutes     int             `json:"minutes"`
		ReduceOnly  bool            `json:"reduceOnly"`
		Randomize   bool            `json:"randomize"`
		Timestamp   int64           `json:"timestamp"`
		Trigger     json.RawMessage `json:"trigger"`
		StopPx      json.RawMessage `json:"stopPx"`
	}

	//easyjson:skip
	WebData3 struct {
		UserState     WebData3UserState `json:"userState"`
		PerpDexStates []PerpDexState    `json:"perpDexStates"`
	}

	WebData3UserState struct {
		AgentAddress          *string `json:"agentAddress"`
		AgentValidUntil       *int64  `json:"agentValidUntil"`
		ServerTime            int64   `json:"serverTime"`
		CumLedger             float64 `json:"cumLedger,string"`
		IsVault               bool    `json:"isVault"`
		User                  string  `json:"user"`
		OptOutOfSpotDusting   *bool   `json:"optOutOfSpotDusting,omitempty"`
		DexAbstractionEnabled *bool   `json:"dexAbstractionEnabled,omitempty"`
		Abstraction           string  `json:"abstraction"`
	}

	PerpDexState struct {
		TotalVaultEquity       float64         `json:"totalVaultEquity,string"`
		PerpsAtOpenInterestCap *[]string       `json:"perpsAtOpenInterestCap,omitempty"`
		LeadingVaults          *[]LeadingVault `json:"leadingVaults,omitempty"`
	}

	LeadingVault struct {
		Address string `json:"address"`
		Name    string `json:"name"`
	}
)

var (
	candleNoop = Candle{}
)
