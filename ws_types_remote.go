package hyperliquid

import "github.com/sonirico/vago/fp"

//go:generate easyjson -all

type remoteL2BookSubscriptionPayload struct {
	Type     string `json:"type"`
	Coin     string `json:"coin"`
	NSigFigs int    `json:"nSigFigs,omitempty"`
	Mantissa int    `json:"mantissa,omitempty"`
	Fast     bool   `json:"fast,omitempty"`
}

func (p remoteL2BookSubscriptionPayload) Channel() string {
	return p.Type
}

func (p remoteL2BookSubscriptionPayload) Key() string {
	// Deliberately exclude NSigFigs, Mantissa and Fast.
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

type remoteActiveAssetCtxSubscriptionPayload struct {
	Type string `json:"type"`
	Coin string `json:"coin"`
}

func (p remoteActiveAssetCtxSubscriptionPayload) Channel() string {
	return p.Type
}

func (p remoteActiveAssetCtxSubscriptionPayload) Key() string {
	return keyActiveAssetCtx(p.Coin)
}

type remoteFastAssetCtxsSubscriptionPayload struct {
	Type string `json:"type"`
}

func (p remoteFastAssetCtxsSubscriptionPayload) Channel() string {
	return p.Type
}

func (p remoteFastAssetCtxsSubscriptionPayload) Key() string {
	return keyFastAssetCtxs()
}

type remoteAllDexsAssetCtxsSubscriptionPayload struct {
	Type string `json:"type"`
}

func (p remoteAllDexsAssetCtxsSubscriptionPayload) Channel() string {
	return p.Type
}

func (p remoteAllDexsAssetCtxsSubscriptionPayload) Key() string {
	return keyAllDexsAssetCtxs()
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

type remoteUserEventsSubscriptionPayload struct {
	Type string `json:"type"`
	User string `json:"user"`
}

func (p remoteUserEventsSubscriptionPayload) Channel() string {
	return p.Type
}

func (p remoteUserEventsSubscriptionPayload) Key() string {
	return keyUserEvents(p.User)
}

type remoteUserFundingsSubscriptionPayload struct {
	Type string `json:"type"`
	User string `json:"user"`
}

func (p remoteUserFundingsSubscriptionPayload) Channel() string {
	return p.Type
}

func (p remoteUserFundingsSubscriptionPayload) Key() string {
	return keyUserFundings(p.User)
}

type remoteUserNonFundingLedgerUpdatesSubscriptionPayload struct {
	Type string `json:"type"`
	User string `json:"user"`
}

func (p remoteUserNonFundingLedgerUpdatesSubscriptionPayload) Channel() string {
	return p.Type
}

func (p remoteUserNonFundingLedgerUpdatesSubscriptionPayload) Key() string {
	return keyUserNonFundingLedgerUpdates(p.User)
}

type remoteActiveAssetDataSubscriptionPayload struct {
	Type string `json:"type"`
	User string `json:"user"`
	Coin string `json:"coin"`
}

func (p remoteActiveAssetDataSubscriptionPayload) Channel() string {
	return p.Type
}

func (p remoteActiveAssetDataSubscriptionPayload) Key() string {
	return keyActiveAssetData(p.User, p.Coin)
}

type remoteUserTwapSliceFillsSubscriptionPayload struct {
	Type string `json:"type"`
	User string `json:"user"`
}

func (p remoteUserTwapSliceFillsSubscriptionPayload) Channel() string {
	return p.Type
}

func (p remoteUserTwapSliceFillsSubscriptionPayload) Key() string {
	return keyUserTwapSliceFills(p.User)
}

type remoteUserTwapHistorySubscriptionPayload struct {
	Type string `json:"type"`
	User string `json:"user"`
}

func (p remoteUserTwapHistorySubscriptionPayload) Channel() string {
	return p.Type
}

func (p remoteUserTwapHistorySubscriptionPayload) Key() string {
	return keyUserTwapHistory(p.User)
}

type remoteSpotStateSubscriptionPayload struct {
	Type string `json:"type"`
	User string `json:"user"`
}

func (p remoteSpotStateSubscriptionPayload) Channel() string {
	return p.Type
}

func (p remoteSpotStateSubscriptionPayload) Key() string {
	return keySpotState(p.User)
}

type remoteAllDexsClearinghouseStateSubscriptionPayload struct {
	Type string `json:"type"`
	User string `json:"user"`
}

func (p remoteAllDexsClearinghouseStateSubscriptionPayload) Channel() string {
	return p.Type
}

func (p remoteAllDexsClearinghouseStateSubscriptionPayload) Key() string {
	return keyAllDexsClearinghouseState(p.User)
}

type remoteOutcomeMetaUpdatesSubscriptionPayload struct {
	Type string `json:"type"`
}

func (p remoteOutcomeMetaUpdatesSubscriptionPayload) Channel() string {
	return p.Type
}

func (p remoteOutcomeMetaUpdatesSubscriptionPayload) Key() string {
	return keyOutcomeMetaUpdates()
}

type remoteL2SubscriptionPayload struct {
	Type     string `json:"type"`
	Coin     string `json:"c"`
	NSigFigs int    `json:"s,omitempty"`
	Mantissa int    `json:"m,omitempty"`
}

func (p remoteL2SubscriptionPayload) Channel() string {
	return p.Type
}

func (p remoteL2SubscriptionPayload) Key() string {
	// Deliberately exclude NSigFigs and Mantissa.
	return keyL2(p.Coin)
}

type remotePacSubscriptionPayload struct {
	Type string `json:"type"`
}

func (p remotePacSubscriptionPayload) Channel() string {
	return p.Type
}

func (p remotePacSubscriptionPayload) Key() string {
	return keyPac()
}

type remoteSacSubscriptionPayload struct {
	Type string `json:"type"`
}

func (p remoteSacSubscriptionPayload) Channel() string {
	return p.Type
}

func (p remoteSacSubscriptionPayload) Key() string {
	return keySac()
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

type remoteClearinghouseStateSubscriptionPayload struct {
	Type string  `json:"type"`
	User string  `json:"user"`
	Dex  *string `json:"dex,omitempty"`
}

func (p remoteClearinghouseStateSubscriptionPayload) Channel() string {
	return p.Type
}

func (p remoteClearinghouseStateSubscriptionPayload) Key() string {
	return keyClearinghouseState(p.User, fp.OptionFromPtr(p.Dex))
}

type remoteOpenOrdersSubscriptionPayload struct {
	Type string  `json:"type"`
	User string  `json:"user"`
	Dex  *string `json:"dex,omitempty"`
}

func (p remoteOpenOrdersSubscriptionPayload) Channel() string {
	return p.Type
}

func (p remoteOpenOrdersSubscriptionPayload) Key() string {
	return keyOpenOrders(p.User, fp.OptionFromPtr(p.Dex))
}

type remoteTwapStatesSubscriptionPayload struct {
	Type string  `json:"type"`
	User string  `json:"user"`
	Dex  *string `json:"dex,omitempty"`
}

func (p remoteTwapStatesSubscriptionPayload) Channel() string {
	return p.Type
}

func (p remoteTwapStatesSubscriptionPayload) Key() string {
	return keyTwapStates(p.User, fp.OptionFromPtr(p.Dex))
}

type remoteWebData3SubscriptionPayload struct {
	Type string  `json:"type"`
	User string  `json:"user"`
	Dex  *string `json:"dex,omitempty"`
}

func (p remoteWebData3SubscriptionPayload) Channel() string {
	return p.Type
}

func (p remoteWebData3SubscriptionPayload) Key() string {
	return keyWebData3(p.User, fp.OptionFromPtr(p.Dex))
}
