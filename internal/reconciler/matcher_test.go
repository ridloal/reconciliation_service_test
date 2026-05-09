package reconciler

import (
	"testing"
	"time"

	"github.com/ridloal/reconciliation-service/internal/domain"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeSystemTrx(id string, amount float64, txType domain.TransactionType, dateStr string) domain.SystemTransaction {
	d, _ := time.Parse("2006-01-02", dateStr)
	return domain.SystemTransaction{
		TrxID:           id,
		Amount:          decimal.NewFromFloat(amount),
		Type:            txType,
		TransactionTime: d,
	}
}

func makeBankTrx(id, bank string, amount float64, dateStr string) domain.BankTransaction {
	d, _ := time.Parse("2006-01-02", dateStr)
	return domain.BankTransaction{
		UniqueIdentifier: id,
		BankName:         bank,
		Amount:           decimal.NewFromFloat(amount),
		Date:             d,
	}
}

func TestMatchAll_AllMatched(t *testing.T) {
	sys := []domain.SystemTransaction{
		makeSystemTrx("S1", 500000, domain.TypeDebit, "2024-01-15"),
		makeSystemTrx("S2", 750000, domain.TypeCredit, "2024-01-16"),
	}
	bank := []domain.BankTransaction{
		makeBankTrx("B1", "BCA", -500000, "2024-01-15"),
		makeBankTrx("B2", "BCA", 750000, "2024-01-16"),
	}

	matched, unmatchedSys, unmatchedBank := matchAll(sys, bank)
	assert.Len(t, matched, 2)
	assert.Empty(t, unmatchedSys)
	assert.Empty(t, unmatchedBank)
	for _, p := range matched {
		assert.True(t, p.IsExactMatch)
	}
}

func TestMatchAll_AllUnmatched(t *testing.T) {
	sys := []domain.SystemTransaction{
		makeSystemTrx("S1", 500000, domain.TypeDebit, "2024-01-15"),
	}
	bank := []domain.BankTransaction{
		makeBankTrx("B1", "BCA", -999999, "2024-01-15"), // different amount
	}

	matched, unmatchedSys, unmatchedBank := matchAll(sys, bank)
	assert.Empty(t, matched)
	assert.Len(t, unmatchedSys, 1)
	assert.Len(t, unmatchedBank, 1)
}

func TestMatchAll_PartialMatch(t *testing.T) {
	sys := []domain.SystemTransaction{
		makeSystemTrx("S1", 500000, domain.TypeDebit, "2024-01-15"),
		makeSystemTrx("S2", 300000, domain.TypeDebit, "2024-01-16"), // no bank match
	}
	bank := []domain.BankTransaction{
		makeBankTrx("B1", "BCA", -500000, "2024-01-15"),
		makeBankTrx("B2", "BCA", 999999, "2024-01-17"), // no system match
	}

	matched, unmatchedSys, unmatchedBank := matchAll(sys, bank)
	assert.Len(t, matched, 1)
	assert.Len(t, unmatchedSys, 1)
	assert.Len(t, unmatchedBank, 1)
}

func TestMatchAll_DuplicatePairs(t *testing.T) {
	// 2 identical system trx + 2 identical bank trx — both should match (FIFO)
	sys := []domain.SystemTransaction{
		makeSystemTrx("S1", 500000, domain.TypeDebit, "2024-01-15"),
		makeSystemTrx("S2", 500000, domain.TypeDebit, "2024-01-15"),
	}
	bank := []domain.BankTransaction{
		makeBankTrx("B1", "BCA", -500000, "2024-01-15"),
		makeBankTrx("B2", "BCA", -500000, "2024-01-15"),
	}

	matched, unmatchedSys, unmatchedBank := matchAll(sys, bank)
	assert.Len(t, matched, 2)
	assert.Empty(t, unmatchedSys)
	assert.Empty(t, unmatchedBank)
}

func TestMatchAll_AsymmetricDuplicate(t *testing.T) {
	// 2 system trx, only 1 bank trx with same key → one system left unmatched
	sys := []domain.SystemTransaction{
		makeSystemTrx("S1", 500000, domain.TypeDebit, "2024-01-15"),
		makeSystemTrx("S2", 500000, domain.TypeDebit, "2024-01-15"),
	}
	bank := []domain.BankTransaction{
		makeBankTrx("B1", "BCA", -500000, "2024-01-15"),
	}

	matched, unmatchedSys, unmatchedBank := matchAll(sys, bank)
	assert.Len(t, matched, 1)
	assert.Len(t, unmatchedSys, 1)
	assert.Equal(t, "all_candidates_taken", unmatchedSys[0].Reason)
	assert.Empty(t, unmatchedBank)
}

func TestMatchAll_EmptySystem(t *testing.T) {
	bank := []domain.BankTransaction{
		makeBankTrx("B1", "BCA", -500000, "2024-01-15"),
	}
	matched, unmatchedSys, unmatchedBank := matchAll(nil, bank)
	assert.Empty(t, matched)
	assert.Empty(t, unmatchedSys)
	assert.Len(t, unmatchedBank, 1)
}

func TestMatchAll_EmptyBank(t *testing.T) {
	sys := []domain.SystemTransaction{
		makeSystemTrx("S1", 500000, domain.TypeDebit, "2024-01-15"),
	}
	matched, unmatchedSys, unmatchedBank := matchAll(sys, nil)
	assert.Empty(t, matched)
	assert.Len(t, unmatchedSys, 1)
	assert.Empty(t, unmatchedBank)
}

func TestMatchAll_BothEmpty(t *testing.T) {
	matched, unmatchedSys, unmatchedBank := matchAll(nil, nil)
	assert.Empty(t, matched)
	assert.Empty(t, unmatchedSys)
	assert.Empty(t, unmatchedBank)
}

func TestMatchAll_MultiBankMerge(t *testing.T) {
	sys := []domain.SystemTransaction{
		makeSystemTrx("S1", 500000, domain.TypeDebit, "2024-01-15"),
		makeSystemTrx("S2", 250000, domain.TypeDebit, "2024-01-21"),
		makeSystemTrx("S3", 375000, domain.TypeCredit, "2024-01-22"),
	}
	bank := []domain.BankTransaction{
		makeBankTrx("BCA-001", "BCA", -500000, "2024-01-15"),
		makeBankTrx("BNI-001", "BNI", -250000, "2024-01-21"),
		makeBankTrx("MDR-001", "MANDIRI", 375000, "2024-01-22"),
	}

	matched, unmatchedSys, unmatchedBank := matchAll(sys, bank)
	assert.Len(t, matched, 3)
	assert.Empty(t, unmatchedSys)
	assert.Empty(t, unmatchedBank)
}

func TestBuildBankIndex_Duplicates(t *testing.T) {
	bank := []domain.BankTransaction{
		makeBankTrx("B1", "BCA", -500000, "2024-01-15"),
		makeBankTrx("B2", "BCA", -500000, "2024-01-15"),
	}
	idx := buildBankIndex(bank)
	require.Len(t, idx, 1, "same key should produce one index entry")
	var key MatchKey
	for k := range idx {
		key = k
	}
	assert.Len(t, idx[key], 2, "index entry should contain both bank transactions")
}

func TestMatchAll_CrossBankMatch(t *testing.T) {
	sys := []domain.SystemTransaction{
		makeSystemTrx("S1", 500000, domain.TypeDebit, "2024-01-15"),
	}
	// Two banks have a transaction with the same key — only one should be consumed
	bank := []domain.BankTransaction{
		makeBankTrx("B-BCA", "BCA", -500000, "2024-01-15"),
		makeBankTrx("B-BNI", "BNI", -500000, "2024-01-15"),
	}

	matched, unmatchedSys, unmatchedBank := matchAll(sys, bank)
	assert.Len(t, matched, 1)
	assert.Empty(t, unmatchedSys)
	assert.Len(t, unmatchedBank, 1, "second bank trx should remain unmatched")
}
