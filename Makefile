.PHONY: build test coverage run clean

build:
	go build -o reconcile ./cmd/reconcile

test:
	go test ./internal/... ./test/integration/... -v

unit:
	go test ./internal/... -v

integration:
	go test ./test/integration/... -v

coverage:
	go test ./internal/... ./test/integration/... -coverprofile=coverage.out
	go tool cover -func=coverage.out

run-sample:
	go run ./cmd/reconcile \
		--system=testdata/system_transactions.csv \
		--bank=testdata/bank_bca_statement.csv,testdata/bank_mandiri_statement.csv,testdata/bank_bni_statement.csv \
		--start=2024-01-15 \
		--end=2024-01-22

run-sample-json:
	go run ./cmd/reconcile \
		--system=testdata/system_transactions.csv \
		--bank=testdata/bank_bca_statement.csv,testdata/bank_mandiri_statement.csv,testdata/bank_bni_statement.csv \
		--start=2024-01-15 \
		--end=2024-01-22 \
		--output=json

clean:
	rm -f reconcile reconcile.exe coverage.out
