package hyperliquid

import "github.com/sonirico/vago/fp"

type subscriptable interface {
	Key() string
}

type (
	Trades   []Trade
	WsOrders []WsOrder

	WsFastAssetCtxs map[string]FastAssetCtx
)

func (t Trades) Key() string {
	if len(t) == 0 {
		return ""
	}
	return keyTrades(t[0].Coin)
}

func (a ActiveAssetCtx) Key() string {
	return keyActiveAssetCtx(a.Coin)
}

func (w WsFastAssetCtxs) Key() string {
	return keyFastAssetCtxs()
}

func (w WsAllDexsAssetCtxs) Key() string {
	return keyAllDexsAssetCtxs()
}

func (c Candle) Key() string {
	return keyCandles(c.Symbol, c.Interval)
}

func (c L2Book) Key() string {
	return keyL2Book(c.Coin)
}

func (a AllMids) Key() string {
	return keyAllMids(fp.None[string]())
}

func (n Notification) Key() string {
	// Notification messages are user-specific but don't contain user info in the message itself.
	// The dispatching is handled by the subscription system based on the subscription key.
	return ChannelNotification
}

func (w WsOrders) Key() string {
	// WsOrder messages are user-specific but don't contain user info in the message itself.
	// The dispatching is handled by the subscription system based on the subscription key.
	return ChannelOrderUpdates
}

func (w Bbo) Key() string { return keyBbo(w.Coin) }

func (u UserEvents) Key() string {
	// UserEvents messages are user-specific but don't contain user info in the message itself.
	// The dispatching is handled by the subscription system based on the subscription key.
	return ChannelUser
}

func (m UserFundings) Key() string {
	return keyUserFundings(m.User)
}

func (m WsUserNonFundingLedgerUpdates) Key() string {
	return keyUserNonFundingLedgerUpdates(m.User)
}

func (m UserActiveAssetData) Key() string {
	return keyActiveAssetData(m.User, m.Coin)
}

func (m WsUserTwapSliceFills) Key() string {
	return keyUserTwapSliceFills(m.User)
}

func (m WsUserTwapHistory) Key() string {
	return keyUserTwapHistory(m.User)
}

func (m WsSpotState) Key() string {
	return keySpotState(m.User)
}

func (m WsAllDexsClearinghouseState) Key() string {
	return keyAllDexsClearinghouseState(m.User)
}

func (OutcomeMetaUpdates) Key() string {
	return keyOutcomeMetaUpdates()
}

func (l WsL2) Key() string {
	if l.S != nil {
		return keyL2(l.S.Coin)
	}
	if l.U != nil {
		return keyL2(l.U.Coin)
	}
	return keyL2("")
}

func (WsPac) Key() string {
	return keyPac()
}

func (w WsOrderFills) Key() string {
	return keyUserFills(w.User)
}

func (c ClearinghouseState) Key() string {
	// ClearinghouseState messages are user-specific but don't contain user/dex info in the message itself.
	// The dispatching is handled by the subscription system based on the subscription key.
	return ChannelClearinghouseState
}

func (c ClearinghouseStateMessage) Key() string {
	if c.Dex == "" {
		return key(ChannelClearinghouseState, c.User)
	}
	return key(ChannelClearinghouseState, c.User, c.Dex)
}

func (o OpenOrders) Key() string {
	// OpenOrders messages contain user and dex info, but we use the subscription key for dispatching
	// The subscription key already includes dex, so we just return a generic key
	return ChannelOpenOrders
}

func (t TwapStates) Key() string {
	// TwapStates messages contain user and dex info, but we use the subscription key for dispatching
	// The subscription key already includes dex, so we just return a generic key
	return ChannelTwapStates
}

func (w WebData3) Key() string {
	// WebData3 messages are user-specific but don't contain user/dex info in the message itself.
	// The dispatching is handled by the subscription system based on the subscription key.
	return ChannelWebData3
}
