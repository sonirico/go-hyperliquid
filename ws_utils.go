package hyperliquid

import (
	"bytes"
	"compress/flate"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/sonirico/vago/fp"
)

func key(args ...string) string {
	return strings.Join(args, ":")
}

func keyTrades(coin string) string {
	return key(ChannelTrades, coin)
}

func keyActiveAssetCtx(coin string) string {
	return key(ChannelActiveAssetCtx, coin)
}

func keyFastAssetCtxs() string {
	return key(ChannelFastAssetCtxs)
}

func keyAllDexsAssetCtxs() string {
	return key(ChannelAllDexsAssetCtxs)
}

func keyCandles(symbol, interval string) string {
	return key(ChannelCandle, symbol, interval)
}

func keyL2Book(coin string) string {
	return key(ChannelL2Book, coin)
}

func keyAllMids(_ fp.Option[string]) string {
	// Unfortunately, "dex" parameter is not returned neither in subscription ACK nor in the
	// allMids message, no we are rendered unable to distinguish between different DEXes from
	// subscriber's standpoint.
	// if dex.IsNone() {
	// 	return key(ChannelAllMids)
	// }
	// return key(ChannelAllMids, dex.UnwrapUnsafe())
	return key(ChannelAllMids)
}

func keyNotification(_ string) string {
	// Notification messages are user-specific but don't contain user info in the message itself.
	// The dispatching is handled by the subscription system based on the subscription key.
	return key(ChannelNotification)
}

func keyOrderUpdates(_ string) string {
	// Order updates are user-specific but don't contain user info in the message itself.
	// The dispatching is handled by the subscription system based on the subscription key.
	return key(ChannelOrderUpdates)
}

func keyUserFills(user string) string {
	return key(ChannelUserFills, user)
}

func keyUserEvents(_ string) string {
	// User events are user-specific but don't contain user info in the message itself.
	// The dispatching is handled by the subscription system based on the subscription key.
	return key(ChannelUser)
}

func keyUserFundings(user string) string {
	return key(ChannelUserFundings, user)
}

func keyUserNonFundingLedgerUpdates(user string) string {
	return key(ChannelUserNonFundingLedgerUpdates, user)
}

func keyActiveAssetData(user string, coin string) string {
	return key(ChannelActiveAssetData, user, coin)
}

func keyUserTwapSliceFills(user string) string {
	return key(ChannelUserTwapSliceFills, user)
}

func keyUserTwapHistory(user string) string {
	return key(ChannelUserTwapHistory, user)
}

func keySpotState(user string) string {
	return key(ChannelSpotState, user)
}

func keyAllDexsClearinghouseState(user string) string {
	return key(ChannelAllDexsClearinghouseState, user)
}

func keyOutcomeMetaUpdates() string {
	return key(ChannelOutcomeMetaUpdates)
}

func keyL2(coin string) string {
	return key(ChannelL2, coin)
}

// unmarshalCompressed decodes a base64 string holding a raw DEFLATE stream (RFC 1951, no zlib/gzip wrapper)
// of JSON into v.
func unmarshalCompressed(name string, data []byte, v any) error {
	var b64 string
	if err := json.Unmarshal(data, &b64); err != nil {
		return fmt.Errorf("%s: data is not a string: %w", name, err)
	}

	compressed, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return fmt.Errorf("%s: base64 decode: %w", name, err)
	}

	raw, err := io.ReadAll(flate.NewReader(bytes.NewReader(compressed)))
	if err != nil {
		return fmt.Errorf("%s: inflate: %w", name, err)
	}

	if err := json.Unmarshal(raw, v); err != nil {
		return fmt.Errorf("%s: json decode: %w", name, err)
	}

	return nil
}

func keyBbo(coin string) string {
	return key(ChannelBbo, coin)
}

func keyClearinghouseState(user string, dex fp.Option[string]) string {
	if dex.IsNone() {
		return key(ChannelClearinghouseState, user)
	}
	return key(ChannelClearinghouseState, user, dex.UnwrapUnsafe())
}

func keyOpenOrders(user string, dex fp.Option[string]) string {
	if dex.IsNone() {
		return key(ChannelOpenOrders, user)
	}
	return key(ChannelOpenOrders, user, dex.UnwrapUnsafe())
}

func keyTwapStates(user string, dex fp.Option[string]) string {
	if dex.IsNone() {
		return key(ChannelTwapStates, user)
	}
	return key(ChannelTwapStates, user, dex.UnwrapUnsafe())
}

func keyWebData3(user string, dex fp.Option[string]) string {
	if dex.IsNone() {
		return key(ChannelWebData3, user)
	}
	return key(ChannelWebData3, user, dex.UnwrapUnsafe())
}
