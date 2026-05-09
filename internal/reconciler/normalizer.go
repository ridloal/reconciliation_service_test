package reconciler

import (
	"time"

	"github.com/ridloal/reconciliation-service/internal/domain"
	"github.com/shopspring/decimal"
)

// ExtractDateKey returns the date portion of a time.Time as "YYYY-MM-DD".
func ExtractDateKey(t time.Time) string {
	return t.UTC().Format("2006-01-02")
}

// NormalizeSystemTrx converts a system transaction datetime to a date key.
func NormalizeSystemTrx(trx domain.SystemTransaction) (dateKey string, amount decimal.Decimal, txType domain.TransactionType) {
	return ExtractDateKey(trx.TransactionTime), trx.Amount, trx.Type
}

// NormalizeBankTrx converts a bank transaction to normalized components.
// Amount is returned as absolute value; type is inferred from sign.
func NormalizeBankTrx(trx domain.BankTransaction) (dateKey string, amount decimal.Decimal, txType domain.TransactionType) {
	return ExtractDateKey(trx.Date), trx.NormalizedAmount(), trx.InferredType()
}
