package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/ridloal/reconciliation-service/internal/config"
	"github.com/ridloal/reconciliation-service/internal/reconciler"
	"github.com/ridloal/reconciliation-service/internal/reporter"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testdataPath returns the absolute path to the testdata directory.
func testdataPath(file string) string {
	_, currentFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(currentFile), "..", "..", "testdata")
	return filepath.Join(root, file)
}

func defaultConfig() config.ReconciliationConfig {
	return config.ReconciliationConfig{
		StartDate:       time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		EndDate:         time.Date(2024, 1, 22, 0, 0, 0, 0, time.UTC),
		SystemCSVPaths:  []string{testdataPath("system_transactions.csv")},
		BankCSVPaths: []string{
			testdataPath("bank_bca_statement.csv"),
			testdataPath("bank_mandiri_statement.csv"),
			testdataPath("bank_bni_statement.csv"),
		},
		AmountTolerance: decimal.Zero,
		OutputFormat:    "text",
	}
}

func TestIntegration_FullFlow_PartialMatch(t *testing.T) {
	// Use only BCA (6 trx, covers Jan 15-17) + Mandiri (4 trx, covers Jan 18-20)
	// System has 12 trx Jan 15-22 → 6 system trx (Jan 21-22) will be unmatched
	// Bank has 10 trx in range → all matched, 0 unmatched bank
	cfg := defaultConfig()
	cfg.BankCSVPaths = []string{
		testdataPath("bank_bca_statement.csv"),
		testdataPath("bank_mandiri_statement.csv"),
	}

	result, errs := reconciler.Reconcile(cfg)
	assert.Empty(t, errs)
	require.NotNil(t, result)

	assert.Equal(t, 12, result.TotalSystemTransactions)
	assert.Equal(t, 10, result.TotalBankTransactions)
	assert.Equal(t, 10, result.TotalMatched)
	assert.Equal(t, 2, len(result.UnmatchedSystem), "TRX-011 and TRX-012 have no bank match")
	assert.Empty(t, result.UnmatchedBank)
	assert.False(t, result.TotalMissingFromBank.IsZero())
}

func TestIntegration_FullFlow_AllMatched(t *testing.T) {
	cfg := defaultConfig()
	result, errs := reconciler.Reconcile(cfg)

	assert.Empty(t, errs)
	require.NotNil(t, result)

	// 12 system in range, BCA(6)+Mandiri(4)+BNI(2 within range)=12 bank in range
	assert.Equal(t, 12, result.TotalSystemTransactions)
	assert.Equal(t, 12, result.TotalBankTransactions)
	assert.Equal(t, 12, result.TotalMatched)
	assert.Equal(t, 0, result.TotalUnmatched)
	assert.Empty(t, result.UnmatchedSystem)
	assert.Empty(t, result.UnmatchedBank)
	assert.True(t, result.TotalDiscrepancyAmount.IsZero())
}

func TestIntegration_FullFlow_DateRangeFilter(t *testing.T) {
	// Narrow range: only 2024-01-15, which has 2 system trx and 2 BCA bank trx
	cfg := defaultConfig()
	cfg.StartDate = time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	cfg.EndDate = time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	result, errs := reconciler.Reconcile(cfg)
	assert.Empty(t, errs)
	assert.Equal(t, 2, result.TotalSystemTransactions)
	assert.Equal(t, 2, result.TotalBankTransactions)
	assert.Equal(t, 2, result.TotalMatched)
	assert.Equal(t, 0, result.TotalUnmatched)
}

func TestIntegration_FullFlow_DateBoundaryExclusion(t *testing.T) {
	// Range that excludes BNI/2024/01/25 (outside) - should give 0 unmatched bank
	cfg := defaultConfig()
	cfg.StartDate = time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	cfg.EndDate = time.Date(2024, 1, 22, 0, 0, 0, 0, time.UTC)

	result, _ := reconciler.Reconcile(cfg)
	// BNI/2024/01/25/0003 is 2024-01-25, outside range → should not appear in unmatched bank
	for _, u := range result.UnmatchedBank {
		assert.NotEqual(t, "BNI/2024/01/25/0003", u.Transaction.UniqueIdentifier,
			"transaction outside date range must not appear in results")
	}
}

