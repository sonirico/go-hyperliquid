package hyperliquid

import (
	"fmt"
)

type ActiveAssetCtxSubscriptionParams struct {
	Coin string
}

func (w *WebsocketClient) ActiveAssetCtx(
	params ActiveAssetCtxSubscriptionParams,
	callback func(ActiveAssetCtx, error),
) (*Subscription, error) {
	remotePayload := remoteActiveAssetCtxSubscriptionPayload{
		Type: ChannelActiveAssetCtx,
		Coin: params.Coin,
	}

	return w.subscribe(remotePayload, func(msg any) {
		activeAssetCtx, ok := msg.(ActiveAssetCtx)
		if !ok {
			callback(ActiveAssetCtx{}, fmt.Errorf("SubscribeToActiveAssetCtx invalid message type"))
			return
		}

		callback(activeAssetCtx, nil)
	})
}
