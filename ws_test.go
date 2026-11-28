package hyperliquid

import (
	"bytes"
	"compress/flate"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func TestNewWebsocketClient(t *testing.T) {
	require.PanicsWithValue(t,
		"baseURL must have a scheme set, either wss or ws",
		func() { _ = NewWebsocketClient("foobar.com") },
	)

	require.NotPanics(t,
		func() { _ = NewWebsocketClient(MainnetAPIURL) },
		"Mainnet should always work",
	)

	require.NotPanics(t,
		func() { _ = NewWebsocketClient("") },
		"empty URL should default to Mainnet",
	)
}

func TestWsOptReadTimeout(t *testing.T) {
	client := NewWebsocketClient(MainnetAPIURL, WsOptReadTimeout(42*time.Second))
	require.Equal(t, 42*time.Second, client.readTimeout)
}

// TestReadPumpReconnectsOnTimeout spins up a WebSocket server that accepts
// connections but never sends a message.  The client should time out and
// reconnect, resulting in more than one TCP-level upgrade.
func TestReadPumpReconnectsOnTimeout(t *testing.T) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(*http.Request) bool { return true },
	}
	var connectCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		connectCount.Add(1)
		// Hold the connection open; drain any frames the client sends (e.g. ping)
		// so the TCP link itself stays alive — only application-layer data is absent.
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				_ = conn.Close()
				return
			}
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 200 ms timeout gives us several reconnect cycles within the 3 s context.
	client := NewWebsocketClient(server.URL, WsOptReadTimeout(200*time.Millisecond))
	require.NoError(t, client.Connect(ctx))

	// Allow enough wall-clock time for multiple timeout → reconnect cycles.
	time.Sleep(2 * time.Second)

	require.GreaterOrEqual(t, int(connectCount.Load()), 2,
		"client should have reconnected at least once after read timeout")

	require.NoError(t, client.Close())
}

// TestReconnectBackoffResetsAfterSuccess checks that the reconnect backoff
// starts over from the initial wait on each outage instead of carrying on from
// where the previous outage left it.
func TestReconnectBackoffResetsAfterSuccess(t *testing.T) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(*http.Request) bool { return true },
	}

	// Dial 1 is the initial Connect. Dials 2-4 fail (first outage, the wait
	// doubles 50 -> 100 -> 200 ms), dial 5 succeeds, dials 6-7 fail (second
	// outage), dial 8 succeeds.
	failing := map[int]bool{2: true, 3: true, 4: true, 6: true, 7: true}

	var (
		mu    sync.Mutex
		dials []time.Time
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		dials = append(dials, time.Now())
		n := len(dials)
		mu.Unlock()

		if failing[n] {
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		// Stay silent so the client's read timeout ends the connection.
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				_ = conn.Close()
				return
			}
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const initialWait = 50 * time.Millisecond
	client := NewWebsocketClient(server.URL, WsOptReadTimeout(100*time.Millisecond))
	client.reconnectWait = initialWait
	require.NoError(t, client.Connect(ctx))
	defer func() { require.NoError(t, client.Close()) }()

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(dials) >= 7
	}, 4*time.Second, 10*time.Millisecond, "client should keep reconnecting")

	mu.Lock()
	gap := dials[6].Sub(dials[5]) // wait after the first failed dial of outage 2
	mu.Unlock()

	// Without the reset the wait here would be 400 ms (the doubled value left
	// over from the first outage); with it, it is back to 50 ms.
	require.Less(t, gap, 4*initialWait,
		"backoff should restart from the initial wait after a successful reconnect, got %v", gap)
	require.Equal(t, initialWait, client.reconnectWait,
		"initial reconnect wait must not be changed by the backoff")
}

