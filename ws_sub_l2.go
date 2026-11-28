package hyperliquid

import (
	"encoding/json"
	"fmt"
)

// UnmarshalJSON handles data that is a snapshot under s, a diff under u or a compressed diff under c, see unmarshalCompressed.
// A compressed diff is decoded into U.
func (l *WsL2) UnmarshalJSON(data []byte) error {
	var raw struct {
		S *L2Book         `json:"s"`
		U *WsL2Diff       `json:"u"`
		C json.RawMessage `json:"c"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("l2: %w", err)
	}

	l.S, l.U = raw.S, raw.U
	if raw.C != nil {
		var diff WsL2Diff
		if err := unmarshalCompressed("l2", raw.C, &diff); err != nil {
			return err
		}
		l.U = &diff
	}

	return nil
}

type L2SubscriptionParams struct {
	Coin     string
	NSigFigs int
	Mantissa int
}

func (w *WebsocketClient) L2(
	params L2SubscriptionParams,
	callback func(WsL2, error),
) (*Subscription, error) {
	payload := remoteL2SubscriptionPayload{
		Type:     ChannelL2,
		Coin:     params.Coin,
		NSigFigs: params.NSigFigs,
		Mantissa: params.Mantissa,
	}

	return w.subscribe(payload, func(msg any) {
		l2, ok := msg.(WsL2)
		if !ok {
			callback(WsL2{}, fmt.Errorf("invalid message type"))
			return
		}

		callback(l2, nil)
	})
}
