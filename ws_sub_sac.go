package hyperliquid

import "fmt"

// Data for this type is compressed, see unmarshalCompressed.
func (s *WsSac) UnmarshalJSON(data []byte) error {
	var ctxs map[string]SacAssetCtx
	if err := unmarshalCompressed("sac", data, &ctxs); err != nil {
		return err
	}

	*s = ctxs
	return nil
}

func (w *WebsocketClient) Sac(
	callback func(WsSac, error),
) (*Subscription, error) {
	payload := remoteSacSubscriptionPayload{
		Type: ChannelSac,
	}

	return w.subscribe(payload, func(msg any) {
		sac, ok := msg.(WsSac)
		if !ok {
			callback(WsSac{}, fmt.Errorf("invalid message type"))
			return
		}

		callback(sac, nil)
	})
}
