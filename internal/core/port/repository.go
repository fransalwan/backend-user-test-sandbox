package port

import (
	"context"

	"github.com/fransalwan/backend-user-test-sandbox/internal/core/entity"
)

// WalletRepository defines persistence operations for wallets.
type WalletRepository interface {
	GetByID(ctx context.Context, id string) (*entity.Wallet, error)
	GetByIDForUpdate(ctx context.Context, id string) (*entity.Wallet, error) // Pessimistic Lock
	Create(ctx context.Context, wallet *entity.Wallet) error
	UpdateBalance(ctx context.Context, id string, newBalance int64) error
	UpdateWithOptimisticLock(ctx context.Context, wallet *entity.Wallet) error // Optimistic Lock via version
}

// LedgerRepository defines persistence operations for immutable ledger entries.
type LedgerRepository interface {
	CreateEntry(ctx context.Context, entry *entity.LedgerEntry) error
	GetEntriesByWalletID(ctx context.Context, walletID string) ([]*entity.LedgerEntry, error)
}

// TransactionRepository defines persistence operations for transactions.
type TransactionRepository interface {
	GetByID(ctx context.Context, id string) (*entity.Transaction, error)
	GetByIdempotencyKey(ctx context.Context, key string) (*entity.Transaction, error)
	Create(ctx context.Context, tx *entity.Transaction) error
	UpdateStatus(ctx context.Context, id string, status entity.TransactionStatus) error
}
