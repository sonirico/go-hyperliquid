package hyperliquid

import (
	"context"
	"encoding/json"
	"fmt"
)

const (
	// spotAssetIndexOffset is the offset added to spot asset indices
	spotAssetIndexOffset = 10000
	// builderPerpAssetBase is the base offset for builder-deployed perp asset ids.
	// See Asset IDs docs: asset = 100000 + perpDexIndex*10000 + indexInMeta.
	builderPerpAssetBase = 100000
)

type Info struct {
	debug          bool
	client         *client
	coinToAsset    map[string]int
	assetToDecimal map[int]int
	perpDexName    string
	clientOpts     []ClientOpt
}

func NewInfo(
	ctx context.Context,
	baseURL string,
	skipWS bool,
	meta *Meta,
	spotMeta *SpotMeta,
	perpDexs *MixedArray,
	opts ...InfoOpt,
) *Info {
	info := &Info{
		coinToAsset:    make(map[string]int),
		assetToDecimal: make(map[int]int),
	}

	for _, opt := range opts {
		opt.Apply(info)
	}

	if info.debug {
		info.clientOpts = append(info.clientOpts, clientOptDebugMode())
	}

	info.client = newClient(baseURL, info.clientOpts...)

	if meta == nil {
		var err error
		meta, err = info.Meta(ctx)
		if err != nil {
			panic(err)
		}
	}

	if spotMeta == nil {
		var err error
		spotMeta, err = info.SpotMeta(ctx)
		if err != nil {
			panic(err)
		}
	}

	// Map perp assets
	if info.perpDexName != "" {
		// Builder-deployed perp: compute full asset id as documented.
		if perpDexs == nil {
			var err error
			perpDexsNew, err := info.PerpDexs(ctx)
			perpDexs = &perpDexsNew
			if err != nil {
				panic(err)
			}
		}
		perpDexIndex := -1
		for i, mv := range *perpDexs {
			if mv.Type() != "object" {
				continue
			}
			var pd PerpDex
			if err := mv.Parse(&pd); err == nil && pd.Name == info.perpDexName {
				perpDexIndex = i
				break
			}
		}
		if perpDexIndex < 0 {
			panic(
				fmt.Errorf("unknown perp dex %q (not present in /info perpDexs)", info.perpDexName),
			)
		}
		base := builderPerpAssetBase + perpDexIndex*10000
		for idxInMeta, assetInfo := range meta.Universe {
			assetID := base + idxInMeta
			info.coinToAsset[assetInfo.Name] = assetID
			info.assetToDecimal[assetID] = assetInfo.SzDecimals
		}
	} else {
		// Default perp dex: asset id is just index in meta universe.
		for asset, assetInfo := range meta.Universe {
			info.coinToAsset[assetInfo.Name] = asset
			info.assetToDecimal[asset] = assetInfo.SzDecimals
		}
	}

	// Map spot assets starting at 10000
	for _, spotInfo := range spotMeta.Universe {
		asset := spotInfo.Index + spotAssetIndexOffset
		info.coinToAsset[spotInfo.Name] = asset
		info.assetToDecimal[asset] = spotMeta.Tokens[spotInfo.Tokens[0]].SzDecimals
	}

	return info
}

// postTimeRangeRequest makes a POST request with time range parameters
func (i *Info) postTimeRangeRequest(
	ctx context.Context,
	requestType, user string,
	startTime int64,
	endTime *int64,
	extraParams map[string]any,
) ([]byte, error) {
	payload := map[string]any{
		"type":      requestType,
		"startTime": startTime,
	}
	if user != "" {
		payload["user"] = user
	}
	if endTime != nil {
		payload["endTime"] = *endTime
	}
	for k, v := range extraParams {
		payload[k] = v
	}

	resp, err := i.client.post(ctx, "/info", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s: %w", requestType, err)
	}
	return resp, nil
}

