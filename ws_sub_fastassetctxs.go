package hyperliquid

import (
	"bytes"
	"compress/flate"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
)

// Data for this type is compressed, decode the base64 then uncompress (RFC 1951, raw — no zlib/gzip wrapper).
func (w *WsFastAssetCtxs) UnmarshalJSON(data []byte) error {
	var b64 string
	if err := json.Unmarshal(data, &b64); err != nil {
		return fmt.Errorf("fastAssetCtxs: data is not a string: %w", err)
	}

	compressed, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return fmt.Errorf("fastAssetCtxs: base64 decode: %w", err)
	}

	raw, err := io.ReadAll(flate.NewReader(bytes.NewReader(compressed)))
	if err != nil {
		return fmt.Errorf("fastAssetCtxs: inflate: %w", err)
	}

	var ctxs map[string]FastAssetCtx
	if err := json.Unmarshal(raw, &ctxs); err != nil {
		return fmt.Errorf("fastAssetCtxs: json decode: %w", err)
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
