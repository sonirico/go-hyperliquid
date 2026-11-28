package hyperliquid

import "fmt"

func (w *WebsocketClient) OutcomeMetaUpdates(
	callback func(OutcomeMetaUpdates, error),
) (*Subscription, error) {
	payload := remoteOutcomeMetaUpdatesSubscriptionPayload{
		Type: ChannelOutcomeMetaUpdates,
	}

	return w.subscribe(payload, func(msg any) {
		outcomeMetaUpdatesMsg, ok := msg.(OutcomeMetaUpdates)
		if !ok {
			callback(OutcomeMetaUpdates{}, fmt.Errorf("invalid message type"))
			return
		}

		callback(outcomeMetaUpdatesMsg, nil)
	})
}
