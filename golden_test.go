package hyperliquid

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/stretchr/testify/require"
)

const (
	goldenDir      = "testdata/golden"
	goldenKey      = "0123456789012345678901234567890123456789012345678901234567890123"
	goldenAgentKey = "0000000000000000000000000000000000000000000000000000000000000002"
	goldenNonce    = int64(1700000000000)
	goldenOther    = "0x1111111111111111111111111111111111111111"
	goldenMixed    = "0xAbCdEf1111111111111111111111111111111111"
	goldenToken    = "USDC:0x6d1e7cde53ba9467b783cb7c530ce054"
	goldenCloid    = "0x00000000000000000000000000000001"
)

type goldenType struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type goldenFile struct {
	Kind         string          `json:"kind"`
	Action       json.RawMessage `json:"action"`
	Nonce        int64           `json:"nonce"`
	VaultAddress *string         `json:"vaultAddress"`
	ExpiresAfter *int64          `json:"expiresAfter"`
	IsMainnet    bool            `json:"isMainnet"`
	PrimaryType  string          `json:"primaryType"`
	Types        []goldenType    `json:"types"`
	Signature    SignatureResult `json:"signature"`
}

type goldenSigned struct {
	action    any
	signature SignatureResult
}

// goldenRow holds the exchange method call that sends the action (call). When
// the method signs with the raw private key and bypasses the injected signers,
// or generates its own key, the golden test builds the wire action the way the
// method does instead (direct).
type goldenRow struct {
	name   string
	skip   string
	call   func(ctx context.Context, ex *Exchange) error
	direct func(pk *ecdsa.PrivateKey) (goldenSigned, error)
}

// goldenSigner signs every action with goldenNonce so that the signature is
// comparable with the vector, and records the user-signed typed data.
type goldenSigner struct {
	pk          *ecdsa.PrivateKey
	calls       int
	primaryType string
	types       []apitypes.Type
}

func newGoldenSigner(pk *ecdsa.PrivateKey) *goldenSigner {
	return &goldenSigner{pk: pk}
}

func (s *goldenSigner) SignL1Action(
	_ context.Context,
	action any,
	vaultAddress string,
	_ int64,
	expiresAfter *int64,
	isMainnet bool,
) (SignatureResult, error) {
	s.calls++
	return SignL1Action(s.pk, action, vaultAddress, goldenNonce, expiresAfter, isMainnet)
}

func (s *goldenSigner) SignUserSignedAction(
	_ context.Context,
	action map[string]any,
	payloadTypes []apitypes.Type,
	primaryType string,
	isMainnet bool,
) (SignatureResult, error) {
	s.calls++
	s.primaryType = primaryType
	s.types = payloadTypes
	for _, key := range []string{"nonce", "time"} {
		if _, ok := action[key]; ok {
			action[key] = big.NewInt(goldenNonce)
		}
	}
	return SignUserSignedAction(s.pk, action, payloadTypes, primaryType, isMainnet)
}

type goldenFixture struct {
	exchange *Exchange
	signer   *goldenSigner
	body     map[string]json.RawMessage
}

func newTestGoldenFixture(t *testing.T, pk *ecdsa.PrivateKey) *goldenFixture {
	t.Helper()
	fixture := &goldenFixture{signer: newGoldenSigner(pk)}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(raw, &fixture.body))
		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write([]byte(`{"status":"ok","response":{"type":"default"}}`))
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	meta := &Meta{Universe: []AssetInfo{{Name: "BTC", SzDecimals: 5}, {Name: "ETH", SzDecimals: 4}}}
	fixture.exchange = NewExchange(
		context.Background(), pk, server.URL, meta, "", "", &SpotMeta{}, nil,
		ExchangeOptL1Signer(fixture.signer),
		ExchangeOptUserSignedSigner(fixture.signer),
	)
	return fixture
}

func newTestGoldenKey(t *testing.T, hexKey string) *ecdsa.PrivateKey {
	t.Helper()
	pk, err := crypto.HexToECDSA(hexKey)
	require.NoError(t, err)
	return pk
}

