package hyperliquid

import (
	"context"
	"fmt"

	"github.com/sonirico/vago/slices"
)

type (
	CancelOrderRequest struct {
		Coin    string
		OrderID int64
	}

	CancelOrderResponse struct {
		Statuses MixedArray
	}
)

func (e *Exchange) Cancel(
	coin string,
	oid int64,
) (res *APIResponse[CancelOrderResponse], err error) {
	return e.BulkCancel([]CancelOrderRequest{
		{
			Coin:    coin,
			OrderID: oid,
		},
	})
}

func (e *Exchange) BulkCancel(
	requests []CancelOrderRequest,
) (res *APIResponse[CancelOrderResponse], err error) {
	return e.BulkCancelWithContext(context.Background(), requests)
}

func (e *Exchange) BulkCancelWithContext(
	ctx context.Context,
	requests []CancelOrderRequest,
) (res *APIResponse[CancelOrderResponse], err error) {
	cancels := slices.Map(requests, func(req CancelOrderRequest) CancelOrderWire {
		return CancelOrderWire{
			Asset:   e.info.NameToAsset(req.Coin),
			OrderID: req.OrderID,
		}
	})

	action := CancelAction{
		Type:    "cancel",
		Cancels: cancels,
	}

	if err = e.executeAction(ctx, action, &res); err != nil {
		return
	}

	if res == nil || !res.Ok || res.Status == "err" {
		if res != nil && res.Err != "" {
			return res, fmt.Errorf("%s", res.Err)
		}
		return res, fmt.Errorf("cancel failed")
	}

	if err := res.Data.Statuses.FirstError(); err != nil {
		return res, err
	}

	return
}

type CancelOrderRequestByCloid struct {
	Coin  string
	Cloid string
}

func (e *Exchange) CancelByCloid(
	coin, cloid string,
) (res *APIResponse[CancelOrderResponse], err error) {
	return e.BulkCancelByCloids([]CancelOrderRequestByCloid{
		{
			Coin:  coin,
			Cloid: cloid,
		},
	})
}

func (e *Exchange) BulkCancelByCloids(
	requests []CancelOrderRequestByCloid,
) (res *APIResponse[CancelOrderResponse], err error) {
	return e.BulkCancelByCloidsWithContext(context.Background(), requests)
}

func (e *Exchange) BulkCancelByCloidsWithContext(
	ctx context.Context,
	requests []CancelOrderRequestByCloid,
) (res *APIResponse[CancelOrderResponse], err error) {
	cancels := slices.Map(requests, func(req CancelOrderRequestByCloid) CancelByCloidWire {
		return CancelByCloidWire{
			Asset:    e.info.NameToAsset(req.Coin),
			ClientID: req.Cloid,
		}
	})

	action := CancelByCloidAction{
		Type:    "cancelByCloid",
		Cancels: cancels,
	}

	if err = e.executeAction(ctx, action, &res); err != nil {
		return
	}

	if res == nil || !res.Ok || res.Status == "err" {
		if res != nil && res.Err != "" {
			return res, fmt.Errorf("%s", res.Err)
		}
		return res, fmt.Errorf("cancel failed")
	}

	if err := res.Data.Statuses.FirstError(); err != nil {
		return res, err
	}

	return
}
