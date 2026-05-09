package reconciler

import (
	"testing"
	"time"

	"github.com/ridloal/reconciliation-service/internal/config"
	"github.com/ridloal/reconciliation-service/internal/domain"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func makeTime(dateStr string) time.Time {
	t, _ := time.Parse("2006-01-02", dateStr)
	return t
}

func TestFilterSystemByDate_IncludesBoundaries(t *testing.T) {
	trxs := []domain.SystemTransaction{
		{TrxID: "before", TransactionTime: makeTime("2024-01-14")},
		{TrxID: "start", TransactionTime: makeTime("2024-01-15")},
		{TrxID: "middle", TransactionTime: makeTime("2024-01-16")},
		{TrxID: "end", TransactionTime: makeTime("2024-01-17")},
		{TrxID: "after", TransactionTime: makeTime("2024-01-18")},
	}

	result := filterSystemByDate(trxs, makeTime("2024-01-15"), makeTime("2024-01-17"))
	assert.Len(t, result, 3)
	assert.Equal(t, "start", result[0].TrxID)
	assert.Equal(t, "end", result[2].TrxID)
}

func TestFilterBankByDate_IncludesBoundaries(t *testing.T) {
	trxs := []domain.BankTransaction{
		{UniqueIdentifier: "before", Date: makeTime("2024-01-14")},
		{UniqueIdentifier: "start", Date: makeTime("2024-01-15")},
		{UniqueIdentifier: "end", Date: makeTime("2024-01-17")},
		{UniqueIdentifier: "after", Date: makeTime("2024-01-18")},
	}

	result := filterBankByDate(trxs, makeTime("2024-01-15"), makeTime("2024-01-17"))
	assert.Len(t, result, 2)
}

func TestFilterSystemByDate_EmptyInput(t *testing.T) {
	result := filterSystemByDate(nil, makeTime("2024-01-01"), makeTime("2024-01-31"))
	assert.Empty(t, result)
}

func TestAggregate_Counts(t *testing.T) {
	matched := []domain.MatchedPair{
		{
			SystemTrx:         domain.SystemTransaction{Amount: decimal.NewFromFloat(100)},
			BankTrx:           domain.BankTransaction{Amount: decimal.NewFromFloat(-100)},
			AmountDiscrepancy: decimal.Zero,
			IsExactMatch:      true,
		},
	}
	unmatchedSys := []domain.UnmatchedSystemTrx{
		{Transaction: domain.SystemTransaction{Amount: decimal.NewFromFloat(200)}},
	}
	unmatchedBank := []domain.UnmatchedBankTrx{
		{Transaction: domain.BankTransaction{Amount: decimal.NewFromFloat(-300)}},
	}

	sysTrxs := []domain.SystemTransaction{{}, {}}
	bankTrxs := []domain.BankTransaction{{}}

	cfg := config.ReconciliationConfig{
		StartDate: makeTime("2024-01-01"),
		EndDate:   makeTime("2024-01-31"),
	}

	result := aggregate(cfg, sysTrxs, bankTrxs, matched, unmatchedSys, unmatchedBank)

	assert.Equal(t, 2, result.TotalSystemTransactions)
	assert.Equal(t, 1, result.TotalBankTransactions)
	assert.Equal(t, 1, result.TotalMatched)
	assert.Equal(t, 2, result.TotalUnmatched)
	assert.True(t, result.TotalDiscrepancyAmount.IsZero())
	assert.True(t, result.TotalMissingFromBank.Equal(decimal.NewFromFloat(200)))
	assert.True(t, result.TotalMissingFromSystem.Equal(decimal.NewFromFloat(300)))
}

func TestAggregate_WithDiscrepancy(t *testing.T) {
	matched := []domain.MatchedPair{
		{
			SystemTrx:         domain.SystemTransaction{Amount: decimal.NewFromFloat(500)},
			BankTrx:           domain.BankTransaction{Amount: decimal.NewFromFloat(-499)},
			AmountDiscrepancy: decimal.NewFromFloat(1),
			IsExactMatch:      false,
		},
		{
			SystemTrx:         domain.SystemTransaction{Amount: decimal.NewFromFloat(300)},
			BankTrx:           domain.BankTransaction{Amount: decimal.NewFromFloat(-298)},
			AmountDiscrepancy: decimal.NewFromFloat(2),
			IsExactMatch:      false,
		},
	}

	cfg := config.ReconciliationConfig{
		StartDate: makeTime("2024-01-01"),
		EndDate:   makeTime("2024-01-31"),
	}

	result := aggregate(cfg, nil, nil, matched, nil, nil)
	assert.True(t, result.TotalDiscrepancyAmount.Equal(decimal.NewFromFloat(3)))
}