func loadTestGoldenFiles(t *testing.T) map[string]goldenFile {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(goldenDir, "*.json"))
	require.NoError(t, err)
	files := make(map[string]goldenFile, len(paths))
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		require.NoError(t, err)
		var file goldenFile
		require.NoError(t, json.Unmarshal(raw, &file))
		files[strings.TrimSuffix(filepath.Base(path), ".json")] = file
	}
	return files
}

func signGoldenAgent(pk *ecdsa.PrivateKey, name *string) (goldenSigned, error) {
	agent, err := crypto.HexToECDSA(goldenAgentKey)
	if err != nil {
		return goldenSigned{}, err
	}
	agentAddress := crypto.PubkeyToAddress(agent.PublicKey).Hex()
	agentName := ""
	if name != nil {
		agentName = *name
	}
	sig, err := SignAgent(pk, agentAddress, agentName, goldenNonce, false)
	action := ApproveAgentAction{
		Type:             "approveAgent",
		SignatureChainId: "0x66eee",
		HyperliquidChain: "Testnet",
		AgentAddress:     agentAddress,
		AgentName:        name,
		Nonce:            goldenNonce,
	}
	return goldenSigned{action: action, signature: sig}, err
}

func goldenLimitOrder() CreateOrderRequest {
	return CreateOrderRequest{
		Coin:      "BTC",
		IsBuy:     true,
		Price:     50000,
		Size:      0.001,
		OrderType: OrderType{Limit: &LimitOrderType{Tif: TifGtc}},
	}
}