// TestServerErrorChannel checks that messages on the server's "error" channel
// are dispatched (not rejected as an unknown channel) and passed to the
// handler set with WsOptErrorHandler.
func TestServerErrorChannel(t *testing.T) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(*http.Request) bool { return true },
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		_ = conn.WriteMessage(websocket.TextMessage,
			[]byte(`{"channel":"error","data":"Invalid subscription {\"type\":\"foo\"}"}`))
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer server.Close()

	errCh := make(chan error, 1)
	client := NewWebsocketClient(server.URL, WsOptErrorHandler(func(err error) {
		errCh <- err
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	require.NoError(t, client.Connect(ctx))
	defer func() { require.NoError(t, client.Close()) }()

	select {
	case err := <-errCh:
		var srvErr WsServerError
		require.True(t, errors.As(err, &srvErr))
		require.Equal(t, `Invalid subscription {"type":"foo"}`, srvErr.Message)
		require.Contains(t, err.Error(), "websocket server error")
	case <-ctx.Done():
		t.Fatal("error handler was not called")
	}
}

func TestServerErrorChannelWithoutHandler(t *testing.T) {
	client := NewWebsocketClient(MainnetAPIURL)
	err := client.dispatch(wsMessage{Channel: ChannelError, Data: []byte(`"boom"`)})
	require.NoError(t, err, "error channel should have a dispatcher")

	// Non-string payloads are passed through as raw JSON.
	var got error
	client = NewWebsocketClient(MainnetAPIURL, WsOptErrorHandler(func(err error) { got = err }))
	require.NoError(t, client.dispatch(wsMessage{Channel: ChannelError, Data: []byte(`{"code":1}`)}))
	require.Equal(t, WsServerError{Message: `{"code":1}`}, got)
}

func TestWsMessage_AllMids(t *testing.T) {
	t.Run("covers every key of the message data", func(t *testing.T) {
		raw := newTestWsFixtureData(t, "ws_allMids")

		assertKeysCovered(t, raw, &AllMids{})
	})
}

func TestWsMessage_L2Book(t *testing.T) {
	t.Run("covers every key of the message data", func(t *testing.T) {
		raw := newTestWsFixtureData(t, "ws_l2Book")

		assertKeysCovered(t, raw, &L2Book{})
	})
}

func TestWsMessage_Trades(t *testing.T) {
	t.Run("covers every key of the message data", func(t *testing.T) {
		raw := newTestWsFixtureData(t, "ws_trades")

		assertKeysCovered(t, raw, &Trades{})
	})
}

func TestWsMessage_Bbo(t *testing.T) {
	t.Run("covers every key of the message data", func(t *testing.T) {
		raw := newTestWsFixtureData(t, "ws_bbo")

		assertKeysCovered(t, raw, &Bbo{})
	})
}

func TestWsMessage_Candle(t *testing.T) {
	t.Run("covers every key of the message data", func(t *testing.T) {
		raw := newTestWsFixtureData(t, "ws_candle")

		assertKeysCovered(t, raw, &Candle{})
	})
}

func TestWsMessage_ActiveAssetCtx(t *testing.T) {
	t.Run("covers every key of the perp message data", func(t *testing.T) {
		raw := newTestWsFixtureData(t, "ws_activeAssetCtx")

		assertKeysCovered(t, raw, &ActiveAssetCtx{})
	})

	t.Run("covers every key of the spot message data", func(t *testing.T) {
		raw := newTestWsFixtureData(t, "ws_activeSpotAssetCtx")

		assertKeysCovered(t, raw, &ActiveAssetCtx{})
	})
}

func TestWsMessage_AllDexsAssetCtxs(t *testing.T) {
	t.Run("covers every key of the message data", func(t *testing.T) {
		raw := newTestWsFixtureData(t, "ws_allDexsAssetCtxs")

		assertKeysCovered(t, raw, &WsAllDexsAssetCtxs{})
	})
}

func TestWsMessage_FastAssetCtxs(t *testing.T) {
	t.Run("covers every key of the inflated message data", func(t *testing.T) {
		var compressed string
		require.NoError(t, json.Unmarshal(newTestWsFixtureData(t, "ws_fastAssetCtxs"), &compressed))
		packed, err := base64.StdEncoding.DecodeString(compressed)
		require.NoError(t, err)
		raw, err := io.ReadAll(flate.NewReader(bytes.NewReader(packed)))
		require.NoError(t, err)

		assertKeysCovered(t, raw, &map[string]FastAssetCtx{})
	})
}

func TestWsMessage_ClearinghouseState(t *testing.T) {
	t.Run("covers every key of the message data", func(t *testing.T) {
		raw := newTestWsFixtureData(t, "ws_clearinghouseState")

		assertKeysCovered(t, raw, &ClearinghouseStateMessage{})
	})
}

func TestWsMessage_OpenOrders(t *testing.T) {
	t.Run("covers every key of the message data", func(t *testing.T) {
		raw := newTestWsFixtureData(t, "ws_openOrders")

		assertKeysCovered(t, raw, &OpenOrders{})
	})
}

func TestWsMessage_UserFills(t *testing.T) {
	t.Run("covers every key of the message data", func(t *testing.T) {
		raw := newTestWsFixtureData(t, "ws_userFills")

		assertKeysCovered(t, raw, &WsOrderFills{})
	})
}

func TestWsMessage_OrderUpdates(t *testing.T) {
	t.Run("covers every key of the message data", func(t *testing.T) {
		raw := newTestWsFixtureData(t, "ws_orderUpdates")

		assertKeysCovered(t, raw, &WsOrders{})
	})
}

func TestWsMessage_TwapStates(t *testing.T) {
	t.Run("covers every key of the message data", func(t *testing.T) {
		raw := newTestWsFixtureData(t, "ws_twapStates")

		assertKeysCovered(t, raw, &TwapStates{})
	})
}

func TestWsMessage_UserEvents(t *testing.T) {
	t.Run("covers every key of the message data", func(t *testing.T) {
		raw := newTestWsFixtureData(t, "ws_userEvents")

		assertKeysCovered(t, raw, &UserEvents{})
	})
}

func TestWsMessage_UserFundings(t *testing.T) {
	t.Run("covers every key of the message data", func(t *testing.T) {
		raw := newTestWsFixtureData(t, "ws_userFundings")

		assertKeysCovered(t, raw, &UserFundings{})
	})
}

func TestWsMessage_UserNonFundingLedgerUpdates(t *testing.T) {
	t.Run("covers every key of the message data", func(t *testing.T) {
		raw := newTestWsFixtureData(t, "ws_userNonFundingLedgerUpdates")

		assertKeysCovered(t, raw, &WsUserNonFundingLedgerUpdates{})
	})
}

func TestWsMessage_ActiveAssetData(t *testing.T) {
	t.Run("covers every key of the message data", func(t *testing.T) {
		raw := newTestWsFixtureData(t, "ws_activeAssetData")

		assertKeysCovered(t, raw, &UserActiveAssetData{})
	})
}

func TestWsMessage_UserTwapSliceFills(t *testing.T) {
	t.Run("covers every key of the message data", func(t *testing.T) {
		raw := newTestWsFixtureData(t, "ws_userTwapSliceFills")

		assertKeysCovered(t, raw, &WsUserTwapSliceFills{})
	})
}

func TestWsMessage_UserTwapHistory(t *testing.T) {
	t.Run("covers every key of the message data", func(t *testing.T) {
		raw := newTestWsFixtureData(t, "ws_userTwapHistory")

		assertKeysCovered(t, raw, &WsUserTwapHistory{})
	})
}

func TestWsMessage_SpotState(t *testing.T) {
	t.Run("covers every key of the message data", func(t *testing.T) {
		raw := newTestWsFixtureData(t, "ws_spotState")

		assertKeysCovered(t, raw, &WsSpotState{})
	})
}

func TestWsMessage_AllDexsClearinghouseState(t *testing.T) {
	t.Run("covers every key of the message data", func(t *testing.T) {
		raw := newTestWsFixtureData(t, "ws_allDexsClearinghouseState")

		assertKeysCovered(t, raw, &WsAllDexsClearinghouseState{})
	})
}

func TestWsMessage_OutcomeMetaUpdates(t *testing.T) {
	t.Run("covers every key of the message data", func(t *testing.T) {
		raw := newTestWsFixtureData(t, "ws_outcomeMetaUpdates_docs")

		assertKeysCovered(t, raw, &OutcomeMetaUpdates{})
	})
}

func TestWsMessage_L2(t *testing.T) {
	t.Run("covers every key of the message data", func(t *testing.T) {
		raw := newTestWsFixtureData(t, "ws_l2")

		assertKeysCovered(t, raw, &WsL2{})
	})
}

func TestWsMessage_Pac(t *testing.T) {
	t.Run("covers every key of the inflated message data", func(t *testing.T) {
		var compressed string
		require.NoError(t, json.Unmarshal(newTestWsFixtureData(t, "ws_pac"), &compressed))
		packed, err := base64.StdEncoding.DecodeString(compressed)
		require.NoError(t, err)
		raw, err := io.ReadAll(flate.NewReader(bytes.NewReader(packed)))
		require.NoError(t, err)

		assertKeysCovered(t, raw, &[]Tuple2[string, []PacAssetCtx]{})
	})
}

func TestWsMessage_Sac(t *testing.T) {
	t.Run("covers every key of the inflated message data", func(t *testing.T) {
		var compressed string
		require.NoError(t, json.Unmarshal(newTestWsFixtureData(t, "ws_sac"), &compressed))
		packed, err := base64.StdEncoding.DecodeString(compressed)
		require.NoError(t, err)
		raw, err := io.ReadAll(flate.NewReader(bytes.NewReader(packed)))
		require.NoError(t, err)

		assertKeysCovered(t, raw, &map[string]SacAssetCtx{})
	})
}

func TestWsMessage_WebData3(t *testing.T) {
	t.Run("covers every key of the message data", func(t *testing.T) {
		raw := newTestWsFixtureData(t, "ws_webData3")

		assertKeysCovered(t, raw, &WebData3{})
	})
}

func TestWsMessage_Notification(t *testing.T) {
	t.Run("covers every key of the message data", func(t *testing.T) {
		raw := newTestWsFixtureData(t, "ws_notification_docs")

		assertKeysCovered(t, raw, &Notification{})
	})
}

const (
	testWsWait      = 3 * time.Second
	testWsUser      = "0xb19e5191b3a97657807c704d58e2ffd33aa4ccdf"
	testWsVaultUser = "0xdfc24b077bc1425ad1dea75bcb6f8158e10df303"
	testWsTwapUser  = "0xdcea581d9d582a0053261bbab53604066f293c74"
)

type testWsSubscribe struct {
	client  *WebsocketClient
	payload chan []byte
}

// newTestWsSubscribe starts a fake ws server that answers every frame it
// receives with the fixture and hands the received frames to payload.
func newTestWsSubscribe(t *testing.T, fixture []byte) testWsSubscribe {
	t.Helper()

	upgrader := websocket.Upgrader{
		CheckOrigin: func(*http.Request) bool { return true },
	}
	payload := make(chan []byte, 8)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		for {
			_, frame, err := conn.ReadMessage()
			if err != nil {
				return
			}
			payload <- frame
			if err := conn.WriteMessage(websocket.TextMessage, fixture); err != nil {
				return
			}
		}
	}))
	t.Cleanup(server.Close)

	ctx, cancel := context.WithTimeout(context.Background(), testWsWait)
	t.Cleanup(cancel)
	client := NewWebsocketClient(server.URL)
	require.NoError(t, client.Connect(ctx))
	t.Cleanup(func() { require.NoError(t, client.Close()) })

	return testWsSubscribe{client: client, payload: payload}
}

