package domain

import (
	"errors"
	"math"
	"testing"
)

func TestTransactionTypeValidateAcceptsMVPTypes(t *testing.T) {
	testCases := map[string]TransactionType{
		"deposit":         TransactionTypeDeposit,
		"withdrawal":      TransactionTypeWithdrawal,
		"transfer debit":  TransactionTypeTransferDebit,
		"transfer credit": TransactionTypeTransferCredit,
	}

	for name, transactionType := range testCases {
		t.Run(name, func(t *testing.T) {
			if err := transactionType.Validate(); err != nil {
				t.Fatalf("expected transaction type %q to be valid, got error: %v", transactionType, err)
			}
		})
	}
}

func TestTransactionTypeValidateRejectsEmptyUnknownAndReversal(t *testing.T) {
	testCases := map[string]TransactionType{
		"empty":    "",
		"unknown":  "payment",
		"reversal": "reversal",
	}

	for name, transactionType := range testCases {
		t.Run(name, func(t *testing.T) {
			if err := transactionType.Validate(); !errors.Is(err, ErrInvalidTransactionType) {
				t.Fatalf("expected ErrInvalidTransactionType for %q, got %v", transactionType, err)
			}
		})
	}
}

func TestApplyTransactionIncreasesBalanceForDepositLikeTypes(t *testing.T) {
	testCases := map[string]TransactionType{
		"deposit":         TransactionTypeDeposit,
		"transfer credit": TransactionTypeTransferCredit,
	}

	for name, transactionType := range testCases {
		t.Run(name, func(t *testing.T) {
			balance := mustBalance(t, 100)
			amount := mustAmount(t, 25)

			updated, err := ApplyTransaction(balance, amount, transactionType)
			if err != nil {
				t.Fatalf("expected %q transaction to succeed, got error: %v", transactionType, err)
			}

			if updated.Int64() != 125 {
				t.Fatalf("expected updated balance 125, got %d", updated.Int64())
			}
		})
	}
}

func TestApplyTransactionDecreasesBalanceForWithdrawalLikeTypes(t *testing.T) {
	testCases := map[string]TransactionType{
		"withdrawal":     TransactionTypeWithdrawal,
		"transfer debit": TransactionTypeTransferDebit,
	}

	for name, transactionType := range testCases {
		t.Run(name, func(t *testing.T) {
			balance := mustBalance(t, 100)
			amount := mustAmount(t, 40)

			updated, err := ApplyTransaction(balance, amount, transactionType)
			if err != nil {
				t.Fatalf("expected %q transaction to succeed, got error: %v", transactionType, err)
			}

			if updated.Int64() != 60 {
				t.Fatalf("expected updated balance 60, got %d", updated.Int64())
			}
		})
	}
}

func TestApplyTransactionRejectsInsufficientBalanceAndReturnsOriginalBalance(t *testing.T) {
	testCases := map[string]TransactionType{
		"withdrawal":     TransactionTypeWithdrawal,
		"transfer debit": TransactionTypeTransferDebit,
	}

	for name, transactionType := range testCases {
		t.Run(name, func(t *testing.T) {
			balance := mustBalance(t, 30)
			amount := mustAmount(t, 40)

			updated, err := ApplyTransaction(balance, amount, transactionType)
			if !errors.Is(err, ErrInsufficientBalance) {
				t.Fatalf("expected ErrInsufficientBalance, got %v", err)
			}

			if updated.Int64() != balance.Int64() {
				t.Fatalf("expected original balance %d, got %d", balance.Int64(), updated.Int64())
			}
		})
	}
}

func TestApplyTransactionRejectsInvalidStartingBalanceAndReturnsOriginalBalance(t *testing.T) {
	balance := Balance{value: -1}
	amount := mustAmount(t, 1)

	updated, err := ApplyTransaction(balance, amount, TransactionTypeDeposit)
	if !errors.Is(err, ErrBalanceMustBeNonNegative) {
		t.Fatalf("expected ErrBalanceMustBeNonNegative, got %v", err)
	}

	if updated.Int64() != balance.Int64() {
		t.Fatalf("expected original balance %d, got %d", balance.Int64(), updated.Int64())
	}
}

func TestApplyTransactionRejectsInvalidAmountAndReturnsOriginalBalance(t *testing.T) {
	balance := mustBalance(t, 100)

	updated, err := ApplyTransaction(balance, Amount{}, TransactionTypeDeposit)
	if !errors.Is(err, ErrAmountMustBePositive) {
		t.Fatalf("expected ErrAmountMustBePositive, got %v", err)
	}

	if updated.Int64() != balance.Int64() {
		t.Fatalf("expected original balance %d, got %d", balance.Int64(), updated.Int64())
	}
}

func TestApplyTransactionRejectsInvalidTransactionTypeAndReturnsOriginalBalance(t *testing.T) {
	balance := mustBalance(t, 100)
	amount := mustAmount(t, 1)

	updated, err := ApplyTransaction(balance, amount, "reversal")
	if !errors.Is(err, ErrInvalidTransactionType) {
		t.Fatalf("expected ErrInvalidTransactionType, got %v", err)
	}

	if updated.Int64() != balance.Int64() {
		t.Fatalf("expected original balance %d, got %d", balance.Int64(), updated.Int64())
	}
}

func TestApplyTransactionPrioritizesBalanceThenAmountThenTransactionTypeValidation(t *testing.T) {
	balance := Balance{value: -1}

	updated, err := ApplyTransaction(balance, Amount{}, "reversal")
	if !errors.Is(err, ErrBalanceMustBeNonNegative) {
		t.Fatalf("expected ErrBalanceMustBeNonNegative, got %v", err)
	}

	if updated.Int64() != balance.Int64() {
		t.Fatalf("expected original balance %d, got %d", balance.Int64(), updated.Int64())
	}

	validBalance := mustBalance(t, 100)
	updated, err = ApplyTransaction(validBalance, Amount{}, "reversal")
	if !errors.Is(err, ErrAmountMustBePositive) {
		t.Fatalf("expected ErrAmountMustBePositive, got %v", err)
	}

	if updated.Int64() != validBalance.Int64() {
		t.Fatalf("expected original balance %d, got %d", validBalance.Int64(), updated.Int64())
	}
}

func TestApplyTransactionReturnsOriginalBalanceOnDepositLikeOverflow(t *testing.T) {
	testCases := map[string]TransactionType{
		"deposit":         TransactionTypeDeposit,
		"transfer credit": TransactionTypeTransferCredit,
	}

	for name, transactionType := range testCases {
		t.Run(name, func(t *testing.T) {
			balance := mustBalance(t, math.MaxInt64)
			amount := mustAmount(t, 1)

			updated, err := ApplyTransaction(balance, amount, transactionType)
			if !errors.Is(err, ErrBalanceOverflow) {
				t.Fatalf("expected ErrBalanceOverflow, got %v", err)
			}

			if updated.Int64() != balance.Int64() {
				t.Fatalf("expected original balance %d, got %d", balance.Int64(), updated.Int64())
			}
		})
	}
}
