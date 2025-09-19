package examples

import (
	"context"
	"testing"
)

func TestUpdateLeverage(t *testing.T) {
	exchange := newTestExchange(t)

	leverage := 5 // 5x leverage
	name := "BTC"
	isCross := true // Use cross margin

	resp, err := exchange.UpdateLeverage(context.TODO(), leverage, name, isCross)
	if err != nil {
		t.Fatalf("Failed to update leverage: %v", err)
	}

	t.Logf("Update leverage response: %+v", resp)
}

func TestUpdateIsolatedMargin(t *testing.T) {
	exchange := newTestExchange(t)

	amount := 1000.0 // Amount in USD
	name := "BTC"

	resp, err := exchange.UpdateIsolatedMargin(context.TODO(), amount, name)
	if err != nil {
		t.Fatalf("Failed to update isolated margin: %v", err)
	}

	t.Logf("Update isolated margin response: %+v", resp)
}
