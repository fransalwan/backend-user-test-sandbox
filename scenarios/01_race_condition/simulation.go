package racecondition

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// Strategy defines concurrency handling strategy
type Strategy string

const (
	StrategyNaive       Strategy = "naive"
	StrategyPessimistic Strategy = "pessimistic"
	StrategyOptimistic  Strategy = "optimistic"
	StrategyRedisLock   Strategy = "redis_lock"
)

type SimulationConfig struct {
	Strategy       Strategy
	InitialBalance int64 // cents
	Amount         int64 // cents per request
	Concurrency    int   // number of goroutines
}

type SimulationResult struct {
	Strategy             Strategy `json:"strategy"`
	InitialBalance       int64    `json:"initial_balance"`
	FinalBalance         int64    `json:"final_balance"`
	ExpectedDeduction    int64    `json:"expected_deduction"`
	ActualDeduction      int64    `json:"actual_deduction"`
	SuccessfulRequests   int      `json:"successful_requests"`
	FailedRequests       int      `json:"failed_requests"`
	ConflictRetries      int      `json:"conflict_retries"`
	BalanceDiscrepancy   int64    `json:"balance_discrepancy"` // Anomaly if != 0
	HasOverdraft         bool     `json:"has_overdraft"`
	DurationMilliseconds int64    `json:"duration_ms"`
}

type EventPublisher func(msg string)

// InMemoryWallet simulates a wallet row with fields for locking demonstrations
type InMemoryWallet struct {
	ID        string
	Balance   int64
	Version   int64
	mu        sync.Mutex   // For pessimistic locking simulation (SELECT ... FOR UPDATE)
	redisLock sync.Mutex   // For distributed lock simulation
}

type Simulator struct {
	wallet *InMemoryWallet
}

func NewSimulator(initialBalance int64) *Simulator {
	return &Simulator{
		wallet: &InMemoryWallet{
			ID:      "wallet-alice-001",
			Balance: initialBalance,
			Version: 1,
		},
	}
}

func (s *Simulator) GetWallet() (int64, int64) {
	return s.wallet.Balance, s.wallet.Version
}

func (s *Simulator) Reset(initialBalance int64) {
	s.wallet.mu.Lock()
	defer s.wallet.mu.Unlock()
	s.wallet.Balance = initialBalance
	s.wallet.Version = 1
}

// Run executes the concurrent simulation based on selected strategy
func (s *Simulator) Run(ctx context.Context, cfg SimulationConfig, publish EventPublisher) SimulationResult {
	start := time.Now()
	var (
		wg                 sync.WaitGroup
		successCount       int64
		failedCount        int64
		retryCount         int64
		maxAffordableTx    = cfg.InitialBalance / cfg.Amount
	)

	publish(fmt.Sprintf("⚡ Starting %s simulation: %d goroutines competing to deduct $%0.2f each (Balance: $%0.2f)",
		cfg.Strategy, cfg.Concurrency, float64(cfg.Amount)/100, float64(s.wallet.Balance)/100))

	for i := 1; i <= cfg.Concurrency; i++ {
		wg.Add(1)
		goroutineID := i

		go func(gID int) {
			defer wg.Done()

			switch cfg.Strategy {
			case StrategyNaive:
				s.executeNaive(gID, cfg.Amount, &successCount, &failedCount, publish)
			case StrategyPessimistic:
				s.executePessimistic(gID, cfg.Amount, &successCount, &failedCount, publish)
			case StrategyOptimistic:
				s.executeOptimistic(gID, cfg.Amount, &successCount, &failedCount, &retryCount, publish)
			case StrategyRedisLock:
				s.executeRedisLock(gID, cfg.Amount, &successCount, &failedCount, publish)
			}
		}(goroutineID)
	}

	wg.Wait()
	duration := time.Since(start).Milliseconds()

	finalBalance := s.wallet.Balance
	expectedDeduction := min(int64(cfg.Concurrency), maxAffordableTx) * cfg.Amount
	actualDeduction := cfg.InitialBalance - finalBalance
	discrepancy := expectedDeduction - actualDeduction
	if cfg.Strategy == StrategyNaive && finalBalance < 0 {
		discrepancy = actualDeduction - expectedDeduction
	}

	publish(fmt.Sprintf("🏁 Simulation completed in %dms. Final Balance: $%0.2f | Success: %d | Rejected: %d",
		duration, float64(finalBalance)/100, successCount, failedCount))

	return SimulationResult{
		Strategy:             cfg.Strategy,
		InitialBalance:       cfg.InitialBalance,
		FinalBalance:         finalBalance,
		ExpectedDeduction:    expectedDeduction,
		ActualDeduction:      actualDeduction,
		SuccessfulRequests:   int(successCount),
		FailedRequests:       int(failedCount),
		ConflictRetries:      int(retryCount),
		BalanceDiscrepancy:   discrepancy,
		HasOverdraft:         finalBalance < 0,
		DurationMilliseconds: duration,
	}
}

