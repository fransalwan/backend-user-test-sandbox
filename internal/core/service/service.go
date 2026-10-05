package service

import (
	"context"

	"github.com/fransalwan/backend-user-test-sandbox/internal/core/entity"
)

// WalletService defines high-level business use cases for wallet accounts.
type WalletService interface {
	GetWallet(ctx context.Context, id string) (*entity.Wallet, error)
	CreditWallet(ctx context.Context, walletID string, amount int64) error
	DebitWallet(ctx context.Context, walletID string, amount int64) error
}

// TransferRequest parameters for moving money between wallets.
type TransferRequest struct {
	IdempotencyKey string `json:"idempotency_key"`
	SourceWalletID string `json:"source_wallet_id"`
	TargetWalletID string `json:"target_wallet_id"`
	Amount         int64  `json:"amount"`
}

// TransferService defines use cases for atomic double-entry money transfers.
type TransferService interface {
	Transfer(ctx context.Context, req TransferRequest) (*entity.Transaction, error)
}
