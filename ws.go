package hyperliquid

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
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
}

func NewWebsocketClient(baseURL string) *WebsocketClient {
	if baseURL == "" {
		baseURL = MainnetAPIURL
	}
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		log.Fatalf("invalid URL: %v", err)
	}
	parsedURL.Scheme = "wss"
	parsedURL.Path = "/ws"
	wsURL := parsedURL.String()

	return &WebsocketClient{
		url:           wsURL,
		done:          make(chan struct{}),
		reconnectWait: time.Second,
		subscribers:   make(map[string]*uniqSubscriber),
		msgDispatcherRegistry: map[string]msgDispatcher{
			ChannelTrades:       NewMsgDispatcher[Trades](ChannelTrades),
			ChannelL2Book:       NewMsgDispatcher[L2Book](ChannelL2Book),
			ChannelCandle:       NewMsgDispatcher[Candles](ChannelCandle),
			ChannelAllMids:      NewMsgDispatcher[AllMids](ChannelAllMids),
			ChannelNotification: NewUserSpecificDispatcher[Notification](ChannelNotification),
			ChannelOrderUpdates: NewUserSpecificDispatcher[WsOrders](ChannelOrderUpdates),
			ChannelSubResponse:  NewNoopDispatcher(),
			//"userEvents":  NewMsgDispatcher[[]UserEvent]("userEvents"),
		},
	}
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
					log.Printf("failed to subscribe: %v", err)
				}
			},
			// on unsubscribe
			func(p subscriptable) {
				w.mu.Lock()
				defer w.mu.Unlock()
				delete(w.subscribers, pkey)
				if err := w.sendUnsubscribe(p); err != nil {
					log.Printf("failed to unsubscribe: %v", err)
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
					log.Printf("websocket read error: %v", err)
				}
				return
			}

			var wsMsg wsMessage
			if err := json.Unmarshal(msg, &wsMsg); err != nil {
				log.Printf("websocket message parse error: %v", err)
				continue
			}

			if wsMsg.Channel == ChannelPong {
				continue
			}

			if err := w.dispatch(wsMsg); err != nil {
				log.Printf("failed to dispatch websocket message: %v", err)
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
				log.Printf("ping error: %v", err)
				w.reconnect(ctx)
				return
			}
		}
	}
}

func (w *WebsocketClient) dispatch(msg wsMessage) error {
	//println("[<] " + msg.Channel)
	//println("[<] " + string(msg.Data))

	dispatcher, ok := w.msgDispatcherRegistry[msg.Channel]
	if !ok {
		return fmt.Errorf("no dispatcher for channel: %s", msg.Channel)
	}

	// Read lock is only required for the subscribers since msgDispatcherRegistry is never modified.
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

	// debug
	//bts, _ := json.Marshal(v)
	//println("[>] " + fmt.Sprintf("%s", string(bts)))

	return w.conn.WriteJSON(v)
}
