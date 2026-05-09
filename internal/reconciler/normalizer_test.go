package reconciler

import (
	"testing"
	"time"

	"github.com/ridloal/reconciliation-service/internal/domain"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestExtractDateKey(t *testing.T) {
	cases := []struct {
		input    time.Time
		expected string
	}{
		{time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC), "2024-01-15"},
		{time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), "2024-01-15"},
		{time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC), "2024-12-31"},
	}
	for _, c := range cases {
		assert.Equal(t, c.expected, ExtractDateKey(c.input))
	}
}

func TestNormalizeSystemTrx(t *testing.T) {
	trx := domain.SystemTransaction{
		Amount:          decimal.NewFromFloat(500000),
		Type:            domain.TypeDebit,
		TransactionTime: time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC),
	}
	date, amount, txType := NormalizeSystemTrx(trx)
	assert.Equal(t, "2024-01-15", date)
	assert.True(t, amount.Equal(decimal.NewFromFloat(500000)))
	assert.Equal(t, domain.TypeDebit, txType)
}

func TestNormalizeBankTrx_Negative(t *testing.T) {
	trx := domain.BankTransaction{
		Amount: decimal.NewFromFloat(-500000),
		Date:   time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
	}
	date, amount, txType := NormalizeBankTrx(trx)
	assert.Equal(t, "2024-01-15", date)
	assert.True(t, amount.Equal(decimal.NewFromFloat(500000)), "should be absolute value")
	assert.Equal(t, domain.TypeDebit, txType)
}

func TestNormalizeBankTrx_Positive(t *testing.T) {
	trx := domain.BankTransaction{
		Amount: decimal.NewFromFloat(750000),
		Date:   time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC),
	}
	date, amount, txType := NormalizeBankTrx(trx)
	assert.Equal(t, "2024-01-16", date)
	assert.True(t, amount.Equal(decimal.NewFromFloat(750000)))
	assert.Equal(t, domain.TypeCredit, txType)
}
