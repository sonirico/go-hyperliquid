package hyperliquid

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sonirico/vago/lol"
	"github.com/sonirico/vago/maps"
)

const (
	// pingInterval is the interval for sending ping messages to keep WebSocket alive
	pingInterval = 50 * time.Second
)

type Subscription struct {
	ID      string
	Payload any
	Close   func()
}

type WebsocketClient struct {
	url                   string
	conn                  *websocket.Conn
	mu                    sync.RWMutex
	writeMu               sync.Mutex
	subscribers           map[string]*uniqSubscriber
	msgDispatcherRegistry map[string]msgDispatcher
	nextSubID             atomic.Int64
	done                  chan struct{}
	closeOnce             sync.Once
	reconnectWait         time.Duration
	debug                 bool
	logger                lol.Logger
}

var upstreamHosts map[string]struct{}

func init() {
	mustHost := func(s string) string {
		u, err := url.Parse(s)
		if err != nil {
			panic(fmt.Sprintf("invalid upstream URL %q: %v", s, err))
		}
		return strings.ToLower(u.Hostname())
	}
	upstreamHosts = map[string]struct{}{
		mustHost(MainnetAPIURL): {},
		mustHost(TestnetAPIURL): {},
	}
}

func isUpstream(u *url.URL) bool {
	_, ok := upstreamHosts[strings.ToLower(u.Hostname())]
	return ok
}

func NewWebsocketClient(baseURL string, opts ...WsOpt) *WebsocketClient {
	if baseURL == "" {
		baseURL = MainnetAPIURL
	}
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		log.Fatalf("invalid URL: %v", err)
	}

	// the current usage expects a full address (https://api.hyp..) to keep compatibility check if
	// that host is set and just use the old method. any new caller with their own endpoint will be
	// forced to provide a full URI
	if isUpstream(parsedURL) {
		parsedURL.Scheme = "wss"
		parsedURL.Path = "/ws"
	} else {
		switch parsedURL.Scheme {
		case "https":
			parsedURL.Scheme = "wss"
		case "http":
			parsedURL.Scheme = "ws"
		case "":
			// baseURL has no scheme set, odd
			panic("baseURL must have a scheme set, either wss or ws")
		}
	}

	wsURL := parsedURL.String()

	cli := &WebsocketClient{
		url:           wsURL,
		done:          make(chan struct{}),
		reconnectWait: time.Second,
		subscribers:   make(map[string]*uniqSubscriber),
		msgDispatcherRegistry: map[string]msgDispatcher{
			ChannelPong:           NewPongDispatcher(),
			ChannelTrades:         NewMsgDispatcher[Trades](ChannelTrades),
			ChannelActiveAssetCtx: NewMsgDispatcher[ActiveAssetCtx](ChannelActiveAssetCtx),
			ChannelL2Book:         NewMsgDispatcher[L2Book](ChannelL2Book),
			ChannelCandle:         NewMsgDispatcher[Candle](ChannelCandle),
			ChannelAllMids:        NewMsgDispatcher[AllMids](ChannelAllMids),
			ChannelNotification:   NewMsgDispatcher[Notification](ChannelNotification),
			ChannelOrderUpdates:   NewMsgDispatcher[WsOrders](ChannelOrderUpdates),
			ChannelWebData2:       NewMsgDispatcher[WebData2](ChannelWebData2),
			ChannelBbo:            NewMsgDispatcher[Bbo](ChannelBbo),
			ChannelUserFills:      NewMsgDispatcher[WsOrderFills](ChannelUserFills),
			ChannelSubResponse:    NewNoopDispatcher(),
			ChannelClearinghouseState: NewMsgDispatcher[ClearinghouseStateMessage](
				ChannelClearinghouseState,
			),
			ChannelOpenOrders: NewMsgDispatcher[OpenOrders](ChannelOpenOrders),
			ChannelTwapStates: NewMsgDispatcher[TwapStates](ChannelTwapStates),
			ChannelWebData3:   NewMsgDispatcher[WebData3](ChannelWebData3),
		},
	}

	for _, opt := range opts {
		opt.Apply(cli)
	}

	return cli
}

func (w *WebsocketClient) Connect(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.conn != nil {
		return nil
	}

	dialer := websocket.Dialer{}

	//nolint:bodyclose // WebSocket connections don't have response bodies to close
	conn, _, err := dialer.DialContext(ctx, w.url, nil)
	if err != nil {
		return fmt.Errorf("websocket dial: %w", err)
	}

	w.conn = conn

	go w.readPump(ctx)
	go w.pingPump(ctx)

	return w.resubscribeAll()
}

type Handler[T subscriptable] func(wsMessage) (T, error)

