package hyperliquid

import "fmt"

type OrderFillsSubscriptionParams struct {
	User string
}

func (w *WebsocketClient) OrderFills(
	params OrderFillsSubscriptionParams,
	callback func(WsOrderFills, error),
) (*Subscription, error) {
	payload := remoteOrderFillsSubscriptionPayload{
		Type: ChannelUserFills,
		User: params.User,
	}

	return w.subscribe(payload, func(msg any) {
		orders, ok := msg.(WsOrderFills)
		if !ok {
			callback(WsOrderFills{}, fmt.Errorf("invalid message type"))
			return
		}

		callback(orders, nil)
	})
}
