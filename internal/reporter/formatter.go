package reporter

import (
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// FormatAmount formats a decimal as Indonesian Rupiah notation.
// e.g. 1500000.50 → "Rp 1.500.000,50"
func FormatAmount(d decimal.Decimal) string {
	isNeg := d.IsNegative()
	abs := d.Abs()

	parts := strings.Split(abs.StringFixed(2), ".")
	intPart := parts[0]
	decPart := parts[1]

	// Insert thousand separators (dot in Indonesian notation)
	var result []byte
	for i, ch := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			result = append(result, '.')
		}
		result = append(result, byte(ch))
	}

	formatted := fmt.Sprintf("Rp %s,%s", string(result), decPart)
	if isNeg {
		formatted = "-" + formatted
	}
	return formatted
}

// FormatDate formats a time.Time as "YYYY-MM-DD".
func FormatDate(t time.Time) string {
	return t.Format("2006-01-02")
}

// FormatDateTime formats a time.Time as "YYYY-MM-DD HH:MM:SS".
func FormatDateTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// Separator returns a horizontal line of the given width.
func Separator(width int) string {
	return strings.Repeat("=", width)
}

// ThinSeparator returns a thin horizontal line of the given width.
func ThinSeparator(width int) string {
	return strings.Repeat("-", width)
}

// TruncateStr truncates a string to max length, appending "…" if cut.
func TruncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
