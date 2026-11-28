package hyperliquid

import "fmt"

// Data for this type is compressed, see unmarshalCompressed.
func (p *WsPac) UnmarshalJSON(data []byte) error {
	var ctxs []Tuple2[string, []PacAssetCtx]
	if err := unmarshalCompressed("pac", data, &ctxs); err != nil {
		return err
	}

	*p = ctxs
	return nil
}

func (w *WebsocketClient) Pac(
	callback func(WsPac, error),
) (*Subscription, error) {
	payload := remotePacSubscriptionPayload{
		Type: ChannelPac,
	}

	return w.subscribe(payload, func(msg any) {
		pac, ok := msg.(WsPac)
		if !ok {
			callback(WsPac{}, fmt.Errorf("invalid message type"))
			return
		}

		callback(pac, nil)
	})
}
