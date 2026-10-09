package distributed

import (
	"context"
	"sync"
	"testing"
)

type MockWallet struct {
	mu      sync.Mutex
	balance int64
}

func (m *MockWallet) GetBalance() int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.balance
}

func (m *MockWallet) Deduct(amount int64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.balance < amount {
		return false
	}
	m.balance -= amount
	return true
}

func (m *MockWallet) Credit(amount int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.balance += amount
}

func TestDualWriteHazard_CausesFinancialLoss(t *testing.T) {
	wallet := &MockWallet{balance: 100000} // $1,000.00
	sim := NewSimulator()

	cfg := SimulationConfig{
		Mode:          ModeDualWriteHazard,
		AmountCents:   15000, // $150.00
		BankTarget:    "Bank Mandiri",
		InjectFailure: true,
		FailureType:   "500_INTERNAL_ERROR",
	}

	result := sim.Run(context.Background(), cfg, wallet, func(string) {})

	if !result.HasFinancialLoss {
		t.Errorf("diharapkan terjadi kerugian finansial (has_financial_loss == true)")
	}

	if result.CompensationRan {
		t.Errorf("pada mode dual write naif tidak boleh ada kompensasi otomatis")
	}

	// Saldo terpotong tapi uang tidak sampai ke gateway
	expectedFinal := int64(100000 - 15000) // $850.00
	if result.FinalBalance != expectedFinal {
		t.Errorf("saldo akhir salah: diharapkan %d, didapat %d", expectedFinal, result.FinalBalance)
	}
}

func TestSagaCompensating_PerformsAutoRefund(t *testing.T) {
	wallet := &MockWallet{balance: 100000} // $1,000.00
	sim := NewSimulator()

	cfg := SimulationConfig{
		Mode:          ModeSagaCompensating,
		AmountCents:   20000, // $200.00
		BankTarget:    "Bank Central Asia",
		InjectFailure: true,
		FailureType:   "504_GATEWAY_TIMEOUT",
	}

	result := sim.Run(context.Background(), cfg, wallet, func(string) {})

	if result.HasFinancialLoss {
		t.Errorf("pada Saga dengan kompensasi, tidak boleh ada kerugian finansial")
	}

	if !result.CompensationRan {
		t.Errorf("diharapkan transaksi kompensasi dijalankan (compensation_ran == true)")
	}

	if result.CompensatedAmount != 20000 {
		t.Errorf("nominal kompensasi salah: diharapkan 20000, didapat %d", result.CompensatedAmount)
	}

	// Saldo harus kembali utuh ke $1,000.00
	if result.FinalBalance != 100000 {
		t.Errorf("saldo akhir nasabah harus kembali utuh 100000, didapat %d", result.FinalBalance)
	}
}

func TestSagaSuccessPath_MarksOutboxPublished(t *testing.T) {
	wallet := &MockWallet{balance: 100000} // $1,000.00
	sim := NewSimulator()

	cfg := SimulationConfig{
		Mode:          ModeSagaSuccessPath,
		AmountCents:   10000, // $100.00
		BankTarget:    "Bank BCA",
		InjectFailure: false,
	}

	result := sim.Run(context.Background(), cfg, wallet, func(string) {})

	if !result.GatewaySuccess {
		t.Errorf("diharapkan gateway sukses")
	}

	if result.FinalStatus != "COMPLETED" {
		t.Errorf("status akhir diharapkan COMPLETED, didapat %s", result.FinalStatus)
	}

	if result.FinalBalance != 90000 {
		t.Errorf("saldo akhir salah: diharapkan 90000, didapat %d", result.FinalBalance)
	}

	events := sim.GetOutboxEvents()
	if len(events) != 1 {
		t.Fatalf("diharapkan 1 outbox event, didapat %d", len(events))
	}

	if events[0].Status != OutboxStatusPublished {
		t.Errorf("status outbox harus PUBLISHED, didapat %s", events[0].Status)
	}
}

