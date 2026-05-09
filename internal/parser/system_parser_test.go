package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTempCSV(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "system_*.csv")
	require.NoError(t, err)
	_, err = f.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	return f.Name()
}

func TestParseSystemCSV_ValidData(t *testing.T) {
	csv := `trxID,amount,type,transactionTime
TRX-001,500000.00,DEBIT,2024-01-15 08:30:00
TRX-002,1250000.50,CREDIT,2024-01-15 09:15:22
TRX-003,750000.00,DEBIT,2024-01-16 10:00:00`

	path := writeTempCSV(t, csv)
	trxs, errs := ParseSystemCSV(path)

	assert.Empty(t, errs)
	require.Len(t, trxs, 3)
	assert.Equal(t, "TRX-001", trxs[0].TrxID)
	assert.True(t, trxs[0].Amount.Equal(decimal.NewFromFloat(500000.00)))
	assert.Equal(t, filepath.Base(path), trxs[0].SourceFile)
}

func TestParseSystemCSV_EmptyFile(t *testing.T) {
	csv := "trxID,amount,type,transactionTime\n"
	path := writeTempCSV(t, csv)
	trxs, errs := ParseSystemCSV(path)
	assert.Empty(t, errs)
	assert.Empty(t, trxs)
}

func TestParseSystemCSV_InvalidAmount(t *testing.T) {
	csv := `trxID,amount,type,transactionTime
TRX-001,abc,DEBIT,2024-01-15 08:30:00`
	path := writeTempCSV(t, csv)
	trxs, errs := ParseSystemCSV(path)
	assert.Empty(t, trxs)
	assert.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "amount")
}

func TestParseSystemCSV_NegativeAmount(t *testing.T) {
	csv := `trxID,amount,type,transactionTime
TRX-001,-500000,DEBIT,2024-01-15 08:30:00`
	path := writeTempCSV(t, csv)
	trxs, errs := ParseSystemCSV(path)
	assert.Empty(t, trxs)
	assert.Len(t, errs, 1)
}

func TestParseSystemCSV_ZeroAmount(t *testing.T) {
	csv := `trxID,amount,type,transactionTime
TRX-001,0.00,DEBIT,2024-01-15 08:30:00`
	path := writeTempCSV(t, csv)
	trxs, errs := ParseSystemCSV(path)
	assert.Empty(t, trxs)
	assert.Len(t, errs, 1)
}

func TestParseSystemCSV_InvalidType(t *testing.T) {
	csv := `trxID,amount,type,transactionTime
TRX-001,500000,TRANSFER,2024-01-15 08:30:00`
	path := writeTempCSV(t, csv)
	trxs, errs := ParseSystemCSV(path)
	assert.Empty(t, trxs)
	assert.Len(t, errs, 1)
}

func TestParseSystemCSV_InvalidDate(t *testing.T) {
	csv := `trxID,amount,type,transactionTime
TRX-001,500000,DEBIT,15/01/2024`
	path := writeTempCSV(t, csv)
	// "15/01/2024" does not match datetime formats; it matches date format DD/MM/YYYY
	// which is valid — so let's use something truly invalid
	csv2 := `trxID,amount,type,transactionTime
TRX-001,500000,DEBIT,not-a-date`
	path2 := writeTempCSV(t, csv2)
	trxs, errs := ParseSystemCSV(path2)
	_ = path // suppress unused
	assert.Empty(t, trxs)
	assert.Len(t, errs, 1)
}

func TestParseSystemCSV_MissingColumns(t *testing.T) {
	csv := "trxID,amount,type\nTRX-001,500000,DEBIT\n"
	path := writeTempCSV(t, csv)
	_, errs := ParseSystemCSV(path)
	assert.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "transactiontime")
}

func TestParseSystemCSV_ExtraColumns(t *testing.T) {
	csv := `trxID,amount,type,transactionTime,extra_col
TRX-001,500000,DEBIT,2024-01-15 08:30:00,ignored`
	path := writeTempCSV(t, csv)
	trxs, errs := ParseSystemCSV(path)
	assert.Empty(t, errs)
	assert.Len(t, trxs, 1)
}

func TestParseSystemCSV_WhitespaceTrim(t *testing.T) {
	csv := `trxID,amount,type,transactionTime
  TRX-001  ,  500000.00  ,  DEBIT  ,  2024-01-15 08:30:00  `
	path := writeTempCSV(t, csv)
	trxs, errs := ParseSystemCSV(path)
	assert.Empty(t, errs)
	require.Len(t, trxs, 1)
	assert.Equal(t, "TRX-001", trxs[0].TrxID)
}

func TestParseSystemCSV_FileNotFound(t *testing.T) {
	_, errs := ParseSystemCSV("/nonexistent/path/file.csv")
	assert.Len(t, errs, 1)
}
