package hyperliquid

import "fmt"

type UserNonFundingLedgerUpdatesSubscriptionParams struct {
	User string
}

func (w *WebsocketClient) UserNonFundingLedgerUpdates(
	params UserNonFundingLedgerUpdatesSubscriptionParams,
	callback func(WsUserNonFundingLedgerUpdates, error),
) (*Subscription, error) {
	payload := remoteUserNonFundingLedgerUpdatesSubscriptionPayload{
		Type: ChannelUserNonFundingLedgerUpdates,
		User: params.User,
	}

	return w.subscribe(payload, func(msg any) {
		userNonFundingLedgerUpdatesMsg, ok := msg.(WsUserNonFundingLedgerUpdates)
		if !ok {
			callback(WsUserNonFundingLedgerUpdates{}, fmt.Errorf("invalid message type"))
			return
		}

		callback(userNonFundingLedgerUpdatesMsg, nil)
	})
}
