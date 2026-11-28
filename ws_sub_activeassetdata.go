package hyperliquid

import "fmt"

type ActiveAssetDataSubscriptionParams struct {
	User string
	Coin string
}

func (w *WebsocketClient) ActiveAssetData(
	params ActiveAssetDataSubscriptionParams,
	callback func(UserActiveAssetData, error),
) (*Subscription, error) {
	payload := remoteActiveAssetDataSubscriptionPayload{
		Type: ChannelActiveAssetData,
		User: params.User,
		Coin: params.Coin,
	}

	return w.subscribe(payload, func(msg any) {
		activeAssetDataMsg, ok := msg.(UserActiveAssetData)
		if !ok {
			callback(UserActiveAssetData{}, fmt.Errorf("invalid message type"))
			return
		}

		callback(activeAssetDataMsg, nil)
	})
}