func (f testWsSubscribe) requireSubscription(t *testing.T, want string) {
	t.Helper()

	select {
	case frame := <-f.payload:
		require.JSONEq(t, `{"method":"subscribe","subscription":`+want+`}`, string(frame))
	case <-time.After(testWsWait):
		t.Fatal("no subscribe frame received")
	}
}

func requireTestWsMessage[T any](t *testing.T, got <-chan T) T {
	t.Helper()

	select {
	case msg := <-got:
		return msg
	case <-time.After(testWsWait):
		t.Fatal("callback was not called")
		panic("unreachable")
	}
}

func TestActiveAssetCtxSubscription(t *testing.T) {
	t.Run("delivers a spot activeAssetCtx message", func(t *testing.T) {
		f := newTestWsSubscribe(t, newTestLiveFixture(t, "ws_activeSpotAssetCtx"))
		got := make(chan ActiveAssetCtx, 1)

		_, err := f.client.ActiveAssetCtx(
			ActiveAssetCtxSubscriptionParams{Coin: "@107"},
			func(msg ActiveAssetCtx, err error) {
				if err == nil {
					got <- msg
				}
			},
		)
		require.NoError(t, err)

		msg := requireTestWsMessage(t, got)
		require.Equal(t, "@107", msg.Coin)
		require.Equal(t, 62.572, msg.Ctx.MarkPx)
	})
}

