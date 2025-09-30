package hyperliquid

import "github.com/sonirico/vago/fp"

//go:generate easyjson -all

type remoteL2BookSubscriptionPayload struct {
	Type     string `json:"type"`
	Coin     string `json:"coin"`
	NSigFigs int    `json:"nSigFigs,omitempty"`
	Mantissa int    `json:"mantissa,omitempty"`
}

func (p remoteL2BookSubscriptionPayload) Channel() string {
	return p.Type
}

func (p remoteL2BookSubscriptionPayload) Key() string {
	// Deliberately exclude NSigFigs and Mantissa.
	return keyL2Book(p.Coin)
}

type remoteTradesSubscriptionPayload struct {
	Type string `json:"type"`
	Coin string `json:"coin"`
}

func (p remoteTradesSubscriptionPayload) Channel() string {
	return p.Type
}

func (p remoteTradesSubscriptionPayload) Key() string {
	return keyTrades(p.Coin)
}

type remoteCandlesSubscriptionPayload struct {
	Type     string `json:"type"`
	Coin     string `json:"coin"`
	Interval string `json:"interval"`
}

func (p remoteCandlesSubscriptionPayload) Channel() string {
	return p.Type
}

func (p remoteCandlesSubscriptionPayload) Key() string {
	return keyCandles(p.Coin, p.Interval)
}

type remoteAllMidsSubscriptionPayload struct {
	Type string  `json:"type"`
	Dex  *string `json:"dex,omitempty"`
}

func (p remoteAllMidsSubscriptionPayload) Channel() string {
	return p.Type
}

func (p remoteAllMidsSubscriptionPayload) Key() string {
	return keyAllMids(fp.OptionFromPtr(p.Dex))
}

type remoteNotificationSubscriptionPayload struct {
	Type string `json:"type"`
	User string `json:"user"`
}

func (p remoteNotificationSubscriptionPayload) Channel() string {
	return p.Type
}

func (p remoteNotificationSubscriptionPayload) Key() string {
	return keyNotification(p.User)
}

type remoteOrderUpdatesSubscriptionPayload struct {
	Type string `json:"type"`
	User string `json:"user"`
}

func (p remoteOrderUpdatesSubscriptionPayload) Channel() string {
	return p.Type
}

func (p remoteOrderUpdatesSubscriptionPayload) Key() string {
	return keyOrderUpdates(p.User)
}

type remoteOrderFillsSubscriptionPayload struct {
	Type string `json:"type"`
	User string `json:"user"`
}

func (p remoteOrderFillsSubscriptionPayload) Channel() string {
	return p.Type
}

func (p remoteOrderFillsSubscriptionPayload) Key() string {
	return keyUserFills(p.User)
}

type remoteWebData2SubscriptionPayload struct {
	Type string `json:"type"`
	User string `json:"user"`
}

func (p remoteWebData2SubscriptionPayload) Channel() string {
	return p.Type
}

func (p remoteWebData2SubscriptionPayload) Key() string {
	return keyWebData2(p.User)
}

type remoteBboSubscriptionPayload struct {
	Type string `json:"type"`
	Coin string `json:"coin"`
}

func (p remoteBboSubscriptionPayload) Channel() string {
	return p.Type
}

func (p remoteBboSubscriptionPayload) Key() string {
	// Deliberately exclude NSigFigs and Mantissa.
	return keyBbo(p.Coin)
}