func parseMetaResponse(resp []byte) (*Meta, error) {
	var meta map[string]json.RawMessage
	if err := json.Unmarshal(resp, &meta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal meta response: %w", err)
	}

	var universe []AssetInfo
	if err := json.Unmarshal(meta["universe"], &universe); err != nil {
		return nil, fmt.Errorf("failed to unmarshal universe: %w", err)
	}

	var marginTables [][]any
	if err := json.Unmarshal(meta["marginTables"], &marginTables); err != nil {
		return nil, fmt.Errorf("failed to unmarshal margin tables: %w", err)
	}

	marginTablesResult := make([]MarginTable, len(marginTables))
	for i, marginTable := range marginTables {
		id := marginTable[0].(float64)
		tableBytes, err := json.Marshal(marginTable[1])
		if err != nil {
			return nil, fmt.Errorf("failed to marshal margin table data: %w", err)
		}

		var marginTableData map[string]any
		if err := json.Unmarshal(tableBytes, &marginTableData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal margin table data: %w", err)
		}

		marginTiersBytes, err := json.Marshal(marginTableData["marginTiers"])
		if err != nil {
			return nil, fmt.Errorf("failed to marshal margin tiers: %w", err)
		}

		var marginTiers []MarginTier
		if err := json.Unmarshal(marginTiersBytes, &marginTiers); err != nil {
			return nil, fmt.Errorf("failed to unmarshal margin tiers: %w", err)
		}

		marginTablesResult[i] = MarginTable{
			ID:          int(id),
			Description: marginTableData["description"].(string),
			MarginTiers: marginTiers,
		}
	}

	return &Meta{
		Universe:     universe,
		MarginTables: marginTablesResult,
	}, nil
}

// Meta retrieves perpetuals metadata
// If dex is empty string, returns metadata for the first perp dex (default)
func (i *Info) Meta(ctx context.Context, dex ...string) (*Meta, error) {
	payload := map[string]any{
		"type": "meta",
	}
	if len(dex) > 0 && dex[0] != "" {
		payload["dex"] = dex[0]
	}

	resp, err := i.client.post(ctx, "/info", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch meta: %w", err)
	}

	return parseMetaResponse(resp)
}

func (i *Info) SpotMeta(ctx context.Context) (*SpotMeta, error) {
	resp, err := i.client.post(ctx, "/info", map[string]any{
		"type": "spotMeta",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch spot meta: %w", err)
	}

	var spotMeta SpotMeta
	if err := json.Unmarshal(resp, &spotMeta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal spot meta response: %w", err)
	}

	return &spotMeta, nil
}

func (i *Info) CoinToAsset(coin string) (int, bool) {
	result, ok := i.coinToAsset[coin]
	return result, ok
}

// UserState retrieves user's perpetuals account summary
// If dex is empty string, returns state for the first perp dex (default)
func (i *Info) UserState(ctx context.Context, address string, dex ...string) (*UserState, error) {
	payload := map[string]any{
		"type": "clearinghouseState",
		"user": address,
	}
	if len(dex) > 0 && dex[0] != "" {
		payload["dex"] = dex[0]
	}

	resp, err := i.client.post(ctx, "/info", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user state: %w", err)
	}

	var result UserState
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user state: %w", err)
	}
	return &result, nil
}

func (i *Info) SpotUserState(ctx context.Context, address string) (*SpotUserState, error) {
	resp, err := i.client.post(ctx, "/info", map[string]any{
		"type": "spotClearinghouseState",
		"user": address,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch spot user state: %w", err)
	}

	var result SpotUserState
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal spot user state: %w", err)
	}
	return &result, nil
}

// OpenOrders retrieves user's open orders
// If dex is empty string, returns orders for the first perp dex (default)
// Note: Spot open orders are only included with the first perp dex
func (i *Info) OpenOrders(ctx context.Context, address string, dex ...string) ([]OpenOrder, error) {
	payload := map[string]any{
		"type": "openOrders",
		"user": address,
	}
	if len(dex) > 0 && dex[0] != "" {
		payload["dex"] = dex[0]
	}

	resp, err := i.client.post(ctx, "/info", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch open orders: %w", err)
	}

	var result []OpenOrder
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal open orders: %w", err)
	}
	return result, nil
}

// FrontendOpenOrders retrieves user's open orders with frontend info
// If dex is empty string, returns orders for the first perp dex (default)
// Note: Spot open orders are only included with the first perp dex
func (i *Info) FrontendOpenOrders(
	ctx context.Context,
	address string,
	dex ...string,
) ([]FrontendOpenOrder, error) {
	payload := map[string]any{
		"type": "frontendOpenOrders",
		"user": address,
	}
	if len(dex) > 0 && dex[0] != "" {
		payload["dex"] = dex[0]
	}

	resp, err := i.client.post(ctx, "/info", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch frontend open orders: %w", err)
	}

	var result []FrontendOpenOrder
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal frontend open orders: %w", err)
	}
	return result, nil
}

