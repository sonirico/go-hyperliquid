package hyperliquid

import "fmt"

type WebData3SubscriptionParams struct {
	User string
	Dex  *string
}

func (w *WebsocketClient) WebData3(
	params WebData3SubscriptionParams,
	callback func(WebData3, error),
) (*Subscription, error) {
	payload := remoteWebData3SubscriptionPayload{
		Type: ChannelWebData3,
		User: params.User,
		Dex:  params.Dex,
	}

	return w.subscribe(payload, func(msg any) {
		webdata3, ok := msg.(WebData3)
		if !ok {
			callback(WebData3{}, fmt.Errorf("invalid message type"))
			return
		}

		callback(webdata3, nil)
	})
}
