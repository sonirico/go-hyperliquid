package hyperliquid

import (
	"context"
	"encoding/json"
	"fmt"
)

// TwapWire is the wire format of a TWAP order.
// The field order is part of the signed msgpack payload and must not change.
type TwapWire struct {
	Asset      int    `json:"a" msgpack:"a"`
	IsBuy      bool   `json:"b" msgpack:"b"`
	Size       string `json:"s" msgpack:"s"`
	ReduceOnly bool   `json:"r" msgpack:"r"`
	Minutes    int    `json:"m" msgpack:"m"`
	Randomize  bool   `json:"t" msgpack:"t"`
}

// TwapOrderAction represents the twapOrder action.
type TwapOrderAction struct {
	Type string   `json:"type" msgpack:"type"`
	Twap TwapWire `json:"twap" msgpack:"twap"`
}

// TwapCancelAction represents the twapCancel action.
type TwapCancelAction struct {
	Type   string `json:"type" msgpack:"type"`
	Asset  int    `json:"a"    msgpack:"a"`
	TwapID int64  `json:"t"    msgpack:"t"`
}

// TwapOrderRequest describes a TWAP order to place.
type TwapOrderRequest struct {
	Coin       string
	IsBuy      bool
	Size       float64
	ReduceOnly bool
	// Minutes is the TWAP duration in minutes.
	Minutes int
	// Randomize enables random order timing.
	Randomize bool
}

// TwapRunning holds the id of a TWAP order that was accepted.
type TwapRunning struct {
	TwapID int64 `json:"twapId"`
}

// TwapOrderStatus is the status of a twapOrder action.
type TwapOrderStatus struct {
	Running *TwapRunning `json:"running,omitempty"`
	Error   string       `json:"error,omitempty"`
}

// TwapOrderResponse is the data of a twapOrder response.
type TwapOrderResponse struct {
	Status TwapOrderStatus `json:"status"`
}

// TwapCancelStatus is the status of a twapCancel action: the string
// "success", or an object with an error message.
type TwapCancelStatus struct {
	Success bool
	Error   string
}

func (s *TwapCancelStatus) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		s.Success = str == "success"
		if !s.Success {
			s.Error = str
		}
		return nil
	}
	var obj struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	s.Error = obj.Error
	return nil
}

// TwapCancelResponse is the data of a twapCancel response.
type TwapCancelResponse struct {
	Status TwapCancelStatus `json:"status"`
}

// TwapOrder places a TWAP order.
// See https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api/exchange-endpoint#place-a-twap-order
func (e *Exchange) TwapOrder(
	ctx context.Context,
	req TwapOrderRequest,
) (res *APIResponse[TwapOrderResponse], err error) {
	asset, ok := e.info.CoinToAsset(req.Coin)
	if !ok {
		return nil, fmt.Errorf("coin %s not found in info", req.Coin)
	}

	sizeWire, err := floatToWire(req.Size)
	if err != nil {
		return nil, fmt.Errorf("failed to wire size: %w", err)
	}

	action := TwapOrderAction{
		Type: "twapOrder",
		Twap: TwapWire{
			Asset:      asset,
			IsBuy:      req.IsBuy,
			Size:       sizeWire,
			ReduceOnly: req.ReduceOnly,
			Minutes:    req.Minutes,
			Randomize:  req.Randomize,
		},
	}

	if err = e.executeAction(ctx, action, &res); err != nil {
		return
	}

	if res == nil || !res.Ok {
		if res != nil && res.Err != "" {
			return res, fmt.Errorf("%s", res.Err)
		}
		return res, fmt.Errorf("twap order failed")
	}

	if res.Data.Status.Error != "" {
		return res, fmt.Errorf("%s", res.Data.Status.Error)
	}

	return
}

// TwapCancel cancels a running TWAP order.
// See https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api/exchange-endpoint#cancel-a-twap-order
func (e *Exchange) TwapCancel(
	ctx context.Context,
	coin string,
	twapID int64,
) (res *APIResponse[TwapCancelResponse], err error) {
	asset, ok := e.info.CoinToAsset(coin)
	if !ok {
		return nil, fmt.Errorf("coin %s not found in info", coin)
	}

	action := TwapCancelAction{
		Type:   "twapCancel",
		Asset:  asset,
		TwapID: twapID,
	}

	if err = e.executeAction(ctx, action, &res); err != nil {
		return
	}

	if res == nil || !res.Ok {
		if res != nil && res.Err != "" {
			return res, fmt.Errorf("%s", res.Err)
		}
		return res, fmt.Errorf("twap cancel failed")
	}

	if res.Data.Status.Error != "" {
		return res, fmt.Errorf("%s", res.Data.Status.Error)
	}

	return
}
