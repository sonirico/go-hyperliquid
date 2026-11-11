package hyperliquid

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// newInfoTestServer serves a single /info request: it records the request body
// and answers with response.
func newInfoTestServer(t *testing.T, response string) (*Info, *map[string]any) {
	t.Helper()

	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/info", r.URL.Path)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(body, &got))
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, response)
	}))
	t.Cleanup(server.Close)

	// Pass empty metadata so NewInfo doesn't query meta and spotMeta first.
	return NewInfo(context.TODO(), server.URL, true, &Meta{}, &SpotMeta{}, nil), &got
}

func TestUserRateLimit(t *testing.T) {
	info, req := newInfoTestServer(
		t,
		`{"cumVlm":"2854574.593578","nRequestsUsed":2890,"nRequestsCap":2864574,"nRequestsSurplus":0}`,
	)

	res, err := info.UserRateLimit(context.TODO(), "0x0000000000000000000000000000000000000001")
	require.NoError(t, err)
	require.Equal(t, map[string]any{
		"type": "userRateLimit",
		"user": "0x0000000000000000000000000000000000000001",
	}, *req)
	require.Equal(t, &UserRateLimit{
		CumVlm:           "2854574.593578",
		NRequestsUsed:    2890,
		NRequestsCap:     2864574,
		NRequestsSurplus: 0,
	}, res)
}

func TestUserRole(t *testing.T) {
	const user = "0x0000000000000000000000000000000000000001"

	tests := []struct {
		name     string
		response string
		want     UserRole
	}{
		{"user", `{"role":"user"}`, UserRole{Role: UserRoleUser}},
		{"missing", `{"role":"missing"}`, UserRole{Role: UserRoleMissing}},
		{"vault", `{"role":"vault"}`, UserRole{Role: UserRoleVault}},
		{
			"agent",
			`{"role":"agent","data":{"user":"0x0000000000000000000000000000000000000002"}}`,
			UserRole{
				Role: UserRoleAgent,
				Data: &UserRoleData{User: "0x0000000000000000000000000000000000000002"},
			},
		},
		{
			"subAccount",
			`{"role":"subAccount","data":{"master":"0x0000000000000000000000000000000000000003"}}`,
			UserRole{
				Role: UserRoleSubAccount,
				Data: &UserRoleData{Master: "0x0000000000000000000000000000000000000003"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, req := newInfoTestServer(t, tt.response)

			res, err := info.UserRole(context.TODO(), user)
			require.NoError(t, err)
			require.Equal(t, map[string]any{"type": "userRole", "user": user}, *req)
			require.Equal(t, &tt.want, res)
		})
	}
}

func TestMaxBuilderFee(t *testing.T) {
	info, req := newInfoTestServer(t, `1`)

	res, err := info.MaxBuilderFee(
		context.TODO(),
		"0x0000000000000000000000000000000000000001",
		"0x0000000000000000000000000000000000000002",
	)
	require.NoError(t, err)
	require.Equal(t, map[string]any{
		"type":    "maxBuilderFee",
		"user":    "0x0000000000000000000000000000000000000001",
		"builder": "0x0000000000000000000000000000000000000002",
	}, *req)
	require.Equal(t, 1, res)
}

func TestPredictedFundings(t *testing.T) {
	info, req := newInfoTestServer(t, `[
		["AVAX", [
			["BinPerp", {"fundingRate": "0.0001", "nextFundingTime": 1733961600000, "fundingIntervalHours": 8}],
			["HlPerp", {"fundingRate": "0.0000125", "nextFundingTime": 1733958000000, "fundingIntervalHours": 1}],
			["BybitPerp", null]
		]],
		["BTC", [
			["HlPerp", {"fundingRate": "0.00001", "nextFundingTime": 1733958000000}]
		]]
	]`)

	res, err := info.PredictedFundings(context.TODO())
	require.NoError(t, err)
	require.Equal(t, map[string]any{"type": "predictedFundings"}, *req)

	eight, one := 8, 1
	require.Equal(t, []CoinPredictedFundings{
		{
			Coin: "AVAX",
			Venues: []VenuePredictedFunding{
				{Venue: "BinPerp", Funding: &PredictedFunding{
					FundingRate: "0.0001", NextFundingTime: 1733961600000, FundingIntervalHours: &eight,
				}},
				{Venue: "HlPerp", Funding: &PredictedFunding{
					FundingRate: "0.0000125", NextFundingTime: 1733958000000, FundingIntervalHours: &one,
				}},
				{Venue: "BybitPerp", Funding: nil},
			},
		},
		{
			Coin: "BTC",
			Venues: []VenuePredictedFunding{
				{Venue: "HlPerp", Funding: &PredictedFunding{
					FundingRate: "0.00001", NextFundingTime: 1733958000000,
				}},
			},
		},
	}, res)
}

func TestPredictedFundings_InvalidResponse(t *testing.T) {
	info, _ := newInfoTestServer(t, `[["AVAX"]]`)

	_, err := info.PredictedFundings(context.TODO())
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to unmarshal predicted fundings")
}