func TestUserEventsSubscription(t *testing.T) {
	t.Run("sends the payload and delivers the fixture", func(t *testing.T) {
		f := newTestWsSubscribe(t, newTestLiveFixture(t, "ws_userEvents"))
		got := make(chan UserEvents, 1)

		_, err := f.client.UserEvents(
			UserEventsSubscriptionParams{User: testWsUser},
			func(msg UserEvents, err error) {
				if err == nil {
					got <- msg
				}
			},
		)
		require.NoError(t, err)

		f.requireSubscription(t, `{"type":"userEvents","user":"`+testWsUser+`"}`)
		msg := requireTestWsMessage(t, got)
		require.Len(t, msg.Fills, 1)
		require.Equal(t, "BTC", msg.Fills[0].Coin)
	})
}

func TestUserFundingsSubscription(t *testing.T) {
	t.Run("sends the payload and delivers the fixture", func(t *testing.T) {
		f := newTestWsSubscribe(t, newTestLiveFixture(t, "ws_userFundings"))
		got := make(chan UserFundings, 1)

		_, err := f.client.UserFundings(
			UserFundingsSubscriptionParams{User: testWsUser},
			func(msg UserFundings, err error) {
				if err == nil {
					got <- msg
				}
			},
		)
		require.NoError(t, err)

		f.requireSubscription(t, `{"type":"userFundings","user":"`+testWsUser+`"}`)
		msg := requireTestWsMessage(t, got)
		require.True(t, msg.IsSnapshot)
	})
}