// AllMids retrieves mids for all coins
// If dex is empty string, returns mids for the first perp dex (default)
// Note: Spot mids are only included with the first perp dex
func (i *Info) AllMids(ctx context.Context, dex ...string) (map[string]string, error) {
	payload := map[string]any{
		"type": "allMids",
	}
	if len(dex) > 0 && dex[0] != "" {
		payload["dex"] = dex[0]
	}

	resp, err := i.client.post(ctx, "/info", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch all mids: %w", err)
	}

	var result map[string]string
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal all mids: %w", err)
	}
	return result, nil
}

func (i *Info) UserFills(ctx context.Context, params UserFillsParams) ([]Fill, error) {
	payload := map[string]any{
		"type": "userFills",
		"user": params.Address,
	}
	if params.AggregateByTime != nil {
		payload["aggregateByTime"] = *params.AggregateByTime
	}

	resp, err := i.client.post(ctx, "/info", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user fills: %w", err)
	}

	var result []Fill
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user fills: %w", err)
	}
	return result, nil
}

func (i *Info) HistoricalOrders(ctx context.Context, address string) ([]OrderQueryResponse, error) {
	resp, err := i.client.post(ctx, "/info", map[string]any{
		"type": "historicalOrders",
		"user": address,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch historical orders: %w", err)
	}

	var result []OrderQueryResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal historical orders: %w", err)
	}
	return result, nil
}

func (i *Info) UserFillsByTime(
	ctx context.Context,
	address string,
	startTime int64,
	endTime *int64,
	aggregateByTime *bool,
) ([]Fill, error) {
	var extraParams = make(map[string]any, 0)
	if aggregateByTime != nil {
		extraParams["aggregateByTime"] = aggregateByTime
	}

	resp, err := i.postTimeRangeRequest(
		ctx,
		"userFillsByTime",
		address,
		startTime,
		endTime,
		extraParams,
	)
	if err != nil {
		return nil, err
	}

	var result []Fill
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user fills by time: %w", err)
	}
	return result, nil
}

// MetaAndAssetCtxs retrieves perpetuals metadata and asset contexts
// If params.Dex is nil or empty string, returns data for the first perp dex (default)
func (i *Info) MetaAndAssetCtxs(ctx context.Context, params MetaAndAssetCtxsParams) (*MetaAndAssetCtxs, error) {
	// Internal payload struct with fixed Type field
	payload := struct {
		Type string  `json:"type"`
		Dex  *string `json:"dex,omitempty"`
	}{
		Type: "metaAndAssetCtxs",
		Dex:  params.Dex,
	}

	resp, err := i.client.post(ctx, "/info", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch meta and asset contexts: %w", err)
	}

	var result []any
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal meta and asset contexts: %w", err)
	}

	if len(result) < 2 {
		return nil, fmt.Errorf("expected at least 2 elements in response, got %d", len(result))
	}

	metaBytes, err := json.Marshal(result[0])
	if err != nil {
		return nil, fmt.Errorf("failed to marshal meta data: %w", err)
	}

	meta, err := parseMetaResponse(metaBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse meta: %w", err)
	}

	ctxsBytes, err := json.Marshal(result[1])
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ctxs data: %w", err)
	}

	var ctxs []AssetCtx
	if err := json.Unmarshal(ctxsBytes, &ctxs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ctxs: %w", err)
	}

	metaAndAssetCtxs := &MetaAndAssetCtxs{
		Meta: *meta,
		Ctxs: ctxs,
	}

	return metaAndAssetCtxs, nil
}

func (i *Info) SpotMetaAndAssetCtxs(ctx context.Context) (*SpotMetaAndAssetCtxs, error) {
	resp, err := i.client.post(ctx, "/info", map[string]any{
		"type": "spotMetaAndAssetCtxs",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch spot meta and asset contexts: %w", err)
	}

	var result []any
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal spot meta and asset contexts: %w", err)
	}

	if len(result) < 2 {
		return nil, fmt.Errorf("expected at least 2 elements in response, got %d", len(result))
	}

	// Unmarshal the first element (SpotMeta)
	metaBytes, err := json.Marshal(result[0])
	if err != nil {
		return nil, fmt.Errorf("failed to marshal meta data: %w", err)
	}

	var meta SpotMeta
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal meta: %w", err)
	}

	// Unmarshal the second element ([]SpotAssetCtx)
	ctxsBytes, err := json.Marshal(result[1])
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ctxs data: %w", err)
	}

	var ctxs []SpotAssetCtx
	if err := json.Unmarshal(ctxsBytes, &ctxs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ctxs: %w", err)
	}

	return &SpotMetaAndAssetCtxs{
		Meta: meta,
		Ctxs: ctxs,
	}, nil
}

