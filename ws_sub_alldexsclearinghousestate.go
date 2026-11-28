package hyperliquid

import "fmt"

type AllDexsClearinghouseStateSubscriptionParams struct {
	User string
}

func (w *WebsocketClient) AllDexsClearinghouseState(
	params AllDexsClearinghouseStateSubscriptionParams,
	callback func(WsAllDexsClearinghouseState, error),
) (*Subscription, error) {
	payload := remoteAllDexsClearinghouseStateSubscriptionPayload{
		Type: ChannelAllDexsClearinghouseState,
		User: params.User,
	}

	return w.subscribe(payload, func(msg any) {
		allDexsClearinghouseStateMsg, ok := msg.(WsAllDexsClearinghouseState)
		if !ok {
			callback(WsAllDexsClearinghouseState{}, fmt.Errorf("invalid message type"))
			return
		}

		callback(allDexsClearinghouseStateMsg, nil)
	})
}
