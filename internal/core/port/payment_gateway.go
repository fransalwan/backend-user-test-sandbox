package port

import "context"

// PaymentChargeRequest contains parameters to process an external payment charge.
type PaymentChargeRequest struct {
	IdempotencyKey string `json:"idempotency_key"`
	Amount         int64  `json:"amount"`
	Currency       string `json:"currency"`
	Source         string `json:"source"`
}

// PaymentChargeResponse contains the result of a payment charge attempt.
type PaymentChargeResponse struct {
	ExternalTransactionID string `json:"external_transaction_id"`
	Status                string `json:"status"` // SUCCESS, PENDING, FAILED
}

// PaymentGateway defines communication with third-party payment providers (Stripe, Midtrans, etc.).
type PaymentGateway interface {
	Charge(ctx context.Context, req PaymentChargeRequest) (*PaymentChargeResponse, error)
	Refund(ctx context.Context, externalTxID string, amount int64) error
}
