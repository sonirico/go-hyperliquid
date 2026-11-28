package hyperliquid

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"testing/synctest"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestNextNonce(t *testing.T) {
	cases := []struct {
		name          string
		preLastOffset int64         // preLast = baseNow + preLastOffset
		advance       time.Duration // fake time to advance before calling nextNonce
	}{
		{
			name:          "fresh (last < now) uses now",
			preLastOffset: -50, // last is behind the clock
			advance:       0,
		},
		{
			name:          "same millisecond increments",
			preLastOffset: 0, // last == now
			advance:       0,
		},
		{
			name:          "clock behind last increments",
			preLastOffset: 100, // last > now
			advance:       0,
		},
		{
			name:          "after time advances uses new now",
			preLastOffset: 0,                     // equal first
			advance:       10 * time.Millisecond, // then clock moves forward
		},
	}

	for _, tc := range cases {
		// Each table case runs in its own bubble.
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				var e Exchange

				baseNow := time.Now().UnixMilli() // fake time start of this bubble
				preLast := baseNow + tc.preLastOffset
				e.lastNonce.Store(preLast)

				if tc.advance > 0 {
					time.Sleep(tc.advance) // advances fake clock instantly within the bubble
				}

				got := e.nextNonce()

				now := baseNow + tc.advance.Milliseconds()
				want := now
				if now <= preLast {
					want = preLast + 1
				}

				if got != want {
					t.Fatalf(
						"nextNonce()=%d, want %d (preLast=%d now=%d advance=%s)",
						got,
						want,
						preLast,
						now,
						tc.advance,
					)
				}
				if stored := e.lastNonce.Load(); stored != want {
					t.Fatalf("lastNonce=%d, want %d", stored, want)
				}
			})
		})
	}
}

func TestNextNonce_SequentialMonotonicity(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var e Exchange

		base := time.Now().UnixMilli()

		// Call multiple times without advancing fake time: must strictly increase by 1 each call.
		const n = 5
		vals := make([]int64, 0, n)
		for i := 0; i < n; i++ {
			vals = append(vals, e.nextNonce())
		}
		for i := 0; i < n; i++ {
			want := base + int64(i)
			if vals[i] != want {
				t.Fatalf("seq[%d]=%d, want %d", i, vals[i], want)
			}
		}

		// Advance 7ms; next should jump to base+7
		time.Sleep(7 * time.Millisecond)
		next := e.nextNonce()
		if next != base+7 {
			t.Fatalf("after advance got %d, want %d", next, base+7)
		}
	})
}

func TestNextNonce_ConcurrencyUniqueness(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var e Exchange
		base := time.Now().UnixMilli()

		const N = 1000
		results := make([]int64, N)
		var wg sync.WaitGroup
		wg.Add(N)
		for i := 0; i < N; i++ {
			go func(i int) {
				defer wg.Done()
				results[i] = e.nextNonce()
			}(i)
		}
		wg.Wait()

		seen := make(map[int64]struct{}, N)
		var min, max int64
		for i, v := range results {
			if _, dup := seen[v]; dup {
				t.Fatalf("duplicate nonce: %d", v)
			}
			seen[v] = struct{}{}
			if i == 0 || v < min {
				min = v
			}
			if i == 0 || v > max {
				max = v
			}
		}

		if min != base {
			t.Fatalf("min=%d want %d", min, base)
		}
		if max != base+N-1 {
			t.Fatalf("max=%d want %d", max, base+N-1)
		}
	})
}

