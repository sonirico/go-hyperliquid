package hyperliquid

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// roundToDecimals rounds a float64 to the specified number of decimals.
func roundToDecimals(value float64, decimals int) float64 {
	pow := math.Pow(10, float64(decimals))
	return math.Round(value*pow) / pow
}

// parseFloat parses a string to float64, returns 0.0 if parsing fails.
func parseFloat(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0.0
	}
	return f
}

// abs returns the absolute value of a float64.
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// formatFloat formats a float64 to string with 6 decimal places.
func formatFloat(f float64) string {
	return fmt.Sprintf("%.6f", f)
}

// floatToWire converts a float64 to a wire-compatible string format
func floatToWire(x float64) (string, error) {
	// Format to 8 decimal places
	rounded := fmt.Sprintf("%.8f", x)

	// Check if rounding causes significant error
	parsed, err := strconv.ParseFloat(rounded, 64)
	if err != nil {
		return "", err
	}

	if math.Abs(parsed-x) >= 1e-12 {
		return "", fmt.Errorf("float_to_wire causes rounding: %f", x)
	}

	// Handle -0 case
	if rounded == "-0.00000000" {
		rounded = "0.00000000"
	}

	// Remove trailing zeros and decimal point if not needed
	result := strings.TrimRight(rounded, "0")
	result = strings.TrimRight(result, ".")

	return result, nil
}

func roundToSignificantFigures(price float64, sigFigs int) float64 {
	if price == 0 {
		return 0
	}

	// Work with the absolute value of the price to simplify calculations. We will restore the sign later.
	absPrice := math.Abs(price)

	// Determine the integer part of the absolute price (e.g., for 123.45, integerPart is 123).
	integerPart := math.Floor(absPrice)

	// Calculate the number of digits in the integer part.
	// This helps in deciding if we're rounding to an integer or including fractional parts.
	numIntegerDigits := 0
	if integerPart > 0 {
		// Count the number of digits in the integer part.
		temp := int(integerPart)
		for temp > 0 {
			temp /= 10
			numIntegerDigits++
		}

		if numIntegerDigits >= sigFigs {
			// Returning the integer part, keeping the original sign.
			// We do need to preserve the whole integer part, even though it may result in more significant figures than requested.
			return math.Copysign(integerPart, price)
		}

		sigFigsLeft := sigFigs - numIntegerDigits

		// Round the float64 to the number of significant figures left.
		rounded := roundToDecimals(absPrice, sigFigsLeft)

		// Return the rounded number, applying the original sign.
		return math.Copysign(rounded, price)
	} else {
		// Working with a fraction, multiply by 10 until we have an integer so we can use roundToDecimals.
		multiplications := 0
		for absPrice < 1 {
			absPrice *= 10
			multiplications++
		}

		// Round the integer to the number of significant figures.
		// Using sigFigs-1 since the integer part is already counted as a significant figure.
		rounded := roundToDecimals(absPrice, sigFigs-1)

		// Divide by 10^multiplications to get the original fraction.
		return math.Copysign(rounded/math.Pow(10, float64(multiplications)), price)
	}
}
