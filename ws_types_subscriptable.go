package hyperliquid

import "github.com/sonirico/vago/fp"

type subscriptable interface {
	Key() string
}

type (
	Trades   []Trade
	Candles  []Candle
	WsOrders []WsOrder
)

func (t Trades) Key() string {
	if len(t) == 0 {
		return ""
	}
	return keyTrades(t[0].Coin)
}

func (c Candles) Key() string {
	if len(c) == 0 {
		return ""
	}
	return keyCandles(c[0].Symbol, c[0].Interval)
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

func (w WebData2) Key() string {
	// WebData2 messages are user-specific but don't contain user info in the message itself.
	// The dispatching is handled by the subscription system based on the subscription key.
	return ChannelWebData2
}

func (w Bbo) Key() string { return keyBbo(w.Coin) }

func (w WsOrderFills) Key() string {
	return keyOrderFills(w.User)
}
