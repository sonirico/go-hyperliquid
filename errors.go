package hyperliquid

import (
	"fmt"
	"strings"
)

//go:generate easyjson -all

type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"msg"`
	Data    any    `json:"data,omitempty"`
}

func (e APIError) Error() string {
	return fmt.Sprintf("API error %d: %s", e.Code, e.Message)
}

type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error on field %s: %s", e.Field, e.Message)
}

// IsWalletDoesNotExistError checks if the error is a "wallet does not exist" error from the API
func IsWalletDoesNotExistError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "does not exist") &&
		(strings.Contains(errMsg, "wallet") || strings.Contains(errMsg, "user"))
}
