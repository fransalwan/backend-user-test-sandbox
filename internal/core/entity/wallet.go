package entity

import "time"

// Wallet represents a customer wallet balance state.
// Money balances are represented as int64 (in smallest currency units, e.g. cents)
// to prevent floating-point rounding errors and ensure strict financial accuracy.
type Wallet struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Balance   int64     `json:"balance"`
	Version   int64     `json:"version"` // Optimistic locking version
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
