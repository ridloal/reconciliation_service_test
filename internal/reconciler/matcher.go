package reconciler

import (
	"fmt"

	"github.com/ridloal/reconciliation-service/internal/domain"
	"github.com/shopspring/decimal"
)

// MatchKey is the composite key used to group transactions for matching.
// Because system IDs and bank IDs use different formats, matching is done
// by (date, absolute-amount, transaction-type).
type MatchKey struct {
	Date   string
	Amount string // decimal string for map-safe equality
	Type   domain.TransactionType
}

func newMatchKey(date string, amount decimal.Decimal, txType domain.TransactionType) MatchKey {
	return MatchKey{Date: date, Amount: amount.String(), Type: txType}
}

// bankIndex maps a MatchKey to all bank transactions that share that key.
// Multiple transactions can share a key (duplicate scenario).
type bankIndex map[MatchKey][]domain.BankTransaction

// buildBankIndex builds a lookup index from a slice of bank transactions.
func buildBankIndex(bankTrxs []domain.BankTransaction) bankIndex {
	idx := make(bankIndex, len(bankTrxs))
	for _, trx := range bankTrxs {
		date, amount, txType := NormalizeBankTrx(trx)
		key := newMatchKey(date, amount, txType)
		idx[key] = append(idx[key], trx)
	}
	return idx
}

// matchAll performs the O(n) matching between system and bank transactions.
//
// Strategy:
//  1. Build a hash index of bank transactions keyed by (date, amount, type).
//  2. For each system transaction look up the index and take the first unused
//     bank candidate (FIFO deduplication).
//  3. Bank transactions not consumed by step 2 are unmatched.
func matchAll(
	systemTrxs []domain.SystemTransaction,
	bankTrxs []domain.BankTransaction,
) (matched []domain.MatchedPair, unmatchedSys []domain.UnmatchedSystemTrx, unmatchedBank []domain.UnmatchedBankTrx) {

	idx := buildBankIndex(bankTrxs)

	// Track which bank transactions have been paired.
	// Key: "<unique_identifier>@<bank_name>" to handle same IDs across banks.
	usedBankIDs := make(map[string]bool, len(bankTrxs))

	for _, sysTrx := range systemTrxs {
		date, amount, txType := NormalizeSystemTrx(sysTrx)
		key := newMatchKey(date, amount, txType)

		candidates := idx[key]
		paired := false

		for _, candidate := range candidates {
			compositeID := bankCompositeID(candidate)
			if usedBankIDs[compositeID] {
				continue
			}
			// Found an unused candidate — pair them.
			usedBankIDs[compositeID] = true
			discrepancy := sysTrx.Amount.Sub(candidate.NormalizedAmount()).Abs()
			matched = append(matched, domain.MatchedPair{
				SystemTrx:         sysTrx,
				BankTrx:           candidate,
				AmountDiscrepancy: discrepancy,
				IsExactMatch:      discrepancy.IsZero(),
			})
			paired = true
			break
		}

		if !paired {
			reason := "no_bank_match"
			if len(candidates) > 0 {
				reason = "all_candidates_taken"
			}
			unmatchedSys = append(unmatchedSys, domain.UnmatchedSystemTrx{
				Transaction: sysTrx, Reason: reason,
			})
		}
	}

	// Collect bank transactions that were never paired.
	for _, bankTrx := range bankTrxs {
		if !usedBankIDs[bankCompositeID(bankTrx)] {
			unmatchedBank = append(unmatchedBank, domain.UnmatchedBankTrx{
				Transaction: bankTrx, Reason: "no_system_match",
			})
		}
	}

	return matched, unmatchedSys, unmatchedBank
}

func bankCompositeID(trx domain.BankTransaction) string {
	return fmt.Sprintf("%s@%s", trx.UniqueIdentifier, trx.BankName)
}
