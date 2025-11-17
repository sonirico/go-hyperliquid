package hyperliquid

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestL2Book_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		book     L2Book
		expected string
	}{
		{
			name: "empty_book",
			book: L2Book{
				Coin:   "BTC",
				Levels: [][]Level{},
				Time:   1234567890,
			},
			expected: `{"coin":"BTC","levels":[],"time":1234567890}`,
		},
		{
			name: "book_with_levels",
			book: L2Book{
				Coin: "ETH",
				Levels: [][]Level{
					{
						{N: 1, Px: 3000.0, Sz: 1.5},
						{N: 2, Px: 3001.0, Sz: 2.0},
					},
					{
						{N: 1, Px: 2999.0, Sz: 0.8},
					},
				},
				Time: 1234567891,
			},
			expected: `{"coin":"ETH","levels":[[{"n":1,"px":"3000","sz":"1.5"},{"n":2,"px":"3001","sz":"2"}],[{"n":1,"px":"2999","sz":"0.8"}]],"time":1234567891}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			jsonData, err := json.Marshal(tt.book)
			require.NoError(t, err, "marshaling should not fail")
			assert.JSONEq(t, tt.expected, string(jsonData), "marshaled JSON should match expected")

			// Test round-trip
			var unmarshaled L2Book
			err = json.Unmarshal(jsonData, &unmarshaled)
			require.NoError(t, err, "unmarshaling should not fail")
			assert.Equal(t, tt.book, unmarshaled, "round-trip should preserve data")
		})
	}
}

func TestLevel_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		level    Level
		expected string
	}{
		{
			name: "integer_values",
			level: Level{
				N:  1,
				Px: 50000.0,
				Sz: 1.0,
			},
			expected: `{"n":1,"px":"50000","sz":"1"}`,
		},
		{
			name: "decimal_values",
			level: Level{
				N:  5,
				Px: 3000.5,
				Sz: 0.123456,
			},
			expected: `{"n":5,"px":"3000.5","sz":"0.123456"}`,
		},
		{
			name: "zero_values",
			level: Level{
				N:  0,
				Px: 0.0,
				Sz: 0.0,
			},
			expected: `{"n":0,"px":"0","sz":"0"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			jsonData, err := json.Marshal(tt.level)
			require.NoError(t, err, "marshaling should not fail")
			assert.JSONEq(t, tt.expected, string(jsonData), "marshaled JSON should match expected")

			// Test round-trip
			var unmarshaled Level
			err = json.Unmarshal(jsonData, &unmarshaled)
			require.NoError(t, err, "unmarshaling should not fail")
			assert.Equal(t, tt.level, unmarshaled, "round-trip should preserve data")
		})
	}
}

func TestPosition_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		pos      Position
		expected string
	}{
		{
			name: "long_position",
			pos: Position{
				Coin:           "BTC",
				EntryPx:        stringPtr("50000.0"),
				Leverage:       Leverage{Type: "cross", Value: 10},
				LiquidationPx:  stringPtr("45000.0"),
				MarginUsed:     "5000.0",
				PositionValue:  "50000.0",
				ReturnOnEquity: "0.05",
				Szi:            "1.0",
				UnrealizedPnl:  "2500.0",
				CumFunding: &CumFunding{
					AllTime:     "0.03621",
					SinceChange: "-0.001129",
					SinceOpen:   "-0.001129",
				},
			},
			expected: `{"coin":"BTC","entryPx":"50000.0","leverage":{"type":"cross","value":10},"liquidationPx":"45000.0","marginUsed":"5000.0","positionValue":"50000.0","returnOnEquity":"0.05","szi":"1.0","unrealizedPnl":"2500.0","cumFunding":{"allTime":"0.03621","sinceOpen":"-0.001129","sinceChange":"-0.001129"}}`,
		},
		{
			name: "no_position",
			pos: Position{
				Coin:           "ETH",
				EntryPx:        nil,
				Leverage:       Leverage{Type: "isolated", Value: 5, RawUsd: stringPtr("1000.0")},
				LiquidationPx:  nil,
				MarginUsed:     "0.0",
				PositionValue:  "0.0",
				ReturnOnEquity: "0.0",
				Szi:            "0.0",
				UnrealizedPnl:  "0.0",
			},
			expected: `{"coin":"ETH","entryPx":null,"leverage":{"type":"isolated","value":5,"rawUsd":"1000.0"},"liquidationPx":null,"marginUsed":"0.0","positionValue":"0.0","returnOnEquity":"0.0","szi":"0.0","unrealizedPnl":"0.0"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			jsonData, err := json.Marshal(tt.pos)
			require.NoError(t, err, "marshaling should not fail")
			assert.JSONEq(t, tt.expected, string(jsonData), "marshaled JSON should match expected")

			// Test round-trip
			var unmarshaled Position
			err = json.Unmarshal(jsonData, &unmarshaled)
			require.NoError(t, err, "unmarshaling should not fail")
			assert.Equal(t, tt.pos, unmarshaled, "round-trip should preserve data")
		})
	}
}

func TestLeverage_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		leverage Leverage
		expected string
	}{
		{
			name: "cross_leverage",
			leverage: Leverage{
				Type:  "cross",
				Value: 10,
			},
			expected: `{"type":"cross","value":10}`,
		},
		{
			name: "isolated_leverage_with_rawusd",
			leverage: Leverage{
				Type:   "isolated",
				Value:  5,
				RawUsd: stringPtr("1000.0"),
			},
			expected: `{"type":"isolated","value":5,"rawUsd":"1000.0"}`,
		},
		{
			name: "isolated_leverage_without_rawusd",
			leverage: Leverage{
				Type:   "isolated",
				Value:  20,
				RawUsd: nil,
			},
			expected: `{"type":"isolated","value":20}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			jsonData, err := json.Marshal(tt.leverage)
			require.NoError(t, err, "marshaling should not fail")
			assert.JSONEq(t, tt.expected, string(jsonData), "marshaled JSON should match expected")

			// Test round-trip
			var unmarshaled Leverage
			err = json.Unmarshal(jsonData, &unmarshaled)
			require.NoError(t, err, "unmarshaling should not fail")
			assert.Equal(t, tt.leverage, unmarshaled, "round-trip should preserve data")
		})
	}
}

