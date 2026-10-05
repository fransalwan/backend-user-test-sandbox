package mock

import (
	"context"

	"github.com/fransalwan/backend-user-test-sandbox/internal/core/port"
)

// PaymentGatewayMock implements port.PaymentGateway for testing.
type PaymentGatewayMock struct {
	ChargeFunc func(ctx context.Context, req port.PaymentChargeRequest) (*port.PaymentChargeResponse, error)
	RefundFunc func(ctx context.Context, externalTxID string, amount int64) error
}

func (m *PaymentGatewayMock) Charge(ctx context.Context, req port.PaymentChargeRequest) (*port.PaymentChargeResponse, error) {
	if m.ChargeFunc != nil {
		return m.ChargeFunc(ctx, req)
	}
	return &port.PaymentChargeResponse{
		ExternalTransactionID: "mock-tx-12345",
		Status:                "SUCCESS",
	}, nil
}

func (m *PaymentGatewayMock) Refund(ctx context.Context, externalTxID string, amount int64) error {
	if m.RefundFunc != nil {
		return m.RefundFunc(ctx, externalTxID, amount)
	}
	return nil
}
