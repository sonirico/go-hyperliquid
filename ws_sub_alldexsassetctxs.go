package hyperliquid

import (
	"fmt"
)

func (w *WebsocketClient) AllDexsAssetCtxs(
	callback func(WsAllDexsAssetCtxs, error),
) (*Subscription, error) {
	remotePayload := remoteAllDexsAssetCtxsSubscriptionPayload{
		Type: ChannelAllDexsAssetCtxs,
	}

	return w.subscribe(remotePayload, func(msg any) {
		allDexsAssetCtxs, ok := msg.(WsAllDexsAssetCtxs)
		if !ok {
			callback(
				WsAllDexsAssetCtxs{}, fmt.Errorf("SubscribeToAllDexsAssetCtxs invalid message type"),
			)
			return
		}

		callback(allDexsAssetCtxs, nil)
	})
}
