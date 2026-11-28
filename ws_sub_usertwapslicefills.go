package hyperliquid

import "fmt"

type UserTwapSliceFillsSubscriptionParams struct {
	User string
}

func (w *WebsocketClient) UserTwapSliceFills(
	params UserTwapSliceFillsSubscriptionParams,
	callback func(WsUserTwapSliceFills, error),
) (*Subscription, error) {
	payload := remoteUserTwapSliceFillsSubscriptionPayload{
		Type: ChannelUserTwapSliceFills,
		User: params.User,
	}

	return w.subscribe(payload, func(msg any) {
		userTwapSliceFillsMsg, ok := msg.(WsUserTwapSliceFills)
		if !ok {
			callback(WsUserTwapSliceFills{}, fmt.Errorf("invalid message type"))
			return
		}

		callback(userTwapSliceFillsMsg, nil)
	})
}
