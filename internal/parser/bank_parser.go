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

// ParseBankCSV reads a bank statement CSV and returns parsed records.
// bankName is used to tag each transaction; if empty, it is derived from the filename.
func ParseBankCSV(filePath, bankName string) ([]domain.BankTransaction, []error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, []error{fmt.Errorf("cannot open bank CSV %s: %w", filePath, err)}
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.TrimLeadingSpace = true

	headers, err := reader.Read()
	if err != nil {
		return nil, []error{fmt.Errorf("cannot read header from %s: %w", filePath, err)}
	}

	colIdx, err := mapBankColumns(headers)
	if err != nil {
		return nil, []error{&domain.ConfigError{Message: fmt.Sprintf("%s: %v", filePath, err)}}
	}

	fileName := filepath.Base(filePath)
	if bankName == "" {
		bankName = deriveBankName(fileName)
	}

	var transactions []domain.BankTransaction
	var parseErrors []error
	lineNum := 1

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

		trx, err := parseBankRow(row, colIdx, fileName, bankName, lineNum)
		if err != nil {
			parseErrors = append(parseErrors, err)
			continue
		}
		transactions = append(transactions, *trx)
	}

	return transactions, parseErrors
}

// known column name variants (case-insensitive)
var knownIDColumns     = []string{"unique_identifier", "id", "ref_no", "reference", "trxid"}
var knownAmountColumns = []string{"amount", "nominal", "jumlah", "debit_credit"}
var knownDateColumns   = []string{"date", "tanggal", "transaction_date", "trx_date"}

func mapBankColumns(headers []string) (map[string]int, error) {
	normalized := make(map[string]string, len(headers)) // lower → original
	idx := make(map[string]int, len(headers))
	for i, h := range headers {
		lower := strings.ToLower(strings.TrimSpace(h))
		normalized[lower] = h
		idx[lower] = i
	}

	result := make(map[string]int, 3)
	find := func(candidates []string, canonical string) error {
		for _, c := range candidates {
			if i, ok := idx[c]; ok {
				result[canonical] = i
				return nil
			}
		}
		return fmt.Errorf("missing required column (tried: %s)", strings.Join(candidates, ", "))
	}

	if err := find(knownIDColumns, "id"); err != nil {
		return nil, err
	}
	if err := find(knownAmountColumns, "amount"); err != nil {
		return nil, err
	}
	if err := find(knownDateColumns, "date"); err != nil {
		return nil, err
	}
	return result, nil
}

func parseBankRow(row []string, col map[string]int, file, bankName string, line int) (*domain.BankTransaction, error) {
	get := func(key string) string {
		return strings.TrimSpace(row[col[key]])
	}

	uid := get("id")
	if uid == "" {
		return nil, &domain.ParseError{File: file, LineNumber: line, Field: "unique_identifier", Message: "empty value"}
	}

	rawAmount := get("amount")
	amount, err := decimal.NewFromString(rawAmount)
	if err != nil {
		return nil, &domain.ParseError{File: file, LineNumber: line, Field: "amount", Message: "invalid decimal: " + rawAmount}
	}
	if amount.IsZero() {
		return nil, &domain.ValidationError{Field: "amount", Value: rawAmount, Message: "amount must be non-zero"}
	}

	rawDate := get("date")
	date, err := parseDate(rawDate)
	if err != nil {
		return nil, &domain.ParseError{File: file, LineNumber: line, Field: "date", Message: "invalid date: " + rawDate}
	}

	return &domain.BankTransaction{
		UniqueIdentifier: uid,
		Amount:           amount,
		Date:             date,
		BankName:         bankName,
		SourceFile:       file,
		LineNumber:       line,
		RawAmount:        rawAmount,
	}, nil
}

// deriveBankName extracts a readable bank name from a filename.
// e.g. "bank_bca_statement.csv" → "bca"
func deriveBankName(filename string) string {
	name := strings.ToLower(strings.TrimSuffix(filename, filepath.Ext(filename)))
	parts := strings.Split(name, "_")
	// strip generic tokens
	filtered := make([]string, 0, len(parts))
	skip := map[string]bool{"bank": true, "statement": true, "csv": true}
	for _, p := range parts {
		if !skip[p] {
			filtered = append(filtered, p)
		}
	}
	if len(filtered) == 0 {
		return strings.ToUpper(name)
	}
	return strings.ToUpper(strings.Join(filtered, "_"))
}
