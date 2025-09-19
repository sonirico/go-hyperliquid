package hyperliquid

import (
	"context"
	"encoding/json"
	"fmt"
)

const (
	// spotAssetIndexOffset is the offset added to spot asset indices
	spotAssetIndexOffset = 10000
)

type Info struct {
	debug          bool
	client         *Client
	coinToAsset    map[string]int
	nameToCoin     map[string]string
	assetToDecimal map[int]int
	clientOpts     []ClientOpt
}

func NewInfo(ctx context.Context, baseURL string, skipWS bool, meta *Meta, spotMeta *SpotMeta, opts ...InfoOpt) *Info {
	info := &Info{
		coinToAsset:    make(map[string]int),
		nameToCoin:     make(map[string]string),
		assetToDecimal: make(map[int]int),
	}

	for _, opt := range opts {
		opt.Apply(info)
	}

	if info.debug {
		info.clientOpts = append(info.clientOpts, ClientOptDebugMode())
	}

	info.client = NewClient(baseURL, info.clientOpts...)

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
	for asset, assetInfo := range meta.Universe {
		info.coinToAsset[assetInfo.Name] = asset
		info.nameToCoin[assetInfo.Name] = assetInfo.Name
		info.assetToDecimal[asset] = assetInfo.SzDecimals
	}

	// Map spot assets starting at 10000
	for _, spotInfo := range spotMeta.Universe {
		asset := spotInfo.Index + spotAssetIndexOffset
		info.coinToAsset[spotInfo.Name] = asset
		info.nameToCoin[spotInfo.Name] = spotInfo.Name
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

func (i *Info) Meta(ctx context.Context) (*Meta, error) {
	resp, err := i.client.post(ctx, "/info", map[string]any{
		"type": "meta",
	})
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

func (i *Info) NameToAsset(name string) int {
	coin := i.nameToCoin[name]
	return i.coinToAsset[coin]
}

func (i *Info) UserState(ctx context.Context, address string) (*UserState, error) {
	resp, err := i.client.post(ctx, "/info", map[string]any{
		"type": "clearinghouseState",
		"user": address,
	})
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

func (i *Info) OpenOrders(ctx context.Context, address string) ([]OpenOrder, error) {
	resp, err := i.client.post(ctx, "/info", map[string]any{
		"type": "openOrders",
		"user": address,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch open orders: %w", err)
	}

	var result []OpenOrder
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal open orders: %w", err)
	}
	return result, nil
}

func (i *Info) FrontendOpenOrders(ctx context.Context, address string) ([]OpenOrder, error) {
	resp, err := i.client.post(ctx, "/info", map[string]any{
		"type": "frontendOpenOrders",
		"user": address,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch frontend open orders: %w", err)
	}

	var result []OpenOrder
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal frontend open orders: %w", err)
	}
	return result, nil
}

func (i *Info) AllMids(ctx context.Context) (map[string]string, error) {
	resp, err := i.client.post(ctx, "/info", map[string]any{
		"type": "allMids",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch all mids: %w", err)
	}

	var result map[string]string
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal all mids: %w", err)
	}
	return result, nil
}

func (i *Info) UserFills(ctx context.Context, address string) ([]Fill, error) {
	resp, err := i.client.post(ctx, "/info", map[string]any{
		"type": "userFills",
		"user": address,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user fills: %w", err)
	}

	var result []Fill
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user fills: %w", err)
	}
	return result, nil
}

func (i *Info) UserFillsByTime(
	ctx context.Context,
	address string,
	startTime int64,
	endTime *int64,
) ([]Fill, error) {
	resp, err := i.postTimeRangeRequest(ctx, "userFillsByTime", address, startTime, endTime, nil)
	if err != nil {
		return nil, err
	}

	var result []Fill
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user fills by time: %w", err)
	}
	return result, nil
}

func (i *Info) MetaAndAssetCtxs(ctx context.Context) (*MetaAndAssetCtxs, error) {
	resp, err := i.client.post(ctx, "/info", map[string]any{
		"type": "metaAndAssetCtxs",
	})
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
	coin := i.nameToCoin[name]
	resp, err := i.postTimeRangeRequest(
		ctx,
		"fundingHistory",
		"",
		startTime,
		endTime,
		map[string]any{"coin": coin},
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

func (i *Info) L2Snapshot(ctx context.Context, name string) (*L2Book, error) {
	resp, err := i.client.post(ctx, "/info", map[string]any{
		"type": "l2Book",
		"coin": i.nameToCoin[name],
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
		"coin":      i.nameToCoin[name],
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
func (i *Info) PerpDexs(ctx context.Context) ([]string, error) {
	resp, err := i.client.post(ctx, "/info", map[string]any{
		"type": "perpDexs",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch perp dexs: %w", err)
	}

	var result []string
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal perp dexs: %w", err)
	}
	return result, nil
}
