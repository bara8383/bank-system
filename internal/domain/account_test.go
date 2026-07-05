package domain

import (
	"errors"
	"testing"
)

func TestAccountStatusValidateAcceptsSupportedStatuses(t *testing.T) {
	testCases := map[string]AccountStatus{
		"active":    AccountStatusActive,
		"suspended": AccountStatusSuspended,
		"closed":    AccountStatusClosed,
	}

	for name, status := range testCases {
		t.Run(name, func(t *testing.T) {
			if err := status.Validate(); err != nil {
				t.Fatalf("expected status %q to be valid, got error: %v", status, err)
			}
		})
	}
}

func TestAccountStatusValidateRejectsEmptyAndUnknownStatuses(t *testing.T) {
	testCases := map[string]AccountStatus{
		"empty":   "",
		"unknown": "pending",
	}

	for name, status := range testCases {
		t.Run(name, func(t *testing.T) {
			if err := status.Validate(); !errors.Is(err, ErrInvalidAccountStatus) {
				t.Fatalf("expected ErrInvalidAccountStatus for %q, got %v", status, err)
			}
		})
	}
}

func TestEnsureAccountCanTransactAcceptsActiveStatus(t *testing.T) {
	if err := EnsureAccountCanTransact(AccountStatusActive); err != nil {
		t.Fatalf("expected active account to transact, got error: %v", err)
	}
}

func TestEnsureAccountCanTransactRejectsSuspendedAndClosedStatuses(t *testing.T) {
	testCases := map[string]AccountStatus{
		"suspended": AccountStatusSuspended,
		"closed":    AccountStatusClosed,
	}

	for name, status := range testCases {
		t.Run(name, func(t *testing.T) {
			if err := EnsureAccountCanTransact(status); !errors.Is(err, ErrAccountNotActive) {
				t.Fatalf("expected ErrAccountNotActive for %q, got %v", status, err)
			}
		})
	}
}

func TestEnsureAccountCanTransactRejectsUnknownStatusAsInvalidStatus(t *testing.T) {
	err := EnsureAccountCanTransact("pending")
	if !errors.Is(err, ErrInvalidAccountStatus) {
		t.Fatalf("expected ErrInvalidAccountStatus for unknown status, got %v", err)
	}
	if errors.Is(err, ErrAccountNotActive) {
		t.Fatalf("expected unknown status not to be treated as ErrAccountNotActive, got %v", err)
	}
}
