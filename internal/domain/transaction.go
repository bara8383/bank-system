package domain

import "errors"

const (
	TransactionTypeDeposit        TransactionType = "deposit"
	TransactionTypeWithdrawal     TransactionType = "withdrawal"
	TransactionTypeTransferDebit  TransactionType = "transfer_debit"
	TransactionTypeTransferCredit TransactionType = "transfer_credit"
)

var ErrInvalidTransactionType = errors.New("invalid transaction type")

// TransactionType represents the MVP-supported transaction type values used to
// decide how a transaction amount affects an account balance.
type TransactionType string

// Validate confirms the transaction type is one of the MVP-supported values.
func (t TransactionType) Validate() error {
	switch t {
	case TransactionTypeDeposit, TransactionTypeWithdrawal, TransactionTypeTransferDebit, TransactionTypeTransferCredit:
		return nil
	default:
		return ErrInvalidTransactionType
	}
}

// ApplyTransaction returns the balance after applying amount according to the
// transaction type. The returned balance represents the candidate balance_after
// value for a future persisted transaction row; this helper does not create or
// persist transaction history.
func ApplyTransaction(balance Balance, amount Amount, transactionType TransactionType) (Balance, error) {
	if err := balance.Validate(); err != nil {
		return balance, err
	}

	if err := amount.Validate(); err != nil {
		return balance, err
	}

	if err := transactionType.Validate(); err != nil {
		return balance, err
	}

	switch transactionType {
	case TransactionTypeDeposit, TransactionTypeTransferCredit:
		return AddBalance(balance, amount)
	case TransactionTypeWithdrawal, TransactionTypeTransferDebit:
		return SubtractBalance(balance, amount)
	default:
		return balance, ErrInvalidTransactionType
	}
}
