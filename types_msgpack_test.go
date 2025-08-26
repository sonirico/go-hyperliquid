package hyperliquid

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"log"
	"testing"

	"github.com/vmihailenco/msgpack/v5"
)

type OrderWireOld struct {
	Asset      int            `json:"a"`
	IsBuy      bool           `json:"b"`
	LimitPx    string         `json:"p"`
	Size       string         `json:"s"`
	ReduceOnly bool           `json:"r"`
	OrderType  map[string]any `json:"t"`
	Cloid      *string        `json:"c,omitempty"`
}

type OrderWireNew struct {
	Asset      int           `json:"a"           msgpack:"a"`
	IsBuy      bool          `json:"b"           msgpack:"b"`
	LimitPx    string        `json:"p"           msgpack:"p"`
	Size       string        `json:"s"           msgpack:"s"`
	ReduceOnly bool          `json:"r"           msgpack:"r"`
	OrderType  orderWireType `json:"t"           msgpack:"t"`
	Cloid      *string       `json:"c,omitempty" msgpack:"c,omitempty"`
}

func Test_Msgpack_Field_Ordering(t *testing.T) {
	// Test data
	orderType := map[string]any{
		"limit": map[string]any{
			"tif": "Gtc",
		},
	}

	orderTypeNew := orderWireType{
		Limit: &orderWireTypeLimit{
			Tif: TifGtc,
		},
	}

	// Test with old struct (no msgpack tags)
	oldOrder := OrderWireOld{
		Asset:      0,
		IsBuy:      true,
		LimitPx:    "40000",
		Size:       "0.001",
		ReduceOnly: false,
		OrderType:  orderType,
	}

	// Test with new struct (with msgpack tags)
	newOrder := OrderWireNew{
		Asset:      0,
		IsBuy:      true,
		LimitPx:    "40000",
		Size:       "0.001",
		ReduceOnly: false,
		OrderType:  orderTypeNew,
	}

	// Serialize both with msgpack
	var bufOld, bufNew bytes.Buffer

	encOld := msgpack.NewEncoder(&bufOld)
	encOld.SetSortMapKeys(true)

	encNew := msgpack.NewEncoder(&bufNew)
	encNew.SetSortMapKeys(true)

	err := encOld.Encode(oldOrder)
	if err != nil {
		log.Fatal(err)
	}

	err = encNew.Encode(newOrder)
	if err != nil {
		log.Fatal(err)
	}

	oldBytes := bufOld.Bytes()
	newBytes := bufNew.Bytes()

	fmt.Printf("Old struct (no msgpack tags): %s\n", hex.EncodeToString(oldBytes))
	fmt.Printf("New struct (with msgpack tags): %s\n", hex.EncodeToString(newBytes))
	fmt.Printf("Are they identical? %v\n", bytes.Equal(oldBytes, newBytes))
}
