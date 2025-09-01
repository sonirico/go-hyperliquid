package hyperliquid

import "fmt"

type BboSubscriptionParams struct {
	Coin string
}

func (w *WebsocketClient) Bbo(
	params BboSubscriptionParams,
	callback func(Bbo, error),
) (*Subscription, error) {
	remotePayload := remoteBboSubscriptionPayload{
		Type: ChannelBbo,
		Coin: params.Coin,
	}

	return w.subscribe(remotePayload, func(msg any) {
		bbo, ok := msg.(Bbo)
		if !ok {
			callback(Bbo{}, fmt.Errorf("invalid message type"))
			return
		}

		callback(bbo, nil)
	})
}
