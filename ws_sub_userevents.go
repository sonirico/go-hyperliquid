package hyperliquid

import "fmt"

type UserEventsSubscriptionParams struct {
	User string
}

func (w *WebsocketClient) UserEvents(
	params UserEventsSubscriptionParams,
	callback func(UserEvents, error),
) (*Subscription, error) {
	payload := remoteUserEventsSubscriptionPayload{
		Type: "userEvents",
		User: params.User,
	}

	return w.subscribe(payload, func(msg any) {
		events, ok := msg.(UserEvents)
		if !ok {
			callback(UserEvents{}, fmt.Errorf("invalid message type"))
			return
		}

		callback(events, nil)
	})
}