func TestIntegration_FullFlow_MultiBankFiles(t *testing.T) {
	cfg := defaultConfig()
	result, errs := reconciler.Reconcile(cfg)

	assert.Empty(t, errs)
	// Verify transactions from all 3 banks are matched
	bankNames := make(map[string]bool)
	for _, p := range result.MatchedPairs {
		bankNames[p.BankTrx.BankName] = true
	}
	assert.True(t, bankNames["BCA"], "BCA transactions should be matched")
	assert.True(t, bankNames["MANDIRI"], "Mandiri transactions should be matched")
	assert.True(t, bankNames["BNI"], "BNI transactions should be matched")
}

func TestIntegration_FullFlow_JSONOutput(t *testing.T) {
	cfg := defaultConfig()
	result, _ := reconciler.Reconcile(cfg)

	// Write JSON to temp file and verify it parses
	tmpFile, err := os.CreateTemp(t.TempDir(), "report_*.json")
	require.NoError(t, err)
	defer tmpFile.Close()

	err = reporter.GenerateJSONReport(result, tmpFile)
	require.NoError(t, err)
	tmpFile.Close()

	data, err := os.ReadFile(tmpFile.Name())
	require.NoError(t, err)

	var parsed reporter.JSONReport
	require.NoError(t, json.Unmarshal(data, &parsed))
	assert.Equal(t, "2024-01-15", parsed.Period.StartDate)
	assert.Equal(t, 12, parsed.Summary.TotalMatched)
}

func TestIntegration_ErrorHandling_MissingFile(t *testing.T) {
	cfg := defaultConfig()
	cfg.SystemCSVPaths = []string{"/nonexistent/file.csv"}

	result, errs := reconciler.Reconcile(cfg)
	assert.NotEmpty(t, errs, "should return errors for missing file")
	require.NotNil(t, result)
	assert.Equal(t, 0, result.TotalSystemTransactions)
}

func TestIntegration_DateRange_ExactBoundary(t *testing.T) {
	// Transactions on exactly start_date and end_date must be included
	cfg := defaultConfig()
	cfg.StartDate = time.Date(2024, 1, 21, 0, 0, 0, 0, time.UTC)
	cfg.EndDate = time.Date(2024, 1, 22, 0, 0, 0, 0, time.UTC)
	// System: TRX-011 (Jan 21), TRX-012 (Jan 22) = 2
	// Bank:   BNI/01/21, BNI/01/22 = 2

	result, errs := reconciler.Reconcile(cfg)
	assert.Empty(t, errs)
	assert.Equal(t, 2, result.TotalSystemTransactions)
	assert.Equal(t, 2, result.TotalBankTransactions)
	assert.Equal(t, 2, result.TotalMatched)
}

func TestIntegration_SingleBankFile(t *testing.T) {
	cfg := defaultConfig()
	cfg.BankCSVPaths = []string{testdataPath("bank_bca_statement.csv")}
	cfg.StartDate = time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	cfg.EndDate = time.Date(2024, 1, 17, 0, 0, 0, 0, time.UTC)
	// System: TRX-001..006 (6 trx), BCA: 6 trx → all matched

	result, errs := reconciler.Reconcile(cfg)
	assert.Empty(t, errs)
	assert.Equal(t, 6, result.TotalSystemTransactions)
	assert.Equal(t, 6, result.TotalBankTransactions)
	assert.Equal(t, 6, result.TotalMatched)
}

func TestIntegration_FinancialSummary_ZeroDiscrepancy(t *testing.T) {
	cfg := defaultConfig()
	result, _ := reconciler.Reconcile(cfg)
	assert.True(t, result.TotalDiscrepancyAmount.IsZero(),
		"all matched amounts are identical so discrepancy should be 0")
}

func TestIntegration_UnmatchedSystemHasAmounts(t *testing.T) {
	// Create config where one system trx has no bank match
	cfg := defaultConfig()
	cfg.BankCSVPaths = []string{testdataPath("bank_bca_statement.csv")} // only BCA, 6 trx
	cfg.StartDate = time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	cfg.EndDate = time.Date(2024, 1, 22, 0, 0, 0, 0, time.UTC)
	// System has 12 trx in range, BCA only has 6 → 6 unmatched system

	result, _ := reconciler.Reconcile(cfg)
	assert.Equal(t, 6, result.TotalMatched)
	assert.Equal(t, 6, len(result.UnmatchedSystem))
	assert.False(t, result.TotalMissingFromBank.IsZero(), "missing from bank should be > 0")
}
