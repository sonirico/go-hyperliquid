package hyperliquid

import (
	"context"
	"errors"
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
