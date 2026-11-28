package hyperliquid

import "fmt"

type SpotStateSubscriptionParams struct {
	User string
}

func (w *WebsocketClient) SpotState(
	params SpotStateSubscriptionParams,
	callback func(WsSpotState, error),
) (*Subscription, error) {
	payload := remoteSpotStateSubscriptionPayload{
		Type: ChannelSpotState,
		User: params.User,
	}

	return w.subscribe(payload, func(msg any) {
		spotStateMsg, ok := msg.(WsSpotState)
		if !ok {
			callback(WsSpotState{}, fmt.Errorf("invalid message type"))
			return
		}

		callback(spotStateMsg, nil)
	})
}
