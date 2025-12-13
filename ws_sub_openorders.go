package hyperliquid

import "fmt"

type OpenOrdersSubscriptionParams struct {
	User string
	Dex  *string
}

func (w *WebsocketClient) OpenOrders(
	params OpenOrdersSubscriptionParams,
	callback func(OpenOrders, error),
) (*Subscription, error) {
	payload := remoteOpenOrdersSubscriptionPayload{
		Type: ChannelOpenOrders,
		User: params.User,
		Dex:  params.Dex,
	}

	return w.subscribe(payload, func(msg any) {
		orders, ok := msg.(OpenOrders)
		if !ok {
			callback(OpenOrders{}, fmt.Errorf("invalid message type"))
			return
		}

		callback(orders, nil)
	})
}
