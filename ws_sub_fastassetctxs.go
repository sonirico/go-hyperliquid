package hyperliquid

import "fmt"

// Data for this type is compressed, decode the base64 then uncompress (RFC 1951, raw — no zlib/gzip wrapper).
func (w *WsFastAssetCtxs) UnmarshalJSON(data []byte) error {
	var ctxs map[string]FastAssetCtx
	if err := unmarshalCompressed("fastAssetCtxs", data, &ctxs); err != nil {
		return err
	}

	*w = ctxs
	return nil
}

func (w *WebsocketClient) FastAssetCtxs(
	callback func(WsFastAssetCtxs, error),
) (*Subscription, error) {
	remotePayload := remoteFastAssetCtxsSubscriptionPayload{
		Type: ChannelFastAssetCtxs,
	}

	return w.subscribe(remotePayload, func(msg any) {
		fastAssetCtx, ok := msg.(WsFastAssetCtxs)
		if !ok {
			callback(WsFastAssetCtxs{}, fmt.Errorf("SubscribeToFastAssetCtxs invalid message type"))
			return
		}

		callback(fastAssetCtx, nil)
	})
}
