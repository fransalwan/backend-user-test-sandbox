package entity

import "time"

// TransactionStatus represents the lifecycle state of a financial transaction.
type TransactionStatus string

const (
	TransactionStatusPending   TransactionStatus = "PENDING"
	TransactionStatusCompleted TransactionStatus = "COMPLETED"
	TransactionStatusFailed    TransactionStatus = "FAILED"
)

// Transaction represents a financial money movement between wallets or external sources.
type Transaction struct {
	ID             string            `json:"id"`
	IdempotencyKey string            `json:"idempotency_key"`
	SourceWalletID string            `json:"source_wallet_id,omitempty"`
	TargetWalletID string            `json:"target_wallet_id,omitempty"`
	Amount         int64             `json:"amount"`
	Status         TransactionStatus `json:"status"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}