func newTestStatusErrExchange(t *testing.T) *Exchange {
	t.Helper()
	pk, err := crypto.HexToECDSA(goldenKey)
	require.NoError(t, err)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"status":"err","response":"boom"}`)
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)
	meta := &Meta{Universe: []AssetInfo{{Name: "BTC", SzDecimals: 5}}}
	return NewExchange(context.Background(), pk, server.URL, meta, "", "", &SpotMeta{}, nil)
}

func TestActionError_StatusErr(t *testing.T) {
	maxGas := 1000000
	scheduleTime := int64(4102444800000)
	bot := "bot"

	type testCase struct {
		name string
		call func(ctx context.Context, ex *Exchange) error
	}

	testCases := []testCase{
		{"ScheduleCancel", func(ctx context.Context, ex *Exchange) error {
			_, err := ex.ScheduleCancel(ctx, &scheduleTime)
			return err
		}},
		{"UpdateLeverage", func(ctx context.Context, ex *Exchange) error {
			_, err := ex.UpdateLeverage(ctx, 5, "BTC", true)
			return err
		}},
		{"UpdateIsolatedMargin", func(ctx context.Context, ex *Exchange) error {
			_, err := ex.UpdateIsolatedMargin(ctx, 10, "BTC")
			return err
		}},
		{"SetReferrer", func(ctx context.Context, ex *Exchange) error {
			_, err := ex.SetReferrer(ctx, "ABC")
			return err
		}},
		{"CreateSubAccount", func(ctx context.Context, ex *Exchange) error {
			_, err := ex.CreateSubAccount(ctx, "sub1")
			return err
		}},
		{"UsdClassTransfer", func(ctx context.Context, ex *Exchange) error {
			_, err := ex.UsdClassTransfer(ctx, 1.5, true)
			return err
		}},
		{"SubAccountTransfer", func(ctx context.Context, ex *Exchange) error {
			_, err := ex.SubAccountTransfer(ctx, goldenOther, true, 1000000)
			return err
		}},
		{"SubAccountSpotTransfer", func(ctx context.Context, ex *Exchange) error {
			_, err := ex.SubAccountSpotTransfer(ctx, goldenOther, true, goldenToken, 1.5)
			return err
		}},
		{"VaultUsdTransfer", func(ctx context.Context, ex *Exchange) error {
			_, err := ex.VaultUsdTransfer(ctx, goldenOther, true, 1000000)
			return err
		}},
		{"CreateVault", func(ctx context.Context, ex *Exchange) error {
			_, err := ex.CreateVault(ctx, "vault", "description", 1000000)
			return err
		}},
		{"VaultModify", func(ctx context.Context, ex *Exchange) error {
			_, err := ex.VaultModify(ctx, goldenOther, true, false)
			return err
		}},
		{"VaultDistribute", func(ctx context.Context, ex *Exchange) error {
			_, err := ex.VaultDistribute(ctx, goldenOther, 1000000)
			return err
		}},
		{"UsdTransfer", func(ctx context.Context, ex *Exchange) error {
			_, err := ex.UsdTransfer(ctx, 1.5, goldenOther)
			return err
		}},
		{"SpotTransfer", func(ctx context.Context, ex *Exchange) error {
			_, err := ex.SpotTransfer(ctx, 1.5, goldenOther, goldenToken)
			return err
		}},
		{"UseBigBlocks", func(ctx context.Context, ex *Exchange) error {
			_, err := ex.UseBigBlocks(ctx, true)
			return err
		}},
		{"TokenDelegate", func(ctx context.Context, ex *Exchange) error {
			_, err := ex.TokenDelegate(ctx, goldenOther, 100000000, false)
			return err
		}},
		{"WithdrawFromBridge", func(ctx context.Context, ex *Exchange) error {
			_, err := ex.WithdrawFromBridge(ctx, 1.5, goldenOther)
			return err
		}},
		{"SendToEVMWithData", func(ctx context.Context, ex *Exchange) error {
			_, err := ex.SendToEVMWithData(
				ctx, goldenToken, "1.5", "", goldenOther, "hex", 1337, 100000, "0x",
			)
			return err
		}},
		{"ApproveAgent", func(ctx context.Context, ex *Exchange) error {
			_, _, err := ex.ApproveAgent(ctx, &bot)
			return err
		}},
		{"ApproveBuilderFee", func(ctx context.Context, ex *Exchange) error {
			_, err := ex.ApproveBuilderFee(ctx, goldenOther, "0.001%")
			return err
		}},
		{"ConvertToMultiSigUser", func(ctx context.Context, ex *Exchange) error {
			_, err := ex.ConvertToMultiSigUser(ctx, []string{goldenOther}, 1)
			return err
		}},
		{"SpotDeployRegisterToken", func(ctx context.Context, ex *Exchange) error {
			_, err := ex.SpotDeployRegisterToken(ctx, "TKN", 2, 8, 1000000, "Token")
			return err
		}},
		{"PerpDeployRegisterAsset", func(ctx context.Context, ex *Exchange) error {
			_, err := ex.PerpDeployRegisterAsset(ctx, "test", &maxGas, AssetRequest{
				Coin: "test:X", SzDecimals: 2, OraclePx: "10.0", MarginTableID: 10,
			}, nil)
			return err
		}},
		{"PerpDeployRegisterAsset2", func(ctx context.Context, ex *Exchange) error {
			_, err := ex.PerpDeployRegisterAsset2(ctx, "test", &maxGas, AssetRequest2{
				Coin: "test:X", SzDecimals: 2, OraclePx: "10.0", MarginTableID: 10, MarginMode: "strictIsolated",
			}, nil)
			return err
		}},
		{"PerpDeployHaltTrading", func(ctx context.Context, ex *Exchange) error {
			_, err := ex.PerpDeployHaltTrading(ctx, "test:X", true)
			return err
		}},
		{"PerpDeploySetOracle", func(ctx context.Context, ex *Exchange) error {
			_, err := ex.PerpDeploySetOracle(
				ctx, "test",
				map[string]string{"test:X": "10.0"},
				[]map[string]string{{"test:X": "10.0"}},
				map[string]string{"test:X": "10.0"},
			)
			return err
		}},
		{"Reserve", func(ctx context.Context, ex *Exchange) error {
			_, err := ex.Reserve(ctx, 1)
			return err
		}},
		{"UserSetAbstraction", func(ctx context.Context, ex *Exchange) error {
			_, err := ex.UserSetAbstraction(ctx, goldenOther, "unifiedAccount")
			return err
		}},
		{"SendAsset", func(ctx context.Context, ex *Exchange) error {
			_, err := ex.SendAsset(ctx, goldenOther, "", "spot", goldenToken, 1.5)
			return err
		}},
		{"BorrowLend", func(ctx context.Context, ex *Exchange) error {
			_, err := ex.BorrowLend(ctx, 1.5, "supply", 0)
			return err
		}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newTestStatusErrExchange(t)

			err := tc.call(context.Background(), ex)

			require.ErrorContains(t, err, "boom")
		})
	}
}

func TestPostAction_VaultAddress(t *testing.T) {
	pk, err := crypto.HexToECDSA(goldenKey)
	require.NoError(t, err)

	type testCase struct {
		name     string
		call     func(ctx context.Context, ex *Exchange) error
		expected any
	}

	testCases := []testCase{
		{"SendAsset", func(ctx context.Context, ex *Exchange) error {
			_, err := ex.SendAsset(ctx, goldenOther, "", "spot", goldenToken, 1.5)
			return err
		}, nil},
		{"UsdClassTransfer", func(ctx context.Context, ex *Exchange) error {
			_, err := ex.UsdClassTransfer(ctx, 1.5, true)
			return err
		}, nil},
		{"SetReferrer", func(ctx context.Context, ex *Exchange) error {
			_, err := ex.SetReferrer(ctx, "ABC")
			return err
		}, goldenOther},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server, recorded := newUserSignedTestServer(t)
			ex := NewExchange(
				context.Background(),
				pk,
				server.URL,
				&Meta{},
				goldenOther,
				"",
				&SpotMeta{},
				&MixedArray{},
			)

			callErr := tc.call(context.Background(), ex)

			require.NoError(t, callErr)
			bodies := recorded()
			require.Len(t, bodies, 1)
			vault, ok := bodies[0]["vaultAddress"]
			require.True(t, ok, "vaultAddress missing from the payload")
			require.Equal(t, tc.expected, vault)
		})
	}
}

func TestInjectedL1Signer(t *testing.T) {
	pk, err := crypto.HexToECDSA(goldenKey)
	require.NoError(t, err)
	maxGas := 1000000

	type testCase struct {
		name string
		call func(ctx context.Context, ex *Exchange) error
	}

	testCases := []testCase{
		{"SetReferrer", func(ctx context.Context, ex *Exchange) error {
			_, err := ex.SetReferrer(ctx, "ABC")
			return err
		}},
		{"Reserve", func(ctx context.Context, ex *Exchange) error {
			_, err := ex.Reserve(ctx, 1)
			return err
		}},
		{"PerpDeployRegisterAsset2", func(ctx context.Context, ex *Exchange) error {
			_, err := ex.PerpDeployRegisterAsset2(ctx, "test", &maxGas, AssetRequest2{
				Coin: "test:X", SzDecimals: 2, OraclePx: "10.0", MarginTableID: 10, MarginMode: "strictIsolated",
			}, nil)
			return err
		}},
		{"PerpDeployHaltTrading", func(ctx context.Context, ex *Exchange) error {
			_, err := ex.PerpDeployHaltTrading(ctx, "test:X", true)
			return err
		}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fixture := newTestGoldenFixture(t, pk)

			callErr := tc.call(context.Background(), fixture.exchange)

			require.NoError(t, callErr)
			require.Equal(t, 1, fixture.signer.calls)
		})
	}
}
