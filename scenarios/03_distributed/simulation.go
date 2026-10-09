package distributed

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// OutboxStatus mencatat status event dalam tabel Transactional Outbox.
type OutboxStatus string

const (
	OutboxStatusPending   OutboxStatus = "PENDING"
	OutboxStatusPublished OutboxStatus = "PUBLISHED"
	OutboxStatusFailed    OutboxStatus = "FAILED"
)

// OutboxEvent merepresentasikan rekaman event outbox yang disimpan atomik dalam 1 DB transaction.
type OutboxEvent struct {
	ID            string       `json:"id"`
	TransactionID string       `json:"transaction_id"`
	EventType     string       `json:"event_type"` // e.g. "WITHDRAWAL_INITIATED"
	Payload       string       `json:"payload"`
	Status        OutboxStatus `json:"status"`
	CreatedAt     time.Time    `json:"created_at"`
	PublishedAt   *time.Time   `json:"published_at,omitempty"`
}

type SimulationMode string

const (
	ModeDualWriteHazard    SimulationMode = "dual_write_hazard"    // Bahaya Dual-Write: gateway gagal, saldo tidak kembali
	ModeSagaCompensating   SimulationMode = "saga_compensating"   // Saga: gateway gagal, otomatis auto-refund kompensasi
	ModeSagaSuccessPath    SimulationMode = "saga_success"        // Saga Sukses: gateway sukses, outbox published
)

type SimulationConfig struct {
	Mode           SimulationMode
	AmountCents    int64
	BankTarget     string
	InjectFailure  bool
	FailureType    string // "500_INTERNAL_ERROR" atau "TIMEOUT"
}

type SimulationResult struct {
	Mode               SimulationMode `json:"mode"`
	TransactionID      string         `json:"transaction_id"`
	InitialBalance     int64          `json:"initial_balance"`
	FinalBalance       int64          `json:"final_balance"`
	AmountCents        int64          `json:"amount_cents"`
	LocalDebitSuccess  bool           `json:"local_debit_success"`
	OutboxRecorded     bool           `json:"outbox_recorded"`
	GatewaySuccess     bool           `json:"gateway_success"`
	CompensationRan    bool           `json:"compensation_ran"`
	CompensatedAmount  int64          `json:"compensated_amount"`
	HasFinancialLoss   bool           `json:"has_financial_loss"` // Nasabah rugi uang
	FinalStatus        string         `json:"final_status"`
	DurationMs         int64          `json:"duration_ms"`
}

type EventPublisher func(msg string)

type SharedWallet interface {
	GetBalance() int64
	Deduct(amount int64) bool
	Credit(amount int64)
}

// Simulator mengorkestrasi skenario distributed transactions, outbox, dan Saga kompensasi.
type Simulator struct {
	mu          sync.Mutex
	outboxStore []*OutboxEvent
}

func NewSimulator() *Simulator {
	return &Simulator{
		outboxStore: make([]*OutboxEvent, 0),
	}
}

func (s *Simulator) GetOutboxEvents() []*OutboxEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	copied := make([]*OutboxEvent, len(s.outboxStore))
	copy(copied, s.outboxStore)
	return copied
}

func (s *Simulator) ClearOutbox() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.outboxStore = make([]*OutboxEvent, 0)
}

