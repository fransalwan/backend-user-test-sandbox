package racecondition

import (
	"context"
	"testing"
)

func TestPessimisticLocking_PreventsOverdraft(t *testing.T) {
	initialBalance := int64(100000) // $1,000.00
	sim := NewSimulator(initialBalance)

	cfg := SimulationConfig{
		Strategy:       StrategyPessimistic,
		InitialBalance: initialBalance,
		Amount:         2500, // $25.00
		Concurrency:    50,   // Total attempt = $1,250.00
	}

	result := sim.Run(context.Background(), cfg, func(string) {})

	if result.HasOverdraft {
		t.Fatalf("expected no overdraft, but got final balance %d", result.FinalBalance)
	}

	if result.FinalBalance < 0 {
		t.Fatalf("balance dipped below zero: %d", result.FinalBalance)
	}

	// 100000 / 2500 = 40 max successful transactions
	expectedSuccess := 40
	if result.SuccessfulRequests != expectedSuccess {
		t.Errorf("expected %d successful requests, got %d", expectedSuccess, result.SuccessfulRequests)
	}

	expectedFailed := 10
	if result.FailedRequests != expectedFailed {
		t.Errorf("expected %d failed requests, got %d", expectedFailed, result.FailedRequests)
	}

	if result.FinalBalance != 0 {
		t.Errorf("expected final balance to be 0, got %d", result.FinalBalance)
	}
}

func TestOptimisticLocking_ConsistentFinalBalance(t *testing.T) {
	initialBalance := int64(50000) // $500.00
	sim := NewSimulator(initialBalance)

	cfg := SimulationConfig{
		Strategy:       StrategyOptimistic,
		InitialBalance: initialBalance,
		Amount:         5000, // $50.00
		Concurrency:    20,   // Total attempt = $1,000.00
	}

	result := sim.Run(context.Background(), cfg, func(string) {})

	if result.HasOverdraft {
		t.Fatalf("expected no overdraft under optimistic locking, got balance %d", result.FinalBalance)
	}

	if result.FinalBalance < 0 {
		t.Fatalf("balance dipped below zero: %d", result.FinalBalance)
	}
}
