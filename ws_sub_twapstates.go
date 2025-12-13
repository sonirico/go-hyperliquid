package hyperliquid

import "fmt"

type TwapStatesSubscriptionParams struct {
	User string
	Dex  *string
}

func (w *WebsocketClient) TwapStates(
	params TwapStatesSubscriptionParams,
	callback func(TwapStates, error),
) (*Subscription, error) {
	payload := remoteTwapStatesSubscriptionPayload{
		Type: ChannelTwapStates,
		User: params.User,
		Dex:  params.Dex,
	}

	return w.subscribe(payload, func(msg any) {
		states, ok := msg.(TwapStates)
		if !ok {
			callback(TwapStates{}, fmt.Errorf("invalid message type"))
			return
		}

		callback(states, nil)
	})
}
