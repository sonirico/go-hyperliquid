package hyperliquid

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/signer/core/apitypes"
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

	l1Signer         L1ActionSigner
	userSignedSigner UserSignedActionSigner
	agentSigner      AgentSigner

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

func (e *Exchange) signL1Action(
	ctx context.Context,
	action any,
	vault string,
	ts int64,
	exp *int64,
	mainnet bool,
) (SignatureResult, error) {
	if e.l1Signer != nil {
		return e.l1Signer.SignL1Action(ctx, action, vault, ts, exp, mainnet)
	}
	return SignL1Action(e.privateKey, action, vault, ts, exp, mainnet)
}

func (e *Exchange) signUserSignedAction(
	ctx context.Context,
	action map[string]any,
	payloadTypes []apitypes.Type,
	primaryType string,
	mainnet bool,
) (SignatureResult, error) {
	if e.userSignedSigner != nil {
		return e.userSignedSigner.SignUserSignedAction(
			ctx,
			action,
			payloadTypes,
			primaryType,
			mainnet,
		)
	}
	return SignUserSignedAction(e.privateKey, action, payloadTypes, primaryType, mainnet)
}

func (e *Exchange) signAgent(
	ctx context.Context,
	agentAddress, agentName string,
	nonce int64,
	mainnet bool,
) (SignatureResult, error) {
	if e.agentSigner != nil {
		return e.agentSigner.SignAgent(ctx, agentAddress, agentName, nonce, mainnet)
	}
	return SignAgent(e.privateKey, agentAddress, agentName, nonce, mainnet)
}

// signAndPost signs an L1 action and posts it, returning the raw response body.
func (e *Exchange) signAndPost(ctx context.Context, action any) ([]byte, error) {
	nonce := e.nextNonce()

	sig, err := e.signL1Action(
		ctx,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	return e.postAction(ctx, action, sig, nonce)
}

// executeAction executes an action and unmarshals the response into the given result.
// The result type is responsible for reporting a rejected action; use
// executeActionChecked for result types that do not decode the status envelope.
func (e *Exchange) executeAction(ctx context.Context, action, result any) error {
	resp, err := e.signAndPost(ctx, action)
	if err != nil {
		return err
	}

	return json.Unmarshal(resp, result)
}

// executeActionChecked is executeAction for result types that ignore the
// {"status":"err","response":"..."} envelope. Without this check such a
// response unmarshals into a zero-value result and reports no error.
func (e *Exchange) executeActionChecked(ctx context.Context, action, result any) error {
	resp, err := e.signAndPost(ctx, action)
	if err != nil {
		return err
	}

	if err := exchangeActionError(resp); err != nil {
		return err
	}

	return json.Unmarshal(resp, result)
}

// ExchangeActionError is returned when the exchange rejects an action,
// as opposed to the request failing to be sent or decoded.
type ExchangeActionError struct {
	Message string
}

func (e *ExchangeActionError) Error() string {
	return e.Message
}

// exchangeActionError reports the exchange's rejection of an action, or nil if
// the response is not a rejection. A body that does not parse as the envelope
// is left for the caller's json.Unmarshal to reject.
func exchangeActionError(resp []byte) error {
	var envelope struct {
		Status   string          `json:"status"`
		Response json.RawMessage `json:"response"`
	}
	if err := json.Unmarshal(resp, &envelope); err != nil {
		return nil
	}
	if envelope.Status != "err" {
		return nil
	}
	var msg string
	if err := json.Unmarshal(envelope.Response, &msg); err == nil && msg != "" {
		return &ExchangeActionError{Message: msg}
	}
	if len(envelope.Response) > 0 {
		return &ExchangeActionError{Message: string(envelope.Response)}
	}
	return &ExchangeActionError{Message: "exchange action failed"}
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
	if e.debug { //nolint:staticcheck // Empty branch for future debugging
		// if jsonPayload, err := json.MarshalIndent(payload, "", "  "); err == nil {
		// 	println("=== OUTGOING EXCHANGE PAYLOAD ===")
		// 	println(string(jsonPayload))
		// 	println("=================================")
		// }
	}

	return e.client.post(ctx, "/exchange", payload)
}
