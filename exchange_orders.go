package hyperliquid

import (
	"context"
	"encoding/json"
	"fmt"
)

type CreateOrderRequest struct {
	Coin          string
	IsBuy         bool
	Price         float64
	Size          float64
	ReduceOnly    bool
	OrderType     OrderType
	ClientOrderID *string
}

func (s *CreateOrderRequest) String() string {
	data, _ := json.Marshal(s)
	return string(data)
}

type OrderStatusResting struct {
	Oid      int64   `json:"oid"`
	ClientID *string `json:"cloid"`
	Status   string  `json:"status"`
}

type OrderStatusFilled struct {
	TotalSz string `json:"totalSz"`
	AvgPx   string `json:"avgPx"`
	Oid     int    `json:"oid"`
}

type OrderStatus struct {
	Resting *OrderStatusResting `json:"resting,omitempty"`
	Filled  *OrderStatusFilled  `json:"filled,omitempty"`
	Error   *string             `json:"error,omitempty"`
}

func (s *OrderStatus) String() string {
	data, _ := json.Marshal(s)
	return string(data)
}

type OrderResponse struct {
	Statuses []OrderStatus
}

func newOrderTypeWire(o CreateOrderRequest) orderWireType {
	if o.OrderType.Limit != nil {
		return orderWireType{
			Limit: &orderWireTypeLimit{
				Tif: o.OrderType.Limit.Tif,
			},
		}
	}

	if o.OrderType.Trigger != nil {
		return orderWireType{
			Trigger: &orderWireTypeTrigger{
				TriggerPx: o.OrderType.Trigger.TriggerPx,
				IsMarket:  o.OrderType.Trigger.IsMarket,
				Tpsl:      o.OrderType.Trigger.Tpsl,
			},
		}
	}

	return orderWireType{}
}

func newCreateOrderAction(
	e *Exchange,
	orders []CreateOrderRequest,
	info *BuilderInfo,
) (OrderAction, error) {
	orderRequests := make([]OrderWire, len(orders))
	for i, order := range orders {
		priceWire, err := floatToWire(order.Price)
		if err != nil {
			return OrderAction{}, fmt.Errorf("failed to wire price for order %d: %w", i, err)
		}

		sizeWire, err := floatToWire(order.Size)
		if err != nil {
			return OrderAction{}, fmt.Errorf("failed to wire size for order %d: %w", i, err)
		}

		orderWire := OrderWire{
			Asset:      e.info.NameToAsset(order.Coin),
			IsBuy:      order.IsBuy,
			LimitPx:    priceWire,
			Size:       sizeWire,
			ReduceOnly: order.ReduceOnly,
			Cloid:      order.ClientOrderID,
			OrderType:  newOrderTypeWire(order),
		}

		orderRequests[i] = orderWire
	}

	res := OrderAction{
		Type:     "order",
		Orders:   orderRequests,
		Grouping: string(GroupingNA),
		Builder:  info,
	}

	return res, nil
}

func (e *Exchange) Order(
	ctx context.Context,
	req CreateOrderRequest,
	builder *BuilderInfo,
) (result OrderStatus, err error) {
	resp, err := e.BulkOrders(ctx, []CreateOrderRequest{req}, builder)
	if err != nil {
		return
	}

	if !resp.Ok {
		err = fmt.Errorf("failed to create order: %s", resp.Err)
		return
	}

	data := resp.Data
	if len(data.Statuses) == 0 {
		err = fmt.Errorf("no status for order: %s", resp.Err)
		return
	}

	return data.Statuses[0], nil
}

func (e *Exchange) BulkOrders(
	ctx context.Context,
	orders []CreateOrderRequest,
	builder *BuilderInfo,
) (result *APIResponse[OrderResponse], err error) {
	action, err := newCreateOrderAction(e, orders, builder)
	if err != nil {
		return nil, err
	}
	err = e.executeAction(ctx, action, &result)
	if err != nil {
		return nil, err
	}

	if result != nil {
		// check if any of the statuses has an error set
		for _, s := range result.Data.Statuses {
			if s.Error != nil {
				return result, fmt.Errorf("%s", *s.Error)
			}
		}
	}

	return
}

type ModifyOrderRequest struct {
	Oid   any // can be int64 or Cloid
	Order CreateOrderRequest
}

func newModifyOrderAction(
	e *Exchange,
	modifyRequest ModifyOrderRequest,
) (ModifyAction, error) {
	priceWire, err := floatToWire(modifyRequest.Order.Price)
	if err != nil {
		return ModifyAction{}, fmt.Errorf("failed to wire price: %w", err)
	}

	sizeWire, err := floatToWire(modifyRequest.Order.Size)
	if err != nil {
		return ModifyAction{}, fmt.Errorf("failed to wire size: %w", err)
	}

	// Build order type map with proper field ordering
	orderTypeMap := make(map[string]any)
	if modifyRequest.Order.OrderType.Limit != nil {
		orderTypeMap["limit"] = map[string]any{
			"tif": modifyRequest.Order.OrderType.Limit.Tif,
		}
	} else if modifyRequest.Order.OrderType.Trigger != nil {
		orderTypeMap["trigger"] = map[string]any{
			"triggerPx": modifyRequest.Order.OrderType.Trigger.TriggerPx,
			"isMarket":  modifyRequest.Order.OrderType.Trigger.IsMarket,
			"tpsl":      modifyRequest.Order.OrderType.Trigger.Tpsl,
		}
	}

	order := OrderWire{
		Asset:      e.info.NameToAsset(modifyRequest.Order.Coin),
		IsBuy:      modifyRequest.Order.IsBuy,
		LimitPx:    priceWire,
		Size:       sizeWire,
		ReduceOnly: modifyRequest.Order.ReduceOnly,
		Cloid:      modifyRequest.Order.ClientOrderID,
		OrderType:  newOrderTypeWire(modifyRequest.Order),
	}

	return ModifyAction{
		Type:  "modify",
		Oid:   modifyRequest.Oid,
		Order: order,
	}, nil
}

