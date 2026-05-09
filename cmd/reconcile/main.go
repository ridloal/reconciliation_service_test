package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ridloal/reconciliation-service/internal/config"
	"github.com/ridloal/reconciliation-service/internal/reconciler"
	"github.com/ridloal/reconciliation-service/internal/reporter"
	"github.com/shopspring/decimal"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		systemFlag    = flag.String("system", "", "Path to system transactions CSV (required)")
		bankFlag      = flag.String("bank", "", "Comma-separated paths to bank statement CSV files (required)")
		startFlag     = flag.String("start", "", "Start date for reconciliation, format YYYY-MM-DD (required)")
		endFlag       = flag.String("end", "", "End date for reconciliation, format YYYY-MM-DD (required)")
		outputFmt     = flag.String("output", "text", "Output format: text (default) or json")
		outputFile    = flag.String("out-file", "", "Write output to file instead of stdout")
		toleranceFlag = flag.String("tolerance", "0", "Amount tolerance for discrepancy (default 0)")
		verbose       = flag.Bool("verbose", false, "Show additional details")
	)
	flag.Parse()

	// Validate required flags
	if *systemFlag == "" {
		flag.Usage()
		return fmt.Errorf("--system is required")
	}
	if *bankFlag == "" {
		flag.Usage()
		return fmt.Errorf("--bank is required")
	}
	if *startFlag == "" || *endFlag == "" {
		flag.Usage()
		return fmt.Errorf("--start and --end are required")
	}

	startDate, err := time.Parse("2006-01-02", *startFlag)
	if err != nil {
		return fmt.Errorf("invalid --start date %q: must be YYYY-MM-DD", *startFlag)
	}
	endDate, err := time.Parse("2006-01-02", *endFlag)
	if err != nil {
		return fmt.Errorf("invalid --end date %q: must be YYYY-MM-DD", *endFlag)
	}
	if endDate.Before(startDate) {
		return fmt.Errorf("--end date must be >= --start date")
	}

	tolerance, err := decimal.NewFromString(*toleranceFlag)
	if err != nil {
		return fmt.Errorf("invalid --tolerance %q", *toleranceFlag)
	}

	bankPaths := splitAndTrim(*bankFlag)

	cfg := config.ReconciliationConfig{
		StartDate:       startDate,
		EndDate:         endDate,
		SystemCSVPaths:  []string{*systemFlag},
		BankCSVPaths:    bankPaths,
		AmountTolerance: tolerance,
		Verbose:         *verbose,
		OutputFormat:    *outputFmt,
		OutputFile:      *outputFile,
	}

	// Run reconciliation
	result, parseErrors := reconciler.Reconcile(cfg)

	// Log non-fatal parse errors to stderr
	if len(parseErrors) > 0 {
		fmt.Fprintf(os.Stderr, "\n[WARNINGS] %d row(s) skipped during parsing:\n", len(parseErrors))
		for _, e := range parseErrors {
			fmt.Fprintf(os.Stderr, "  - %v\n", e)
		}
		fmt.Fprintln(os.Stderr)
	}

	// Determine output writer
	out := os.Stdout
	if *outputFile != "" {
		f, err := os.Create(*outputFile)
		if err != nil {
			return fmt.Errorf("cannot create output file %s: %w", *outputFile, err)
		}
		defer f.Close()
		out = f
	}

	// Write report
	switch strings.ToLower(*outputFmt) {
	case "json":
		return reporter.GenerateJSONReport(result, out)
	default:
		reporter.GenerateTextReport(result, out)
		return nil
	}
}

func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
