package money

import (
	"errors"
	"math"
	"strings"
)

var (
	ErrInvalidCurrency = errors.New("invalid currency code")
	ErrInvalidAmount   = errors.New("amount must be non-negative")
	ErrAmountOverflow  = errors.New("amount overflow")
)

type Currency string

func ParseCurrency(value string) (Currency, error) {
	value = strings.ToUpper(strings.TrimSpace(value))
	if len(value) != 3 {
		return "", ErrInvalidCurrency
	}
	for _, character := range value {
		if character < 'A' || character > 'Z' {
			return "", ErrInvalidCurrency
		}
	}
	return Currency(value), nil
}

func MultiplyMinorUnits(unitPrice, quantity int64) (int64, error) {
	if unitPrice < 0 || quantity < 0 {
		return 0, ErrInvalidAmount
	}
	if unitPrice != 0 && quantity > math.MaxInt64/unitPrice {
		return 0, ErrAmountOverflow
	}
	return unitPrice * quantity, nil
}

func AddMinorUnits(total, amount int64) (int64, error) {
	if total < 0 || amount < 0 {
		return 0, ErrInvalidAmount
	}
	if total > math.MaxInt64-amount {
		return 0, ErrAmountOverflow
	}
	return total + amount, nil
}
