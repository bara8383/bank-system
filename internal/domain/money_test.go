package domain

import (
	"errors"
	"math"
	"testing"
)

func TestNewAmountAcceptsPositiveTransactionAmount(t *testing.T) {
	amount, err := NewAmount(1)
	if err != nil {
		t.Fatalf("expected positive amount to be accepted, got error: %v", err)
	}

	if amount.Int64() != 1 {
		t.Fatalf("expected amount 1, got %d", amount.Int64())
	}
}

func TestNewAmountRejectsZeroAndNegativeTransactionAmounts(t *testing.T) {
	testCases := map[string]int64{
		"zero":     0,
		"negative": -1,
	}

	for name, value := range testCases {
		t.Run(name, func(t *testing.T) {
			_, err := NewAmount(value)
			if !errors.Is(err, ErrAmountMustBePositive) {
				t.Fatalf("expected ErrAmountMustBePositive for %d, got %v", value, err)
			}
		})
	}
}

func TestAmountValidateRejectsZeroValueAmount(t *testing.T) {
	if err := (Amount{}).Validate(); !errors.Is(err, ErrAmountMustBePositive) {
		t.Fatalf("expected ErrAmountMustBePositive for zero value amount, got %v", err)
	}
}

func TestAmountValidateAcceptsPositiveAmount(t *testing.T) {
	amount := mustAmount(t, 1)

	if err := amount.Validate(); err != nil {
		t.Fatalf("expected positive amount to be valid, got error: %v", err)
	}
}

func TestNewBalanceAcceptsZeroAndPositiveBalances(t *testing.T) {
	testCases := map[string]int64{
		"zero":     0,
		"positive": 1,
	}

	for name, value := range testCases {
		t.Run(name, func(t *testing.T) {
			balance, err := NewBalance(value)
			if err != nil {
				t.Fatalf("expected balance %d to be accepted, got error: %v", value, err)
			}

			if balance.Int64() != value {
				t.Fatalf("expected balance %d, got %d", value, balance.Int64())
			}
		})
	}
}

func TestNewBalanceRejectsNegativeBalance(t *testing.T) {
	_, err := NewBalance(-1)
	if !errors.Is(err, ErrBalanceMustBeNonNegative) {
		t.Fatalf("expected ErrBalanceMustBeNonNegative, got %v", err)
	}
}

func TestBalanceValidateAcceptsZeroAndPositiveBalances(t *testing.T) {
	testCases := map[string]Balance{
		"zero":     {},
		"positive": mustBalance(t, 1),
	}

	for name, balance := range testCases {
		t.Run(name, func(t *testing.T) {
			if err := balance.Validate(); err != nil {
				t.Fatalf("expected balance %d to be valid, got error: %v", balance.Int64(), err)
			}
		})
	}
}

func TestBalanceValidateRejectsNegativeBalance(t *testing.T) {
	balance := Balance{value: -1}

	if err := balance.Validate(); !errors.Is(err, ErrBalanceMustBeNonNegative) {
		t.Fatalf("expected ErrBalanceMustBeNonNegative, got %v", err)
	}
}

func TestAddBalanceIncreasesBalance(t *testing.T) {
	balance := mustBalance(t, 100)
	amount := mustAmount(t, 25)

	updated, err := AddBalance(balance, amount)
	if err != nil {
		t.Fatalf("expected add balance to succeed, got error: %v", err)
	}

	if updated.Int64() != 125 {
		t.Fatalf("expected updated balance 125, got %d", updated.Int64())
	}
}

func TestAddBalanceRejectsInvalidAmountAndReturnsOriginalBalance(t *testing.T) {
	balance := mustBalance(t, 100)

	updated, err := AddBalance(balance, Amount{})
	if !errors.Is(err, ErrAmountMustBePositive) {
		t.Fatalf("expected ErrAmountMustBePositive, got %v", err)
	}

	if updated.Int64() != balance.Int64() {
		t.Fatalf("expected original balance %d, got %d", balance.Int64(), updated.Int64())
	}
}

func TestAddBalanceRejectsInvalidStartingBalanceAndReturnsOriginalBalance(t *testing.T) {
	balance := Balance{value: -1}
	amount := mustAmount(t, 1)

	updated, err := AddBalance(balance, amount)
	if !errors.Is(err, ErrBalanceMustBeNonNegative) {
		t.Fatalf("expected ErrBalanceMustBeNonNegative, got %v", err)
	}

	if updated.Int64() != balance.Int64() {
		t.Fatalf("expected original balance %d, got %d", balance.Int64(), updated.Int64())
	}
}