func (i *Info) FundingHistory(
	ctx context.Context,
	name string,
	startTime int64,
	endTime *int64,
) ([]FundingHistory, error) {
	resp, err := i.postTimeRangeRequest(
		ctx,
		"fundingHistory",
		"",
		startTime,
		endTime,
		map[string]any{"coin": name},
	)
	if err != nil {
		return nil, err
	}

	var result []FundingHistory
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal funding history: %w", err)
	}
	return result, nil
}

func (i *Info) UserFundingHistory(
	ctx context.Context,
	user string,
	startTime int64,
	endTime *int64,
) ([]UserFundingHistory, error) {
	resp, err := i.postTimeRangeRequest(ctx, "userFunding", user, startTime, endTime, nil)
	if err != nil {
		return nil, err
	}

	var result []UserFundingHistory
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user funding history: %w", err)
	}
	return result, nil
}

func (i *Info) UserNonFundingLedgerUpdates(
	ctx context.Context,
	user string,
	startTime int64,
	endTime *int64,
) ([]UserNonFundingLedgerUpdates, error) {
	resp, err := i.postTimeRangeRequest(
		ctx,
		"userNonFundingLedgerUpdates",
		user,
		startTime,
		endTime,
		nil,
	)
	if err != nil {
		return nil, err
	}

	var result []UserNonFundingLedgerUpdates
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user non-funding ledger updates: %w", err)
	}
	return result, nil
}

func (i *Info) L2Snapshot(ctx context.Context, name string) (*L2Book, error) {
	resp, err := i.client.post(ctx, "/info", map[string]any{
		"type": "l2Book",
		"coin": name,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch L2 snapshot: %w", err)
	}

	var result L2Book
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal L2 snapshot: %w", err)
	}
	return &result, nil
}

func (i *Info) CandlesSnapshot(
	ctx context.Context,
	name, interval string,
	startTime, endTime int64,
) ([]Candle, error) {
	req := map[string]any{
		"coin":      name,
		"interval":  interval,
		"startTime": startTime,
		"endTime":   endTime,
	}

	resp, err := i.client.post(ctx, "/info", map[string]any{
		"type": "candleSnapshot",
		"req":  req,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch candles snapshot: %w", err)
	}

	var result []Candle
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal candles snapshot: %w", err)
	}
	return result, nil
}

func (i *Info) UserFees(ctx context.Context, address string) (*UserFees, error) {
	resp, err := i.client.post(ctx, "/info", map[string]any{
		"type": "userFees",
		"user": address,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user fees: %w", err)
	}

	var result UserFees
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user fees: %w", err)
	}
	return &result, nil
}

func (i *Info) UserActiveAssetData(
	ctx context.Context,
	address string,
	coin string,
) (*UserActiveAssetData, error) {
	resp, err := i.client.post(ctx, "/info", map[string]any{
		"type": "activeAssetData",
		"user": address,
		"coin": coin,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user active asset data: %w", err)
	}

	var result UserActiveAssetData
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user active asset data: %w", err)
	}
	return &result, nil
}

func (i *Info) UserStakingSummary(ctx context.Context, address string) (*StakingSummary, error) {
	resp, err := i.client.post(ctx, "/info", map[string]any{
		"type": "delegatorSummary",
		"user": address,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch staking summary: %w", err)
	}

	var result StakingSummary
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal staking summary: %w", err)
	}
	return &result, nil
}

func (i *Info) UserStakingDelegations(
	ctx context.Context,
	address string,
) ([]StakingDelegation, error) {
	resp, err := i.client.post(ctx, "/info", map[string]any{
		"type": "delegations",
		"user": address,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch staking delegations: %w", err)
	}

	var result []StakingDelegation
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal staking delegations: %w", err)
	}
	return result, nil
}

func (i *Info) UserStakingRewards(ctx context.Context, address string) ([]StakingReward, error) {
	resp, err := i.client.post(ctx, "/info", map[string]any{
		"type": "delegatorRewards",
		"user": address,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch staking rewards: %w", err)
	}

	var result []StakingReward
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal staking rewards: %w", err)
	}
	return result, nil
}

