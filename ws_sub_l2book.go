package hyperliquid

import "fmt"

type L2BookSubscriptionParams struct {
	Coin     string
	NSigFigs int
	Mantissa int
	Fast     bool
}

func (w *WebsocketClient) L2Book(
	params L2BookSubscriptionParams,
	callback func(L2Book, error),
) (*Subscription, error) {
	remotePayload := remoteL2BookSubscriptionPayload{
		Type:     ChannelL2Book,
		Coin:     params.Coin,
		NSigFigs: params.NSigFigs,
		Mantissa: params.Mantissa,
		Fast:     params.Fast,
	}

	return w.subscribe(remotePayload, func(msg any) {
		orderbook, ok := msg.(L2Book)
		if !ok {
			callback(L2Book{}, fmt.Errorf("invalid message type"))
			return
		}

		callback(orderbook, nil)
	})
}