func TestUserState_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		state    UserState
		expected string
	}{
		{
			name: "user_state_with_positions",
			state: UserState{
				AssetPositions: []AssetPosition{
					{
						Position: Position{
							Coin:           "BTC",
							EntryPx:        stringPtr("50000.0"),
							Leverage:       Leverage{Type: "cross", Value: 10},
							LiquidationPx:  stringPtr("45000.0"),
							MarginUsed:     "5000.0",
							PositionValue:  "50000.0",
							ReturnOnEquity: "0.05",
							Szi:            "1.0",
							UnrealizedPnl:  "2500.0",
						},
						Type: "oneWay",
					},
				},
				CrossMarginSummary: MarginSummary{
					AccountValue:    "100000.0",
					TotalMarginUsed: "5000.0",
					TotalNtlPos:     "50000.0",
					TotalRawUsd:     "100000.0",
				},
				MarginSummary: MarginSummary{
					AccountValue:    "100000.0",
					TotalMarginUsed: "5000.0",
					TotalNtlPos:     "50000.0",
					TotalRawUsd:     "100000.0",
				},
				Withdrawable: "95000.0",
			},
			expected: `{"assetPositions":[{"position":{"coin":"BTC","entryPx":"50000.0","leverage":{"type":"cross","value":10},"liquidationPx":"45000.0","marginUsed":"5000.0","positionValue":"50000.0","returnOnEquity":"0.05","szi":"1.0","unrealizedPnl":"2500.0"},"type":"oneWay"}],"crossMarginSummary":{"accountValue":"100000.0","totalMarginUsed":"5000.0","totalNtlPos":"50000.0","totalRawUsd":"100000.0"},"marginSummary":{"accountValue":"100000.0","totalMarginUsed":"5000.0","totalNtlPos":"50000.0","totalRawUsd":"100000.0"},"withdrawable":"95000.0"}`,
		},
		{
			name: "empty_user_state",
			state: UserState{
				AssetPositions: []AssetPosition{},
				CrossMarginSummary: MarginSummary{
					AccountValue:    "0.0",
					TotalMarginUsed: "0.0",
					TotalNtlPos:     "0.0",
					TotalRawUsd:     "0.0",
				},
				MarginSummary: MarginSummary{
					AccountValue:    "0.0",
					TotalMarginUsed: "0.0",
					TotalNtlPos:     "0.0",
					TotalRawUsd:     "0.0",
				},
				Withdrawable: "0.0",
			},
			expected: `{"assetPositions":[],"crossMarginSummary":{"accountValue":"0.0","totalMarginUsed":"0.0","totalNtlPos":"0.0","totalRawUsd":"0.0"},"marginSummary":{"accountValue":"0.0","totalMarginUsed":"0.0","totalNtlPos":"0.0","totalRawUsd":"0.0"},"withdrawable":"0.0"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			jsonData, err := json.Marshal(tt.state)
			require.NoError(t, err, "marshaling should not fail")
			assert.JSONEq(t, tt.expected, string(jsonData), "marshaled JSON should match expected")

			// Test round-trip
			var unmarshaled UserState
			err = json.Unmarshal(jsonData, &unmarshaled)
			require.NoError(t, err, "unmarshaling should not fail")
			assert.Equal(t, tt.state, unmarshaled, "round-trip should preserve data")
		})
	}
}

func TestOpenOrder_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		order    OpenOrder
		expected string
	}{
		{
			name: "buy_order",
			order: OpenOrder{
				Coin:      "BTC",
				LimitPx:   49500.0,
				Oid:       12345,
				Side:      "B",
				Size:      0.5,
				Timestamp: 1234567890,
			},
			expected: `{"coin":"BTC","limitPx":"49500","oid":12345,"side":"B","sz":"0.5","timestamp":1234567890}`,
		},
		{
			name: "sell_order",
			order: OpenOrder{
				Coin:      "ETH",
				LimitPx:   3100.0,
				Oid:       67890,
				Side:      "A",
				Size:      2.0,
				Timestamp: 1234567891,
			},
			expected: `{"coin":"ETH","limitPx":"3100","oid":67890,"side":"A","sz":"2","timestamp":1234567891}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			jsonData, err := json.Marshal(tt.order)
			require.NoError(t, err, "marshaling should not fail")
			assert.JSONEq(t, tt.expected, string(jsonData), "marshaled JSON should match expected")

			// Test round-trip
			var unmarshaled OpenOrder
			err = json.Unmarshal(jsonData, &unmarshaled)
			require.NoError(t, err, "unmarshaling should not fail")
			assert.Equal(t, tt.order, unmarshaled, "round-trip should preserve data")
		})
	}
}

func TestLevel_UnmarshalJSON_StringToFloat(t *testing.T) {
	// Test that px and sz fields can be unmarshaled from string values
	jsonData := `{"n":1,"px":"50000.123","sz":"1.456789"}`

	var level Level
	err := json.Unmarshal([]byte(jsonData), &level)
	require.NoError(t, err, "unmarshaling should not fail")

	assert.Equal(t, 1, level.N)
	assert.Equal(t, 50000.123, level.Px)
	assert.Equal(t, 1.456789, level.Sz)
}
