package reporter

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/ridloal/reconciliation-service/internal/domain"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sampleResult() *domain.ReconciliationResult {
	d := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	return &domain.ReconciliationResult{
		StartDate:               time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:                 time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
		ProcessedAt:             time.Date(2024, 2, 1, 10, 0, 0, 0, time.UTC),
		SystemFiles:             []string{"system_transactions.csv"},
		BankFiles:               []string{"bank_bca.csv"},
		TotalSystemTransactions: 2,
		TotalBankTransactions:   2,
		TotalProcessed:          4,
		TotalMatched:            1,
		TotalUnmatched:          2,
		MatchedPairs: []domain.MatchedPair{
			{
				SystemTrx: domain.SystemTransaction{TrxID: "TRX-001", Amount: decimal.NewFromFloat(500000), Type: domain.TypeDebit, TransactionTime: d},
				BankTrx:   domain.BankTransaction{UniqueIdentifier: "BCA-001", BankName: "BCA", Amount: decimal.NewFromFloat(-500000), Date: d},
				AmountDiscrepancy: decimal.Zero,
				IsExactMatch:      true,
			},
		},
		UnmatchedSystem: []domain.UnmatchedSystemTrx{
			{Transaction: domain.SystemTransaction{TrxID: "TRX-002", Amount: decimal.NewFromFloat(375000), Type: domain.TypeCredit, TransactionTime: d}, Reason: "no_bank_match"},
		},
		UnmatchedBank: []domain.UnmatchedBankTrx{
			{Transaction: domain.BankTransaction{UniqueIdentifier: "BCA-002", BankName: "BCA", Amount: decimal.NewFromFloat(-100000), Date: d}, Reason: "no_system_match"},
		},
		TotalDiscrepancyAmount: decimal.Zero,
		TotalMissingFromBank:   decimal.NewFromFloat(375000),
		TotalMissingFromSystem: decimal.NewFromFloat(100000),
	}
}

func TestGenerateTextReport_ContainsKeyFields(t *testing.T) {
	var buf bytes.Buffer
	GenerateTextReport(sampleResult(), &buf)
	out := buf.String()

	assert.Contains(t, out, "RECONCILIATION SUMMARY REPORT")
	assert.Contains(t, out, "2024-01-01")
	assert.Contains(t, out, "2024-01-31")
	assert.Contains(t, out, "TRX-001")
	assert.Contains(t, out, "BCA-001")
	assert.Contains(t, out, "TRX-002")
	assert.Contains(t, out, "BCA-002")
	assert.Contains(t, out, "no_bank_match")
	assert.Contains(t, out, "no_system_match")
	assert.Contains(t, out, "END OF REPORT")
}

func TestGenerateTextReport_Counts(t *testing.T) {
	var buf bytes.Buffer
	GenerateTextReport(sampleResult(), &buf)
	out := buf.String()
	assert.Contains(t, out, "Total Matched")
	assert.Contains(t, out, "Total Unmatched")
}

func TestGenerateJSONReport_ValidJSON(t *testing.T) {
	var buf bytes.Buffer
	err := GenerateJSONReport(sampleResult(), &buf)
	require.NoError(t, err)

	var parsed JSONReport
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))

	assert.Equal(t, "2024-01-01", parsed.Period.StartDate)
	assert.Equal(t, "2024-01-31", parsed.Period.EndDate)
	assert.Equal(t, 1, parsed.Summary.TotalMatched)
	assert.Equal(t, 2, parsed.Summary.TotalUnmatched)
	require.Len(t, parsed.Matched, 1)
	assert.Equal(t, "TRX-001", parsed.Matched[0].SystemTrxID)
	require.Len(t, parsed.UnmatchedSystem, 1)
	require.Len(t, parsed.UnmatchedBank, 1)
}

func TestFormatAmount_Indonesian(t *testing.T) {
	cases := []struct {
		input    float64
		expected string
	}{
		{1500000.50, "Rp 1.500.000,50"},
		{500000.00, "Rp 500.000,00"},
		{0.00, "Rp 0,00"},
		{1000000000.00, "Rp 1.000.000.000,00"},
	}
	for _, c := range cases {
		result := FormatAmount(decimal.NewFromFloat(c.input))
		assert.Equal(t, c.expected, result, "input: %v", c.input)
	}
}

func TestFormatDate(t *testing.T) {
	t1 := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)
	assert.Equal(t, "2024-01-15", FormatDate(t1))
}

func TestGenerateJSONReport_EmptyResult(t *testing.T) {
	r := &domain.ReconciliationResult{
		StartDate:              time.Now(),
		EndDate:                time.Now(),
		ProcessedAt:            time.Now(),
		TotalDiscrepancyAmount: decimal.Zero,
		TotalMissingFromBank:   decimal.Zero,
		TotalMissingFromSystem: decimal.Zero,
	}
	var buf bytes.Buffer
	err := GenerateJSONReport(r, &buf)
	require.NoError(t, err)
	assert.True(t, strings.Contains(buf.String(), `"matched":null`) ||
		strings.Contains(buf.String(), `"matched":[]`) ||
		!strings.Contains(buf.String(), `"matched":[{`))
}