func (i *Info) QueryOrderByOid(
	ctx context.Context,
	user string,
	oid int64,
) (*OrderQueryResult, error) {
	resp, err := i.client.post(ctx, "/info", map[string]any{
		"type": "orderStatus",
		"user": user,
		"oid":  oid,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch order status: %w", err)
	}

	var result OrderQueryResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal order status: %w", err)
	}
	return &result, nil
}

func (i *Info) QueryOrderByCloid(
	ctx context.Context,
	user, cloid string,
) (*OrderQueryResult, error) {
	resp, err := i.client.post(ctx, "/info", map[string]any{
		"type": "orderStatus",
		"user": user,
		"oid":  cloid,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch order status by cloid: %w", err)
	}

	var result OrderQueryResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal order status: %w", err)
	}
	return &result, nil
}

func (i *Info) QueryReferralState(ctx context.Context, user string) (*ReferralState, error) {
	resp, err := i.client.post(ctx, "/info", map[string]any{
		"type": "referral",
		"user": user,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch referral state: %w", err)
	}

	var result ReferralState
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal referral state: %w", err)
	}
	return &result, nil
}

func (i *Info) QuerySubAccounts(ctx context.Context, user string) ([]SubAccount, error) {
	resp, err := i.client.post(ctx, "/info", map[string]any{
		"type": "subAccounts",
		"user": user,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sub accounts: %w", err)
	}

	var result []SubAccount
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal sub accounts: %w", err)
	}
	return result, nil
}

func (i *Info) QueryUserToMultiSigSigners(
	ctx context.Context,
	multiSigUser string,
) ([]MultiSigSigner, error) {
	resp, err := i.client.post(ctx, "/info", map[string]any{
		"type": "userToMultiSigSigners",
		"user": multiSigUser,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch multi-sig signers: %w", err)
	}

	var result []MultiSigSigner
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal multi-sig signers: %w", err)
	}
	return result, nil
}

// PerpDexs returns the list of available perpetual dexes
// Returns an array where each element can be nil (for the default dex) or a PerpDex object
// The first element is always null (representing the default dex)
func (i *Info) PerpDexs(ctx context.Context) (MixedArray, error) {
	resp, err := i.client.post(ctx, "/info", map[string]any{
		"type": "perpDexs",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch perp dexs: %w", err)
	}

	var result MixedArray
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal perp dexs: %w", err)
	}
	return result, nil
}

func (i *Info) TokenDetails(ctx context.Context, tokenId string) (*TokenDetail, error) {
	resp, err := i.client.post(ctx, "/info", map[string]any{
		"type":    "tokenDetails",
		"tokenId": tokenId,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch token detail: %w", err)
	}

	var tokenDetail TokenDetail
	if err := json.Unmarshal(resp, &tokenDetail); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token detail response: %w", err)
	}

	return &tokenDetail, nil
}

// PerpDexLimits retrieves builder-deployed perp market limits
// dex must be a non-empty string (the empty string is not allowed for this endpoint)
func (i *Info) PerpDexLimits(ctx context.Context, dex string) (*PerpDexLimits, error) {
	if dex == "" {
		return nil, fmt.Errorf("dex parameter is required for perpDexLimits")
	}

	resp, err := i.client.post(ctx, "/info", map[string]any{
		"type": "perpDexLimits",
		"dex":  dex,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch perp dex limits: %w", err)
	}

	var result PerpDexLimits
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal perp dex limits: %w", err)
	}
	return &result, nil
}

// PerpDexStatus retrieves perp market status
// If dex is empty string, returns status for the first perp dex (default)
func (i *Info) PerpDexStatus(ctx context.Context, dex string) (*PerpDexStatus, error) {
	payload := map[string]any{
		"type": "perpDexStatus",
	}
	if dex != "" {
		payload["dex"] = dex
	}

	resp, err := i.client.post(ctx, "/info", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch perp dex status: %w", err)
	}

	var result PerpDexStatus
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal perp dex status: %w", err)
	}
	return &result, nil
}

// PerpDeployAuctionStatus retrieves information about the Perp Deploy Auction
func (i *Info) PerpDeployAuctionStatus(ctx context.Context) (*PerpDeployAuctionStatus, error) {
	resp, err := i.client.post(ctx, "/info", map[string]any{
		"type": "perpDeployAuctionStatus",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch perp deploy auction status: %w", err)
	}

	var result PerpDeployAuctionStatus
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal perp deploy auction status: %w", err)
	}
	return &result, nil
}
