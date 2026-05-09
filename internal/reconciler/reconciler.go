package reconciler

import (
	"sync"
	"time"

	"github.com/ridloal/reconciliation-service/internal/config"
	"github.com/ridloal/reconciliation-service/internal/domain"
	"github.com/ridloal/reconciliation-service/internal/parser"
	"github.com/shopspring/decimal"
)

// Reconcile runs the full reconciliation pipeline:
//  1. Parse all CSV files (bank files parsed concurrently).
//  2. Filter transactions to the configured date range.
//  3. Match system transactions against bank transactions.
//  4. Aggregate and return the result.
func Reconcile(cfg config.ReconciliationConfig) (*domain.ReconciliationResult, []error) {
	var allErrors []error

	// --- Parse system transactions ---
	var systemTrxs []domain.SystemTransaction
	for _, path := range cfg.SystemCSVPaths {
		trxs, errs := parser.ParseSystemCSV(path)
		allErrors = append(allErrors, errs...)
		systemTrxs = append(systemTrxs, trxs...)
	}

	// --- Parse bank transactions concurrently ---
	bankTrxs, bankErrs := parseBankFilesConcurrently(cfg.BankCSVPaths)
	allErrors = append(allErrors, bankErrs...)

	// --- Filter by date range ---
	filteredSystem := filterSystemByDate(systemTrxs, cfg.StartDate, cfg.EndDate)
	filteredBank := filterBankByDate(bankTrxs, cfg.StartDate, cfg.EndDate)

	// --- Match ---
	matched, unmatchedSys, unmatchedBank := matchAll(filteredSystem, filteredBank)

	// --- Aggregate ---
	result := aggregate(
		cfg,
		filteredSystem, filteredBank,
		matched, unmatchedSys, unmatchedBank,
	)

	return result, allErrors
}

// parseBankFilesConcurrently parses each bank CSV in its own goroutine.
func parseBankFilesConcurrently(paths []string) ([]domain.BankTransaction, []error) {
	type result struct {
		trxs []domain.BankTransaction
		errs []error
	}

	ch := make(chan result, len(paths))
	var wg sync.WaitGroup

	for _, path := range paths {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			trxs, errs := parser.ParseBankCSV(p, "")
			ch <- result{trxs, errs}
		}(path)
	}

	wg.Wait()
	close(ch)

	var allTrxs []domain.BankTransaction
	var allErrs []error
	for r := range ch {
		allTrxs = append(allTrxs, r.trxs...)
		allErrs = append(allErrs, r.errs...)
	}
	return allTrxs, allErrs
}

// filterSystemByDate keeps only system transactions whose date falls within [start, end].
func filterSystemByDate(trxs []domain.SystemTransaction, start, end time.Time) []domain.SystemTransaction {
	start = truncateToDay(start)
	end = truncateToDay(end)
	filtered := make([]domain.SystemTransaction, 0, len(trxs))
	for _, t := range trxs {
		d := truncateToDay(t.TransactionTime.UTC())
		if !d.Before(start) && !d.After(end) {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

// filterBankByDate keeps only bank transactions whose date falls within [start, end].
func filterBankByDate(trxs []domain.BankTransaction, start, end time.Time) []domain.BankTransaction {
	start = truncateToDay(start)
	end = truncateToDay(end)
	filtered := make([]domain.BankTransaction, 0, len(trxs))
	for _, t := range trxs {
		d := truncateToDay(t.Date.UTC())
		if !d.Before(start) && !d.After(end) {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

func truncateToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// aggregate builds the final ReconciliationResult from matched/unmatched slices.
func aggregate(
	cfg config.ReconciliationConfig,
	systemTrxs []domain.SystemTransaction,
	bankTrxs []domain.BankTransaction,
	matched []domain.MatchedPair,
	unmatchedSys []domain.UnmatchedSystemTrx,
	unmatchedBank []domain.UnmatchedBankTrx,
) *domain.ReconciliationResult {

	totalDiscrepancy := decimal.Zero
	for _, p := range matched {
		totalDiscrepancy = totalDiscrepancy.Add(p.AmountDiscrepancy)
	}

	missingFromBank := decimal.Zero
	for _, u := range unmatchedSys {
		missingFromBank = missingFromBank.Add(u.Transaction.Amount)
	}

	missingFromSystem := decimal.Zero
	for _, u := range unmatchedBank {
		missingFromSystem = missingFromSystem.Add(u.Transaction.NormalizedAmount())
	}

	totalUnmatched := len(unmatchedSys) + len(unmatchedBank)

	return &domain.ReconciliationResult{
		StartDate:   cfg.StartDate,
		EndDate:     cfg.EndDate,
		ProcessedAt: time.Now(),
		SystemFiles: cfg.SystemCSVPaths,
		BankFiles:   cfg.BankCSVPaths,

		TotalSystemTransactions: len(systemTrxs),
		TotalBankTransactions:   len(bankTrxs),
		TotalProcessed:          len(systemTrxs) + len(bankTrxs),
		TotalMatched:            len(matched),
		TotalUnmatched:          totalUnmatched,

		MatchedPairs:    matched,
		UnmatchedSystem: unmatchedSys,
		UnmatchedBank:   unmatchedBank,

		TotalDiscrepancyAmount: totalDiscrepancy,
		TotalMissingFromBank:   missingFromBank,
		TotalMissingFromSystem: missingFromSystem,
	}
}
