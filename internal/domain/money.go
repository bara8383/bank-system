package domain

import (
	"errors"
	"math"
)

const CurrencyJPY = "JPY"

var (
	ErrAmountMustBePositive     = errors.New("amount must be greater than zero")
	ErrBalanceMustBeNonNegative = errors.New("balance must be greater than or equal to zero")
	ErrInsufficientBalance      = errors.New("insufficient balance")
	ErrBalanceOverflow          = errors.New("balance overflow")
)

// Amount represents a transaction amount in the smallest currency unit.
// MVP scope fixes the currency to JPY, so 1 means 1 yen.
type Amount struct {
	value int64
}

// NewAmount validates and creates a positive transaction amount.
func NewAmount(value int64) (Amount, error) {
	if value <= 0 {
		return Amount{}, ErrAmountMustBePositive
	}

	return Amount{value: value}, nil
}

// Int64 returns the amount in the smallest currency unit.
func (a Amount) Int64() int64 {
	return a.value
}

// Balance represents an account balance in the smallest currency unit.
// MVP scope fixes the currency to JPY, so 1 means 1 yen.
type Balance struct {
	value int64
}

// NewBalance validates and creates a non-negative balance.
func NewBalance(value int64) (Balance, error) {
	if value < 0 {
		return Balance{}, ErrBalanceMustBeNonNegative
	}

	return Balance{value: value}, nil
}

// Int64 returns the balance in the smallest currency unit.
func (b Balance) Int64() int64 {
	return b.value
}

// AddBalance returns the balance after applying a deposit-like increase.
func AddBalance(balance Balance, amount Amount) (Balance, error) {
	if amount.value <= 0 {
		return balance, ErrAmountMustBePositive
	}

	if balance.value > math.MaxInt64-amount.value {
		return balance, ErrBalanceOverflow
	}

	return Balance{value: balance.value + amount.value}, nil
}

// SubtractBalance returns the balance after applying a withdrawal-like decrease.
// When the amount exceeds the current balance, the original balance is returned
// with ErrInsufficientBalance so callers can keep state unchanged.
func SubtractBalance(balance Balance, amount Amount) (Balance, error) {
	if amount.value <= 0 {
		return balance, ErrAmountMustBePositive
	}

	if amount.value > balance.value {
		return balance, ErrInsufficientBalance
	}

	return Balance{value: balance.value - amount.value}, nil
}
