package parser

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTempBankCSV(t *testing.T, content string) string {
	t.Helper()
	return writeTempCSV(t, content)
}

func TestParseBankCSV_ValidData(t *testing.T) {
	csv := `unique_identifier,amount,date
BCA-001,-500000.00,2024-01-15
BCA-002,1250000.50,2024-01-15`
	path := writeTempBankCSV(t, csv)
	trxs, errs := ParseBankCSV(path, "BCA")
	assert.Empty(t, errs)
	require.Len(t, trxs, 2)
	assert.True(t, trxs[0].Amount.Equal(decimal.NewFromFloat(-500000.00)))
	assert.Equal(t, "BCA", trxs[0].BankName)
}

func TestParseBankCSV_NormalizedAmount(t *testing.T) {
	csv := `unique_identifier,amount,date
BCA-001,-750000.00,2024-01-16`
	path := writeTempBankCSV(t, csv)
	trxs, errs := ParseBankCSV(path, "BCA")
	assert.Empty(t, errs)
	require.Len(t, trxs, 1)
	// NormalizedAmount should return absolute value
	assert.True(t, trxs[0].NormalizedAmount().Equal(decimal.NewFromFloat(750000.00)))
	// InferredType should be DEBIT for negative
	assert.Equal(t, "DEBIT", string(trxs[0].InferredType()))
}

func TestParseBankCSV_AllPositive(t *testing.T) {
	csv := `unique_identifier,amount,date
BCA-001,500000.00,2024-01-15
BCA-002,750000.00,2024-01-16`
	path := writeTempBankCSV(t, csv)
	trxs, errs := ParseBankCSV(path, "BCA")
	assert.Empty(t, errs)
	assert.Len(t, trxs, 2)
	for _, trx := range trxs {
		assert.Equal(t, "CREDIT", string(trx.InferredType()))
	}
}

func TestParseBankCSV_MixedSigns(t *testing.T) {
	csv := `unique_identifier,amount,date
BCA-001,-500000.00,2024-01-15
BCA-002,750000.00,2024-01-15`
	path := writeTempBankCSV(t, csv)
	trxs, errs := ParseBankCSV(path, "BCA")
	assert.Empty(t, errs)
	assert.Len(t, trxs, 2)
	assert.Equal(t, "DEBIT", string(trxs[0].InferredType()))
	assert.Equal(t, "CREDIT", string(trxs[1].InferredType()))
}

func TestParseBankCSV_InvalidDate(t *testing.T) {
	csv := `unique_identifier,amount,date
BCA-001,-500000.00,not-a-date`
	path := writeTempBankCSV(t, csv)
	trxs, errs := ParseBankCSV(path, "BCA")
	assert.Empty(t, trxs)
	assert.Len(t, errs, 1)
}

func TestParseBankCSV_ZeroAmount(t *testing.T) {
	csv := `unique_identifier,amount,date
BCA-001,0.00,2024-01-15`
	path := writeTempBankCSV(t, csv)
	trxs, errs := ParseBankCSV(path, "BCA")
	assert.Empty(t, trxs)
	assert.Len(t, errs, 1)
}

func TestParseBankCSV_LargeDecimal(t *testing.T) {
	csv := `unique_identifier,amount,date
BCA-001,1234567890.123456,2024-01-15`
	path := writeTempBankCSV(t, csv)
	trxs, errs := ParseBankCSV(path, "BCA")
	assert.Empty(t, errs)
	require.Len(t, trxs, 1)
	assert.Equal(t, "1234567890.123456", trxs[0].Amount.String())
}

func TestParseBankCSV_DeriveBankName(t *testing.T) {
	csv := `unique_identifier,amount,date
X001,-100000,2024-01-15`
	path := writeTempBankCSV(t, csv)
	trxs, errs := ParseBankCSV(path, "MANDIRI")
	assert.Empty(t, errs)
	require.Len(t, trxs, 1)
	assert.Equal(t, "MANDIRI", trxs[0].BankName)
}

func TestParseBankCSV_MissingColumns(t *testing.T) {
	csv := "unique_identifier,amount\nBCA-001,-500000\n"
	path := writeTempBankCSV(t, csv)
	_, errs := ParseBankCSV(path, "BCA")
	assert.Len(t, errs, 1)
}

func TestParseBankCSV_EmptyFile(t *testing.T) {
	csv := "unique_identifier,amount,date\n"
	path := writeTempBankCSV(t, csv)
	trxs, errs := ParseBankCSV(path, "BCA")
	assert.Empty(t, errs)
	assert.Empty(t, trxs)
}

func TestDeriveBankName(t *testing.T) {
	cases := []struct {
		filename string
		expected string
	}{
		{"bank_bca_statement.csv", "BCA"},
		{"bank_mandiri_statement.csv", "MANDIRI"},
		{"bni_statement.csv", "BNI"},
		{"transactions.csv", "TRANSACTIONS"},
	}
	for _, c := range cases {
		assert.Equal(t, c.expected, deriveBankName(c.filename), c.filename)
	}
}
