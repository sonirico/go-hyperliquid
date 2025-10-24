// Package hyperliquid provides a Go client library for the Hyperliquid exchange API.
// It includes support for both REST API and WebSocket connections, allowing users to
// access market data, manage orders, and handle user account operations.
package hyperliquid

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/sonirico/vago/lol"
)

const (
	MainnetAPIURL = "https://api.hyperliquid.xyz"
	TestnetAPIURL = "https://api.hyperliquid-testnet.xyz"
	LocalAPIURL   = "http://localhost:3001"

	// httpErrorStatusCode is the minimum status code considered an error
	httpErrorStatusCode = 400
)

type client struct {
	logger     lol.Logger
	debug      bool
	baseURL    string
	httpClient *http.Client
}

func newClient(baseURL string, opts ...ClientOpt) *client {
	if baseURL == "" {
		baseURL = MainnetAPIURL
	}

	cli := &client{
		baseURL:    baseURL,
		httpClient: new(http.Client),
	}

	for _, opt := range opts {
		opt.Apply(cli)
	}

	return cli
}

func (c *client) post(ctx context.Context, path string, payload any) ([]byte, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := c.baseURL + path
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		url,
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	if c.debug {
		c.logger.WithFields(lol.Fields{
			"method": "POST",
			"url":    url,
			"body":   string(jsonData),
		}).Debug("HTTP request")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body := make([]byte, 0)
	if resp.Body != nil {
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}
	}

	if c.debug {
		c.logger.WithFields(lol.Fields{
			"status": resp.Status,
			"body":   string(body),
		}).Debug("HTTP response")
	}

	if resp.StatusCode >= httpErrorStatusCode {
		if !json.Valid(body) {
			return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
		}
		var apiErr APIError
		if err := json.Unmarshal(body, &apiErr); err != nil {
			return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
		}
		return nil, apiErr
	}

	return body, nil
}
