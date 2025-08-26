package hyperliquid

import (
	"crypto/ecdsa"
	"encoding/json"
	"time"
)

type Exchange struct {
	debug        bool
	client       *Client
	privateKey   *ecdsa.PrivateKey
	vault        string
	accountAddr  string
	info         *Info
	expiresAfter *int64
}

func NewExchange(
	privateKey *ecdsa.PrivateKey,
	baseURL string,
	meta *Meta,
	vaultAddr, accountAddr string,
	spotMeta *SpotMeta,
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

	var (
		clientOpts []ClientOpt
		infoOpts   []InfoOpt
	)
	if ex.debug {
		clientOpts = append(clientOpts, ClientOptDebugMode())
		infoOpts = append(infoOpts, InfoOptDebugMode())
	}

	ex.client = NewClient(baseURL, clientOpts...)
	ex.info = NewInfo(baseURL, true, meta, spotMeta, infoOpts...)

	return ex
}

// executeAction executes an action and unmarshals the response into the given result
func (e *Exchange) executeAction(action, result any) error {
	timestamp := time.Now().UnixMilli()

	sig, err := SignL1Action(
		e.privateKey,
		action,
		e.vault,
		timestamp,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return err
	}

	resp, err := e.postAction(action, sig, timestamp)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(resp, result); err != nil {
		return err
	}

	return nil
}

func (e *Exchange) postAction(
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

	return e.client.post("/exchange", payload)
}
