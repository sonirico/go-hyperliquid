package hyperliquid

import "fmt"

type ClearinghouseStateSubscriptionParams struct {
	User string
	Dex  *string
}

func (w *WebsocketClient) ClearinghouseState(
	params ClearinghouseStateSubscriptionParams,
	callback func(ClearinghouseState, error),
) (*Subscription, error) {
	payload := remoteClearinghouseStateSubscriptionPayload{
		Type: ChannelClearinghouseState,
		User: params.User,
		Dex:  params.Dex,
	}

	return w.subscribe(payload, func(msg any) {
		state, ok := msg.(ClearinghouseState)
		if !ok {
			callback(ClearinghouseState{}, fmt.Errorf("invalid message type"))
			return
		}

		callback(state, nil)
	})
}
