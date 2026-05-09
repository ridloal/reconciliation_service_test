package reporter

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/ridloal/reconciliation-service/internal/domain"
)

const reportWidth = 80

// GenerateTextReport writes a human-readable reconciliation report to w.
func GenerateTextReport(r *domain.ReconciliationResult, w io.Writer) {
	sep := Separator(reportWidth)
	thin := ThinSeparator(reportWidth)

	// Header
	fmt.Fprintln(w, sep)
	fmt.Fprintln(w, center("RECONCILIATION SUMMARY REPORT", reportWidth))
	fmt.Fprintln(w, sep)
	fmt.Fprintf(w, "%-20s: %s s/d %s\n", "Period", FormatDate(r.StartDate), FormatDate(r.EndDate))
	fmt.Fprintf(w, "%-20s: %s\n", "Processed At", FormatDateTime(r.ProcessedAt))
	fmt.Fprintf(w, "%-20s: %s\n", "System File(s)", strings.Join(fileNames(r.SystemFiles), ", "))
	fmt.Fprintf(w, "%-20s: %s\n", "Bank File(s)", strings.Join(fileNames(r.BankFiles), ", "))
	fmt.Fprintln(w, thin)

	// Transaction counts
	fmt.Fprintln(w, "\nTRANSACTION COUNTS")
	fmt.Fprintln(w, thin)
	fmt.Fprintf(w, "%-45s: %d\n", "Total System Transactions (in range)", r.TotalSystemTransactions)
	fmt.Fprintf(w, "%-45s: %d\n", "Total Bank Transactions  (in range)", r.TotalBankTransactions)
	fmt.Fprintf(w, "%-45s: %d\n", "Total Processed", r.TotalProcessed)
	fmt.Fprintf(w, "%-45s: %d\n", "Total Matched", r.TotalMatched)
	fmt.Fprintf(w, "%-45s: %d\n", "Total Unmatched", r.TotalUnmatched)
	fmt.Fprintf(w, "  %-43s: %d\n", "- Unmatched from System", len(r.UnmatchedSystem))
	fmt.Fprintf(w, "  %-43s: %d\n", "- Unmatched from Bank", len(r.UnmatchedBank))

	// Financial summary
	fmt.Fprintln(w, "\nFINANCIAL SUMMARY")
	fmt.Fprintln(w, thin)
	fmt.Fprintf(w, "%-45s: %s\n", "Total Discrepancy Amount", FormatAmount(r.TotalDiscrepancyAmount))
	fmt.Fprintf(w, "%-45s: %s\n", "Total Missing from Bank", FormatAmount(r.TotalMissingFromBank))
	fmt.Fprintf(w, "%-45s: %s\n", "Total Missing from System", FormatAmount(r.TotalMissingFromSystem))

	// Matched transactions table
	if len(r.MatchedPairs) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, sep)
		fmt.Fprintf(w, "MATCHED TRANSACTIONS (%d)\n", len(r.MatchedPairs))
		fmt.Fprintln(w, sep)
		fmt.Fprintf(w, " %-4s | %-18s | %-24s | %-8s | %-10s | %-16s | %-16s | %s\n",
			"No", "System TrxID", "Bank ID", "Bank", "Date", "Sys Amount", "Bank Amount", "Discrepancy")
		fmt.Fprintln(w, ThinSeparator(reportWidth+30))
		for i, p := range r.MatchedPairs {
			fmt.Fprintf(w, " %-4d | %-18s | %-24s | %-8s | %-10s | %-16s | %-16s | %s\n",
				i+1,
				TruncateStr(p.SystemTrx.TrxID, 18),
				TruncateStr(p.BankTrx.UniqueIdentifier, 24),
				TruncateStr(p.BankTrx.BankName, 8),
				FormatDate(p.BankTrx.Date),
				FormatAmount(p.SystemTrx.Amount),
				FormatAmount(p.BankTrx.NormalizedAmount()),
				FormatAmount(p.AmountDiscrepancy),
			)
		}
	}

	// Unmatched system transactions
	if len(r.UnmatchedSystem) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, sep)
		fmt.Fprintf(w, "UNMATCHED SYSTEM TRANSACTIONS (%d)\n", len(r.UnmatchedSystem))
		fmt.Fprintln(w, sep)
		fmt.Fprintf(w, " %-4s | %-18s | %-16s | %-6s | %-10s | %s\n",
			"No", "TrxID", "Amount", "Type", "Date", "Reason")
		fmt.Fprintln(w, ThinSeparator(reportWidth))
		for i, u := range r.UnmatchedSystem {
			fmt.Fprintf(w, " %-4d | %-18s | %-16s | %-6s | %-10s | %s\n",
				i+1,
				TruncateStr(u.Transaction.TrxID, 18),
				FormatAmount(u.Transaction.Amount),
				string(u.Transaction.Type),
				FormatDate(u.Transaction.TransactionTime),
				u.Reason,
			)
		}
	}

	// Unmatched bank transactions
	if len(r.UnmatchedBank) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, sep)
		fmt.Fprintf(w, "UNMATCHED BANK TRANSACTIONS (%d)\n", len(r.UnmatchedBank))
		fmt.Fprintln(w, sep)
		fmt.Fprintf(w, " %-4s | %-24s | %-8s | %-16s | %-6s | %-10s | %s\n",
			"No", "Bank ID", "Bank", "Amount", "Type", "Date", "Reason")
		fmt.Fprintln(w, ThinSeparator(reportWidth))
		for i, u := range r.UnmatchedBank {
			fmt.Fprintf(w, " %-4d | %-24s | %-8s | %-16s | %-6s | %-10s | %s\n",
				i+1,
				TruncateStr(u.Transaction.UniqueIdentifier, 24),
				TruncateStr(u.Transaction.BankName, 8),
				FormatAmount(u.Transaction.NormalizedAmount()),
				string(u.Transaction.InferredType()),
				FormatDate(u.Transaction.Date),
				u.Reason,
			)
		}
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, sep)
	fmt.Fprintln(w, center("END OF REPORT", reportWidth))
	fmt.Fprintln(w, sep)
}

