package parser

import (
	"fmt"
	"strings"
	"time"

	"github.com/ridloal/reconciliation-service/internal/domain"
)

var dateTimeFormats = []string{
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05 -0700",
	"2006-01-02",
}

var dateFormats = []string{
	"2006-01-02",
	"02/01/2006",
	"01/02/2006",
	"2006/01/02",
}

func parseDateTime(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	for _, layout := range dateTimeFormats {
		if t, err := time.Parse(layout, raw); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized datetime format: %q", raw)
}

func parseDate(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	for _, layout := range dateFormats {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.UTC().Truncate(24 * time.Hour), nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized date format: %q", raw)
}

func parseTransactionType(raw string) (domain.TransactionType, error) {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "DEBIT":
		return domain.TypeDebit, nil
	case "CREDIT":
		return domain.TypeCredit, nil
	default:
		return "", fmt.Errorf("unknown transaction type %q, expected DEBIT or CREDIT", raw)
	}
}
