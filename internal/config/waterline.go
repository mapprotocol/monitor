package config

import (
	"math/big"
	"strings"

	"github.com/shopspring/decimal"
)

// ParseNativeWaterLine parses a balance threshold in human-readable native
// token units and returns the chain's smallest unit. Legacy integer values
// already written in the smallest unit are accepted for compatibility.
func ParseNativeWaterLine(value string, decimals int32) (*big.Int, bool) {
	value = strings.TrimSpace(value)
	if value == "" || decimals < 0 {
		return nil, false
	}

	if shouldTreatAsSmallestUnit(value, decimals) {
		waterLine, ok := new(big.Int).SetString(value, 10)
		return waterLine, ok
	}

	amount, err := decimal.NewFromString(value)
	if err != nil || amount.IsNegative() {
		return nil, false
	}

	scaled := amount.Shift(decimals)
	if !scaled.Equal(scaled.Truncate(0)) {
		return nil, false
	}

	waterLine := new(big.Int)
	waterLine.SetString(scaled.StringFixed(0), 10)
	return waterLine, true
}

func shouldTreatAsSmallestUnit(value string, decimals int32) bool {
	if strings.ContainsAny(value, ".+-") {
		return false
	}

	digits := strings.TrimLeft(value, "0")
	if digits == "" {
		return false
	}

	// Existing configs used wei/yocto-style integers, including small
	// thresholds like 400000000000000 (0.0004 ETH). Native unit thresholds this
	// large are not realistic for balance alarms, so keep those values compatible.
	return len(digits) >= 12
}