// JSONReport is the JSON-serializable representation of a reconciliation result.
type JSONReport struct {
	Period struct {
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
	} `json:"period"`
	ProcessedAt string     `json:"processed_at"`
	Summary     JSONSummary `json:"summary"`
	Matched     []JSONMatchedPair    `json:"matched"`
	UnmatchedSystem []JSONUnmatchedSystem `json:"unmatched_system"`
	UnmatchedBank   []JSONUnmatchedBank   `json:"unmatched_bank"`
}

type JSONSummary struct {
	TotalSystemTransactions int    `json:"total_system_transactions"`
	TotalBankTransactions   int    `json:"total_bank_transactions"`
	TotalProcessed          int    `json:"total_processed"`
	TotalMatched            int    `json:"total_matched"`
	TotalUnmatched          int    `json:"total_unmatched"`
	UnmatchedSystemCount    int    `json:"unmatched_system_count"`
	UnmatchedBankCount      int    `json:"unmatched_bank_count"`
	TotalDiscrepancyAmount  string `json:"total_discrepancy_amount"`
	TotalMissingFromBank    string `json:"total_missing_from_bank"`
	TotalMissingFromSystem  string `json:"total_missing_from_system"`
}

type JSONMatchedPair struct {
	SystemTrxID  string `json:"system_trx_id"`
	BankID       string `json:"bank_id"`
	BankName     string `json:"bank_name"`
	Date         string `json:"date"`
	SystemAmount string `json:"system_amount"`
	BankAmount   string `json:"bank_amount"`
	Discrepancy  string `json:"discrepancy"`
	IsExactMatch bool   `json:"is_exact_match"`
}

type JSONUnmatchedSystem struct {
	TrxID  string `json:"trx_id"`
	Amount string `json:"amount"`
	Type   string `json:"type"`
	Date   string `json:"date"`
	Reason string `json:"reason"`
}

type JSONUnmatchedBank struct {
	BankID   string `json:"bank_id"`
	BankName string `json:"bank_name"`
	Amount   string `json:"amount"`
	Type     string `json:"type"`
	Date     string `json:"date"`
	Reason   string `json:"reason"`
}

// GenerateJSONReport writes a JSON reconciliation report to w.
func GenerateJSONReport(r *domain.ReconciliationResult, w io.Writer) error {
	report := JSONReport{}
	report.Period.StartDate = FormatDate(r.StartDate)
	report.Period.EndDate = FormatDate(r.EndDate)
	report.ProcessedAt = r.ProcessedAt.Format(time.RFC3339)
	report.Summary = JSONSummary{
		TotalSystemTransactions: r.TotalSystemTransactions,
		TotalBankTransactions:   r.TotalBankTransactions,
		TotalProcessed:          r.TotalProcessed,
		TotalMatched:            r.TotalMatched,
		TotalUnmatched:          r.TotalUnmatched,
		UnmatchedSystemCount:    len(r.UnmatchedSystem),
		UnmatchedBankCount:      len(r.UnmatchedBank),
		TotalDiscrepancyAmount:  r.TotalDiscrepancyAmount.StringFixed(2),
		TotalMissingFromBank:    r.TotalMissingFromBank.StringFixed(2),
		TotalMissingFromSystem:  r.TotalMissingFromSystem.StringFixed(2),
	}
	for _, p := range r.MatchedPairs {
		report.Matched = append(report.Matched, JSONMatchedPair{
			SystemTrxID:  p.SystemTrx.TrxID,
			BankID:       p.BankTrx.UniqueIdentifier,
			BankName:     p.BankTrx.BankName,
			Date:         FormatDate(p.BankTrx.Date),
			SystemAmount: p.SystemTrx.Amount.StringFixed(2),
			BankAmount:   p.BankTrx.NormalizedAmount().StringFixed(2),
			Discrepancy:  p.AmountDiscrepancy.StringFixed(2),
			IsExactMatch: p.IsExactMatch,
		})
	}
	for _, u := range r.UnmatchedSystem {
		report.UnmatchedSystem = append(report.UnmatchedSystem, JSONUnmatchedSystem{
			TrxID:  u.Transaction.TrxID,
			Amount: u.Transaction.Amount.StringFixed(2),
			Type:   string(u.Transaction.Type),
			Date:   FormatDate(u.Transaction.TransactionTime),
			Reason: u.Reason,
		})
	}
	for _, u := range r.UnmatchedBank {
		report.UnmatchedBank = append(report.UnmatchedBank, JSONUnmatchedBank{
			BankID:   u.Transaction.UniqueIdentifier,
			BankName: u.Transaction.BankName,
			Amount:   u.Transaction.NormalizedAmount().StringFixed(2),
			Type:     string(u.Transaction.InferredType()),
			Date:     FormatDate(u.Transaction.Date),
			Reason:   u.Reason,
		})
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}

func center(s string, width int) string {
	if len(s) >= width {
		return s
	}
	pad := (width - len(s)) / 2
	return strings.Repeat(" ", pad) + s
}

func fileNames(paths []string) []string {
	names := make([]string, len(paths))
	for i, p := range paths {
		parts := strings.Split(strings.ReplaceAll(p, "\\", "/"), "/")
		names[i] = parts[len(parts)-1]
	}
	return names
}
