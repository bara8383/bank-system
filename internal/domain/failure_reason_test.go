package domain

import (
	"errors"
	"fmt"
	"testing"
)

func TestFailureReasonValidateAcceptsSupportedReasons(t *testing.T) {
	testCases := map[string]FailureReason{
		"invalid amount":           FailureReasonInvalidAmount,
		"invalid balance state":    FailureReasonInvalidBalanceState,
		"insufficient balance":     FailureReasonInsufficientBalance,
		"balance overflow":         FailureReasonBalanceOverflow,
		"invalid account status":   FailureReasonInvalidAccountStatus,
		"account not active":       FailureReasonAccountNotActive,
		"invalid transaction type": FailureReasonInvalidTransactionType,
		"internal error":           FailureReasonInternalError,
	}

	for name, reason := range testCases {
		t.Run(name, func(t *testing.T) {
			if err := reason.Validate(); err != nil {
				t.Fatalf("expected failure reason %q to be valid, got error: %v", reason, err)
			}
		})
	}
}

func TestFailureReasonValidateRejectsUnsafeAndUnknownReasons(t *testing.T) {
	testCases := map[string]FailureReason{
		"empty":                          "",
		"unknown":                        "database_timeout",
		"raw amount error":               FailureReason(ErrAmountMustBePositive.Error()),
		"raw balance state error":        FailureReason(ErrBalanceMustBeNonNegative.Error()),
		"raw insufficient balance error": FailureReason(ErrInsufficientBalance.Error()),
		"secret-like value":              "password=super-secret",
	}

	for name, reason := range testCases {
		t.Run(name, func(t *testing.T) {
			if err := reason.Validate(); !errors.Is(err, ErrInvalidFailureReason) {
				t.Fatalf("expected ErrInvalidFailureReason for %q, got %v", reason, err)
			}
		})
	}
}

func TestFailureReasonFromErrorMapsKnownDomainErrors(t *testing.T) {
	testCases := map[string]struct {
		err  error
		want FailureReason
	}{
		"amount must be positive": {
			err:  ErrAmountMustBePositive,
			want: FailureReasonInvalidAmount,
		},
		"balance must be non-negative": {
			err:  ErrBalanceMustBeNonNegative,
			want: FailureReasonInvalidBalanceState,
		},
		"insufficient balance": {
			err:  ErrInsufficientBalance,
			want: FailureReasonInsufficientBalance,
		},
		"balance overflow": {
			err:  ErrBalanceOverflow,
			want: FailureReasonBalanceOverflow,
		},
		"invalid account status": {
			err:  ErrInvalidAccountStatus,
			want: FailureReasonInvalidAccountStatus,
		},
		"account not active": {
			err:  ErrAccountNotActive,
			want: FailureReasonAccountNotActive,
		},
		"invalid transaction type": {
			err:  ErrInvalidTransactionType,
			want: FailureReasonInvalidTransactionType,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, ok := FailureReasonFromError(tc.err)
			if !ok {
				t.Fatalf("expected %v to be mapped", tc.err)
			}
			if got != tc.want {
				t.Fatalf("expected failure reason %q, got %q", tc.want, got)
			}
			if err := got.Validate(); err != nil {
				t.Fatalf("expected mapped failure reason %q to be valid, got error: %v", got, err)
			}
		})
	}
}

func TestFailureReasonFromErrorMapsWrappedDomainErrors(t *testing.T) {
	wrapped := fmt.Errorf("domain validation failed: %w", ErrInvalidTransactionType)

	got, ok := FailureReasonFromError(wrapped)
	if !ok {
		t.Fatalf("expected wrapped domain error to be mapped")
	}
	if got != FailureReasonInvalidTransactionType {
		t.Fatalf("expected failure reason %q, got %q", FailureReasonInvalidTransactionType, got)
	}
}

func TestFailureReasonFromErrorLeavesNilAndUnknownErrorsUnclassified(t *testing.T) {
	testCases := map[string]error{
		"nil":     nil,
		"unknown": errors.New("database connection refused"),
	}

	for name, err := range testCases {
		t.Run(name, func(t *testing.T) {
			got, ok := FailureReasonFromError(err)
			if ok {
				t.Fatalf("expected %v to be unclassified", err)
			}
			if got != "" {
				t.Fatalf("expected empty failure reason, got %q", got)
			}
		})
	}
}

func TestSafeFailureReasonFromErrorMapsKnownDomainErrors(t *testing.T) {
	testCases := map[string]struct {
		err  error
		want FailureReason
	}{
		"known domain error": {
			err:  ErrInsufficientBalance,
			want: FailureReasonInsufficientBalance,
		},
		"wrapped domain error": {
			err:  fmt.Errorf("domain validation failed: %w", ErrInvalidTransactionType),
			want: FailureReasonInvalidTransactionType,
		},
		"joined known and unknown error": {
			err:  errors.Join(ErrAccountNotActive, errors.New("database connection refused")),
			want: FailureReasonAccountNotActive,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, ok := SafeFailureReasonFromError(tc.err)
			if !ok {
				t.Fatalf("expected %v to be mapped", tc.err)
			}
			if got != tc.want {
				t.Fatalf("expected failure reason %q, got %q", tc.want, got)
			}
			if err := got.Validate(); err != nil {
				t.Fatalf("expected mapped failure reason %q to be valid, got error: %v", got, err)
			}
		})
	}
}

func TestSafeFailureReasonFromErrorLeavesNilUnclassified(t *testing.T) {
	got, ok := SafeFailureReasonFromError(nil)
	if ok {
		t.Fatal("expected nil error to be unclassified")
	}
	if got != "" {
		t.Fatalf("expected empty failure reason, got %q", got)
	}
}

func TestSafeFailureReasonFromErrorMapsUnknownErrorsToInternalError(t *testing.T) {
	unknownErr := errors.New("database password=super-secret connection refused")

	got, ok := SafeFailureReasonFromError(unknownErr)
	if !ok {
		t.Fatal("expected unknown non-nil error to be classified")
	}
	if got != FailureReasonInternalError {
		t.Fatalf("expected failure reason %q, got %q", FailureReasonInternalError, got)
	}
	if got == FailureReason(unknownErr.Error()) {
		t.Fatal("expected raw error message not to be returned as failure reason")
	}
	if err := got.Validate(); err != nil {
		t.Fatalf("expected internal error failure reason to be valid, got error: %v", err)
	}
}