func TestUserNonFundingLedgerUpdatesSubscription(t *testing.T) {
	t.Run("sends the payload and delivers the fixture", func(t *testing.T) {
		f := newTestWsSubscribe(t, newTestLiveFixture(t, "ws_userNonFundingLedgerUpdates"))
		got := make(chan WsUserNonFundingLedgerUpdates, 1)

		_, err := f.client.UserNonFundingLedgerUpdates(
			UserNonFundingLedgerUpdatesSubscriptionParams{User: testWsVaultUser},
			func(msg WsUserNonFundingLedgerUpdates, err error) {
				if err == nil {
					got <- msg
				}
			},
		)
		require.NoError(t, err)

		f.requireSubscription(t, `{"type":"userNonFundingLedgerUpdates","user":"`+testWsVaultUser+`"}`)
		msg := requireTestWsMessage(t, got)
		require.True(t, msg.IsSnapshot)
		require.NotEmpty(t, msg.NonFundingLedgerUpdates)
	})
}

func TestActiveAssetDataSubscription(t *testing.T) {
	t.Run("sends the payload and delivers the fixture", func(t *testing.T) {
		f := newTestWsSubscribe(t, newTestLiveFixture(t, "ws_activeAssetData"))
		got := make(chan UserActiveAssetData, 1)

		_, err := f.client.ActiveAssetData(
			ActiveAssetDataSubscriptionParams{User: testWsUser, Coin: "BTC"},
			func(msg UserActiveAssetData, err error) {
				if err == nil {
					got <- msg
				}
			},
		)
		require.NoError(t, err)

		f.requireSubscription(t, `{"type":"activeAssetData","user":"`+testWsUser+`","coin":"BTC"}`)
		msg := requireTestWsMessage(t, got)
		require.Equal(t, "BTC", msg.Coin)
	})
}

