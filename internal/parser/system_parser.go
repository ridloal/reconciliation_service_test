package parser

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ridloal/reconciliation-service/internal/domain"
	"github.com/shopspring/decimal"
)

// ParseSystemCSV reads a system transaction CSV and returns parsed records.
// Rows that fail validation are skipped; errors for those rows are collected
// and returned alongside the valid records so the caller can log them.
func ParseSystemCSV(filePath string) ([]domain.SystemTransaction, []error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, []error{fmt.Errorf("cannot open system CSV %s: %w", filePath, err)}
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.TrimLeadingSpace = true

	headers, err := reader.Read()
	if err != nil {
		return nil, []error{fmt.Errorf("cannot read header from %s: %w", filePath, err)}
	}

	colIdx, err := mapSystemColumns(headers)
	if err != nil {
		return nil, []error{&domain.ConfigError{Message: fmt.Sprintf("%s: %v", filePath, err)}}
	}

	fileName := filepath.Base(filePath)
	var transactions []domain.SystemTransaction
	var parseErrors []error
	lineNum := 1 // header was line 1

	for {
		lineNum++
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			parseErrors = append(parseErrors, &domain.ParseError{
				File: fileName, LineNumber: lineNum, Field: "row", Message: err.Error(),
			})
			continue
		}

		trx, err := parseSystemRow(row, colIdx, fileName, lineNum)
		if err != nil {
			parseErrors = append(parseErrors, err)
			continue
		}
		transactions = append(transactions, *trx)
	}

	return transactions, parseErrors
}

// mapSystemColumns returns a map of canonical column name → index.
func mapSystemColumns(headers []string) (map[string]int, error) {
	required := []string{"trxid", "amount", "type", "transactiontime"}
	idx := make(map[string]int, len(headers))

	for i, h := range headers {
		idx[strings.ToLower(strings.TrimSpace(h))] = i
	}

	for _, r := range required {
		if _, ok := idx[r]; !ok {
			return nil, fmt.Errorf("missing required column %q", r)
		}
	}
	return idx, nil
}

func parseSystemRow(row []string, col map[string]int, file string, line int) (*domain.SystemTransaction, error) {
	get := func(key string) string {
		return strings.TrimSpace(row[col[key]])
	}

	trxID := get("trxid")
	if trxID == "" {
		return nil, &domain.ParseError{File: file, LineNumber: line, Field: "trxID", Message: "empty value"}
	}

	rawAmount := get("amount")
	amount, err := decimal.NewFromString(rawAmount)
	if err != nil {
		return nil, &domain.ParseError{File: file, LineNumber: line, Field: "amount", Message: "invalid decimal: " + rawAmount}
	}
	if amount.IsNegative() {
		return nil, &domain.ValidationError{Field: "amount", Value: rawAmount, Message: "system transaction amount must be positive"}
	}
	if amount.IsZero() {
		return nil, &domain.ValidationError{Field: "amount", Value: rawAmount, Message: "amount must be non-zero"}
	}

	rawType := get("type")
	txType, err := parseTransactionType(rawType)
	if err != nil {
		return nil, &domain.ParseError{File: file, LineNumber: line, Field: "type", Message: err.Error()}
	}

	rawTime := get("transactiontime")
	txTime, err := parseDateTime(rawTime)
	if err != nil {
		return nil, &domain.ParseError{File: file, LineNumber: line, Field: "transactionTime", Message: "invalid datetime: " + rawTime}
	}

	return &domain.SystemTransaction{
		TrxID:           trxID,
		Amount:          amount,
		Type:            txType,
		TransactionTime: txTime,
		SourceFile:      file,
		LineNumber:      line,
	}, nil
}
