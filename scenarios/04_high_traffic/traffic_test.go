package hightraffic

import (
	"context"
	"testing"
)

func TestTokenBucket_PreventsOverselling(t *testing.T) {
	initialStock := int64(30)
	sim := NewSimulator(initialStock)

	cfg := SimulationConfig{
		Strategy:        StrategyTokenBucket,
		InitialStock:    initialStock,
		TotalRequests:   100,
		RateLimitMaxRPS: 25,
	}

	result := sim.Run(context.Background(), cfg, func(string) {})

	if result.HasOverselling {
		t.Fatalf("tidak boleh terjadi overselling pada token bucket")
	}

	if result.FinalStock < 0 {
		t.Fatalf("stok akhir tidak boleh minus: %d", result.FinalStock)
	}

	if result.RateLimitedDrops == 0 {
		t.Errorf("diharapkan ada request yang terkena rate limit 429")
	}
}
