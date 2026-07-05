package domain

import "errors"

const (
	AccountStatusActive    AccountStatus = "active"
	AccountStatusSuspended AccountStatus = "suspended"
	AccountStatusClosed    AccountStatus = "closed"
)

var (
	ErrInvalidAccountStatus = errors.New("invalid account status")
	ErrAccountNotActive     = errors.New("account is not active")
)

// AccountStatus represents the account lifecycle status used by the MVP
// domain layer. It maps docs terms as active=有効, suspended=停止中,
// and closed=解約済み.
type AccountStatus string

// Validate confirms the account status is one of the MVP-supported values.
func (s AccountStatus) Validate() error {
	switch s {
	case AccountStatusActive, AccountStatusSuspended, AccountStatusClosed:
		return nil
	default:
		return ErrInvalidAccountStatus
	}
}

// EnsureAccountCanTransact confirms an account may proceed to balance-changing
// operations such as deposit, withdrawal, or transfer.
func EnsureAccountCanTransact(status AccountStatus) error {
	if err := status.Validate(); err != nil {
		return err
	}

	if status != AccountStatusActive {
		return ErrAccountNotActive
	}

	return nil
}
