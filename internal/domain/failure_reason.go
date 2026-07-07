package domain

import "errors"

const (
	FailureReasonInvalidAmount          FailureReason = "invalid_amount"
	FailureReasonInvalidBalanceState    FailureReason = "invalid_balance_state"
	FailureReasonInsufficientBalance    FailureReason = "insufficient_balance"
	FailureReasonBalanceOverflow        FailureReason = "balance_overflow"
	FailureReasonInvalidAccountStatus   FailureReason = "invalid_account_status"
	FailureReasonAccountNotActive       FailureReason = "account_not_active"
	FailureReasonInvalidTransactionType FailureReason = "invalid_transaction_type"
)

var ErrInvalidFailureReason = errors.New("invalid failure reason")

// FailureReason is a stable, safe category for domain failures that may be
// reused by API responses, audit failure_reason fields, and structured logs.
// It must not contain raw request bodies, secrets, tokens, session IDs, or
// unvalidated free-form input.
type FailureReason string

// Validate confirms the failure reason is one of the MVP-supported safe
// categories. Raw error messages and free-form values are intentionally
// rejected so callers do not persist or expose sensitive details by accident.
func (r FailureReason) Validate() error {
	switch r {
	case FailureReasonInvalidAmount,
		FailureReasonInvalidBalanceState,
		FailureReasonInsufficientBalance,
		FailureReasonBalanceOverflow,
		FailureReasonInvalidAccountStatus,
		FailureReasonAccountNotActive,
		FailureReasonInvalidTransactionType:
		return nil
	default:
		return ErrInvalidFailureReason
	}
}

// FailureReasonFromError maps known domain sentinel errors to stable, safe
// failure categories. Unknown errors are left unclassified instead of exposing
// err.Error() as a category.
func FailureReasonFromError(err error) (FailureReason, bool) {
	switch {
	case errors.Is(err, ErrAmountMustBePositive):
		return FailureReasonInvalidAmount, true
	case errors.Is(err, ErrBalanceMustBeNonNegative):
		return FailureReasonInvalidBalanceState, true
	case errors.Is(err, ErrInsufficientBalance):
		return FailureReasonInsufficientBalance, true
	case errors.Is(err, ErrBalanceOverflow):
		return FailureReasonBalanceOverflow, true
	case errors.Is(err, ErrInvalidAccountStatus):
		return FailureReasonInvalidAccountStatus, true
	case errors.Is(err, ErrAccountNotActive):
		return FailureReasonAccountNotActive, true
	case errors.Is(err, ErrInvalidTransactionType):
		return FailureReasonInvalidTransactionType, true
	default:
		return "", false
	}
}
