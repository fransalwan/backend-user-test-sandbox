package hightraffic

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

type Strategy string

const (
	StrategyUnthrottled    Strategy = "unthrottled"     // Langsung ke DB tanpa proteksi (Overselling & Crash)
	StrategyTokenBucket    Strategy = "token_bucket"    // Redis Token Bucket Rate Limiting + Atomic Cache
	StrategyWorkerQueue    Strategy = "worker_queue"    // Antrean Leaky Bucket Worker Pool
)

type SimulationConfig struct {
	Strategy       Strategy
	InitialStock   int64 // Jumlah voucher / kuota promo terbatas (misal: 50 atau 100)
	TotalRequests  int   // Serbuan request konkuren (misal: 250 atau 500)
	RateLimitMaxRPS int  // Batas maksimal request per detik
}

type SimulationResult struct {
	Strategy          Strategy `json:"strategy"`
	InitialStock      int64    `json:"initial_stock"`
	FinalStock        int64    `json:"final_stock"`
	TotalRequests     int      `json:"total_requests"`
	SuccessfulClaims  int      `json:"successful_claims"`
	RateLimitedDrops  int      `json:"rate_limited_drops"` // Kena 429 Too Many Requests
	OutOfStockDrops   int      `json:"out_of_stock_drops"` // Kena 410 Kuota Habis
	OversoldItems     int64    `json:"oversold_items"`     // Anomali kuota jebol minus
	HasOverselling    bool     `json:"has_overselling"`
	DurationMs        int64    `json:"duration_ms"`
}

type EventPublisher func(msg string)

type Simulator struct {
	mu    sync.Mutex
	stock int64
}

func NewSimulator(initialStock int64) *Simulator {
	return &Simulator{
		stock: initialStock,
	}
}

func (s *Simulator) Reset(stock int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stock = stock
}

func (s *Simulator) GetStock() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stock
}

// Run mengeksekusi simulasi serbuan Flash Sale / High Traffic
func (s *Simulator) Run(ctx context.Context, cfg SimulationConfig, publish EventPublisher) SimulationResult {
	start := time.Now()
	s.Reset(cfg.InitialStock)

	publish(fmt.Sprintf("⚡ [Tahap 4 - High Traffic] Memulai serbuan Flash Sale: %d request berebut %d voucher (Strategi: %s)...",
		cfg.TotalRequests, cfg.InitialStock, cfg.Strategy))

	var (
		wg               sync.WaitGroup
		successCount     int64
		rateLimitDrops   int64
		outOfStockDrops  int64
	)

	// Token bucket semaphore (kapasitas burst)
	bucketTokens := int64(cfg.RateLimitMaxRPS)
	if bucketTokens <= 0 {
		bucketTokens = 50
	}

	for i := 1; i <= cfg.TotalRequests; i++ {
		wg.Add(1)
		reqID := i

		go func(id int) {
			defer wg.Done()

			switch cfg.Strategy {
			case StrategyUnthrottled:
				// Naif: Read snapshot stok -> delay -> decrement (Terjadi race condition & overselling)
				current := s.stock
				time.Sleep(time.Duration(1+rand.Intn(4)) * time.Millisecond) // Simulasi latency DB
				if current > 0 {
					s.stock = current - 1
					atomic.AddInt64(&successCount, 1)
					if id%15 == 0 {
						publish(fmt.Sprintf("⚠️ [Req #%03d] Klaim voucher diterima tanpa rate limiting (Sisa di memori: %d)", id, s.stock))
					}
				} else {
					atomic.AddInt64(&outOfStockDrops, 1)
				}

			case StrategyTokenBucket:
				// Proteksi Tingkat 1: Rate Limiter Token Bucket
				tokensLeft := atomic.AddInt64(&bucketTokens, -1)
				if tokensLeft < 0 {
					atomic.AddInt64(&rateLimitDrops, 1)
					if id%20 == 0 {
						publish(fmt.Sprintf("🛑 [Req #%03d] DITOLAK (HTTP 429 Too Many Requests): Melebihi kapasitas Rate Limiter!", id))
					}
					return
				}

				// Proteksi Tingkat 2: Redis Atomic DECRBY (Simulasi atomic decrement)
				s.mu.Lock()
				if s.stock > 0 {
					s.stock--
					s.mu.Unlock()
					atomic.AddInt64(&successCount, 1)
					if id%15 == 0 {
						publish(fmt.Sprintf("✅ [Req #%03d] Sukses klaim voucher (Sisa kuota: %d)", id, s.stock))
					}
				} else {
					s.mu.Unlock()
					atomic.AddInt64(&outOfStockDrops, 1)
				}

			case StrategyWorkerQueue:
				// Worker Pool Decoupling
				s.mu.Lock()
				if s.stock > 0 {
					s.stock--
					s.mu.Unlock()
					atomic.AddInt64(&successCount, 1)
				} else {
					s.mu.Unlock()
					atomic.AddInt64(&outOfStockDrops, 1)
				}
			}
		}(reqID)
	}

	wg.Wait()
	duration := time.Since(start).Milliseconds()

	finalStock := s.stock
	oversold := int64(0)
	if finalStock < 0 {
		oversold = -finalStock
	}

	publish(fmt.Sprintf("🏁 Flash Sale selesai dalam %d ms. Berhasil: %d | Ditolak Rate Limiter (429): %d | Kehabisan Stok (410): %d | Sisa Stok: %d",
		duration, successCount, rateLimitDrops, outOfStockDrops, finalStock))

	return SimulationResult{
		Strategy:          cfg.Strategy,
		InitialStock:      cfg.InitialStock,
		FinalStock:        finalStock,
		TotalRequests:     cfg.TotalRequests,
		SuccessfulClaims:  int(successCount),
		RateLimitedDrops:  int(rateLimitDrops),
		OutOfStockDrops:   int(outOfStockDrops),
		OversoldItems:     oversold,
		HasOverselling:    finalStock < 0,
		DurationMs:        duration,
	}
}