func goldenRows(userAddress string) []goldenRow {
	cloid := goldenCloid
	oid := int64(123)
	scheduleTime := int64(4102444800000)
	maxGas := 1000000
	nSeededLevels := 3
	bot := "bot"
	assetRequest2 := AssetRequest2{
		Coin: "test:X", SzDecimals: 2, OraclePx: "10.0", MarginTableID: 10, MarginMode: "strictIsolated",
	}
	cloidOrder := goldenLimitOrder()
	cloidOrder.ClientOrderID = &cloid
	triggerOrder := goldenLimitOrder()
	triggerOrder.IsBuy = false
	triggerOrder.ReduceOnly = true
	triggerOrder.OrderType = OrderType{
		Trigger: &TriggerOrderType{TriggerPx: 49000, IsMarket: true, Tpsl: StopLoss},
	}
	modify := ModifyOrderRequest{Oid: &oid, Order: goldenLimitOrder()}

	return []goldenRow{
		{name: "order_limit", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.Order(ctx, goldenLimitOrder(), nil)
			return err
		}},
		{name: "order_cloid_builder", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.Order(ctx, cloidOrder, &BuilderInfo{Builder: goldenOther, Fee: 10})
			return err
		}},
		{name: "order_trigger", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.Order(ctx, triggerOrder, nil)
			return err
		}},
		{name: "modify", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.ModifyOrder(ctx, modify)
			return err
		}},
		{name: "batch_modify", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.BulkModifyOrders(ctx, []ModifyOrderRequest{modify})
			return err
		}},
		{name: "cancel", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.Cancel(ctx, "BTC", oid)
			return err
		}},
		{name: "cancel_by_cloid", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.CancelByCloid(ctx, "BTC", goldenCloid)
			return err
		}},
		{name: "schedule_cancel", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.ScheduleCancel(ctx, &scheduleTime)
			return err
		}},
		{name: "update_leverage", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.UpdateLeverage(ctx, 5, "BTC", true)
			return err
		}},
		{name: "set_referrer", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.SetReferrer(ctx, "ABC")
			return err
		}},
		{name: "create_sub_account", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.CreateSubAccount(ctx, "sub1")
			return err
		}},
		{name: "usd_class_transfer", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.UsdClassTransfer(ctx, 1.5, true)
			return err
		}},
		{name: "sub_account_transfer", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.SubAccountTransfer(ctx, goldenOther, true, 1000000)
			return err
		}},
		{name: "sub_account_spot_transfer", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.SubAccountSpotTransfer(ctx, goldenOther, true, goldenToken, 1.5)
			return err
		}},
		{name: "vault_transfer", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.VaultUsdTransfer(ctx, goldenOther, true, 1000000)
			return err
		}},
		{name: "vault_modify", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.VaultModify(ctx, goldenOther, true, false)
			return err
		}},
		{name: "vault_distribute", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.VaultDistribute(ctx, goldenOther, 1000000)
			return err
		}},
		{name: "usd_send", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.UsdTransfer(ctx, 1.5, goldenOther)
			return err
		}},
		{name: "spot_send", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.SpotTransfer(ctx, 1.5, goldenOther, goldenToken)
			return err
		}},
		{name: "evm_user_modify", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.UseBigBlocks(ctx, true)
			return err
		}},
		{name: "token_delegate", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.TokenDelegate(ctx, goldenOther, 100000000, false)
			return err
		}},
		{name: "withdraw3", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.WithdrawFromBridge(ctx, 1.5, goldenOther)
			return err
		}},
		{name: "approve_agent_named", call: func(ctx context.Context, ex *Exchange) error {
			_, _, err := ex.ApproveAgent(ctx, &bot)
			return err
		}, direct: func(pk *ecdsa.PrivateKey) (goldenSigned, error) {
			return signGoldenAgent(pk, &bot)
		}},
		{name: "approve_agent_unnamed", call: func(ctx context.Context, ex *Exchange) error {
			_, _, err := ex.ApproveAgent(ctx, nil)
			return err
		}, direct: func(pk *ecdsa.PrivateKey) (goldenSigned, error) {
			return signGoldenAgent(pk, nil)
		}},
		{name: "approve_builder_fee", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.ApproveBuilderFee(ctx, goldenOther, "0.001%")
			return err
		}},
		{name: "convert_to_multi_sig_user", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.ConvertToMultiSigUser(ctx, []string{goldenOther}, 1)
			return err
		}},
		{name: "perp_deploy_register_asset", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.PerpDeployRegisterAsset(ctx, "test", &maxGas, AssetRequest{
				Coin: "test:X", SzDecimals: 2, OraclePx: "10.0", MarginTableID: 10,
			}, nil)
			return err
		}},
		{name: "perp_deploy_register_asset2", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.PerpDeployRegisterAsset2(ctx, "test", &maxGas, assetRequest2, nil)
			return err
		}},
		{name: "perp_deploy_halt_trading", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.PerpDeployHaltTrading(ctx, "test:X", true)
			return err
		}},
		{name: "perp_deploy_set_oracle", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.PerpDeploySetOracle(
				ctx, "test",
				map[string]string{"test:X": "10.0"},
				[]map[string]string{{"test:X": "10.0"}},
				map[string]string{"test:X": "10.0"},
			)
			return err
		}},
		{name: "user_set_abstraction", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.UserSetAbstraction(ctx, userAddress, "unifiedAccount")
			return err
		}},
		{name: "send_asset", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.SendAsset(ctx, goldenOther, "", "spot", goldenToken, 1.5)
			return err
		}},
		{name: "reserve_request_weight", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.Reserve(ctx, 1)
			return err
		}},
		{name: "twap_order", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.TwapOrder(ctx, TwapOrderRequest{Coin: "BTC", IsBuy: true, Size: 0.1, Minutes: 30})
			return err
		}},
		{name: "twap_cancel", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.TwapCancel(ctx, "BTC", 77738308)
			return err
		}},
		{name: "spotDeploy_registerToken2", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.SpotDeployRegisterToken(ctx, "TKN", 2, 8, 1000000, "Token")
			return err
		}},
		{name: "spotDeploy_userGenesis", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.SpotDeployUserGenesis(
				ctx, 1, [][2]string{{goldenOther, "100"}}, []SpotDeployExistingTokenWei{{Token: 0, Wei: "50"}},
			)
			return err
		}},
		{name: "spotDeploy_userGenesis_mixedCase", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.SpotDeployUserGenesis(ctx, 1, [][2]string{{goldenMixed, "100"}}, nil)
			return err
		}},
		{name: "spotDeploy_enableFreezePrivilege", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.SpotDeployEnableFreezePrivilege(ctx, 1)
			return err
		}},
		{name: "spotDeploy_freezeUser", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.SpotDeployFreezeUser(ctx, 1, goldenOther, true)
			return err
		}},
		{name: "spotDeploy_revokeFreezePrivilege", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.SpotDeployRevokeFreezePrivilege(ctx, 1)
			return err
		}},
		{name: "spotDeploy_genesis", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.SpotDeployGenesis(ctx, 1, "1000000", false)
			return err
		}},
		{name: "spotDeploy_genesis_noHyperliquidity", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.SpotDeployGenesis(ctx, 1, "1000000", true)
			return err
		}},
		{name: "spotDeploy_registerSpot", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.SpotDeployRegisterSpot(ctx, 1, 0)
			return err
		}},
		{name: "spotDeploy_registerHyperliquidity", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.SpotDeployRegisterHyperliquidity(ctx, 0, 1.5, 0.5, 10, nil)
			return err
		}},
		{name: "spotDeploy_registerHyperliquidity_nSeededLevels", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.SpotDeployRegisterHyperliquidity(ctx, 0, 1.5, 0.5, 10, &nSeededLevels)
			return err
		}},
		{name: "spotDeploy_setDeployerTradingFeeShare", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.SpotDeploySetDeployerTradingFeeShare(ctx, 1, "50%")
			return err
		}},
		{name: "spotDeploy_enableQuoteToken", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.SpotDeployEnableQuoteToken(ctx, 1)
			return err
		}},
		{name: "cSigner_unjailSelf", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.CSignerUnjailSelf(ctx)
			return err
		}},
		{name: "cSigner_jailSelf", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.CSignerJailSelf(ctx)
			return err
		}},
		{name: "cValidator_register", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.CValidatorRegister(ctx, "1.2.3.4", "val", "desc", false, 500, goldenOther, true, 100000000)
			return err
		}},
		{name: "borrowLend", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.BorrowLend(ctx, 1.5, "supply", 0)
			return err
		}},
	}
}

