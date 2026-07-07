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
	FailureReasonInternalError          FailureReason = "internal_error"
)

var ErrInvalidFailureReason = errors.New("invalid failure reason")

// FailureReason is a stable, safe category for domain and audit failures that
// may be reused by audit failure_reason fields and safe structured logs. Public
// API response codes and HTTP status mapping are intentionally not finalized by
// this domain helper. FailureReason must not contain raw request bodies,
// secrets, tokens, session IDs, or unvalidated free-form input.
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
		FailureReasonInvalidTransactionType,
		FailureReasonInternalError:
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

// SafeFailureReasonFromError maps known domain sentinel errors to stable, safe
// failure categories and falls back to internal_error for unknown non-nil
// errors. This helper is intended for audit failure_reason fields and safe
// structured logs so callers do not persist or expose err.Error(), raw request
// bodies, secrets, or other sensitive details. It is not the final public API
// response body or HTTP status code contract.
func SafeFailureReasonFromError(err error) (FailureReason, bool) {
	if reason, ok := FailureReasonFromError(err); ok {
		return reason, true
	}

	if err == nil {
		return "", false
	}

	return FailureReasonInternalError, true
}
