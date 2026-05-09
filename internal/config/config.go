package config

import (
	"time"

	"github.com/shopspring/decimal"
)

// ReconciliationConfig holds all parameters for a reconciliation run.
type ReconciliationConfig struct {
	StartDate       time.Time
	EndDate         time.Time
	SystemCSVPaths  []string
	BankCSVPaths    []string
	AmountTolerance decimal.Decimal
	Verbose         bool
	OutputFormat    string // "text" or "json"
	OutputFile      string // empty = stdout
}
