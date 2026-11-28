package hyperliquid

import "fmt"

type UserTwapHistorySubscriptionParams struct {
	User string
}

func (w *WebsocketClient) UserTwapHistory(
	params UserTwapHistorySubscriptionParams,
	callback func(WsUserTwapHistory, error),
) (*Subscription, error) {
	payload := remoteUserTwapHistorySubscriptionPayload{
		Type: ChannelUserTwapHistory,
		User: params.User,
	}

	return w.subscribe(payload, func(msg any) {
		userTwapHistoryMsg, ok := msg.(WsUserTwapHistory)
		if !ok {
			callback(WsUserTwapHistory{}, fmt.Errorf("invalid message type"))
			return
		}

		callback(userTwapHistoryMsg, nil)
	})
}
