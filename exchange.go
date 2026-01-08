package hyperliquid

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"sync/atomic"
	"time"
)

type Exchange struct {
	debug        bool
	client       *client
	privateKey   *ecdsa.PrivateKey
	vault        string
	accountAddr  string
	dex          string
	info         *Info
	expiresAfter *int64
	lastNonce    atomic.Int64

	clientOpts []ClientOpt
	infoOpts   []InfoOpt
}

func NewExchange(
	ctx context.Context,
	privateKey *ecdsa.PrivateKey,
	baseURL string,
	meta *Meta,
	vaultAddr, accountAddr string,
	spotMeta *SpotMeta,
	perpDexs *MixedArray,
	opts ...ExchangeOpt,
) *Exchange {
	ex := &Exchange{
		privateKey:  privateKey,
		vault:       vaultAddr,
		accountAddr: accountAddr,
	}

	for _, opt := range opts {
		opt.Apply(ex)
	}

	if ex.debug {
		ex.clientOpts = append(ex.clientOpts, clientOptDebugMode())
		ex.infoOpts = append(ex.infoOpts, InfoOptDebugMode())
	}

	ex.client = newClient(baseURL, ex.clientOpts...)
	ex.info = NewInfo(ctx, baseURL, true, meta, spotMeta, perpDexs, ex.infoOpts...)

	return ex
}

// nextNonce returns either the current timestamp in milliseconds or incremented by one to prevent duplicates
// Nonces must be within (T - 2 days, T + 1 day), where T is the unix millisecond timestamp on the block of the transaction.
// See https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api/nonces-and-api-wallets#hyperliquid-nonces
func (e *Exchange) nextNonce() int64 {
	// it's possible that at exactly the same time a nextNonce is requested
	for {
		last := e.lastNonce.Load()
		candidate := time.Now().UnixMilli()

		if candidate <= last {
			candidate = last + 1
		}

		// Try to publish our candidate; if someone beat us, retry.
		if e.lastNonce.CompareAndSwap(last, candidate) {
			return candidate
		}
	}
}

func (e *Exchange) Info() *Info {
	return e.info
}

// PerpDex returns the configured builder perp dex name (e.g. "flx"), or empty string for default dex.
func (e *Exchange) PerpDex() string {
	return e.dex
}

// SetExpiresAfter sets the expiration time for actions
// If expiresAfter is nil, actions will not have an expiration time
// If expiresAfter is set, actions will include this expiration nonce
func (e *Exchange) SetExpiresAfter(expiresAfter *int64) {
	e.expiresAfter = expiresAfter
}

// SetLastNonce allows for resuming from a persisted nonce, e.g. the nonce was stored before a restart
// Only useful if a lot of increments happen for unique nonces. Most users do not need this.
func (e *Exchange) SetLastNonce(n int64) {
	e.lastNonce.Store(n)
}

// executeAction executes an action and unmarshals the response into the given result
func (e *Exchange) executeAction(ctx context.Context, action, result any) error {
	nonce := e.nextNonce()

	sig, err := SignL1Action(
		e.privateKey,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return err
	}

	resp, err := e.postAction(ctx, action, sig, nonce)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(resp, result); err != nil {
		return err
	}

	return nil
}

func (e *Exchange) postAction(
	ctx context.Context,
	action any,
	signature SignatureResult,
	nonce int64,
) ([]byte, error) {
	payload := map[string]any{
		"action":    action,
		"nonce":     nonce,
		"signature": signature,
	}

	if e.vault != "" {
		// Handle vault address based on action type
		if actionMap, ok := action.(map[string]any); ok {
			if actionMap["type"] != "usdClassTransfer" {
				payload["vaultAddress"] = e.vault
			} else {
				payload["vaultAddress"] = nil
			}
		} else {
			// For struct types, we need to use reflection or type assertion
			// For now, assume it's not usdClassTransfer
			payload["vaultAddress"] = e.vault
		}
	}

	// Add expiration time if set
	if e.expiresAfter != nil {
		payload["expiresAfter"] = *e.expiresAfter
	}

	// Debug logging
	if e.debug {
		//if jsonPayload, err := json.MarshalIndent(payload, "", "  "); err == nil {
		//	println("=== OUTGOING EXCHANGE PAYLOAD ===")
		//	println(string(jsonPayload))
		//	println("=================================")
		//}
	}

	return e.client.post(ctx, "/exchange", payload)
}
