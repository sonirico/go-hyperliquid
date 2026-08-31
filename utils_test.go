package hyperliquid

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRoundToSignificantFigures(t *testing.T) {
	tests := []struct {
		name     string
		price    float64
		sigFigs  int
		expected float64
	}{
		{
			name:     "keeps all digits",
			price:    123.456789,
			sigFigs:  9,
			expected: 123.456789,
		},
		{
			name:     "keeps 2 of the 3 decimal places",
			price:    123.453,
			sigFigs:  5,
			expected: 123.45,
		},
		{
			name:     "if integer part has more significant figures, return whole integer part",
			price:    110454,
			sigFigs:  5,
			expected: 110454,
		},
		{
			name:     "even if sigFigs is 0, return the whole integer part",
			price:    24,
			sigFigs:  0,
			expected: 24,
		},
		{
			name:     "test rounding a fraction",
			price:    0.00252312,
			sigFigs:  5,
			expected: 0.0025231,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rounded := roundToSignificantFigures(test.price, test.sigFigs)
			assert.Equal(t, test.expected, rounded)
		})
	}
}

func TestParseFloatStrict(t *testing.T) {
	// Valid numbers parse the same as parseFloat.
	for _, s := range []string{"0", "1.5", "12345.6789", "-3.2", "0.00000001"} {
		got, err := parseFloatStrict(s)
		assert.NoError(t, err, "input %q", s)
		assert.Equal(t, parseFloat(s), got, "input %q", s)
	}

	// Invalid input must error, NOT silently return 0 the way parseFloat does.
	// This is the difference that matters for financial values off the wire:
	// a malformed mid price must not become a market order priced at 0.
	for _, s := range []string{"", "abc", "1.2.3", "NaNaN", "0x10", " "} {
		_, err := parseFloatStrict(s)
		assert.Error(t, err, "input %q must be rejected", s)
		// Document the dangerous legacy behaviour the strict variant guards against.
		assert.Equal(t, 0.0, parseFloat(s), "parseFloat silently returns 0 for %q", s)
	}
}
