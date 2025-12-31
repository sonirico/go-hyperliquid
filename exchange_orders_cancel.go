package hyperliquid

import (
	"context"
	"fmt"
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
	ctx context.Context,
	coin string,
	oid int64,
) (res *APIResponse[CancelOrderResponse], err error) {
	return e.BulkCancel(ctx, []CancelOrderRequest{
		{
			Coin:    coin,
			OrderID: oid,
		},
	})
}

func (e *Exchange) BulkCancel(
	ctx context.Context,
	requests []CancelOrderRequest,
) (res *APIResponse[CancelOrderResponse], err error) {
	cancels := make([]CancelOrderWire, 0, len(requests))
	for _, req := range requests {
		asset, ok := e.info.CoinToAsset(req.Coin)
		if !ok {
			return nil, fmt.Errorf("coin %s not found in info", req.Coin)
		}
		cancels = append(cancels, CancelOrderWire{
			Asset:   asset,
			OrderID: req.OrderID,
		})
	}

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
	ctx context.Context,
	coin, cloid string,
) (res *APIResponse[CancelOrderResponse], err error) {
	return e.BulkCancelByCloids(ctx, []CancelOrderRequestByCloid{
		{
			Coin:  coin,
			Cloid: cloid,
		},
	})
}

func (e *Exchange) BulkCancelByCloids(
	ctx context.Context,
	requests []CancelOrderRequestByCloid,
) (res *APIResponse[CancelOrderResponse], err error) {
	cancels := make([]CancelByCloidWire, len(requests))
	for i, req := range requests {
		normalizedCloid, err := normalizeCloid(&req.Cloid)
		if err != nil {
			return nil, fmt.Errorf("invalid cloid for cancel request %d: %w", i, err)
		}
		if normalizedCloid == nil {
			return nil, fmt.Errorf("cloid is required for cancel by cloid request %d", i)
		}
		asset, ok := e.info.CoinToAsset(req.Coin)
		if !ok {
			return nil, fmt.Errorf("coin %s not found in info", req.Coin)
		}

		cancels[i] = CancelByCloidWire{
			Asset:    asset,
			ClientID: *normalizedCloid,
		}
	}

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
