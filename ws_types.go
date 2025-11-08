package hyperliquid

import (
	"encoding/json"
)

//go:generate easyjson -all

const (
	ChannelPong           string = "pong"
	ChannelTrades         string = "trades"
	ChannelActiveAssetCtx string = "activeAssetCtx"
	ChannelL2Book         string = "l2Book"
	ChannelCandle         string = "candle"
	ChannelAllMids        string = "allMids"
	ChannelNotification   string = "notification"
	ChannelOrderUpdates   string = "orderUpdates"
	ChannelUserFills      string = "userFills"
	ChannelWebData2       string = "webData2"
	ChannelBbo            string = "bbo"
	ChannelSubResponse    string = "subscriptionResponse"
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

	SharedAssetCtx struct {
		DayNtlVlm float64 `json:"dayNtlVlm,string"`
		PrevDayPx float64 `json:"prevDayPx,string"`
		MarkPx    float64 `json:"markPx,string"`
		MidPx     float64 `json:"midPx,string"`

		// PerpsAssetCtx
		Funding      float64 `json:"funding,string,omitempty"`
		OpenInterest float64 `json:"openInterest,string,omitempty"`
		OraclePx     float64 `json:"oraclePx,string,omitempty"`

		// SpotAssetCtx
		CirculatingSupply float64 `json:"circulatingSupply,string,omitempty"`
	}

	AllMids struct {
		Mids map[string]string `json:"mids"`
	}

	Notification struct {
		Notification string `json:"notification"`
	}

	//easyjson:skip
	WebData2 struct {
		ClearinghouseState     *ClearinghouseState `json:"clearinghouseState,omitempty"`
		LeadingVaults          []any               `json:"leadingVaults,omitempty"`
		TotalVaultEquity       string              `json:"totalVaultEquity,omitempty"`
		OpenOrders             []WsBasicOrder      `json:"openOrders,omitempty"`
		AgentAddress           *string             `json:"agentAddress,omitempty"`
		AgentValidUntil        *int64              `json:"agentValidUntil,omitempty"`
		CumLedger              string              `json:"cumLedger,omitempty"`
		Meta                   *WebData2Meta       `json:"meta,omitempty"`
		AssetCtxs              []AssetCtx          `json:"assetCtxs,omitempty"`
		ServerTime             int64               `json:"serverTime,omitempty"`
		IsVault                bool                `json:"isVault,omitempty"`
		User                   string              `json:"user,omitempty"`
		TwapStates             []any               `json:"twapStates,omitempty"`
		SpotState              *SpotState          `json:"spotState,omitempty"`
		SpotAssetCtxs          []SpotAssetCtx      `json:"spotAssetCtxs,omitempty"`
		PerpsAtOpenInterestCap []string            `json:"perpsAtOpenInterestCap,omitempty"`
	}

	//easyjson:skip
	WebData2Meta struct {
		Universe     []WebData2AssetInfo                `json:"universe,omitempty"`
		MarginTables []Tuple2[int, WebData2MarginTable] `json:"marginTables,omitempty"`
	}

	WebData2AssetInfo struct {
		SzDecimals    int    `json:"szDecimals,omitempty"`
		Name          string `json:"name,omitempty"`
		MaxLeverage   int    `json:"maxLeverage,omitempty"`
		MarginTableID int    `json:"marginTableId,omitempty"`
		IsDelisted    bool   `json:"isDelisted,omitempty"`
		OnlyIsolated  bool   `json:"onlyIsolated,omitempty"`
	}

	WebData2MarginTable struct {
		Description string               `json:"description,omitempty"`
		MarginTiers []WebData2MarginTier `json:"marginTiers,omitempty"`
	}

	WebData2MarginTier struct {
		LowerBound  string `json:"lowerBound,omitempty"`
		MaxLeverage int    `json:"maxLeverage,omitempty"`
	}

	ClearinghouseState struct {
		MarginSummary              *MarginSummary  `json:"marginSummary,omitempty"`
		CrossMarginSummary         *MarginSummary  `json:"crossMarginSummary,omitempty"`
		CrossMaintenanceMarginUsed string          `json:"crossMaintenanceMarginUsed,omitempty"`
		Withdrawable               string          `json:"withdrawable,omitempty"`
		AssetPositions             []AssetPosition `json:"assetPositions,omitempty"`
		Time                       int64           `json:"time,omitempty"`
	}

	SpotState struct {
		Balances []SpotBalance `json:"balances,omitempty"`
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
)

var (
	candleNoop = Candle{}
)
