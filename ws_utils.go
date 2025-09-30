package hyperliquid

import (
	"strings"

	"github.com/sonirico/vago/fp"
)

func key(args ...string) string {
	return strings.Join(args, ":")
}

func keyTrades(coin string) string {
	return key(ChannelTrades, coin)
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

func keyOrderFills(user string) string {
	// Order fills are user-specific but don't contain user info in the message itself.
	// The dispatching is handled by the subscription system based on the subscription key.
	return key(ChannelUserFills, user)
}

func keyWebData2(_ string) string {
	// WebData2 messages are user-specific but don't contain user info in the message itself.
	// The dispatching is handled by the subscription system based on the subscription key.
	return key(ChannelWebData2)
}

func keyBbo(coin string) string {
	return key(ChannelBbo, coin)
}
