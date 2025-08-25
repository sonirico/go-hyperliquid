package hyperliquid

import (
	"encoding/json"
)

//go:generate easyjson -all

const (
	ChannelTrades       string = "trades"
	ChannelL2Book       string = "l2Book"
	ChannelCandle       string = "candle"
	ChannelAllMids      string = "allMids"
	ChannelNotification string = "notification"
	ChannelOrderUpdates string = "orderUpdates"
	ChannelSubResponse  string = "subscriptionResponse"
	ChannelPong         string = "pong"
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

	AllMids struct {
		Mids map[string]string `json:"mids"`
	}

	Notification struct {
		Notification string `json:"notification"`
	}

	WsOrder struct {
		Order           WsBasicOrder `json:"order"`
		Status          string       `json:"status"`
		StatusTimestamp int64        `json:"statusTimestamp"`
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

	L2Book struct {
		Coin   string    `json:"coin"`
		Levels [][]Level `json:"levels"`
		Time   int64     `json:"time"`
	}

	Level struct {
		N  int     `json:"n"`
		Px float64 `json:"px,string"`
		Sz float64 `json:"sz,string"`
	}

	Candle struct {
		Timestamp int64  `json:"T"`
		Close     string `json:"c"`
		High      string `json:"h"`
		Interval  string `json:"i"`
		Low       string `json:"l"`
		Number    int    `json:"n"`
		Open      string `json:"o"`
		Symbol    string `json:"s"`
		Time      int64  `json:"t"`
		Volume    string `json:"v"`
	}
)