// 1. NAIVE (Classic Race Hazard - No locking, lost updates & balance overdraft)
func (s *Simulator) executeNaive(gID int, amount int64, success, failed *int64, publish EventPublisher) {
	// Step 1: Read snapshot
	current := s.wallet.Balance

	// Simulate network I/O or database latency (1-5ms)
	time.Sleep(time.Duration(1+rand.Intn(4)) * time.Millisecond)

	// Step 2: Check balance in application memory (Naive)
	if current < amount {
		atomic.AddInt64(failed, 1)
		publish(fmt.Sprintf("❌ [Goroutine #%02d] Rejected: Insufficient balance ($%0.2f < $%0.2f)", gID, float64(current)/100, float64(amount)/100))
		return
	}

	// Step 3: Write back new balance (Overwrites concurrent updates!)
	s.wallet.Balance = current - amount
	atomic.AddInt64(success, 1)
	publish(fmt.Sprintf("⚠️ [Goroutine #%02d] NAIVE WRITE: Deducted $%0.2f (Saw balance: $%0.2f)", gID, float64(amount)/100, float64(current)/100))
}

// 2. PESSIMISTIC LOCKING (Simulates SELECT ... FOR UPDATE)
func (s *Simulator) executePessimistic(gID int, amount int64, success, failed *int64, publish EventPublisher) {
	publish(fmt.Sprintf("⏳ [Goroutine #%02d] Waiting for Row Lock (SELECT ... FOR UPDATE)", gID))

	// Acquire exclusive lock on wallet row
	s.wallet.mu.Lock()
	defer s.wallet.mu.Unlock()

	publish(fmt.Sprintf("🔒 [Goroutine #%02d] Lock Acquired. Current Balance: $%0.2f", gID, float64(s.wallet.Balance)/100))

	// Accurate atomic check under lock
	if s.wallet.Balance < amount {
		atomic.AddInt64(failed, 1)
		publish(fmt.Sprintf("🚫 [Goroutine #%02d] Rejected under lock: Insufficient Balance ($%0.2f)", gID, float64(s.wallet.Balance)/100))
		return
	}

	s.wallet.Balance -= amount
	atomic.AddInt64(success, 1)
	publish(fmt.Sprintf("✅ [Goroutine #%02d] Success: Deducted $%0.2f | New Balance: $%0.2f", gID, float64(amount)/100, float64(s.wallet.Balance)/100))
}

// 3. OPTIMISTIC LOCKING (Simulates WHERE version = $v AND balance >= $amount)
func (s *Simulator) executeOptimistic(gID int, amount int64, success, failed, retries *int64, publish EventPublisher) {
	maxRetries := 5

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Read version and balance
		currentBalance := s.wallet.Balance
		currentVersion := s.wallet.Version

		if currentBalance < amount {
			atomic.AddInt64(failed, 1)
			publish(fmt.Sprintf("🚫 [Goroutine #%02d] Rejected: Insufficient balance ($%0.2f)", gID, float64(currentBalance)/100))
			return
		}

		// Simulating brief processing
		time.Sleep(time.Duration(rand.Intn(3)) * time.Millisecond)

		// Atomic compare-and-swap (simulating DB query: UPDATE ... WHERE version = currentVersion)
		s.wallet.mu.Lock()
		if s.wallet.Version == currentVersion {
			s.wallet.Balance -= amount
			s.wallet.Version++
			s.wallet.mu.Unlock()

			atomic.AddInt64(success, 1)
			publish(fmt.Sprintf("✅ [Goroutine #%02d] Commit OK at Version %d (attempt %d)", gID, currentVersion, attempt))
			return
		}
		s.wallet.mu.Unlock()

		// Conflict detected!
		atomic.AddInt64(retries, 1)
		publish(fmt.Sprintf("🔄 [Goroutine #%02d] Version Conflict! Expected %d, but row changed. Retrying (%d/%d)...",
			gID, currentVersion, attempt, maxRetries))

		// Exponential jitter backoff
		time.Sleep(time.Duration(attempt*2) * time.Millisecond)
	}

	atomic.AddInt64(failed, 1)
	publish(fmt.Sprintf("💥 [Goroutine #%02d] Aborted after exceeding max retries", gID))
}

// 4. REDIS DISTRIBUTED LOCK (SETNX simulation)
func (s *Simulator) executeRedisLock(gID int, amount int64, success, failed *int64, publish EventPublisher) {
	publish(fmt.Sprintf("🔑 [Goroutine #%02d] Requesting Redis SETNX lock 'lock:wallet:001'", gID))

	s.wallet.redisLock.Lock()
	defer s.wallet.redisLock.Unlock()

	publish(fmt.Sprintf("🔐 [Goroutine #%02d] Redis Lock Acquired.", gID))

	if s.wallet.Balance < amount {
		atomic.AddInt64(failed, 1)
		publish(fmt.Sprintf("🚫 [Goroutine #%02d] Rejected: Insufficient funds ($%0.2f)", gID, float64(s.wallet.Balance)/100))
		return
	}

	s.wallet.Balance -= amount
	atomic.AddInt64(success, 1)
	publish(fmt.Sprintf("✅ [Goroutine #%02d] Redis-Protected Deduct: $%0.2f | New Balance: $%0.2f", gID, float64(amount)/100, float64(s.wallet.Balance)/100))
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
