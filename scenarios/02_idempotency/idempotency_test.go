package idempotency

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

func TestWithoutIdempotency_CausesDoubleSpending(t *testing.T) {
	wallet := &MockWallet{balance: 100000} // $1,000.00
	sim := NewSimulator()

	cfg := SimulationConfig{
		Mode:         ModeWithoutIdempotency,
		Key:          "test-key-no-idem",
		AmountCents:  5000, // $50.00
		RetriesCount: 4,    // 4 duplicate requests
	}

	result := sim.Run(context.Background(), cfg, wallet, func(string) {})

	if !result.HasDoubleSpending {
		t.Errorf("diharapkan terjadi double-spending pada mode without_idempotency")
	}

	if result.ProcessedCount != 4 {
		t.Errorf("diharapkan 4 pemotongan terjadi, didapat %d", result.ProcessedCount)
	}

	expectedFinal := int64(100000 - (5000 * 4)) // $800.00
	if result.FinalBalance != expectedFinal {
		t.Errorf("saldo akhir tidak sesuai: diharapkan %d, didapat %d", expectedFinal, result.FinalBalance)
	}
}

func TestWithIdempotency_PreventsDoubleSpending(t *testing.T) {
	wallet := &MockWallet{balance: 100000} // $1,000.00
	sim := NewSimulator()

	cfg := SimulationConfig{
		Mode:         ModeWithIdempotency,
		Key:          "test-key-idem-safe",
		AmountCents:  5000, // $50.00
		RetriesCount: 5,    // 5 duplicate requests
	}

	result := sim.Run(context.Background(), cfg, wallet, func(string) {})

	if result.HasDoubleSpending {
		t.Errorf("tidak boleh terjadi double-spending dengan kunci idempotensi")
	}

	if result.ProcessedCount != 1 {
		t.Errorf("hanya tepat 1 transaksi yang boleh memotong saldo, didapat %d", result.ProcessedCount)
	}

	if result.CachedReplays != 4 {
		t.Errorf("diharapkan 4 request dilayani dari cache response, didapat %d", result.CachedReplays)
	}

	expectedFinal := int64(100000 - 5000) // $950.00
	if result.FinalBalance != expectedFinal {
		t.Errorf("saldo akhir salah: diharapkan %d, didapat %d", expectedFinal, result.FinalBalance)
	}
}

func TestInFlightConflict_DetectsConcurrentRequests(t *testing.T) {
	wallet := &MockWallet{balance: 100000}
	sim := NewSimulator()

	cfg := SimulationConfig{
		Mode:         ModeInFlightConflict,
		Key:          "test-in-flight-key",
		AmountCents:  2500,
		RetriesCount: 3,
	}

	result := sim.Run(context.Background(), cfg, wallet, func(string) {})

	if result.InFlightConflicts == 0 {
		t.Errorf("diharapkan ada deteksi in-flight conflict HTTP 409")
	}

	if result.ProcessedCount != 1 {
		t.Errorf("hanya boleh 1 request yang berhasil diproses, didapat %d", result.ProcessedCount)
	}
}

func TestPayloadMismatch_DetectsTampering(t *testing.T) {
	wallet := &MockWallet{balance: 100000}
	sim := NewSimulator()

	cfg := SimulationConfig{
		Mode:        ModePayloadMismatch,
		Key:         "test-tampered-key",
		AmountCents: 2500,
	}

	result := sim.Run(context.Background(), cfg, wallet, func(string) {})

	if result.PayloadMismatches != 1 {
		t.Errorf("diharapkan 1 payload mismatch terdeteksi, didapat %d", result.PayloadMismatches)
	}
}
