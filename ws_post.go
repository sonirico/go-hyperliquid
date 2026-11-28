package hyperliquid

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// Post sends an info or action request over the websocket and waits for its response.
func (w *WebsocketClient) Post(
	ctx context.Context,
	requestType string,
	payload any,
) (json.RawMessage, error) {
	id := w.nextPostID.Add(1)
	respCh := make(chan wsPostResult, 1)

	w.postMu.Lock()
	w.pendingPosts[id] = respCh
	w.postMu.Unlock()
	defer func() {
		w.postMu.Lock()
		delete(w.pendingPosts, id)
		w.postMu.Unlock()
	}()

	if err := w.writeJSON(wsPostCommand{
		Method:  "post",
		ID:      id,
		Request: wsPostRequest{Type: requestType, Payload: payload},
	}); err != nil {
		return nil, fmt.Errorf("websocket post: %w", err)
	}

	select {
	case result := <-respCh:
		if result.Err != nil {
			return nil, fmt.Errorf("websocket post: %w", result.Err)
		}
		if result.Response.Type == "error" {
			var text string
			if err := json.Unmarshal(result.Response.Payload, &text); err != nil {
				text = string(result.Response.Payload)
			}
			return nil, errors.New(text)
		}
		return result.Response.Payload, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
