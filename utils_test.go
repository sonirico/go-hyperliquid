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
			name:     "fraction below 1 should consider 0 as a significant figure",
			price:    0.12,
			sigFigs:  2,
			expected: 0.1,
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
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rounded, err := roundToSignificantFigures(test.price, test.sigFigs)
			assert.NoError(t, err)
			assert.Equal(t, test.expected, rounded)
		})
	}
}