func TestUserTwapSliceFillsSubscription(t *testing.T) {
	t.Run("sends the payload and delivers the fixture", func(t *testing.T) {
		f := newTestWsSubscribe(t, newTestLiveFixture(t, "ws_userTwapSliceFills"))
		got := make(chan WsUserTwapSliceFills, 1)

		_, err := f.client.UserTwapSliceFills(
			UserTwapSliceFillsSubscriptionParams{User: testWsTwapUser},
			func(msg WsUserTwapSliceFills, err error) {
				if err == nil {
					got <- msg
				}
			},
		)
		require.NoError(t, err)

		f.requireSubscription(t, `{"type":"userTwapSliceFills","user":"`+testWsTwapUser+`"}`)
		msg := requireTestWsMessage(t, got)
		require.True(t, msg.IsSnapshot)
		require.NotEmpty(t, msg.TwapSliceFills)
	})
}

func TestUserTwapHistorySubscription(t *testing.T) {
	t.Run("sends the payload and delivers the fixture", func(t *testing.T) {
		f := newTestWsSubscribe(t, newTestLiveFixture(t, "ws_userTwapHistory"))
		got := make(chan WsUserTwapHistory, 1)

		_, err := f.client.UserTwapHistory(
			UserTwapHistorySubscriptionParams{User: testWsTwapUser},
			func(msg WsUserTwapHistory, err error) {
				if err == nil {
					got <- msg
				}
			},
		)
		require.NoError(t, err)

		f.requireSubscription(t, `{"type":"userTwapHistory","user":"`+testWsTwapUser+`"}`)
		msg := requireTestWsMessage(t, got)
		require.True(t, msg.IsSnapshot)
		require.NotEmpty(t, msg.History)
	})
}

func TestSpotStateSubscription(t *testing.T) {
	t.Run("sends the payload and delivers the fixture", func(t *testing.T) {
		f := newTestWsSubscribe(t, newTestLiveFixture(t, "ws_spotState"))
		got := make(chan WsSpotState, 1)

		_, err := f.client.SpotState(
			SpotStateSubscriptionParams{User: testWsUser},
			func(msg WsSpotState, err error) {
				if err == nil {
					got <- msg
				}
			},
		)
		require.NoError(t, err)

		f.requireSubscription(t, `{"type":"spotState","user":"`+testWsUser+`"}`)
		msg := requireTestWsMessage(t, got)
		require.Equal(t, testWsUser, msg.User)
	})
}

func TestAllDexsClearinghouseStateSubscription(t *testing.T) {
	t.Run("sends the payload and delivers the fixture", func(t *testing.T) {
		f := newTestWsSubscribe(t, newTestLiveFixture(t, "ws_allDexsClearinghouseState"))
		got := make(chan WsAllDexsClearinghouseState, 1)

		_, err := f.client.AllDexsClearinghouseState(
			AllDexsClearinghouseStateSubscriptionParams{User: testWsUser},
			func(msg WsAllDexsClearinghouseState, err error) {
				if err == nil {
					got <- msg
				}
			},
		)
		require.NoError(t, err)

		f.requireSubscription(t, `{"type":"allDexsClearinghouseState","user":"`+testWsUser+`"}`)
		msg := requireTestWsMessage(t, got)
		require.NotEmpty(t, msg.ClearinghouseStates)
		require.Equal(t, "", msg.ClearinghouseStates[0].First)
	})
}

