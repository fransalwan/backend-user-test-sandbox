package entity

import "time"

// LedgerDirection defines the direction of funds for an immutable ledger entry.
type LedgerDirection string

const (
	DirectionDebit  LedgerDirection = "DEBIT"
	DirectionCredit LedgerDirection = "CREDIT"
)

// LedgerEntry represents an immutable double-entry ledger record.
// All financial state changes must produce balanced debit/credit entries.
type LedgerEntry struct {
	ID            string          `json:"id"`
	WalletID      string          `json:"wallet_id"`
	Amount        int64           `json:"amount"` // Absolute value in smallest currency units
	Direction     LedgerDirection `json:"direction"`
	TransactionID string          `json:"transaction_id"`
	CreatedAt     time.Time       `json:"created_at"`
}
