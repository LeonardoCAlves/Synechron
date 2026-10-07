package money

import (
	"errors"
	"math"
	"testing"
)

func TestParseCurrency(t *testing.T) {
	currency, err := ParseCurrency(" cad ")
	if err != nil {
		t.Fatalf("ParseCurrency() error = %v", err)
	}
	if currency != "CAD" {
		t.Fatalf("ParseCurrency() = %q, want CAD", currency)
	}

	for _, value := range []string{"", "CA", "C$D", "123"} {
		if _, err := ParseCurrency(value); !errors.Is(err, ErrInvalidCurrency) {
			t.Errorf("ParseCurrency(%q) error = %v, want %v", value, err, ErrInvalidCurrency)
		}
	}
}

func TestMinorUnitArithmeticChecksOverflow(t *testing.T) {
	if _, err := MultiplyMinorUnits(math.MaxInt64, 2); !errors.Is(err, ErrAmountOverflow) {
		t.Fatalf("MultiplyMinorUnits() error = %v, want %v", err, ErrAmountOverflow)
	}
	if _, err := AddMinorUnits(math.MaxInt64, 1); !errors.Is(err, ErrAmountOverflow) {
		t.Fatalf("AddMinorUnits() error = %v, want %v", err, ErrAmountOverflow)
	}
	if _, err := AddMinorUnits(-1, 1); !errors.Is(err, ErrInvalidAmount) {
		t.Fatalf("AddMinorUnits() error = %v, want %v", err, ErrInvalidAmount)
	}
}
