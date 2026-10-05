package idempotency

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// IdempotencyStatus merepresentasikan status siklus hidup kunci idempotensi.
type IdempotencyStatus string

const (
	StatusProcessing IdempotencyStatus = "PROCESSING"
	StatusCompleted  IdempotencyStatus = "COMPLETED"
	StatusFailed     IdempotencyStatus = "FAILED"
)

// Record menyimpan data respons dan status pemrosesan untuk suatu kunci idempotensi.
type Record struct {
	Key          string            `json:"key"`
	RequestHash  string            `json:"request_hash"`
	Status       IdempotencyStatus `json:"status"`
	ResponseCode int               `json:"response_code"`
	ResponseBody string            `json:"response_body"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// Store adalah penyimpanan in-memory thread-safe untuk kunci idempotensi.
type Store struct {
	mu      sync.RWMutex
	records map[string]*Record
}

func NewStore() *Store {
	return &Store{
		records: make(map[string]*Record),
	}
}

func (s *Store) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = make(map[string]*Record)
}

// ComputeHash menghasilkan string SHA-256 dari payload permintaan untuk verifikasi integritas data.
func ComputeHash(payload string) string {
	hash := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(hash[:])
}

type SimulationMode string

const (
	ModeWithoutIdempotency SimulationMode = "without_idempotency" // Bahaya double-spending
	ModeWithIdempotency    SimulationMode = "with_idempotency"    // Aman & idempoten
	ModeInFlightConflict   SimulationMode = "in_flight_conflict"  // Simulasi 409 Conflict saat request masih diproses
	ModePayloadMismatch    SimulationMode = "payload_mismatch"    // Kunci sama tapi isi berbeda (422)
)

type SimulationConfig struct {
	Mode         SimulationMode
	Key          string
	AmountCents  int64
	RetriesCount int // Jumlah percobaan / request duplikat
}

type SimulationResult struct {
	Mode              SimulationMode `json:"mode"`
	Key               string         `json:"key"`
	TotalRequests     int            `json:"total_requests"`
	ProcessedCount    int            `json:"processed_count"`    // Berapa transaksi yang benar-benar memotong saldo
	CachedReplays     int            `json:"cached_replays"`     // Berapa respons yang disajikan dari cache
	InFlightConflicts int            `json:"in_flight_conflicts"`// Berapa request yang kena 409 Conflict
	PayloadMismatches int            `json:"payload_mismatches"` // Berapa request ditolak karena manipulasi payload
	InitialBalance    int64          `json:"initial_balance"`
	FinalBalance      int64          `json:"final_balance"`
	TotalDeducted     int64          `json:"total_deducted"`
	HasDoubleSpending bool           `json:"has_double_spending"`
	DurationMs        int64          `json:"duration_ms"`
}

type EventPublisher func(msg string)

// SharedWallet merepresentasikan dompet yang dimutasi dalam simulasi.
type SharedWallet interface {
	GetBalance() int64
	Deduct(amount int64) bool
}

// Simulator mengorkestrasi skenario pengujian idempotensi.
type Simulator struct {
	store *Store
}

func NewSimulator() *Simulator {
	return &Simulator{
		store: NewStore(),
	}
}

func (s *Simulator) GetStore() *Store {
	return s.store
}

func (s *Simulator) Reset() {
	s.store.Clear()
}

// Run mengeksekusi simulasi idempotensi sesuai mode yang dipilih.
func (s *Simulator) Run(ctx context.Context, cfg SimulationConfig, wallet SharedWallet, publish EventPublisher) SimulationResult {
	start := time.Now()
	initialBalance := wallet.GetBalance()

	var (
		processedCount    int64
		cachedReplays     int64
		inFlightConflicts int64
		payloadMismatches int64
	)

	publish(fmt.Sprintf("🚀 [Skenario 02] Memulai simulasi mode '%s' dengan %d permintaan duplikat...", cfg.Mode, cfg.RetriesCount))

	switch cfg.Mode {
	case ModeWithoutIdempotency:
		// Simulasi Tanpa Idempotensi: Setiap request memotong saldo secara langsung (Bahaya Double Spending!)
		var wg sync.WaitGroup
		for i := 1; i <= cfg.RetriesCount; i++ {
			wg.Add(1)
			reqID := i
			go func(id int) {
				defer wg.Done()
				publish(fmt.Sprintf("⚠️ [Request #%d] Masuk tanpa Idempotency-Key. Memproses pemotongan saldo...", id))
				time.Sleep(10 * time.Millisecond) // Simulasi I/O

				if wallet.Deduct(cfg.AmountCents) {
					atomic.AddInt64(&processedCount, 1)
					publish(fmt.Sprintf("💸 [Request #%d] BAHAYA: Saldo berhasil dipotong $%0.2f (Pemotongan berulang ke-%d!)",
						id, float64(cfg.AmountCents)/100, id))
				} else {
					publish(fmt.Sprintf("❌ [Request #%d] Gagal: Saldo tidak mencukupi.", id))
				}
			}(reqID)
		}
		wg.Wait()

	case ModeWithIdempotency:
		// Simulasi Dengan Kunci Idempotensi: Request serial atau dengan retry network
		payload := fmt.Sprintf(`{"amount":%d,"action":"transfer"}`, cfg.AmountCents)
		payloadHash := ComputeHash(payload)

		for i := 1; i <= cfg.RetriesCount; i++ {
			reqID := i
			publish(fmt.Sprintf("📨 [Request #%d] Mengirim permintaan dengan Idempotency-Key: '%s'", reqID, cfg.Key))

			s.store.mu.Lock()
			existing, exists := s.store.records[cfg.Key]

			if !exists {
				// Request pertama: Daftarkan status PROCESSING
				s.store.records[cfg.Key] = &Record{
					Key:         cfg.Key,
					RequestHash: payloadHash,
					Status:      StatusProcessing,
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}
				s.store.mu.Unlock()

				publish(fmt.Sprintf("🔒 [Request #%d] Kunci baru ditemukan. Mengunci status 'PROCESSING'...", reqID))
				time.Sleep(15 * time.Millisecond) // Simulasi eksekusi mutasi ledger

				if wallet.Deduct(cfg.AmountCents) {
					atomic.AddInt64(&processedCount, 1)
					s.store.mu.Lock()
					rec := s.store.records[cfg.Key]
					rec.Status = StatusCompleted
					rec.ResponseCode = 200
					rec.ResponseBody = fmt.Sprintf(`{"status":"SUCCESS","deducted":%d}`, cfg.AmountCents)
					rec.UpdatedAt = time.Now()
					s.store.mu.Unlock()

					publish(fmt.Sprintf("✅ [Request #%d] TRANSAKSI ASLI BERHASIL: Saldo dipotong $%0.2f. Status disimpan 'COMPLETED'.",
						reqID, float64(cfg.AmountCents)/100))
				}
			} else {
				// Request duplikat: Periksa status yang tersimpan
				if existing.Status == StatusCompleted {
					s.store.mu.Unlock()
					atomic.AddInt64(&cachedReplays, 1)
					publish(fmt.Sprintf("🛡️ [Request #%d] DEDUPLIKASI AKTIF: Kunci '%s' sudah COMPLETED. Mengembalikan Cached Response (HTTP 200) tanpa memotong saldo!",
						reqID, cfg.Key))
				} else if existing.Status == StatusProcessing {
					s.store.mu.Unlock()
					atomic.AddInt64(&inFlightConflicts, 1)
					publish(fmt.Sprintf("⛔ [Request #%d] IN-FLIGHT CONFLICT (HTTP 409): Permintaan dengan kunci ini sedang berjalan!", reqID))
				} else {
					s.store.mu.Unlock()
				}
			}
			time.Sleep(5 * time.Millisecond)
		}

	case ModeInFlightConflict:
		// Simulasi Request Paralel Cepat: Request B tiba saat Request A masih status PROCESSING
		payload := fmt.Sprintf(`{"amount":%d}`, cfg.AmountCents)
		payloadHash := ComputeHash(payload)

		var wg sync.WaitGroup
		for i := 1; i <= cfg.RetriesCount; i++ {
			wg.Add(1)
			reqID := i
			go func(id int) {
				defer wg.Done()

				s.store.mu.Lock()
				existing, exists := s.store.records[cfg.Key]

				if !exists {
					// Pemenang pertama yang mendapat lock
					s.store.records[cfg.Key] = &Record{
						Key:         cfg.Key,
						RequestHash: payloadHash,
						Status:      StatusProcessing,
						CreatedAt:   time.Now(),
						UpdatedAt:   time.Now(),
					}
					s.store.mu.Unlock()

					publish(fmt.Sprintf("🥇 [Request #%d] Pertama tiba! Mengunci status PROCESSING. Memproses gateway selama 100ms...", id))
					time.Sleep(100 * time.Millisecond) // Sengaja dibuat lama untuk menguji in-flight collision

					wallet.Deduct(cfg.AmountCents)
					atomic.AddInt64(&processedCount, 1)

					s.store.mu.Lock()
					rec := s.store.records[cfg.Key]
					rec.Status = StatusCompleted
					rec.ResponseCode = 200
					rec.ResponseBody = `{"status":"SUCCESS"}`
					s.store.mu.Unlock()
					publish(fmt.Sprintf("🏁 [Request #%d] Selesai & disimpan sebagai COMPLETED.", id))
				} else {
					// Request lain yang tiba sebelum pemenang selesai
					if existing.Status == StatusProcessing {
						s.store.mu.Unlock()
						atomic.AddInt64(&inFlightConflicts, 1)
						publish(fmt.Sprintf("🚫 [Request #%d] DITOLAK (HTTP 409 Conflict): Request sebelumnya masih PROCESSING!", id))
					} else {
						s.store.mu.Unlock()
						atomic.AddInt64(&cachedReplays, 1)
						publish(fmt.Sprintf("🛡️ [Request #%d] Menyajikan respons cache.", id))
					}
				}
			}(reqID)
		}
		wg.Wait()

	case ModePayloadMismatch:
		// Simulasi Manipulasi Payload: Kunci sama tapi amount diubah oleh penyerang/bug klien
		originalPayload := fmt.Sprintf(`{"amount":%d}`, cfg.AmountCents)
		s.store.mu.Lock()
		s.store.records[cfg.Key] = &Record{
			Key:          cfg.Key,
			RequestHash:  ComputeHash(originalPayload),
			Status:       StatusCompleted,
			ResponseCode: 200,
			ResponseBody: `{"status":"SUCCESS"}`,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		s.store.mu.Unlock()

		publish(fmt.Sprintf("🔑 Kunci '%s' tercatat dengan payload asli $%0.2f", cfg.Key, float64(cfg.AmountCents)/100))

		// Request kedua masuk dengan amount berbeda ($500 bukan $25)
		tamperedAmount := cfg.AmountCents * 10
		tamperedPayload := fmt.Sprintf(`{"amount":%d}`, tamperedAmount)
		tamperedHash := ComputeHash(tamperedPayload)

		publish(fmt.Sprintf("🚨 Request baru tiba dengan kunci yang sama ('%s') tetapi payload diubah menjadi $%0.2f!",
			cfg.Key, float64(tamperedAmount)/100))

		s.store.mu.RLock()
		rec := s.store.records[cfg.Key]
		if rec.RequestHash != tamperedHash {
			atomic.AddInt64(&payloadMismatches, 1)
			publish(fmt.Sprintf("🛑 ERROR (HTTP 422 Unprocessable Entity): Hash payload tidak cocok! Potensi manipulasi transaksi digagalkan."))
		}
		s.store.mu.RUnlock()
	}

	finalBalance := wallet.GetBalance()
	totalDeducted := initialBalance - finalBalance
	duration := time.Since(start).Milliseconds()

	hasDoubleSpending := (cfg.Mode == ModeWithoutIdempotency && processedCount > 1)

	publish(fmt.Sprintf("🏁 Simulasi selesai (%d ms). Total Pemotongan Nyata: $%0.2f | Cached Replays: %d | 409 Conflicts: %d",
		duration, float64(totalDeducted)/100, cachedReplays, inFlightConflicts))

	return SimulationResult{
		Mode:              cfg.Mode,
		Key:               cfg.Key,
		TotalRequests:     cfg.RetriesCount,
		ProcessedCount:    int(processedCount),
		CachedReplays:     int(cachedReplays),
		InFlightConflicts: int(inFlightConflicts),
		PayloadMismatches: int(payloadMismatches),
		InitialBalance:    initialBalance,
		FinalBalance:      finalBalance,
		TotalDeducted:     totalDeducted,
		HasDoubleSpending: hasDoubleSpending,
		DurationMs:        duration,
	}
}