func TestGolden(t *testing.T) {
	pk := newTestGoldenKey(t, goldenKey)
	userAddress := strings.ToLower(crypto.PubkeyToAddress(pk.PublicKey).Hex())
	files := loadTestGoldenFiles(t)
	rows := make(map[string]goldenRow)
	for _, row := range goldenRows(userAddress) {
		rows[row.name] = row
	}

	for name := range files {
		_, ok := rows[name]
		require.True(t, ok, "vector file %s has no row", name)
	}
	for name := range rows {
		_, ok := files[name]
		require.True(t, ok, "row %s has no vector file", name)
	}

	for name, row := range rows {
		file := files[name]
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if row.skip != "" {
				t.Skip("known: " + row.skip)
			}

			require.Nil(t, file.VaultAddress)
			require.Nil(t, file.ExpiresAfter)
			require.False(t, file.IsMainnet)
			require.Equal(t, goldenNonce, file.Nonce)

			var action json.RawMessage
			var signature SignatureResult
			if row.direct != nil {
				signed, err := row.direct(pk)
				require.NoError(t, err)
				action, err = json.Marshal(signed.action)
				require.NoError(t, err)
				signature = signed.signature
			} else {
				fixture := newTestGoldenFixture(t, pk)
				callErr := row.call(context.Background(), fixture.exchange)
				t.Logf("method result: %v", callErr)
				require.Equal(t, 1, fixture.signer.calls)
				require.NotNil(t, fixture.body, "method did not post")
				action = fixture.body["action"]
				require.NoError(t, json.Unmarshal(fixture.body["signature"], &signature))
				if file.Kind == "user" {
					require.Equal(t, file.PrimaryType, fixture.signer.primaryType)
					require.Equal(t, toGoldenTypes(fixture.signer.types), file.Types)
				}
			}

			require.JSONEq(t, string(file.Action), string(action))
			require.Equal(t, file.Signature, signature)
		})
	}
}

func toGoldenTypes(types []apitypes.Type) []goldenType {
	out := make([]goldenType, len(types))
	for i, typ := range types {
		out[i] = goldenType{Name: typ.Name, Type: typ.Type}
	}
	return out
}