func newModifyOrdersAction(
	e *Exchange,
	modifyRequests []ModifyOrderRequest,
) (BatchModifyAction, error) {
	modifies := make([]ModifyAction, len(modifyRequests))
	for i, req := range modifyRequests {
		modify, err := newModifyOrderAction(e, req)
		if err != nil {
			return BatchModifyAction{}, fmt.Errorf("failed to create modify request %d: %w", i, err)
		}
		modifies[i] = modify
	}

	return BatchModifyAction{
		Type:     "batchModify",
		Modifies: modifies,
	}, nil
}

// ModifyOrder modifies an existing order
func (e *Exchange) ModifyOrder(
	ctx context.Context,
	req ModifyOrderRequest,
) (result OrderStatus, err error) {
	resp := APIResponse[OrderResponse]{}
	action, err := newModifyOrderAction(e, req)
	if err != nil {
		return result, fmt.Errorf("failed to create modify action: %w", err)
	}

	err = e.executeAction(ctx, action, &resp)
	if err != nil {
		err = fmt.Errorf("failed to modify order: %w", err)
		return
	}

	if !resp.Ok {
		err = fmt.Errorf("failed to modify order: %s", resp.Err)
		return
	}

	data := resp.Data
	if len(data.Statuses) == 0 {
		err = fmt.Errorf("no status for modified order: %s", resp.Err)
		return
	}

	return data.Statuses[0], nil
}

// BulkModifyOrders modifies multiple orders
func (e *Exchange) BulkModifyOrders(
	ctx context.Context,
	modifyRequests []ModifyOrderRequest,
) ([]OrderStatus, error) {
	resp := APIResponse[OrderResponse]{}
	action, err := newModifyOrdersAction(e, modifyRequests)
	if err != nil {
		return nil, fmt.Errorf("failed to create bulk modify action: %w", err)
	}

	err = e.executeAction(ctx, action, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to modify orders: %w", err)
	}

	if !resp.Ok {
		return nil, fmt.Errorf("failed to modify orders: %s", resp.Err)
	}

	data := resp.Data
	if len(data.Statuses) == 0 {
		return nil, fmt.Errorf("no status for modified order: %s", resp.Err)
	}

	return data.Statuses, nil
}

// MarketOpen opens a market position
func (e *Exchange) MarketOpen(
	ctx context.Context,
	name string,
	isBuy bool,
	sz float64,
	px *float64,
	slippage float64,
	cloid *string,
	builder *BuilderInfo,
) (res OrderStatus, err error) {
	slippagePrice, err := e.SlippagePrice(ctx, name, isBuy, slippage, px)
	if err != nil {
		return
	}

	orderType := OrderType{
		Limit: &LimitOrderType{Tif: TifIoc},
	}

	return e.Order(ctx, CreateOrderRequest{
		Coin:          name,
		IsBuy:         isBuy,
		Size:          sz,
		Price:         slippagePrice,
		OrderType:     orderType,
		ReduceOnly:    false,
		ClientOrderID: cloid,
	}, builder)
}

// MarketClose closes a position
func (e *Exchange) MarketClose(
	ctx context.Context,
	coin string,
	sz *float64,
	px *float64,
	slippage float64,
	cloid *string,
	builder *BuilderInfo,
) (OrderStatus, error) {
	address := e.accountAddr
	if address == "" {
		address = e.vault
	}

	userState, err := e.info.UserState(ctx, address)
	if err != nil {
		return OrderStatus{}, err
	}

	for _, assetPos := range userState.AssetPositions {
		pos := assetPos.Position
		if coin != pos.Coin {
			continue
		}

		szi := parseFloat(pos.Szi)
		var size float64
		if sz != nil {
			size = *sz
		} else {
			size = abs(szi)
		}

		isBuy := szi < 0

		slippagePrice, err := e.SlippagePrice(ctx, coin, isBuy, slippage, px)
		if err != nil {
			return OrderStatus{}, err
		}

		orderType := OrderType{
			Limit: &LimitOrderType{Tif: TifIoc},
		}

		return e.Order(ctx, CreateOrderRequest{
			Coin:          coin,
			IsBuy:         isBuy,
			Size:          size,
			Price:         slippagePrice,
			OrderType:     orderType,
			ReduceOnly:    true,
			ClientOrderID: cloid,
		}, builder)
	}

	return OrderStatus{}, fmt.Errorf("position not found for coin: %s", coin)
}
