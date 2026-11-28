package hyperliquid

import "fmt"

type UserFundingsSubscriptionParams struct {
	User string
}

func (w *WebsocketClient) UserFundings(
	params UserFundingsSubscriptionParams,
	callback func(UserFundings, error),
) (*Subscription, error) {
	payload := remoteUserFundingsSubscriptionPayload{
		Type: ChannelUserFundings,
		User: params.User,
	}

	return w.subscribe(payload, func(msg any) {
		userFundingsMsg, ok := msg.(UserFundings)
		if !ok {
			callback(UserFundings{}, fmt.Errorf("invalid message type"))
			return
		}

		callback(userFundingsMsg, nil)
	})
}