func (w *WebsocketClient) subscribe(
	payload subscriptable,
	callback func(any),
) (*Subscription, error) {
	if callback == nil {
		return nil, fmt.Errorf("callback cannot be nil")
	}

	w.mu.Lock()

	pkey := payload.Key()
	subscriber, exists := w.subscribers[pkey]
	if !exists {
		subscriber = newUniqSubscriber(
			pkey,
			payload,
			// on subscribe
			func(p subscriptable) {
				if err := w.sendSubscribe(p); err != nil {
					w.logErrf("failed to subscribe: %v", err)
				}
			},
			// on unsubscribe
			func(p subscriptable) {
				w.mu.Lock()
				defer w.mu.Unlock()
				delete(w.subscribers, pkey)
				if err := w.sendUnsubscribe(p); err != nil {
					w.logErrf("failed to unsubscribe: %v", err)
				}
			},
		)

		w.subscribers[pkey] = subscriber
	}

	w.mu.Unlock()

	nextID := w.nextSubID.Add(1)
	subID := key(pkey, strconv.Itoa(int(nextID)))
	subscriber.subscribe(subID, callback)

	return &Subscription{
		ID: subID,
		Close: func() {
			subscriber.unsubscribe(subID)
		},
	}, nil
}

func (w *WebsocketClient) Close() error {
	var err error
	w.closeOnce.Do(func() {
		err = w.close()
	})
	return err
}

func (w *WebsocketClient) close() error {
	close(w.done)

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.conn != nil {
		return w.conn.Close()
	}

	for _, subscriber := range w.subscribers {
		subscriber.clear()
	}
	return nil
}

// Private methods

func (w *WebsocketClient) readPump(ctx context.Context) {
	defer func() {
		w.mu.Lock()
		if w.conn != nil {
			_ = w.conn.Close() // Ignore close error in defer
			w.conn = nil
		}
		w.mu.Unlock()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		default:
			_, msg, err := w.conn.ReadMessage()
			if err != nil {
				if !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					w.logErrf("websocket read error: %v", err)
				}
				return
			}

			if w.debug {
				w.logDebugf("[<] %s", string(msg))
			}

			var wsMsg wsMessage
			if err := json.Unmarshal(msg, &wsMsg); err != nil {
				w.logErrf("websocket message parse error: %v", err)
				continue
			}

			if err := w.dispatch(wsMsg); err != nil {
				w.logErrf("failed to dispatch websocket message: %v", err)
			}
		}
	}
}

func (w *WebsocketClient) pingPump(ctx context.Context) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.done:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.sendPing(); err != nil {
				w.logErrf("ping error: %v", err)
				w.reconnect(ctx)
				return
			}
		}
	}
}

func (w *WebsocketClient) dispatch(msg wsMessage) error {
	dispatcher, ok := w.msgDispatcherRegistry[msg.Channel]
	if !ok {
		return fmt.Errorf("no dispatcher for channel: %s", msg.Channel)
	}

	w.mu.RLock()
	subscribers := maps.Values(w.subscribers)
	w.mu.RUnlock()

	return dispatcher.Dispatch(subscribers, msg)
}

func (w *WebsocketClient) reconnect(ctx context.Context) {
	for {
		select {
		case <-w.done:
			return
		case <-ctx.Done():
			return
		default:
			if err := w.Connect(ctx); err == nil {
				return
			}
			time.Sleep(w.reconnectWait)
			w.reconnectWait *= 2 // TODO: configurable strategies such as exponential backoff and the like
			if w.reconnectWait > time.Minute {
				w.reconnectWait = time.Minute
			}
		}
	}
}

func (w *WebsocketClient) resubscribeAll() error {
	for _, subscriber := range w.subscribers {
		if err := w.sendSubscribe(subscriber.subscriptionPayload); err != nil {
			return fmt.Errorf("resubscribe: %w", err)
		}
	}
	return nil
}

func (w *WebsocketClient) sendSubscribe(payload subscriptable) error {
	return w.writeJSON(wsCommand{
		Method:       "subscribe",
		Subscription: payload,
	})
}

func (w *WebsocketClient) sendUnsubscribe(payload subscriptable) error {
	return w.writeJSON(wsCommand{
		Method:       "unsubscribe",
		Subscription: payload,
	})
}

func (w *WebsocketClient) sendPing() error {
	return w.writeJSON(wsCommand{Method: "ping"})
}

func (w *WebsocketClient) writeJSON(v any) error {
	w.writeMu.Lock()
	defer w.writeMu.Unlock()

	if w.conn == nil {
		return fmt.Errorf("connection closed")
	}

	if w.debug {
		bts, _ := json.Marshal(v)
		w.logDebugf("[>] %s", string(bts))
	}

	return w.conn.WriteJSON(v)
}

func (w *WebsocketClient) logErrf(fmt string, args ...any) {
	if w.logger == nil {
		return
	}

	w.logger.Errorf(fmt, args...)
}

func (w *WebsocketClient) logDebugf(fmt string, args ...any) {
	if w.logger == nil {
		return
	}

	w.logger.Debugf(fmt, args...)
}