func TestAddBalancePrioritizesInvalidStartingBalanceOverInvalidAmount(t *testing.T) {
	balance := Balance{value: -1}

	updated, err := AddBalance(balance, Amount{})
	if !errors.Is(err, ErrBalanceMustBeNonNegative) {
		t.Fatalf("expected ErrBalanceMustBeNonNegative, got %v", err)
	}

	if updated.Int64() != balance.Int64() {
		t.Fatalf("expected original balance %d, got %d", balance.Int64(), updated.Int64())
	}
}

func TestAddBalanceReturnsOriginalBalanceOnOverflow(t *testing.T) {
	balance := mustBalance(t, math.MaxInt64)
	amount := mustAmount(t, 1)

	updated, err := AddBalance(balance, amount)
	if !errors.Is(err, ErrBalanceOverflow) {
		t.Fatalf("expected ErrBalanceOverflow, got %v", err)
	}

	if updated.Int64() != balance.Int64() {
		t.Fatalf("expected original balance %d, got %d", balance.Int64(), updated.Int64())
	}
}

func TestSubtractBalanceSucceedsWithinBalance(t *testing.T) {
	balance := mustBalance(t, 100)
	amount := mustAmount(t, 40)

	updated, err := SubtractBalance(balance, amount)
	if err != nil {
		t.Fatalf("expected subtract balance to succeed, got error: %v", err)
	}

	if updated.Int64() != 60 {
		t.Fatalf("expected updated balance 60, got %d", updated.Int64())
	}
}

func TestSubtractBalanceRejectsInvalidAmountAndReturnsOriginalBalance(t *testing.T) {
	balance := mustBalance(t, 100)

	updated, err := SubtractBalance(balance, Amount{})
	if !errors.Is(err, ErrAmountMustBePositive) {
		t.Fatalf("expected ErrAmountMustBePositive, got %v", err)
	}

	if updated.Int64() != balance.Int64() {
		t.Fatalf("expected original balance %d, got %d", balance.Int64(), updated.Int64())
	}
}

func TestSubtractBalanceRejectsInvalidStartingBalanceAndReturnsOriginalBalance(t *testing.T) {
	balance := Balance{value: -1}
	amount := mustAmount(t, 1)

	updated, err := SubtractBalance(balance, amount)
	if !errors.Is(err, ErrBalanceMustBeNonNegative) {
		t.Fatalf("expected ErrBalanceMustBeNonNegative, got %v", err)
	}

	if updated.Int64() != balance.Int64() {
		t.Fatalf("expected original balance %d, got %d", balance.Int64(), updated.Int64())
	}
}

func TestSubtractBalancePrioritizesInvalidStartingBalanceOverInvalidAmount(t *testing.T) {
	balance := Balance{value: -1}

	updated, err := SubtractBalance(balance, Amount{})
	if !errors.Is(err, ErrBalanceMustBeNonNegative) {
		t.Fatalf("expected ErrBalanceMustBeNonNegative, got %v", err)
	}

	if updated.Int64() != balance.Int64() {
		t.Fatalf("expected original balance %d, got %d", balance.Int64(), updated.Int64())
	}
}

func TestSubtractBalanceRejectsInsufficientBalanceAndReturnsOriginalBalance(t *testing.T) {
	balance := mustBalance(t, 30)
	amount := mustAmount(t, 40)

	updated, err := SubtractBalance(balance, amount)
	if !errors.Is(err, ErrInsufficientBalance) {
		t.Fatalf("expected ErrInsufficientBalance, got %v", err)
	}

	if updated.Int64() != balance.Int64() {
		t.Fatalf("expected original balance %d, got %d", balance.Int64(), updated.Int64())
	}
}

func mustAmount(t *testing.T, value int64) Amount {
	t.Helper()

	amount, err := NewAmount(value)
	if err != nil {
		t.Fatalf("NewAmount(%d) returned error: %v", value, err)
	}

	return amount
}

func mustBalance(t *testing.T, value int64) Balance {
	t.Helper()

	balance, err := NewBalance(value)
	if err != nil {
		t.Fatalf("NewBalance(%d) returned error: %v", value, err)
	}

	return balance
}