func TestOutcomeMetaUpdatesSubscription(t *testing.T) {
	t.Run("sends the payload and delivers the fixture", func(t *testing.T) {
		f := newTestWsSubscribe(t, newTestLiveFixture(t, "ws_outcomeMetaUpdates_docs"))
		got := make(chan OutcomeMetaUpdates, 1)

		_, err := f.client.OutcomeMetaUpdates(
			func(msg OutcomeMetaUpdates, err error) {
				if err == nil {
					got <- msg
				}
			},
		)
		require.NoError(t, err)

		f.requireSubscription(t, `{"type":"outcomeMetaUpdates"}`)
		msg := requireTestWsMessage(t, got)
		require.Len(t, msg, 4)
		require.Equal(t, 1, msg[0].OutcomeCreated.Outcome)
	})
}

func TestWsL2_UnmarshalJSON(t *testing.T) {
	const diff = `{"c":"BTC","t":1,"l":[[{"p":"10","s":"2"}],[]],"r":[[],[0]]}`
	want := WsL2Diff{
		Coin:    "BTC",
		Time:    1,
		Levels:  [][]WsL2LevelUpdate{{{Px: "10", Sz: "2"}}, {}},
		Removed: [][]int{{}, {0}},
	}

	t.Run("decodes an uncompressed diff", func(t *testing.T) {
		var got WsL2

		require.NoError(t, json.Unmarshal([]byte(`{"u":`+diff+`}`), &got))

		require.Equal(t, &want, got.U)
		require.Nil(t, got.S)
	})

	t.Run("decodes a compressed diff", func(t *testing.T) {
		var packed bytes.Buffer
		writer, err := flate.NewWriter(&packed, flate.DefaultCompression)
		require.NoError(t, err)
		_, err = writer.Write([]byte(diff))
		require.NoError(t, err)
		require.NoError(t, writer.Close())
		var got WsL2

		require.NoError(t, json.Unmarshal(
			[]byte(`{"c":"`+base64.StdEncoding.EncodeToString(packed.Bytes())+`"}`), &got))

		require.Equal(t, &want, got.U)
	})
}

func TestL2Subscription(t *testing.T) {
	t.Run("sends the payload and delivers the fixture", func(t *testing.T) {
		f := newTestWsSubscribe(t, newTestLiveFixture(t, "ws_l2"))
		got := make(chan WsL2, 1)

		_, err := f.client.L2(
			L2SubscriptionParams{Coin: "BTC", NSigFigs: 5},
			func(msg WsL2, err error) {
				if err == nil {
					got <- msg
				}
			},
		)
		require.NoError(t, err)

		f.requireSubscription(t, `{"type":"l2","c":"BTC","s":5}`)
		msg := requireTestWsMessage(t, got)
		require.NotNil(t, msg.S)
		require.Equal(t, "BTC", msg.S.Coin)
	})
}

func TestPacSubscription(t *testing.T) {
	t.Run("sends the payload and delivers the fixture", func(t *testing.T) {
		f := newTestWsSubscribe(t, newTestLiveFixture(t, "ws_pac"))
		got := make(chan WsPac, 1)

		_, err := f.client.Pac(func(msg WsPac, err error) {
			if err == nil {
				got <- msg
			}
		})
		require.NoError(t, err)

		f.requireSubscription(t, `{"type":"pac"}`)
		msg := requireTestWsMessage(t, got)
		require.NotEmpty(t, msg)
		require.Equal(t, "", msg[0].First)
		require.NotEmpty(t, msg[0].Second)
	})
}

func TestSacSubscription(t *testing.T) {
	t.Run("sends the payload and delivers the fixture", func(t *testing.T) {
		f := newTestWsSubscribe(t, newTestLiveFixture(t, "ws_sac"))
		got := make(chan WsSac, 1)

		_, err := f.client.Sac(func(msg WsSac, err error) {
			if err == nil {
				got <- msg
			}
		})
		require.NoError(t, err)

		f.requireSubscription(t, `{"type":"sac"}`)
		msg := requireTestWsMessage(t, got)
		require.NotEmpty(t, msg)
	})
}
