package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

type TransactionType string

const (
	TypeDebit  TransactionType = "DEBIT"
	TypeCredit TransactionType = "CREDIT"
)

// SystemTransaction is a transaction record from the internal system CSV.
type SystemTransaction struct {
	TrxID           string
	Amount          decimal.Decimal // always positive
	Type            TransactionType
	TransactionTime time.Time
	SourceFile      string
	LineNumber      int
}

// BankTransaction is a transaction record from a bank statement CSV.
type BankTransaction struct {
	UniqueIdentifier string
	Amount           decimal.Decimal // may be negative (debit = money out)
	Date             time.Time       // date only, no time component
	BankName         string
	SourceFile       string
	LineNumber       int
	RawAmount        string
}

// NormalizedAmount returns the absolute value of the bank transaction amount.
func (bt *BankTransaction) NormalizedAmount() decimal.Decimal {
	return bt.Amount.Abs()
}

// InferredType infers DEBIT or CREDIT from the sign of Amount.
// Negative amount = money leaving the account = DEBIT.
func (bt *BankTransaction) InferredType() TransactionType {
	if bt.Amount.IsNegative() {
		return TypeDebit
	}
	return TypeCredit
}