// Run mengeksekusi simulasi Saga atau Dual-Write
func (s *Simulator) Run(ctx context.Context, cfg SimulationConfig, wallet SharedWallet, publish EventPublisher) SimulationResult {
	start := time.Now()
	txID := fmt.Sprintf("tx-wd-%d", time.Now().UnixNano()%100000)
	initialBalance := wallet.GetBalance()

	publish(fmt.Sprintf("🚀 [Skenario 03] Memulai permintaan penarikan dana $%0.2f ke %s (Mode: %s)...",
		float64(cfg.AmountCents)/100, cfg.BankTarget, cfg.Mode))

	var (
		localDebitSuccess bool
		outboxRecorded    bool
		gatewaySuccess    bool
		compensationRan   bool
		compensatedAmount int64
		hasFinancialLoss  bool
		finalStatus       string
	)

	// ================= LANGKAH 1: LOCAL TRANSACTION =================
	publish("📦 [Langkah 1/3] Menjalankan Transaksi Database Lokal (BEGIN TRANSACTION)...")
	if !wallet.Deduct(cfg.AmountCents) {
		publish("❌ [Langkah 1/3] Gagal: Saldo tidak mencukupi untuk penarikan dana.")
		return SimulationResult{
			Mode:           cfg.Mode,
			TransactionID:  txID,
			InitialBalance: initialBalance,
			FinalBalance:   wallet.GetBalance(),
			DurationMs:     time.Since(start).Milliseconds(),
			FinalStatus:    "REJECTED_INSUFFICIENT_FUNDS",
		}
	}
	localDebitSuccess = true
	publish(fmt.Sprintf("💳 [Langkah 1/3] Saldo berhasil dipotong $%0.2f. Sisa saldo saat ini: $%0.2f",
		float64(cfg.AmountCents)/100, float64(wallet.GetBalance())/100))

	// Jika memakai arsitektur Outbox: Simpan event outbox atomik dalam transaksi yang sama
	if cfg.Mode != ModeDualWriteHazard {
		outboxEvent := &OutboxEvent{
			ID:            fmt.Sprintf("evt-outbox-%d", time.Now().UnixNano()%100000),
			TransactionID: txID,
			EventType:     "WITHDRAWAL_INITIATED",
			Payload:       fmt.Sprintf(`{"amount":%d,"target":"%s"}`, cfg.AmountCents, cfg.BankTarget),
			Status:        OutboxStatusPending,
			CreatedAt:     time.Now(),
		}
		s.mu.Lock()
		s.outboxStore = append(s.outboxStore, outboxEvent)
		s.mu.Unlock()
		outboxRecorded = true
		publish("📝 [Langkah 1/3] Event dicatat ke tabel 'outbox_events' (Status: PENDING) dalam 1 DB Transaction (COMMIT OK).")
	} else {
		publish("⚠️ [Langkah 1/3] Pola Naif: Tanpa tabel Outbox. Mengandalkan panggilan langsung ke API eksternal...")
	}

	time.Sleep(30 * time.Millisecond) // Simulasi I/O

	// ================= LANGKAH 2: EXTERNAL PAYMENT GATEWAY CALL =================
	publish(fmt.Sprintf("🌐 [Langkah 2/3] Memanggil API Bank Eksternal (%s) untuk pencairan dana...", cfg.BankTarget))
	time.Sleep(50 * time.Millisecond) // Simulasi latency jaringan eksternal

	if cfg.InjectFailure {
		// Simulasi kegagalan pihak ketiga (500 Internal Error atau Jaringan Putus)
		publish(fmt.Sprintf("💥 [Langkah 2/3] KEGAGALAN PIHAK KETIGA: Bank Eksternal merespons '%s'!", cfg.FailureType))
		gatewaySuccess = false
	} else {
		publish(fmt.Sprintf("✅ [Langkah 2/3] Bank Eksternal merespons: 200 OK (Pencairan dana ke %s berhasil).", cfg.BankTarget))
		gatewaySuccess = true
	}

	// ================= LANGKAH 3: SAGA ORCHESTRATION & COMPENSATING TRANSACTIONS =================
	if gatewaySuccess {
		finalStatus = "COMPLETED"
		publish("🎉 [Langkah 3/3] Saga Selesai Sukses: Transaksi berstatus COMPLETED.")
		if outboxRecorded {
			s.mu.Lock()
			for _, evt := range s.outboxStore {
				if evt.TransactionID == txID {
					evt.Status = OutboxStatusPublished
					now := time.Now()
					evt.PublishedAt = &now
				}
			}
			s.mu.Unlock()
			publish("📮 [Outbox Relay] Status outbox event diperbarui menjadi 'PUBLISHED'.")
		}
	} else {
		// GATEWAY GAGAL: Penanganan tergantung arsitektur!
		if cfg.Mode == ModeDualWriteHazard {
			// Anti-pattern Dual-Write: Tanpa Saga kompensasi, saldo nasabah raib begitu saja!
			finalStatus = "FAILED_UNRECOVERED_MONEY_LOST"
			hasFinancialLoss = true
			publish("🚨 [BENCANA FINANSIAL DUAL-WRITE] Saldo nasabah sudah terpotong di DB, tetapi panggilan Bank gagal dan TIDAK ADA kompensasi!")
			publish(fmt.Sprintf("💸 Nasabah rugi $%0.2f! Uang lenyap tanpa jejak ke bank tujuan.", float64(cfg.AmountCents)/100))
		} else {
			// Saga Orchestrator: Otomatis memicu COMPENSATING TRANSACTION (Auto-Refund)
			publish("🛡️ [Langkah 3/3] SAGA ORCHESTRATOR MENDETEKSI KEGAGALAN: Memicu Transaksi Kompensasi (Compensating Transaction)...")
			wallet.Credit(cfg.AmountCents)
			compensationRan = true
			compensatedAmount = cfg.AmountCents
			finalStatus = "COMPENSATED_AUTO_REFUNDED"

			s.mu.Lock()
			for _, evt := range s.outboxStore {
				if evt.TransactionID == txID {
					evt.Status = OutboxStatusFailed
				}
			}
			s.mu.Unlock()

			publish(fmt.Sprintf("↩️ [Kompensasi Selesai] Saldo nasabah berhasil di-REFUND kembali sebesar $%0.2f!", float64(cfg.AmountCents)/100))
			publish(fmt.Sprintf("🔒 Sisa saldo akhir nasabah kembali utuh: $%0.2f. Zero data loss!", float64(wallet.GetBalance())/100))
		}
	}

	finalBalance := wallet.GetBalance()
	duration := time.Since(start).Milliseconds()

	return SimulationResult{
		Mode:              cfg.Mode,
		TransactionID:     txID,
		InitialBalance:    initialBalance,
		FinalBalance:      finalBalance,
		AmountCents:       cfg.AmountCents,
		LocalDebitSuccess: localDebitSuccess,
		OutboxRecorded:    outboxRecorded,
		GatewaySuccess:    gatewaySuccess,
		CompensationRan:   compensationRan,
		CompensatedAmount: compensatedAmount,
		HasFinancialLoss:  hasFinancialLoss,
		FinalStatus:       finalStatus,
		DurationMs:        duration,
	}
}

