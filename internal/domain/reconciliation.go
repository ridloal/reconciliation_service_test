package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// MatchedPair holds a system transaction paired with its bank counterpart.
type MatchedPair struct {
	SystemTrx         SystemTransaction
	BankTrx           BankTransaction
	AmountDiscrepancy decimal.Decimal
	IsExactMatch      bool
}

// UnmatchedSystemTrx is a system transaction with no corresponding bank entry.
type UnmatchedSystemTrx struct {
	Transaction SystemTransaction
	Reason      string
}

// UnmatchedBankTrx is a bank transaction with no corresponding system entry.
type UnmatchedBankTrx struct {
	Transaction BankTransaction
	Reason      string
}

// ReconciliationResult is the complete output of a reconciliation run.
type ReconciliationResult struct {
	StartDate   time.Time
	EndDate     time.Time
	ProcessedAt time.Time
	SystemFiles []string
	BankFiles   []string

	TotalSystemTransactions int
	TotalBankTransactions   int
	TotalProcessed          int
	TotalMatched            int
	TotalUnmatched          int

	MatchedPairs    []MatchedPair
	UnmatchedSystem []UnmatchedSystemTrx
	UnmatchedBank   []UnmatchedBankTrx

	TotalDiscrepancyAmount decimal.Decimal
	TotalMissingFromBank   decimal.Decimal
	TotalMissingFromSystem decimal.Decimal
}
