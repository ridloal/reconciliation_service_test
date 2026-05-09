# Transaction Reconciliation Service

A CLI tool that reconciles internal system transactions against bank statements, identifying matched transactions, unmatched records, and amount discrepancies.

## Features

- Parses system transaction CSV and one or more bank statement CSVs
- Filters transactions by configurable date range
- Matches transactions using composite key: `(date, normalized amount, type)`  
  — handles the case where system IDs and bank IDs use different formats
- Handles duplicate transactions using FIFO deduplication
- Concurrent bank CSV parsing (one goroutine per file)
- Outputs a detailed text report or structured JSON
- Precise decimal arithmetic via `shopspring/decimal` (no float rounding errors)

## Prerequisites

- Go 1.21 or later

## Installation

```bash
git clone <repo-url>
cd reconciliation-service
go mod download
```

## Build

```bash
# Build binary
go build -o reconcile ./cmd/reconcile

# Or via make
make build
```

## Usage

```bash
./reconcile \
  --system=path/to/system_transactions.csv \
  --bank=path/to/bca.csv,path/to/mandiri.csv,path/to/bni.csv \
  --start=2024-01-01 \
  --end=2024-01-31
```

### Parameters

| Parameter     | Description                                              | Required |
|---------------|----------------------------------------------------------|----------|
| `--system`    | Path to the system transactions CSV                      | Yes      |
| `--bank`      | Comma-separated paths to bank statement CSV files        | Yes      |
| `--start`     | Start date for reconciliation (`YYYY-MM-DD`)             | Yes      |
| `--end`       | End date for reconciliation (`YYYY-MM-DD`)               | Yes      |
| `--output`    | Output format: `text` (default) or `json`                | No       |
| `--out-file`  | Write output to a file instead of stdout                 | No       |
| `--tolerance` | Amount tolerance for discrepancy matching (default `0`)  | No       |
| `--verbose`   | Show additional details                                  | No       |

### Quick Start with Sample Data

```bash
go build -o reconcile ./cmd/reconcile

./reconcile \
  --system=testdata/system_transactions.csv \
  --bank=testdata/bank_bca_statement.csv,testdata/bank_mandiri_statement.csv,testdata/bank_bni_statement.csv \
  --start=2024-01-15 \
  --end=2024-01-22
```

### JSON Output

```bash
./reconcile \
  --system=testdata/system_transactions.csv \
  --bank=testdata/bank_bca_statement.csv \
  --start=2024-01-15 \
  --end=2024-01-17 \
  --output=json \
  --out-file=result.json
```

## CSV Formats

### System Transactions CSV

```csv
trxID,amount,type,transactionTime
TRX-2024-001,500000.00,DEBIT,2024-01-15 08:30:00
TRX-2024-002,1250000.50,CREDIT,2024-01-15 09:15:22
```

| Field             | Type     | Notes                              |
|-------------------|----------|------------------------------------|
| `trxID`           | string   | Unique identifier                  |
| `amount`          | decimal  | Always positive                    |
| `type`            | enum     | `DEBIT` or `CREDIT`                |
| `transactionTime` | datetime | Format: `YYYY-MM-DD HH:MM:SS`      |

### Bank Statement CSV

```csv
unique_identifier,amount,date
BCA-TF-20240115-001,-500000.00,2024-01-15
BCA-TF-20240115-002,1250000.50,2024-01-15
```

| Field                | Type    | Notes                                           |
|----------------------|---------|-------------------------------------------------|
| `unique_identifier`  | string  | Bank-specific ID (format varies per bank)       |
| `amount`             | decimal | Can be negative (debit = money leaving account) |
| `date`               | date    | Format: `YYYY-MM-DD`                            |

The bank parser also recognises alternative column names: `id`, `ref_no`, `nominal`, `jumlah`, `tanggal`, etc.

## Matching Strategy

Since system IDs and bank IDs use different formats, direct ID matching is not possible. The service matches transactions using a composite key:

```
MatchKey = (date, |amount|, inferred_type)
```

- Bank amount is normalised to its absolute value; DEBIT/CREDIT is inferred from the sign (negative = DEBIT)
- System datetime is truncated to date for comparison
- Duplicate transactions with the same key are matched in FIFO order

## Running Tests

```bash
# Unit tests
make test

# Unit tests with verbose output
go test ./internal/... -v

# Integration tests
go test ./test/integration/... -v

# All tests with coverage report
make coverage

# Coverage report in browser
go tool cover -html=coverage.out
```

## Project Structure

```
cmd/reconcile/          CLI entry point and flag parsing
internal/config/        Configuration struct
internal/domain/        Core data models and error types
internal/parser/        CSV parsers (system and bank)
internal/reconciler/    Matching engine, normalizer, orchestrator
internal/reporter/      Text and JSON report generators
testdata/               Sample CSV files for manual testing
test/integration/       End-to-end integration tests
```
