package hyperliquid

import "fmt"

type OrderFillsSubscriptionParams struct {
	User string
}

func (w *WebsocketClient) OrderFills(
	params OrderFillsSubscriptionParams,
	callback func([]WsOrderFill, error),
) (*Subscription, error) {
	payload := remoteOrderFillsSubscriptionPayload{
		Type: ChannelOrderFills,
		User: params.User,
	}

	return w.subscribe(payload, func(msg any) {
		orders, ok := msg.(WsOrderFills)
		if !ok {
			callback(nil, fmt.Errorf("invalid message type"))
			return
		}

		callback(orders.Fills, nil)
	})
}
