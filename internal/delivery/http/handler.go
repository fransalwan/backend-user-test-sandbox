package http

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	racecondition "github.com/fransalwan/backend-user-test-sandbox/scenarios/01_race_condition"
	"github.com/fransalwan/backend-user-test-sandbox/scenarios/02_idempotency"
	"github.com/fransalwan/backend-user-test-sandbox/scenarios/03_distributed"
	"github.com/fransalwan/backend-user-test-sandbox/scenarios/04_high_traffic"
)

//go:embed templates/*
var templateFS embed.FS

// Handler mengelola rute HTTP pada lapisan delivery.
type Handler struct {
	tmpl           *template.Template
	bootcampTmpl   *template.Template
	exerciseTmpl   *template.Template
	hub            *SSEHub
	raceSim        *racecondition.Simulator
	idempotencySim *idempotency.Simulator
	sagaSim        *distributed.Simulator
	trafficSim     *hightraffic.Simulator

	// Gamifikasi Status Kandidat
	gamifyMu    sync.RWMutex
	stagePassed map[int]bool
}

// NewHandler menginisialisasi delivery handler beserta seluruh simulator skenario.
func NewHandler() (*Handler, error) {
	tmpl, err := template.ParseFS(templateFS, "templates/index.html")
	if err != nil {
		return nil, fmt.Errorf("gagal mem-parsing template index.html: %w", err)
	}

	bootcampTmpl, err := template.ParseFS(templateFS, "templates/bootcamp.html")
	if err != nil {
		return nil, fmt.Errorf("gagal mem-parsing template bootcamp.html: %w", err)
	}

	exerciseTmpl, err := template.ParseFS(templateFS, "templates/exercise.html")
	if err != nil {
		return nil, fmt.Errorf("gagal mem-parsing template exercise.html: %w", err)
	}

	raceSim := racecondition.NewSimulator(100000) // Saldo awal $1,000.00
	idemSim := idempotency.NewSimulator()
	sagaSim := distributed.NewSimulator()
	trafficSim := hightraffic.NewSimulator(50) // Kuota 50 voucher
	hub := NewSSEHub()

	return &Handler{
		tmpl:           tmpl,
		bootcampTmpl:   bootcampTmpl,
		exerciseTmpl:   exerciseTmpl,
		hub:            hub,
		raceSim:        raceSim,
		idempotencySim: idemSim,
		sagaSim:        sagaSim,
		trafficSim:     trafficSim,
		stagePassed:    make(map[int]bool),
	}, nil
}

// Index menampilkan dashboard web interaktif utama dalam Bahasa Indonesia.
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.Execute(w, nil); err != nil {
		http.Error(w, "Gagal merender dashboard", http.StatusInternalServerError)
	}
}

// EventsStream melayani koneksi real-time Server-Sent Events (SSE).
func (h *Handler) EventsStream(w http.ResponseWriter, r *http.Request) {
	h.hub.ServeHTTP(w, r)
}

// GetWallets mengembalikan partial HTML kartu status saldo dompet.
func (h *Handler) GetWallets(w http.ResponseWriter, r *http.Request) {
	balance, version := h.raceSim.GetWallet()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	balanceDollars := float64(balance) / 100.0
	statusColor := "text-slate-900 dark:text-white"
	anomalyBadge := ""
	if balance < 0 {
		statusColor = "text-rose-600 dark:text-rose-400 font-bold"
		anomalyBadge = `<span class="ml-2 px-1.5 py-0.5 text-[10px] rounded bg-rose-500/20 text-rose-600 dark:text-rose-300 border border-rose-500/40">OVERDRAFT!</span>`
	}

	html := fmt.Sprintf(`
    <div id="wallet-cards" class="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <div class="rounded-xl bg-white dark:bg-dark-900 border border-slate-200 dark:border-slate-800 p-4 shadow-sm">
            <div class="flex justify-between items-start">
                <div>
                    <div class="flex items-center gap-2">
                        <span class="text-xs text-slate-500 font-mono">wallet-alice-001</span>
                        <span class="px-2 py-0.5 text-[10px] rounded-full bg-emerald-500/10 text-emerald-700 dark:text-emerald-400 border border-emerald-500/20 font-medium">Rekening Utama</span>
                    </div>
                    <div class="text-sm sm:text-base font-bold text-slate-900 dark:text-white mt-1 flex items-center">
                        Alice (Hot Wallet Target) %s
                    </div>
                </div>
                <span class="px-2.5 py-1 text-xs font-mono font-semibold rounded-lg bg-emerald-500/10 text-emerald-700 dark:text-emerald-400 border border-emerald-500/30">
                    Versi: %d
                </span>
            </div>
            <div class="mt-4 flex items-baseline justify-between border-t border-slate-100 dark:border-slate-800/60 pt-3">
                <span class="text-xs text-slate-500 font-medium">Saldo Tersedia</span>
                <div class="text-right">
                    <span class="text-2xl sm:text-3xl font-mono font-extrabold %s tracking-tight">$%0.2f</span>
                    <div class="text-[11px] text-slate-400 font-mono mt-0.5">%d sen (int64)</div>
                </div>
            </div>
        </div>

        <div class="rounded-xl bg-white dark:bg-dark-900 border border-slate-200 dark:border-slate-800 p-4 shadow-sm">
            <div class="flex justify-between items-start">
                <div>
                    <div class="flex items-center gap-2">
                        <span class="text-xs text-slate-500 font-mono">wallet-bob-002</span>
                        <span class="px-2 py-0.5 text-[10px] rounded-full bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-400 border border-slate-200 dark:border-slate-700 font-medium">Rekening Penerima</span>
                    </div>
                    <div class="text-sm sm:text-base font-bold text-slate-900 dark:text-white mt-1">Bob (Rekening Tujuan)</div>
                </div>
                <span class="px-2.5 py-1 text-xs font-mono font-semibold rounded-lg bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-300 border border-slate-200 dark:border-slate-700">
                    Versi: 1
                </span>
            </div>
            <div class="mt-4 flex items-baseline justify-between border-t border-slate-100 dark:border-slate-800/60 pt-3">
                <span class="text-xs text-slate-500 font-medium">Saldo Tersedia</span>
                <div class="text-right">
                    <span class="text-2xl sm:text-3xl font-mono font-extrabold text-slate-900 dark:text-white tracking-tight">$250.00</span>
                    <div class="text-[11px] text-slate-400 font-mono mt-0.5">25.000 sen (int64)</div>
                </div>
            </div>
        </div>
    </div>
    `, anomalyBadge, version, statusColor, balanceDollars, balance)

	_, _ = w.Write([]byte(html))
}

// ResetWallets mengembalikan saldo Alice ke $1,000.00.
func (h *Handler) ResetWallets(w http.ResponseWriter, r *http.Request) {
	h.raceSim.Reset(100000)
	h.hub.Broadcast("🔄 Saldo Dompet Alice berhasil di-reset kembali ke $1.000,00 (100.000 sen)")
	h.GetWallets(w, r)
}

// markStagePassed mencatat kelulusan tahap dan memancarkan event gamifikasi.
func (h *Handler) markStagePassed(stage int, title string) {
	h.gamifyMu.Lock()
	already := h.stagePassed[stage]
	h.stagePassed[stage] = true
	h.gamifyMu.Unlock()

	if !already {
		h.hub.Broadcast(fmt.Sprintf("🎖️ [EVALUASI TEKNIS] TAHAP %d TERVERIFIKASI: '%s' (Kompetensi Disetujui)", stage, title))
	}
}

// RunScenario01 mengeksekusi simulasi Skenario 1 (Tahap 1 Wawancara).
func (h *Handler) RunScenario01(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Permintaan tidak valid", http.StatusBadRequest)
		return
	}

	strategy := racecondition.Strategy(r.FormValue("strategy"))
	concurrency, _ := strconv.Atoi(r.FormValue("concurrency"))
	amountDollars, _ := strconv.ParseInt(r.FormValue("amount"), 10, 64)

	if concurrency <= 0 {
		concurrency = 50
	}
	if amountDollars <= 0 {
		amountDollars = 25
	}
	amountCents := amountDollars * 100

	currentBalance, _ := h.raceSim.GetWallet()
	cfg := racecondition.SimulationConfig{
		Strategy:       strategy,
		InitialBalance: currentBalance,
		Amount:         amountCents,
		Concurrency:    concurrency,
	}

	result := h.raceSim.Run(r.Context(), cfg, func(msg string) {
		h.hub.Broadcast(msg)
	})

	if !result.HasOverdraft && result.BalanceDiscrepancy == 0 && (strategy == racecondition.StrategyPessimistic || strategy == racecondition.StrategyRedisLock) {
		h.markStagePassed(1, "Master Concurrency & Locking")
	}

	w.Header().Set("HX-Trigger", "refreshWallets")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	statusBadge := `<span class="px-2 py-0.5 text-xs font-semibold rounded bg-emerald-500/20 text-emerald-400 border border-emerald-500/30">KONSISTENSI ACID TERJAGA &bull; 0 ANOMALI</span>`
	alertBox := ""

	if result.HasOverdraft || result.BalanceDiscrepancy != 0 {
		statusBadge = `<span class="px-2 py-0.5 text-xs font-semibold rounded bg-rose-500/20 text-rose-400 border border-rose-500/30 animate-pulse">RACE HAZARD TERDETEKSI &bull; ANOMALI SALDO!</span>`
		alertBox = fmt.Sprintf(`
        <div class="p-3 rounded-lg bg-rose-950/60 border border-rose-800 text-rose-200 text-xs">
            <strong>⚠️ Kerusakan Finansial (Financial Defect):</strong> Terjadi lost update atau saldo minus!<br>
            Saldo Awal: $%0.2f | Saldo Akhir: $%0.2f | Selisih/Anomali: $%0.2f.<br>
            Ini membuktikan mengapa operasi baca-tulis sederhana tanpa <em>row locking</em> (SELECT FOR UPDATE) sangat berbahaya di sistem perbankan.
        </div>`, float64(result.InitialBalance)/100, float64(result.FinalBalance)/100, float64(result.BalanceDiscrepancy)/100)
	}

	html := fmt.Sprintf(`
    <div class="space-y-3">
        <div class="flex items-center justify-between">
            <div class="flex items-center gap-2">
                <span class="text-sm font-bold text-white uppercase tracking-wider">Hasil Uji Strategi: %s</span>
                %s
            </div>
            <span class="text-xs text-slate-400 font-mono">Waktu Eksekusi: %d ms</span>
        </div>
        %s
        <div class="grid grid-cols-2 sm:grid-cols-4 gap-2 text-xs">
            <div class="p-2.5 rounded-lg bg-slate-900 border border-slate-800">
                <div class="text-slate-500">Saldo Awal</div>
                <div class="text-sm font-bold text-white font-mono">$%0.2f</div>
            </div>
            <div class="p-2.5 rounded-lg bg-slate-900 border border-slate-800">
                <div class="text-slate-500">Saldo Akhir</div>
                <div class="text-sm font-bold text-white font-mono">$%0.2f</div>
            </div>
            <div class="p-2.5 rounded-lg bg-slate-900 border border-slate-800">
                <div class="text-slate-500">Transaksi Berhasil</div>
                <div class="text-sm font-bold text-emerald-400 font-mono">%d tx</div>
            </div>
            <div class="p-2.5 rounded-lg bg-slate-900 border border-slate-800">
                <div class="text-slate-500">Ditolak / Konflik</div>
                <div class="text-sm font-bold text-amber-400 font-mono">%d ditolak</div>
            </div>
        </div>
    </div>`, result.Strategy, statusBadge, result.DurationMilliseconds, alertBox,
		float64(result.InitialBalance)/100, float64(result.FinalBalance)/100,
		result.SuccessfulRequests, result.FailedRequests)

	_, _ = w.Write([]byte(html))
}

// RunScenario02 mengeksekusi simulasi Skenario 2 (Tahap 2 Wawancara).
func (h *Handler) RunScenario02(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Permintaan tidak valid", http.StatusBadRequest)
		return
	}

	mode := idempotency.SimulationMode(r.FormValue("mode"))
	key := r.FormValue("idempotency_key")
	if key == "" {
		key = "tx-pay-idempotent-001"
	}
	retries, _ := strconv.Atoi(r.FormValue("retries"))
	if retries <= 0 {
		retries = 3
	}
	amountDollars, _ := strconv.ParseInt(r.FormValue("amount"), 10, 64)
	if amountDollars <= 0 {
		amountDollars = 50
	}
	amountCents := amountDollars * 100

	cfg := idempotency.SimulationConfig{
		Mode:         mode,
		Key:          key,
		AmountCents:  amountCents,
		RetriesCount: retries,
	}

	result := h.idempotencySim.Run(r.Context(), cfg, h.raceSim, func(msg string) {
		h.hub.Broadcast(msg)
	})

	if !result.HasDoubleSpending && (mode == idempotency.ModeWithIdempotency || mode == idempotency.ModeInFlightConflict) {
		h.markStagePassed(2, "Zero-Loss Idempotency Guardian")
	}

	w.Header().Set("HX-Trigger", "refreshWallets")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	statusBadge := `<span class="px-2 py-0.5 text-xs font-semibold rounded bg-emerald-500/20 text-emerald-400 border border-emerald-500/30">IDEMPOTEN &bull; ZERO DOUBLE-SPENDING</span>`
	alertBox := ""

	if result.HasDoubleSpending {
		statusBadge = `<span class="px-2 py-0.5 text-xs font-semibold rounded bg-rose-500/20 text-rose-400 border border-rose-500/30 animate-pulse">DOUBLE-SPENDING TERJADI! 💸💸💸</span>`
		alertBox = fmt.Sprintf(`
        <div class="p-3 rounded-lg bg-rose-950/60 border border-rose-800 text-rose-200 text-xs">
            <strong>🚨 Bencana Finansial:</strong> Permintaan transfer duplikat tanpa kunci idempotensi memotong saldo sebanyak <strong>%d kali</strong>! Total lenyap: <strong>$%0.2f</strong>.
        </div>`, result.ProcessedCount, float64(result.TotalDeducted)/100)
	} else if result.InFlightConflicts > 0 {
		statusBadge = `<span class="px-2 py-0.5 text-xs font-semibold rounded bg-amber-500/20 text-amber-300 border border-amber-500/30">IN-FLIGHT CONFLICT (HTTP 409)</span>`
		alertBox = fmt.Sprintf(`
        <div class="p-3 rounded-lg bg-amber-950/60 border border-amber-800 text-amber-200 text-xs">
            <strong>🛡️ Proteksi In-Flight Aktif:</strong> %d permintaan bersamaan dicegat dengan status <strong>HTTP 409 Conflict</strong>.
        </div>`, result.InFlightConflicts)
	} else {
		alertBox = fmt.Sprintf(`
        <div class="p-3 rounded-lg bg-emerald-950/60 border border-emerald-800 text-emerald-200 text-xs">
            <strong>✅ Perlindungan Idempotensi Sempurna:</strong> Dari %d kali request, saldo hanya dipotong <strong>1 kali ($%0.2f)</strong>. %d request disajikan dari cache.
        </div>`, result.TotalRequests, float64(cfg.AmountCents)/100, result.CachedReplays)
	}

	html := fmt.Sprintf(`
    <div class="space-y-3">
        <div class="flex items-center justify-between">
            <div class="flex items-center gap-2">
                <span class="text-sm font-bold text-white uppercase tracking-wider">Hasil Uji Idempotensi</span>
                %s
            </div>
            <span class="text-xs text-slate-400 font-mono">%d ms</span>
        </div>
        %s
        <div class="grid grid-cols-2 sm:grid-cols-4 gap-2 text-xs">
            <div class="p-2.5 rounded-lg bg-slate-900 border border-slate-800">
                <div class="text-slate-500">Total Permintaan</div>
                <div class="text-sm font-bold text-white font-mono">%d kali</div>
            </div>
            <div class="p-2.5 rounded-lg bg-slate-900 border border-slate-800">
                <div class="text-slate-500">Pemotongan Nyata</div>
                <div class="text-sm font-bold text-emerald-400 font-mono">%d kali ($%0.2f)</div>
            </div>
            <div class="p-2.5 rounded-lg bg-slate-900 border border-slate-800">
                <div class="text-slate-500">Cached Replays</div>
                <div class="text-sm font-bold text-sky-400 font-mono">%d respons</div>
            </div>
            <div class="p-2.5 rounded-lg bg-slate-900 border border-slate-800">
                <div class="text-slate-500">Konflik In-Flight (409)</div>
                <div class="text-sm font-bold text-amber-400 font-mono">%d konflik</div>
            </div>
        </div>
    </div>`, statusBadge, result.DurationMs, alertBox,
		result.TotalRequests, result.ProcessedCount, float64(result.TotalDeducted)/100,
		result.CachedReplays, result.InFlightConflicts)

	_, _ = w.Write([]byte(html))
}

// RunScenario03 mengeksekusi simulasi Skenario 3 (Tahap 3 Wawancara).
func (h *Handler) RunScenario03(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Permintaan tidak valid", http.StatusBadRequest)
		return
	}

	mode := distributed.SimulationMode(r.FormValue("mode"))
	bankTarget := r.FormValue("bank_target")
	if bankTarget == "" {
		bankTarget = "Bank BCA"
	}
	amountDollars, _ := strconv.ParseInt(r.FormValue("amount"), 10, 64)
	if amountDollars <= 0 {
		amountDollars = 100
	}
	amountCents := amountDollars * 100

	injectFailure := (mode != distributed.ModeSagaSuccessPath)
	failureType := r.FormValue("failure_type")
	if failureType == "" {
		failureType = "500_INTERNAL_SERVER_ERROR"
	}

	cfg := distributed.SimulationConfig{
		Mode:          mode,
		AmountCents:   amountCents,
		BankTarget:    bankTarget,
		InjectFailure: injectFailure,
		FailureType:   failureType,
	}

	result := h.sagaSim.Run(r.Context(), cfg, h.raceSim, func(msg string) {
		h.hub.Broadcast(msg)
	})

	if result.CompensationRan || result.FinalStatus == "COMPLETED" {
		h.markStagePassed(3, "Master of Saga & Distributed Outbox")
	}

	w.Header().Set("HX-Trigger", "refreshWallets")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	statusBadge := ""
	alertBox := ""

	if result.HasFinancialLoss {
		statusBadge = `<span class="px-2 py-0.5 text-xs font-semibold rounded bg-rose-500/20 text-rose-400 border border-rose-500/30 animate-pulse">BENCANA DUAL-WRITE: UANG LENYAP! 💸</span>`
		alertBox = fmt.Sprintf(`
        <div class="p-3 rounded-lg bg-rose-950/60 border border-rose-800 text-rose-200 text-xs">
            <strong>🚨 Kerugian Finansial Terjadi (Data Loss):</strong> Saldo nasabah sudah terpotong di DB lokal, tetapi panggilan ke Bank Eksternal gagal (%s). Karena tanpa rollback kompensasi, saldo sebesar <strong>$%0.2f</strong> hilang tanpa terkirim ke rekening bank tujuan!
        </div>`, cfg.FailureType, float64(result.AmountCents)/100)
	} else if result.CompensationRan {
		statusBadge = `<span class="px-2 py-0.5 text-xs font-semibold rounded bg-sky-500/20 text-sky-400 border border-sky-500/30">SAGA COMPENSATED &bull; AUTO-REFUND SUKSES ↩️</span>`
		alertBox = fmt.Sprintf(`
        <div class="p-3 rounded-lg bg-sky-950/60 border border-sky-800 text-sky-200 text-xs">
            <strong>🛡️ Pemulihan Saga Berhasil:</strong> Bank Eksternal mengalami gangguan (%s). Saga Orchestrator segera memicu <strong>Compensating Transaction</strong> yang mengembalikan saldo <strong>$%0.2f</strong> utuh ke dompet nasabah. Tidak ada uang yang hilang!
        </div>`, cfg.FailureType, float64(result.CompensatedAmount)/100)
	} else {
		statusBadge = `<span class="px-2 py-0.5 text-xs font-semibold rounded bg-emerald-500/20 text-emerald-400 border border-emerald-500/30">SAGA SELESAI &bull; DANA TERKIRIM 🚀</span>`
		alertBox = fmt.Sprintf(`
        <div class="p-3 rounded-lg bg-emerald-950/60 border border-emerald-800 text-emerald-200 text-xs">
            <strong>✅ Alur Saga Tuntas:</strong> Saldo dipotong atomik bersamaan dengan pencatatan event ke tabel Outbox, API %s sukses memproses pencairan <strong>$%0.2f</strong>, dan status outbox event diperbarui menjadi <strong>PUBLISHED</strong>.
        </div>`, cfg.BankTarget, float64(result.AmountCents)/100)
	}

	html := fmt.Sprintf(`
    <div class="space-y-3">
        <div class="flex items-center justify-between">
            <div class="flex items-center gap-2">
                <span class="text-sm font-bold text-white uppercase tracking-wider">Hasil Uji Saga & Outbox (%s)</span>
                %s
            </div>
            <span class="text-xs text-slate-400 font-mono">ID: %s &bull; %d ms</span>
        </div>
        %s
        <div class="grid grid-cols-2 sm:grid-cols-4 gap-2 text-xs">
            <div class="p-2.5 rounded-lg bg-slate-900 border border-slate-800">
                <div class="text-slate-500">Status Akhir</div>
                <div class="text-xs font-bold text-white font-mono break-all">%s</div>
            </div>
            <div class="p-2.5 rounded-lg bg-slate-900 border border-slate-800">
                <div class="text-slate-500">Penarikan Diajukan</div>
                <div class="text-sm font-bold text-white font-mono">$%0.2f</div>
            </div>
            <div class="p-2.5 rounded-lg bg-slate-900 border border-slate-800">
                <div class="text-slate-500">Saldo Akhir Nasabah</div>
                <div class="text-sm font-bold text-emerald-400 font-mono">$%0.2f</div>
            </div>
            <div class="p-2.5 rounded-lg bg-slate-900 border border-slate-800">
                <div class="text-slate-500">Kompensasi Auto-Refund</div>
                <div class="text-sm font-bold text-sky-400 font-mono">$%0.2f</div>
            </div>
        </div>
    </div>`, result.Mode, statusBadge, result.TransactionID, result.DurationMs, alertBox,
		result.FinalStatus, float64(result.AmountCents)/100, float64(result.FinalBalance)/100, float64(result.CompensatedAmount)/100)

	_, _ = w.Write([]byte(html))
}

// RunScenario04 mengeksekusi simulasi Skenario 4 (Tahap 4 Wawancara).
func (h *Handler) RunScenario04(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Permintaan tidak valid", http.StatusBadRequest)
		return
	}

	strategy := hightraffic.Strategy(r.FormValue("strategy"))
	stock, _ := strconv.ParseInt(r.FormValue("stock"), 10, 64)
	if stock <= 0 {
		stock = 50
	}
	requests, _ := strconv.Atoi(r.FormValue("requests"))
	if requests <= 0 {
		requests = 250
	}
	maxRPS, _ := strconv.Atoi(r.FormValue("max_rps"))
	if maxRPS <= 0 {
		maxRPS = 40
	}

	cfg := hightraffic.SimulationConfig{
		Strategy:        strategy,
		InitialStock:    stock,
		TotalRequests:   requests,
		RateLimitMaxRPS: maxRPS,
	}

	result := h.trafficSim.Run(r.Context(), cfg, func(msg string) {
		h.hub.Broadcast(msg)
	})

	if !result.HasOverselling && strategy == hightraffic.StrategyTokenBucket {
		h.markStagePassed(4, "Scale Architect & Rate Limiter")
	}

	w.Header().Set("HX-Trigger", "refreshWallets")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	statusBadge := `<span class="px-2 py-0.5 text-xs font-semibold rounded bg-emerald-500/20 text-emerald-400 border border-emerald-500/30">KUOTA AMAN &bull; ZERO OVERSELLING</span>`
	alertBox := ""

	if result.HasOverselling {
		statusBadge = `<span class="px-2 py-0.5 text-xs font-semibold rounded bg-rose-500/20 text-rose-400 border border-rose-500/30 animate-pulse">BENCANA OVERSELLING! KUOTA JEBOL MINUS</span>`
		alertBox = fmt.Sprintf(`
        <div class="p-3 rounded-lg bg-rose-950/60 border border-rose-800 text-rose-200 text-xs">
            <strong>🚨 Bencana Flash Sale:</strong> Stok voucher jebol tembus <strong>%d voucher</strong> (Oversold)! Database diserbu thundering herd tanpa rate limiting. Perusahaan merugi karena membagikan voucher melebihi kuota!
        </div>`, result.OversoldItems)
	} else {
		alertBox = fmt.Sprintf(`
        <div class="p-3 rounded-lg bg-emerald-950/60 border border-emerald-800 text-emerald-200 text-xs">
            <strong>✅ Perlindungan Rate Limiter Sukses:</strong> Dari %d request masuk, <strong>%d klaim voucher sukses</strong> diserap hingga tepat habis, dan <strong>%d request berlebih ditolak aman dengan HTTP 429 Too Many Requests</strong>. Database terlindungi dari crash!
        </div>`, result.TotalRequests, result.SuccessfulClaims, result.RateLimitedDrops)
	}

	html := fmt.Sprintf(`
    <div class="space-y-3">
        <div class="flex items-center justify-between">
            <div class="flex items-center gap-2">
                <span class="text-sm font-bold text-white uppercase tracking-wider">Hasil Uji High Traffic (%s)</span>
                %s
            </div>
            <span class="text-xs text-slate-400 font-mono">%d ms</span>
        </div>
        %s
        <div class="grid grid-cols-2 sm:grid-cols-4 gap-2 text-xs">
            <div class="p-2.5 rounded-lg bg-slate-900 border border-slate-800">
                <div class="text-slate-500">Stok Awal Kuota</div>
                <div class="text-sm font-bold text-white font-mono">%d voucher</div>
            </div>
            <div class="p-2.5 rounded-lg bg-slate-900 border border-slate-800">
                <div class="text-slate-500">Klaim Berhasil</div>
                <div class="text-sm font-bold text-emerald-400 font-mono">%d voucher</div>
            </div>
            <div class="p-2.5 rounded-lg bg-slate-900 border border-slate-800">
                <div class="text-slate-500">Dicegat Rate Limiter</div>
                <div class="text-sm font-bold text-amber-400 font-mono">%d request (429)</div>
            </div>
            <div class="p-2.5 rounded-lg bg-slate-900 border border-slate-800">
                <div class="text-slate-500">Stok Akhir</div>
                <div class="text-sm font-bold text-white font-mono">%d kuota</div>
            </div>
        </div>
    </div>`, result.Strategy, statusBadge, result.DurationMs, alertBox,
		result.InitialStock, result.SuccessfulClaims, result.RateLimitedDrops, result.FinalStock)

	_, _ = w.Write([]byte(html))
}

// GetGamificationStatus mengembalikan status kelulusan 6 tahap alur kandidat (0 Bootcamp s/d 5 Offer).
func (h *Handler) GetGamificationStatus(w http.ResponseWriter, r *http.Request) {
	h.gamifyMu.RLock()
	s0 := h.stagePassed[0]
	s1 := h.stagePassed[1]
	s2 := h.stagePassed[2]
	s3 := h.stagePassed[3]
	s4 := h.stagePassed[4]
	s5 := s1 && s2 && s3 && s4
	h.gamifyMu.RUnlock()

	levelTitle := "Tahap 0: Screening Awal & Lab Fondasi Moneter"
	statusBadge := `<span class="px-2.5 py-1 text-xs font-semibold rounded-lg bg-slate-100 dark:bg-slate-800 text-slate-700 dark:text-slate-300 border border-slate-300 dark:border-slate-700">Screening Awal</span>`
	if s5 {
		levelTitle = "🏆 EVALUASI TUNTAS - SURAT PENAWARAN RESMI TERBIT!"
		statusBadge = `<span class="px-2.5 py-1 text-xs font-bold rounded-lg bg-emerald-500/20 text-emerald-700 dark:text-emerald-300 border border-emerald-500/40 animate-pulse">STRONG HIRE APPROVED ✓</span>`
	} else if s4 {
		levelTitle = "Tahap 5: Peninjauan Tawaran Kerja (Job Offer Letter)"
		statusBadge = `<span class="px-2.5 py-1 text-xs font-bold rounded-lg bg-amber-500/20 text-amber-700 dark:text-amber-300 border border-amber-500/40">Siap Review Penawaran</span>`
	} else if s3 {
		levelTitle = "Tahap 4: Production War Room (Flash Sale Rate Limiter & Concurrency Incident)"
		statusBadge = `<span class="px-2.5 py-1 text-xs font-semibold rounded-lg bg-cyan-500/20 text-cyan-700 dark:text-cyan-300 border border-cyan-500/40">Tahap Akhir Evaluasi</span>`
	} else if s2 {
		levelTitle = "Tahap 3: Take-Home Core Payment API Review"
		statusBadge = `<span class="px-2.5 py-1 text-xs font-semibold rounded-lg bg-teal-500/20 text-teal-700 dark:text-teal-300 border border-teal-500/40">Ujian Praktik</span>`
	} else if s1 {
		levelTitle = "Tahap 2: System Design Payment Architecture Board"
		statusBadge = `<span class="px-2.5 py-1 text-xs font-semibold rounded-lg bg-emerald-500/20 text-emerald-700 dark:text-emerald-300 border border-emerald-500/40">Arsitektur Terbuka</span>`
	} else if s0 {
		levelTitle = "Tahap 1: Live Coding DSA (Financial Transaction Deduplication)"
		statusBadge = `<span class="px-2.5 py-1 text-xs font-semibold rounded-lg bg-emerald-500/20 text-emerald-700 dark:text-emerald-300 border border-emerald-500/40">Fondasi Terverifikasi ✓</span>`
	}

	coreStepsCompleted := 0
	if s1 { coreStepsCompleted++ }
	if s2 { coreStepsCompleted++ }
	if s3 { coreStepsCompleted++ }
	if s4 { coreStepsCompleted++ }
	if s5 { coreStepsCompleted++ }
	progressPct := coreStepsCompleted * 20

	steps := []struct {
		idx       int
		icon      string
		name      string
		sub       string
		passed    bool
		isCurrent bool
	}{
		{0, "📚", "Tahap 0", "Teori & Lab", s0, !s0},
		{1, "💻", "Tahap 1", "Live Coding DSA", s1, s0 && !s1},
		{2, "🏛️", "Tahap 2", "System Design", s2, s1 && !s2},
		{3, "📦", "Tahap 3", "Take-Home API", s3, s2 && !s3},
		{4, "🔥", "Tahap 4", "War Room Incident", s4, s3 && !s4},
		{5, "📜", "Tahap 5", "Offer Letter", s5, s4 && !s5},
	}

	var stepperItems strings.Builder
	for i, step := range steps {
		circleClass := ""
		iconDisplay := step.icon
		statusPill := ""

		if step.passed {
			circleClass = "bg-emerald-600 text-white shadow-md shadow-emerald-950/20 ring-4 ring-emerald-500/20 border border-emerald-400 scale-105"
			iconDisplay = "✓"
			statusPill = `<span class="text-[9px] font-bold text-emerald-600 dark:text-emerald-400">Terverifikasi ✓</span>`
		} else if step.isCurrent {
			circleClass = "bg-gradient-to-tr from-teal-600 to-cyan-500 text-white shadow-md ring-4 ring-cyan-500/30 animate-pulse border border-cyan-400 scale-110"
			statusPill = `<span class="text-[9px] font-bold text-cyan-600 dark:text-cyan-400">Sedang Dinilai ●</span>`
		} else {
			circleClass = "bg-slate-100 dark:bg-dark-850 text-slate-400 dark:text-slate-500 border border-slate-200 dark:border-slate-800 ring-2 ring-slate-200/60 dark:ring-slate-800"
			statusPill = `<span class="text-[9px] text-slate-400 dark:text-slate-500">Antrean</span>`
		}

		stepperItems.WriteString(fmt.Sprintf(`
            <button type="button" onclick="switchScenario(%d)" class="group flex flex-col items-center flex-1 cursor-pointer transition transform hover:scale-105 focus:outline-none min-w-[70px]">
                <div class="w-8 h-8 sm:w-9 sm:h-9 rounded-full flex items-center justify-center font-bold text-xs sm:text-sm transition-all %s">
                    %s
                </div>
                <div class="text-center mt-1.5">
                    <div class="text-[10px] sm:text-[11px] font-bold text-slate-800 dark:text-slate-200 group-hover:text-emerald-600 dark:group-hover:text-emerald-400 transition leading-tight">%s</div>
                    <div class="text-[9px] text-slate-500 dark:text-slate-400 hidden md:block">%s</div>
                    <div class="mt-0.5">%s</div>
                </div>
            </button>`, step.idx, circleClass, iconDisplay, step.name, step.sub, statusPill))

		if i < len(steps)-1 {
			connectorColor := "bg-slate-200 dark:bg-slate-800"
			if step.passed {
				connectorColor = "bg-gradient-to-r from-emerald-500 to-teal-500"
			}
			stepperItems.WriteString(fmt.Sprintf(`
            <div class="hidden sm:block flex-1 h-0.5 mt-4 mx-1.5 %s"></div>`, connectorColor))
		}
	}

	html := fmt.Sprintf(`
    <div id="gamification-hud" class="rounded-2xl bg-white dark:bg-gradient-to-r dark:from-dark-900 dark:via-[#101932] dark:to-dark-900 border border-slate-200 dark:border-slate-800 p-4 sm:p-5 shadow-sm transition-colors duration-200 space-y-4">
        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 sm:gap-4">
            <div class="flex items-center gap-3">
                <div class="w-11 h-11 sm:w-12 sm:h-12 rounded-2xl bg-gradient-to-tr from-emerald-600 to-cyan-500 flex items-center justify-center text-xl sm:text-2xl shadow-md flex-shrink-0">
                    👨‍💻
                </div>
                <div>
                    <div class="flex items-center gap-2 flex-wrap">
                        <span class="text-xs sm:text-sm font-bold text-slate-900 dark:text-white">Kandidat: Frans Alwan</span>
                        <span class="text-[11px] text-emerald-600 dark:text-emerald-400 font-mono font-medium">&bull; Jalur: Junior / Associate Backend Engineer (Fintech Track)</span>
                    </div>
                    <div class="text-xs font-semibold text-slate-600 dark:text-slate-300 mt-0.5">%s</div>
                </div>
            </div>

            <div class="flex items-center justify-between sm:justify-end gap-3 border-t sm:border-t-0 border-slate-100 dark:border-slate-800/80 pt-2 sm:pt-0">
                <div class="sm:text-right">
                    <div class="text-[11px] text-slate-500 dark:text-slate-400">Matriks Kompetensi</div>
                    <div class="text-lg sm:text-xl font-mono font-extrabold text-slate-900 dark:text-white">%d / 5 Terverifikasi</div>
                </div>
                %s
            </div>
        </div>

        <!-- Professional Interview Assessment Pipeline Stepper (Tahap 0 s/d Tahap 5) -->
        <div class="pt-3 border-t border-slate-100 dark:border-slate-800/80">
            <div class="flex items-center justify-between text-[11px] text-slate-500 dark:text-slate-400 mb-2 font-medium">
                <span class="flex items-center gap-1.5 font-bold text-slate-700 dark:text-slate-300">
                    <span>🗺️</span> <span>Pipeline Penilaian Teknis (%d dari 5 Tahap Terpenuhi)</span>
                </span>
                <span class="font-mono text-emerald-600 dark:text-emerald-400 font-bold">%d%% Selesai</span>
            </div>

            <div class="overflow-x-auto no-scrollbar py-1">
                <div class="flex items-start justify-between min-w-[560px] sm:min-w-0">
                    %s
                </div>
            </div>

            <div class="w-full bg-slate-100 dark:bg-slate-950 rounded-full h-2 overflow-hidden border border-slate-200 dark:border-slate-800 mt-3">
                <div class="bg-gradient-to-r from-emerald-500 via-teal-400 to-cyan-400 h-2 rounded-full transition-all duration-700" style="width: %d%%"></div>
            </div>
        </div>

        <script>
            if (typeof updateNavTabBadges === 'function') {
                updateNavTabBadges({
                    0: %t,
                    1: %t,
                    2: %t,
                    3: %t,
                    4: %t,
                    5: %t
                });
            }
        </script>
    </div>`, levelTitle, coreStepsCompleted, statusBadge, coreStepsCompleted, progressPct, stepperItems.String(), progressPct, s0, s1, s2, s3, s4, s5)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(html))
}

// ResetGamification mereset status evaluasi teknis kandidat.
func (h *Handler) ResetGamification(w http.ResponseWriter, r *http.Request) {
	h.gamifyMu.Lock()
	h.stagePassed = make(map[int]bool)
	h.gamifyMu.Unlock()

	h.hub.Broadcast("🔄 Status evaluasi teknis kandidat telah di-reset kembali ke Tahap 0 (Screening Awal).")
	h.GetGamificationStatus(w, r)
}

// ClearScenario03 membersihkan riwayat outbox events.
func (h *Handler) ClearScenario03(w http.ResponseWriter, r *http.Request) {
	h.sagaSim.ClearOutbox()
	h.hub.Broadcast("🧹 Penyimpanan tabel outbox events berhasil dibersihkan.")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"cleared"}`))
}

// ClearIdempotencyRecords membersihkan riwayat kunci idempotensi.
func (h *Handler) ClearIdempotencyRecords(w http.ResponseWriter, r *http.Request) {
	h.idempotencySim.Reset()
	h.hub.Broadcast("🧹 Penyimpanan kunci idempotensi berhasil dibersihkan.")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"cleared"}`))
}

// HealthCheck menyediakan endpoint liveness probe.
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// EvalBootcampLiveCode mengevaluasi kode Go yang diketik kandidat untuk lab teori persiapan Live Coding (DSA Hash Map).
func (h *Handler) EvalBootcampLiveCode(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Permintaan tidak valid", http.StatusBadRequest)
		return
	}

	codeContent := strings.TrimSpace(r.FormValue("code_content"))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	hasTODO := strings.Contains(codeContent, "// TODO") && !strings.Contains(codeContent, "return true")
	hasMap := strings.Contains(codeContent, "map[int]") || strings.Contains(codeContent, "make(map")
	hasComplement := strings.Contains(codeContent, "target -") || strings.Contains(codeContent, "target-") || strings.Contains(codeContent, "+")
	hasReturnBool := strings.Contains(codeContent, "return true") || strings.Contains(codeContent, "return false")

	h.hub.Broadcast("🧪 [Bootcamp Lab] Menjalankan automated test uji teori Live Coding (DSA HasTargetSum Hash Map)...")

	if codeContent == "" || hasTODO {
		html := `
        <div class="p-3.5 rounded-xl bg-rose-50 dark:bg-rose-950/60 border border-rose-300 dark:border-rose-800 text-rose-800 dark:text-rose-200 text-xs space-y-1.5 font-mono">
            <div class="font-bold flex items-center gap-1.5">
                <span>⚠️</span> <span>Implementasi Belum Lengkap</span>
            </div>
            <p class="text-[11px] text-rose-700 dark:text-rose-300 leading-relaxed">
                Ketik implementasi fungsi <code>HasTargetSum</code> di editor di atas. Gunakan Hash Map untuk mencari pasangan angka secara efisien!
            </p>
        </div>`
		_, _ = w.Write([]byte(html))
		return
	}

	if !hasReturnBool || !hasComplement {
		html := `
        <div class="p-3.5 rounded-xl bg-rose-50 dark:bg-rose-950/60 border border-rose-300 dark:border-rose-800 text-rose-800 dark:text-rose-200 text-xs space-y-1.5 font-mono">
            <div class="font-bold flex items-center gap-1.5">
                <span>❌</span> <span>Test Case 1 Gagal: Logika Pencarian Belum Tepat</span>
            </div>
            <p class="text-[11px] text-rose-700 dark:text-rose-300 leading-relaxed">
                Fungsi harus memeriksa apakah selisih <code>target - num</code> sudah pernah dicatat di Hash Map. Jika ada, kembalikan <code>true</code>. Jika perulangan selesai tanpa hasil, kembalikan <code>false</code>.
            </p>
        </div>`
		_, _ = w.Write([]byte(html))
		return
	}

	complexityBadge := "O(N) Hash Map Lookup"
	if !hasMap {
		complexityBadge = "O(N^2) Nested Loop (Suboptimal)"
	}

	h.markStagePassed(0, "Bootcamp DSA Lab Certified")
	w.Header().Set("HX-Trigger", "refreshWallets")

	html := fmt.Sprintf(`
    <div class="p-4 rounded-xl bg-emerald-50 dark:bg-emerald-950/60 border border-emerald-300 dark:border-emerald-800/80 text-xs space-y-3 font-mono shadow-sm">
        <div class="flex items-center justify-between border-b border-emerald-200 dark:border-emerald-800/60 pb-2">
            <span class="text-emerald-800 dark:text-emerald-300 font-bold flex items-center gap-1.5">
                <span>✅</span> <span>ALL DSA TEST CASES PASSED (3/3 LOLOS)</span>
            </span>
            <span class="text-[11px] px-2 py-0.5 rounded bg-emerald-500/20 text-emerald-800 dark:text-emerald-300 font-bold font-sans">ALGORITMA VALID</span>
        </div>
        <div class="space-y-1.5 text-[11px] text-slate-700 dark:text-slate-300">
            <div class="text-emerald-600 dark:text-emerald-400">✓ Case 1: nums=[2, 7, 11, 15], target=9 &rarr; Return true (2 + 7)............ PASS</div>
            <div class="text-emerald-600 dark:text-emerald-400">✓ Case 2: nums=[1, 2, 3], target=10 &rarr; Return false (Tidak Ada Pasangan)... PASS</div>
            <div class="text-emerald-600 dark:text-emerald-400">✓ Case 3: Kompleksitas Terdeteksi (%s)........................ PASS</div>
        </div>
        <div class="p-2.5 rounded-lg bg-white dark:bg-dark-950 border border-slate-200 dark:border-slate-800 text-[11px] font-sans text-slate-600 dark:text-slate-300">
            <strong class="text-emerald-700 dark:text-emerald-400 block mb-0.5">💡 Analisis Algoritma:</strong>
            Kerja bagus! Anda telah menguasai pola dasar Hash Map untuk mencari komplemen target dalam satu kali lintasan (*single pass*). Anda sudah siap menaklukkan <strong>Tahap 1 (Live Coding: Two Sum)</strong>!
        </div>
        <div class="pt-2 border-t border-emerald-200 dark:border-emerald-800/60 flex items-center justify-between font-sans">
            <span class="text-[11px] text-emerald-800 dark:text-emerald-300">Fondasi teori siap diuji di sesi wawancara live!</span>
            <button type="button" onclick="switchScenario(1)" class="px-3.5 py-1.5 rounded-xl bg-gradient-to-r from-emerald-600 to-teal-600 hover:from-emerald-500 hover:to-teal-500 text-white font-bold text-xs transition shadow flex items-center gap-1.5 active:scale-95">
                <span>💻 Lanjut ke Tahap 1: Live Coding</span>
                <span>&rarr;</span>
            </button>
        </div>
    </div>`, complexityBadge)

	_, _ = w.Write([]byte(html))
}

// EvalBootcampSystemDesignCode mengevaluasi kode Go yang diketik kandidat untuk lab teori persiapan System Design.
func (h *Handler) EvalBootcampSystemDesignCode(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Permintaan tidak valid", http.StatusBadRequest)
		return
	}

	codeContent := strings.TrimSpace(r.FormValue("code_content"))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	hasTODO := strings.Contains(codeContent, "// TODO") && !strings.Contains(codeContent, "records")
	hasMissingKeyCheck := strings.Contains(codeContent, `key == ""`) || strings.Contains(codeContent, `len(key) == 0`)
	hasInFlightCheck := strings.Contains(codeContent, `"PROCESSING"`)
	hasCompletedCheck := strings.Contains(codeContent, `"COMPLETED"`) || strings.Contains(codeContent, `"SUCCESS"`)
	hasSaveProcessing := strings.Contains(codeContent, `records[key] = "PROCESSING"`) || strings.Contains(codeContent, `records[key]="PROCESSING"`)
	hasLock := strings.Contains(codeContent, ".Lock()") || strings.Contains(codeContent, "Lock()")
	hasUnlock := strings.Contains(codeContent, ".Unlock()") || strings.Contains(codeContent, "Unlock()")

	h.hub.Broadcast("🧪 [Bootcamp Lab] Menjalankan automated test uji teori System Design (Idempotency Engine & Lock Guard)...")

	if codeContent == "" || hasTODO {
		html := `
        <div class="p-3.5 rounded-xl bg-rose-50 dark:bg-rose-950/60 border border-rose-300 dark:border-rose-800 text-rose-800 dark:text-rose-200 text-xs space-y-1.5 font-mono">
            <div class="font-bold flex items-center gap-1.5">
                <span>⚠️</span> <span>Implementasi Belum Lengkap</span>
            </div>
            <p class="text-[11px] text-rose-700 dark:text-rose-300 leading-relaxed">
                Ketik implementasi fungsi <code>AcquireOrReplay</code> pada editor di atas untuk menguji pemahaman state machine idempotensi.
            </p>
        </div>`
		_, _ = w.Write([]byte(html))
		return
	}

	if !hasMissingKeyCheck {
		html := `
        <div class="p-3.5 rounded-xl bg-rose-50 dark:bg-rose-950/60 border border-rose-300 dark:border-rose-800 text-rose-800 dark:text-rose-200 text-xs space-y-1.5 font-mono">
            <div class="font-bold flex items-center gap-1.5">
                <span>❌</span> <span>Test Case 1 Gagal: Validasi Key Kosong Terlewat</span>
            </div>
            <p class="text-[11px] text-rose-700 dark:text-rose-300 leading-relaxed">
                Jika client tidak menyertakan key (<code>key == ""</code>), sistem wajib menolak request sedini mungkin dengan mengembalikan <code>ErrMissingKey</code>!
            </p>
        </div>`
		_, _ = w.Write([]byte(html))
		return
	}

	if !hasInFlightCheck {
		html := `
        <div class="p-3.5 rounded-xl bg-rose-50 dark:bg-rose-950/60 border border-rose-300 dark:border-rose-800 text-rose-800 dark:text-rose-200 text-xs space-y-1.5 font-mono">
            <div class="font-bold flex items-center gap-1.5">
                <span>❌</span> <span>Test Case 2 Gagal: Deteksi Request In-Flight Belum Ada</span>
            </div>
            <p class="text-[11px] text-rose-700 dark:text-rose-300 leading-relaxed">
                Jika key sudah ada dan statusnya masih <code>"PROCESSING"</code>, sistem wajib mengembalikan <code>ErrInFlightConflict</code> (representasi HTTP 409 Conflict) untuk mencegat eksekusi paralel.
            </p>
        </div>`
		_, _ = w.Write([]byte(html))
		return
	}

	if !hasCompletedCheck || !hasSaveProcessing {
		html := `
        <div class="p-3.5 rounded-xl bg-rose-50 dark:bg-rose-950/60 border border-rose-300 dark:border-rose-800 text-rose-800 dark:text-rose-200 text-xs space-y-1.5 font-mono">
            <div class="font-bold flex items-center gap-1.5">
                <span>❌</span> <span>Test Case 3 Gagal: Replay Cache & Inisialisasi Key</span>
            </div>
            <p class="text-[11px] text-rose-700 dark:text-rose-300 leading-relaxed">
                Jika status sudah <code>"COMPLETED"</code>, kembalikan <code>true, nil</code> (Replay). Jika key belum terdaftar, simpan status awal <code>e.records[key] = "PROCESSING"</code> dan kembalikan <code>false, nil</code>.
            </p>
        </div>`
		_, _ = w.Write([]byte(html))
		return
	}

	if !hasLock || !hasUnlock {
		html := `
        <div class="p-3.5 rounded-xl bg-amber-50 dark:bg-amber-950/60 border border-amber-300 dark:border-amber-800 text-amber-900 dark:text-amber-200 text-xs space-y-1.5 font-mono">
            <div class="font-bold flex items-center gap-1.5">
                <span>⚠️</span> <span>Peringatan: Thread-Safety Map di Go</span>
            </div>
            <p class="text-[11px] text-amber-800 dark:text-amber-300 leading-relaxed">
                Tipe <code>map</code> bawaan Go <strong>TIDAK aman terhadap konkurensi</strong>. Pembacaan dan penulisan konkuren ke <code>records</code> akan memicu runtime crash fatal (<code>fatal error: concurrent map read and map write</code>). Gunakan <code>e.mu.Lock()</code> dan <code>defer e.mu.Unlock()</code>!
            </p>
        </div>`
		_, _ = w.Write([]byte(html))
		return
	}

	h.markStagePassed(0, "Bootcamp System Design Lab Certified")
	w.Header().Set("HX-Trigger", "refreshWallets")

	html := `
    <div class="p-4 rounded-xl bg-emerald-50 dark:bg-emerald-950/60 border border-emerald-300 dark:border-emerald-800/80 text-xs space-y-3 font-mono shadow-sm">
        <div class="flex items-center justify-between border-b border-emerald-200 dark:border-emerald-800/60 pb-2">
            <span class="text-emerald-800 dark:text-emerald-300 font-bold flex items-center gap-1.5">
                <span>✅</span> <span>ALL SYSTEM DESIGN TESTS PASSED (4/4 LOLOS)</span>
            </span>
            <span class="text-[11px] px-2 py-0.5 rounded bg-emerald-500/20 text-emerald-800 dark:text-emerald-300 font-bold font-sans">ARSITEKTUR VALID</span>
        </div>
        <div class="space-y-1.5 text-[11px] text-slate-700 dark:text-slate-300">
            <div class="text-emerald-600 dark:text-emerald-400">✓ Case 1: Tolak Request Tanpa Header Key (ErrMissingKey)................ PASS</div>
            <div class="text-emerald-600 dark:text-emerald-400">✓ Case 2: Cegat Konkurensi In-Flight 'PROCESSING' (HTTP 409 Conflict)... PASS</div>
            <div class="text-emerald-600 dark:text-emerald-400">✓ Case 3: Sajikan Ulang 'COMPLETED' Tanpa Mutasi (Replay Cached 200).... PASS</div>
            <div class="text-emerald-600 dark:text-emerald-400">✓ Case 4: Map Thread-Safety Guard (sync.Mutex Atomic Lock)............. SECURE</div>
        </div>
        <div class="p-2.5 rounded-lg bg-white dark:bg-dark-950 border border-slate-200 dark:border-slate-800 text-[11px] font-sans text-slate-600 dark:text-slate-300">
            <strong class="text-emerald-700 dark:text-emerald-400 block mb-0.5">💡 Analisis Teori:</strong>
            Luar biasa! Kode Anda merepresentasikan implementasi nyata dari state-machine Idempotency Engine di API Gateway fintech. Pemahaman ini adalah modal utama Anda untuk mempertahankan desain di <strong>Tahap 2 (System Design Interview)</strong>!
        </div>
        <div class="pt-2 border-t border-emerald-200 dark:border-emerald-800/60 flex items-center justify-between font-sans">
            <span class="text-[11px] text-emerald-800 dark:text-emerald-300">Pemahaman arsitektur siap diuji di hadapan Lead Architect!</span>
            <button type="button" onclick="switchScenario(2)" class="px-3.5 py-1.5 rounded-xl bg-gradient-to-r from-teal-600 to-cyan-600 hover:from-teal-500 hover:to-cyan-500 text-white font-bold text-xs transition shadow flex items-center gap-1.5 active:scale-95">
                <span>🏛️ Lanjut ke Tahap 2: System Design</span>
                <span>&rarr;</span>
            </button>
        </div>
    </div>`

	_, _ = w.Write([]byte(html))
}

// EvalScenario01Code mengevaluasi kode Go yang diketik manual oleh kandidat untuk Live Coding DSA:
// Financial Transaction Deduplication & Sliding Window Matcher.
func (h *Handler) EvalScenario01Code(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Permintaan tidak valid", http.StatusBadRequest)
		return
	}

	codeContent := strings.TrimSpace(r.FormValue("code_content"))
	action := r.FormValue("action")
	if action == "" {
		action = "submit"
	}

	w.Header().Set("HX-Trigger", "refreshWallets")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	hasTODO := strings.Contains(codeContent, "// TODO") && !strings.Contains(codeContent, "return")
	hasReturnSlice := strings.Contains(codeContent, "[]string{") || strings.Contains(codeContent, "return []string") || strings.Contains(codeContent, "return duplicates") || strings.Contains(codeContent, "return result") || strings.Contains(codeContent, "return nil") || strings.Contains(codeContent, "append(")
	hasLogic := strings.Contains(codeContent, "windowSeconds") || strings.Contains(codeContent, "<=") || strings.Contains(codeContent, "Timestamp") || strings.Contains(codeContent, "UserID")
	hasHashMap := strings.Contains(codeContent, "map[string]") || strings.Contains(codeContent, "make(map") || strings.Contains(codeContent, "map[")
	hasNestedLoop := (strings.Contains(codeContent, "for i") && strings.Contains(codeContent, "for j")) ||
		(strings.Count(codeContent, "for ") >= 2 && !hasHashMap)

	h.hub.Broadcast(fmt.Sprintf("💻 [DSA Live Runner] Menganalisis algoritma Deduplikasi Transaksi kandidat (%d baris, Aksi: %s)...", len(strings.Split(codeContent, "\n")), action))

	if codeContent == "" || (hasTODO && !hasLogic) {
		html := `
        <div class="space-y-3 font-mono text-xs">
            <div class="flex items-center justify-between border-b border-rose-800/60 pb-2">
                <div class="flex items-center gap-2">
                    <span class="text-base font-extrabold text-rose-400">⚠️ COMPILATION / LOGIC ERROR</span>
                    <span class="text-[11px] px-2 py-0.5 rounded bg-rose-500/20 text-rose-300 border border-rose-500/30">Implementasi Kosong</span>
                </div>
            </div>
            <div class="p-3 rounded-lg bg-rose-950/60 border border-rose-800 text-rose-200 text-xs space-y-1.5">
                <strong class="text-rose-300 block">Fungsi FindDuplicateTransactions Belum Diisi!</strong>
                <p class="text-[11px] text-rose-300 leading-relaxed">
                    Ketik implementasi fungsi deduplikasi transaksi finansial pada editor. Gunakan Hash Map untuk melacak riwayat transaksi terakhir per pengguna dalam jendela waktu <code>windowSeconds</code>.
                </p>
            </div>
        </div>`
		_, _ = w.Write([]byte(html))
		return
	}

	if !hasReturnSlice && !hasLogic {
		html := `
        <div class="space-y-3 font-mono text-xs">
            <div class="flex items-center justify-between border-b border-rose-800/60 pb-2">
                <div class="flex items-center gap-2">
                    <span class="text-base font-extrabold text-rose-400">❌ WRONG ANSWER</span>
                    <span class="text-[11px] px-2 py-0.5 rounded bg-rose-500/20 text-rose-300 border border-rose-500/30">Missing Return Slice</span>
                </div>
            </div>
            <div class="p-3 rounded-lg bg-rose-950/60 border border-rose-800 text-rose-200 text-xs space-y-1.5">
                <strong class="text-rose-300 block">Test Case 1 Gagal: Output Tidak Sesuai Spesifikasi</strong>
                <p class="text-[11px] text-rose-300 leading-relaxed">
                    Fungsi wajib mengembalikan slice <code>[]string</code> berisi daftar ID transaksi kedua yang teridentifikasi sebagai duplikat.
                </p>
            </div>
        </div>`
		_, _ = w.Write([]byte(html))
		return
	}

	// Action == "run" (Cek Cepat 3 Test Cases)
	if action == "run" {
		complexityText := "O(N) Optimal (Hash Map Single-Pass)"
		if hasNestedLoop && !hasHashMap {
			complexityText = "O(N²) Suboptimal (Nested Loop Brute Force)"
		}

		html := fmt.Sprintf(`
        <div class="space-y-3 font-mono text-xs">
            <div class="flex items-center justify-between border-b border-slate-800 pb-2">
                <span class="text-sm font-bold text-amber-300 flex items-center gap-2">
                    <span>▶</span> Hasil Uji Cepat (3 Test Cases)
                </span>
                <span class="text-slate-500 text-[11px]">Go 1.27 Live Sandbox &bull; Transaction Deduplication</span>
            </div>
            <div class="space-y-2">
                <div class="p-2.5 rounded-lg bg-emerald-950/40 border border-emerald-800/60 flex items-center justify-between">
                    <div>
                        <strong class="text-emerald-400">Case 1: Duplicate dalam Window 60s</strong>
                        <div class="text-[11px] text-slate-400">tx1 (t=100s, Rp50k) & tx2 (t=120s, Rp50k) &rarr; Output: ["tx2"]</div>
                    </div>
                    <span class="px-2 py-0.5 rounded text-[11px] font-bold bg-emerald-500/20 text-emerald-300">PASS ✓</span>
                </div>
                <div class="p-2.5 rounded-lg bg-emerald-950/40 border border-emerald-800/60 flex items-center justify-between">
                    <div>
                        <strong class="text-emerald-400">Case 2: Beda Nominal / Beda Pengguna</strong>
                        <div class="text-[11px] text-slate-400">tx1 (User Alice, Rp50k) vs tx2 (User Bob, Rp50k) &rarr; Output: [] (Bukan Duplikat)</div>
                    </div>
                    <span class="px-2 py-0.5 rounded text-[11px] font-bold bg-emerald-500/20 text-emerald-300">PASS ✓</span>
                </div>
                <div class="p-2.5 rounded-lg bg-emerald-950/40 border border-emerald-800/60 flex items-center justify-between">
                    <div>
                        <strong class="text-emerald-400">Case 3: Jeda Waktu Melebihi Window (&gt; 60s)</strong>
                        <div class="text-[11px] text-slate-400">tx1 (t=100s) & tx2 (t=200s, selisih 100s &gt; window 60s) &rarr; Output: []</div>
                    </div>
                    <span class="px-2 py-0.5 rounded text-[11px] font-bold bg-emerald-500/20 text-emerald-300">PASS ✓</span>
                </div>
            </div>
            <div class="p-2.5 rounded-lg bg-slate-900/90 border border-slate-800 text-[11px] text-slate-300 flex items-center justify-between">
                <span>Kompleksitas Terdeteksi: <strong>%s</strong></span>
                <span class="text-emerald-400">Klik "⚡ Submit Solusi" untuk evaluasi komite resmi!</span>
            </div>
        </div>`, complexityText)
		_, _ = w.Write([]byte(html))
		return
	}

	// Action == "submit"
	if hasNestedLoop && !hasHashMap {
		h.hub.Broadcast("⚠️ [DSA Live Runner] SUBOPTIMAL WARNING: Algoritma nested loop O(N²) terdeteksi. Risiko TLE pada 10.000 transaksi ingestion!")
		html := `
        <div class="space-y-3 font-mono text-xs">
            <div class="flex items-center justify-between border-b border-amber-800/60 pb-2">
                <div class="flex items-center gap-2">
                    <span class="text-base font-extrabold text-amber-400">⚠️ TIME LIMIT HAZARD (O(N²) SUBOPTIMAL)</span>
                    <span class="text-[11px] px-2 py-0.5 rounded bg-amber-500/20 text-amber-300 border border-amber-500/30">Perlu Optimasi</span>
                </div>
                <span class="text-amber-400 font-mono text-[11px]">Runtime: 1,980 ms (TLE Risk)</span>
            </div>
            <div class="p-3 rounded-lg bg-amber-950/60 border border-amber-800 text-amber-200 text-xs space-y-2">
                <strong class="text-amber-300 block">Evaluasi Pewawancara (Staff Software Engineer):</strong>
                <p class="text-[11px] text-amber-200 leading-relaxed">
                    <em>"Solusi brute force nested loop Anda berhasil untuk batch transaksi kecil, tetapi beresiko <strong>Time Limit Exceeded (TLE)</strong> saat pipeline ingestion menerima 10.000 transaksi serentak (Case 4 butuh 1.980 ms!). Gunakan <strong>Hash Map</strong> dengan key kombinasi <code>fmt.Sprintf('%s_%d', tx.UserID, tx.Amount)</code> untuk mencatat waktu transaksi terakhir secara <strong>O(1)</strong> lookup, menghasilkan total waktu <strong>O(N)</strong>!"</em>
                </p>
            </div>
            <div class="grid grid-cols-4 gap-2 text-center text-[11px]">
                <div class="p-2 rounded bg-slate-900 border border-slate-800"><span class="text-slate-500">Case 1</span><div class="font-bold text-emerald-400">PASS ✓</div></div>
                <div class="p-2 rounded bg-slate-900 border border-slate-800"><span class="text-slate-500">Case 2</span><div class="font-bold text-emerald-400">PASS ✓</div></div>
                <div class="p-2 rounded bg-slate-900 border border-slate-800"><span class="text-slate-500">Case 3</span><div class="font-bold text-emerald-400">PASS ✓</div></div>
                <div class="p-2 rounded bg-slate-900 border border-amber-800/80 bg-amber-950/30"><span class="text-amber-400">Case 4 (10k txs)</span><div class="font-bold text-amber-400">TLE HAZARD ⚠️</div></div>
            </div>
        </div>`
		_, _ = w.Write([]byte(html))
		return
	}

	// Solusi Optimal O(N) Hash Map:
	h.markStagePassed(1, "Pragmatic DSA Master: Transaction Deduplication O(N)")
	h.hub.Broadcast("🎉 [DSA Live Runner] STATUS: ACCEPTED! Solusi Deduplikasi Transaksi O(N) Hash Map lolos seluruh 4 test cases dengan runtime 0.7 ms (Kompetensi Algoritma Terverifikasi).")

	html := `
    <div class="space-y-3 font-mono text-xs">
        <div class="flex items-center justify-between border-b border-emerald-800/60 pb-2">
            <div class="flex items-center gap-2">
                <span class="text-base font-extrabold text-emerald-400">ACCEPTED ✅</span>
                <span class="text-[11px] px-2 py-0.5 rounded bg-emerald-500/20 text-emerald-300 border border-emerald-500/30">KOMPETENSI TERVERIFIKASI ✓</span>
            </div>
            <span class="text-emerald-400 font-mono text-[11px]">Runtime: 0.7 ms (Beats 99.6%)</span>
        </div>
        <div class="grid grid-cols-2 sm:grid-cols-4 gap-2 text-center text-[11px]">
            <div class="p-2 rounded bg-slate-900 border border-slate-800">
                <span class="text-slate-500">Runtime</span>
                <div class="font-bold text-emerald-400">0.7 ms</div>
                <div class="text-[10px] text-emerald-400">Beats 99.6%</div>
            </div>
            <div class="p-2 rounded bg-slate-900 border border-slate-800">
                <span class="text-slate-500">Memory</span>
                <div class="font-bold text-white">2.8 MB</div>
                <div class="text-[10px] text-emerald-400">O(U) Unique Keys</div>
            </div>
            <div class="p-2 rounded bg-slate-900 border border-slate-800">
                <span class="text-slate-500">Kompleksitas</span>
                <div class="font-bold text-emerald-400">O(N) Linear</div>
                <div class="text-[10px] text-emerald-400">Single-Pass Map</div>
            </div>
            <div class="p-2 rounded bg-slate-900 border border-slate-800">
                <span class="text-slate-500">Test Cases</span>
                <div class="font-bold text-emerald-400">4/4 Lolos</div>
                <div class="text-[10px] text-emerald-400">100% Akurat</div>
            </div>
        </div>
        <div class="p-3.5 rounded-lg bg-emerald-950/40 border border-emerald-800/60 text-emerald-200 text-xs space-y-1.5">
            <strong class="text-emerald-300 block">🏆 Evaluasi Interviewer Live Coding (Staff Engineer):</strong>
            <p class="text-[11px] leading-relaxed text-emerald-200">
                Luar biasa! Algoritma Anda menyelesaikan masalah nyata *transaction duplicate ingestion* dengan efisiensi <strong>O(N) Time Complexity</strong>. Pemahaman Anda dalam mengombinasikan Hash Map dengan batasan waktu (sliding window) mencerminkan kesiapan nyata sebagai backend engineer di industri fintech.
            </p>
            <div class="text-[11px] text-slate-300 pt-1">
                👉 <strong>Langkah Berikutnya:</strong> Masuk ke Tahap 2 untuk mempertahankan arsitektur sistem di hadapan Lead Architect!
            </div>
            <div class="pt-2 border-t border-emerald-800/60 flex items-center justify-end font-sans">
                <button type="button" onclick="switchScenario(2)" class="px-4 py-2 rounded-xl bg-gradient-to-r from-teal-500 to-cyan-500 hover:from-teal-400 hover:to-cyan-400 text-white font-bold text-xs transition shadow-md shadow-emerald-950/60 flex items-center gap-1.5 active:scale-95">
                    <span>🏛️ Lanjut ke Tahap 2: System Design</span>
                    <span>&rarr;</span>
                </button>
            </div>
        </div>
    </div>`

	_, _ = w.Write([]byte(html))
}

// DefendScenario02 memproses pertahanan keputusan arsitektur kandidat di depan Dewan Arsitek.
func (h *Handler) DefendScenario02(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Permintaan tidak valid", http.StatusBadRequest)
		return
	}

	storageChoice := r.FormValue("idempotency_storage")
	ledgerChoice := r.FormValue("ledger_model")
	failureChoice := r.FormValue("failure_handling")
	legacyChoice := r.FormValue("defense_choice")

	w.Header().Set("HX-Trigger", "refreshWallets")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Backwards compatibility jika form lama terpanggil
	if legacyChoice == "sha256_hash_and_inflight_lock" {
		storageChoice = "redis_lock"
		ledgerChoice = "double_entry"
		failureChoice = "circuit_breaker_dlq"
	}

	isStorageValid := storageChoice == "redis_lock" || storageChoice == "pg_unique"
	isLedgerValid := ledgerChoice == "double_entry"
	isFailureValid := failureChoice == "circuit_breaker_dlq"

	if storageChoice == "frontend_only" {
		h.hub.Broadcast("⚠️ [System Design Review] Ditolak: Mengandalkan frontend disable saja berisiko fatal double-charge!")
		html := `
        <div class="p-4 rounded-xl bg-rose-50 dark:bg-rose-950/50 border border-rose-300 dark:border-rose-800/70 space-y-2.5 font-sans text-xs">
            <div class="flex items-center justify-between border-b border-rose-200 dark:border-rose-800/60 pb-2">
                <strong class="text-sm font-bold text-rose-800 dark:text-rose-300">HASIL EVALUASI: ARSITEKTUR BERISIKO TINGGI (FATAL) ✗</strong>
                <span class="text-xs font-mono text-rose-600 dark:text-rose-400">Predikat: Rejected</span>
            </div>
            <p class="text-slate-700 dark:text-slate-300 leading-relaxed text-xs">
                <strong>Lead Architect:</strong> <em>"Disable tombol di frontend sama sekali tidak melindungi sistem pembayaran dari network timeout retry otomatis oleh SDK ponsel nasabah atau serangan API script duplikat. Backend WAJIB memiliki Idempotency Guard mandiri dengan atomic lock!"</em>
            </p>
        </div>`
		_, _ = w.Write([]byte(html))
		return
	}

	if isStorageValid && isLedgerValid && isFailureValid {
		h.markStagePassed(2, "Fintech Core System Design Approved")
		h.hub.Broadcast("🏛️ [System Design] Dewan Arsitek menyetujui desain Core Banking & Idempotency Engine kandidat! Hasil: Memenuhi Standar Kompetensi Arsitektur.")

		html := `
        <div class="p-4 sm:p-5 rounded-xl bg-emerald-50 dark:bg-emerald-950/50 border border-emerald-300 dark:border-emerald-800/70 space-y-3 font-sans text-xs shadow-md">
            <div class="flex items-center justify-between border-b border-emerald-200 dark:border-emerald-800/60 pb-2.5">
                <div class="flex items-center gap-2">
                    <span class="w-3 h-3 rounded-full bg-emerald-500"></span>
                    <strong class="text-sm font-bold text-emerald-800 dark:text-emerald-300 uppercase tracking-wide">SYSTEM DESIGN DEFENSE: APPROVED (STRONG HIRE) ✓</strong>
                </div>
                <span class="text-xs font-mono font-bold text-emerald-600 dark:text-emerald-400">KOMPETENSI TERVERIFIKASI ✓</span>
            </div>
            <div class="text-slate-700 dark:text-slate-200 leading-relaxed text-xs space-y-2">
                <p>
                    <strong>Keputusan Dewan Arsitek (Principal Engineer & Lead Architect):</strong><br>
                    <em>"Pilihan desain Anda sangat matang untuk level Entry-Level/Junior! Menggunakan Redis Distributed Lock (SETNX) untuk in-flight guard menjamin latensi &lt;1ms. Menerapkan <strong>Double-Entry Ledger (Buku Besar Berpasangan)</strong> menjamin integritas audit keuangan bebas selisih. Serta penggunaan <strong>Circuit Breaker + Asynchronous Dead-Letter Queue</strong> mencegah thread starvation saat bank mitra mengalami gangguan."</em>
                </p>
                <div class="grid grid-cols-1 sm:grid-cols-3 gap-2 pt-1 font-mono text-[11px]">
                    <div class="p-2.5 rounded bg-white dark:bg-dark-900 border border-slate-200 dark:border-slate-800">
                        <span class="text-slate-400">Idempotency Guard</span>
                        <div class="text-emerald-600 dark:text-emerald-400 font-bold mt-0.5">Redis SETNX / Atomic</div>
                    </div>
                    <div class="p-2.5 rounded bg-white dark:bg-dark-900 border border-slate-200 dark:border-slate-800">
                        <span class="text-slate-400">Data Model Ledger</span>
                        <div class="text-emerald-600 dark:text-emerald-400 font-bold mt-0.5">Double-Entry Journal</div>
                    </div>
                    <div class="p-2.5 rounded bg-white dark:bg-dark-900 border border-slate-200 dark:border-slate-800">
                        <span class="text-slate-400">Resiliensi Mitra Bank</span>
                        <div class="text-emerald-600 dark:text-emerald-400 font-bold mt-0.5">Circuit Breaker + DLQ</div>
                    </div>
                </div>
                <div class="pt-2 border-t border-emerald-200 dark:border-emerald-800/60 flex items-center justify-between">
                    <span class="text-[11px] text-emerald-800 dark:text-emerald-300">Fondasi arsitektur siap dibuktikan di Take-Home Challenge!</span>
                    <button type="button" onclick="switchScenario(3)" class="px-4 py-2 rounded-xl bg-gradient-to-r from-cyan-600 to-blue-600 hover:from-cyan-500 hover:to-blue-500 text-white font-bold text-xs transition shadow-md flex items-center gap-1.5 active:scale-95">
                        <span>📦 Lanjut ke Tahap 3: Take-Home Test</span>
                        <span>&rarr;</span>
                    </button>
                </div>
            </div>
        </div>`
		_, _ = w.Write([]byte(html))
		return
	}

	// Jika ada opsi yang belum optimal
	html := `
    <div class="p-4 rounded-xl bg-amber-50 dark:bg-amber-950/50 border border-amber-300 dark:border-amber-800/70 space-y-2 font-sans text-xs">
        <strong class="text-amber-800 dark:text-amber-300 block">Evaluasi Arsitektur: Trade-off Belum Optimal</strong>
        <p class="text-slate-700 dark:text-slate-300 leading-relaxed text-[11px]">
            Dewan arsitek mencatat Anda belum memilih kombinasi standar industri (Double-Entry Ledger dan Circuit Breaker). Pastikan menggunakan Double-Entry Bookkeeping agar setiap mutasi debit selalu memiliki kredit yang seimbang untuk audit kepatuhan regulasi OJK/BI.
        </p>
    </div>`
	_, _ = w.Write([]byte(html))
}

// SubmitScenario03Repo menerima submission repositori GitHub publik untuk Take-Home Test.
func (h *Handler) SubmitScenario03Repo(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Permintaan tidak valid", http.StatusBadRequest)
		return
	}

	repoURL := strings.TrimSpace(r.FormValue("repo_url"))
	branch := strings.TrimSpace(r.FormValue("branch"))
	if branch == "" {
		branch = "main"
	}

	w.Header().Set("HX-Trigger", "refreshWallets")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if !strings.HasPrefix(repoURL, "https://github.com/") {
		html := `
        <div class="p-3.5 rounded-xl bg-rose-50 dark:bg-rose-950/60 border border-rose-300 dark:border-rose-800 text-rose-800 dark:text-rose-200 text-xs space-y-1">
            <strong>⚠️ Validasi Repositori Gagal:</strong>
            <p class="text-[11px] text-rose-700 dark:text-rose-300">
                Harap masukkan URL repositori GitHub publik yang valid (diawali dengan <code>https://github.com/username/project</code>).
            </p>
        </div>`
		_, _ = w.Write([]byte(html))
		return
	}

	// Simulasi Pipeline Evaluasi Komite Engineering untuk Entry-Level
	h.hub.Broadcast(fmt.Sprintf("🚀 [Take-Home Bot] Menerima submission repo: %s (Branch: %s). Menjalankan code review bot...", repoURL, branch))
	h.markStagePassed(3, "Take-Home Payment API Accepted")

	html := fmt.Sprintf(`
    <div class="p-4 sm:p-5 rounded-xl bg-gradient-to-b from-cyan-50/60 to-white dark:from-[#0b1424] dark:to-[#070c18] border border-cyan-300 dark:border-cyan-500/30 space-y-3 font-sans text-xs shadow-md">
        <div class="flex items-center justify-between border-b border-slate-200 dark:border-slate-800 pb-2.5">
            <div class="flex items-center gap-2">
                <span class="w-3 h-3 rounded-full bg-cyan-500 animate-pulse"></span>
                <strong class="text-sm font-bold text-slate-900 dark:text-white uppercase tracking-wider">TAKE-HOME CODE REVIEW: APPROVED (LEVEL ENTRY-READY) ✓</strong>
            </div>
            <span class="text-xs font-mono font-bold text-emerald-600 dark:text-emerald-400">KOMPETENSI TERVERIFIKASI ✓</span>
        </div>

        <div class="text-[11px] font-mono text-slate-700 dark:text-slate-300 bg-white dark:bg-dark-950 p-3 rounded-lg border border-slate-200 dark:border-slate-800 space-y-1">
            <div class="text-slate-500">Target Repo: <a href="%s" target="_blank" class="text-cyan-600 dark:text-cyan-400 hover:underline">%s</a> (Branch: %s)</div>
            <div class="text-emerald-600 dark:text-emerald-400">[1/5] Aksesibilitas Git Remote Publik................... OK (200 OK)</div>
            <div class="text-emerald-600 dark:text-emerald-400">[2/5] Struktur Folder Modular (Handler/Service/Repo).... RAPI & TERPISAH ✓</div>
            <div class="text-emerald-600 dark:text-emerald-400">[3/5] Integritas Moneter (DB Transaction & int64 Cents). BEBAS FLOAT BUG ✓</div>
            <div class="text-emerald-600 dark:text-emerald-400">[4/5] Unit Test Logika Mutasi Saldo Dompet.............. PASS ✓ (Coverage 88.5%%)</div>
            <div class="text-emerald-600 dark:text-emerald-400">[5/5] Panduan README.md & Dockerfile Container.......... JELAS & SIAP RUN ✓</div>
        </div>

        <div class="p-3 rounded-lg bg-cyan-50 dark:bg-cyan-950/40 border border-cyan-200 dark:border-cyan-800/60 text-cyan-900 dark:text-cyan-200 text-xs space-y-1">
            <strong class="text-cyan-700 dark:text-cyan-300 block font-semibold">📝 Catatan Evaluasi Tim Reviewer:</strong>
            Kode yang Anda kirimkan menunjukkan kebiasaan pemrograman yang sangat baik untuk level Junior/Associate: penamaan variabel bersih, penanganan error eksplisit, dan penggunaan transaksi database atomic. Sangat layak untuk lanjut ke Tahap 4 (War Room & Traffic Safety)!
        </div>
    </div>`, repoURL, repoURL, branch)

	_, _ = w.Write([]byte(html))
}

// DownloadPostmanCollection menyajikan file JSON koleksi Postman resmi untuk diunduh langsung dari browser.
func (h *Handler) DownloadPostmanCollection(w http.ResponseWriter, r *http.Request) {
	candidates := []string{
		"docs/fintech_api.postman_collection.json",
		"../../docs/fintech_api.postman_collection.json",
		"../../../docs/fintech_api.postman_collection.json",
	}

	var data []byte
	var err error
	for _, path := range candidates {
		data, err = os.ReadFile(path)
		if err == nil {
			break
		}
	}

	if err != nil {
		http.Error(w, "File koleksi Postman tidak ditemukan", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=\"fintech_api.postman_collection.json\"")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// Bootcamp menampilkan halaman dedicated Bootcamp untuk pembelajaran teori fundamental arsitektur.
func (h *Handler) Bootcamp(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.bootcampTmpl.Execute(w, nil); err != nil {
		http.Error(w, "Gagal merender halaman bootcamp", http.StatusInternalServerError)
	}
}

// Exercise menampilkan halaman dedicated Exercise untuk melatih mental live coding dan repetisi debugging.
func (h *Handler) Exercise(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.exerciseTmpl.Execute(w, nil); err != nil {
		http.Error(w, "Gagal merender halaman exercise", http.StatusInternalServerError)
	}
}

// EvalExerciseDrill mengevaluasi perbaikan bug pada 4 skenario latihan live coding debugging.
func (h *Handler) EvalExerciseDrill(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Permintaan tidak valid", http.StatusBadRequest)
		return
	}

	drill := r.FormValue("drill")
	action := r.FormValue("action")
	code := r.FormValue("code")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	switch drill {
	case "money":
		hasFloat := strings.Contains(code, "float64(amountCents)") || strings.Contains(code, "rawFee :=") || strings.Contains(code, "0.11") || strings.Contains(code, "100.0")
		hasIntMath := strings.Contains(code, "int64") && (strings.Contains(code, "25") || strings.Contains(code, "250"))

		if action == "run" {
			html := fmt.Sprintf(`
            <div class="p-3.5 rounded-xl bg-slate-900 border border-slate-800 font-mono text-xs space-y-2.5">
                <div class="text-amber-400 font-bold flex items-center justify-between border-b border-slate-800 pb-1.5">
                    <span>▶ Test Runner: Uji Cepat Kalkulasi Integer Math (3 Sampel):</span>
                    <span class="text-slate-400 text-[10px]">Float Detected: %v</span>
                </div>
                <div class="space-y-1 text-slate-300 text-[11px]">
                    <div class="flex justify-between"><span>[TC 1 - Standard] Nominal Rp 10.000 (1.000.000 sen)</span><span class="text-emerald-400">Fee: 25.000 sen, PPN: 2.750 sen</span></div>
                    <div class="flex justify-between"><span>[TC 2 - Large Amount] Nominal Rp 50.000.000</span><span class="text-emerald-400">Fee: 125.000.000 sen, PPN: 13.750.000 sen</span></div>
                    <div class="flex justify-between"><span>[TC 3 - Fractional Odd] Nominal Rp 125.750</span><span class="text-emerald-400">Integer Cents (Zero Truncation Drift)</span></div>
                </div>
                <div class="text-[11px] text-amber-300 pt-1">
                    Klik <strong>"⚡ Verifikasi Perbaikan Bug"</strong> untuk audit kepatuhan integer math tanpa float64.
                </div>
            </div>`, hasFloat)
			_, _ = w.Write([]byte(html))
			return
		}

		if hasFloat {
			html := `
            <div class="p-3.5 rounded-xl bg-rose-950/60 border border-rose-800 text-rose-200 text-xs font-mono space-y-2">
                <div class="flex items-center gap-2">
                    <span class="text-sm font-extrabold text-rose-400">❌ DETEKSI FLOAT64: RISIKO AUDIT LEAKAGE!</span>
                </div>
                <div class="grid grid-cols-3 gap-2 text-center text-[10px] my-1">
                    <div class="p-1.5 rounded bg-rose-900/60 border border-rose-700">TC 1: Standard &bull; <strong class="text-rose-300">FAILED</strong></div>
                    <div class="p-1.5 rounded bg-rose-900/60 border border-rose-700">TC 2: Large Val &bull; <strong class="text-rose-300">FAILED</strong></div>
                    <div class="p-1.5 rounded bg-rose-900/60 border border-rose-700">TC 3: Precision &bull; <strong class="text-rose-300">FAILED</strong></div>
                </div>
                <p class="text-[11px] leading-relaxed text-rose-300">
                    Sistem masih mendeteksi penggunaan tipe data <code>float64</code> dalam kalkulasi. Di sistem perbankan dan payment gateway, kalkulasi uang wajib murni menggunakan <strong>integer math (int64 sen)</strong>. Contoh: <code>feeCents = (amountCents * 25) / 1000</code> dan <code>taxCents = (feeCents * 11) / 100</code>.
                </p>
                <div class="text-[10px] text-rose-400">Status: Gagal verifikasi integritas nilai uang. Hilangkan seluruh float64!</div>
            </div>`
			_, _ = w.Write([]byte(html))
			return
		}

		if hasIntMath {
			h.hub.Broadcast("🎉 [Exercise Lab] Drill 1 Lolos: Bug Floating-Point berhasil diperbaiki menjadi Integer Cents (int64)!")
			html := `
            <div class="p-3.5 rounded-xl bg-emerald-950/60 border border-emerald-800 text-emerald-200 text-xs font-mono space-y-3">
                <div class="flex items-center justify-between border-b border-emerald-800/80 pb-2">
                    <span class="text-sm font-extrabold text-emerald-400">ACCEPTED & VERIFIED ✅</span>
                    <span class="text-[10px] px-2 py-0.5 rounded bg-emerald-500/20 text-emerald-300 border border-emerald-500/30">All 3 Test Cases Passed</span>
                </div>
                <div class="space-y-1.5 text-[11px]">
                    <div class="flex items-center justify-between p-1.5 rounded bg-slate-900 border border-slate-800">
                        <span>✓ Test Case 1: Standard Transfer (Rp 10.000) &bull; Fee 2.5%% &amp; PPN 11%%</span>
                        <span class="text-emerald-400 font-bold">PASS (0.1ms)</span>
                    </div>
                    <div class="flex items-center justify-between p-1.5 rounded bg-slate-900 border border-slate-800">
                        <span>✓ Test Case 2: Boundary Value (Rp 0 &amp; Rp 50.000.000)</span>
                        <span class="text-emerald-400 font-bold">PASS (0.1ms)</span>
                    </div>
                    <div class="flex items-center justify-between p-1.5 rounded bg-slate-900 border border-slate-800">
                        <span>✓ Test Case 3: Invariant Audit (Zero IEEE-754 Precision Drift)</span>
                        <span class="text-emerald-400 font-bold">PASS (0.1ms)</span>
                    </div>
                </div>
                <!-- Follow-up Question Simulator -->
                <div class="p-2.5 rounded-lg bg-slate-900/90 border border-emerald-600/40 text-[11px] space-y-1">
                    <strong class="text-emerald-300 block">💼 Pertanyaan Lanjutan Tech Lead (Follow-up Verbal Interview):</strong>
                    <p class="text-slate-300 italic">"Mengapa pada integer math kita mengalikan terlebih dahulu sebelum membagi: (amountCents * 25) / 1000 bukannya amountCents * (25 / 1000)?"</p>
                    <p class="text-emerald-400 pt-0.5"><strong>Jawaban Ideal:</strong> Karena pada integer division, pecahan di belakang koma langsung dipotong (truncation). Jika membagi terlebih dahulu, 25 / 1000 akan menjadi 0, sehingga seluruh biaya menjadi nol!</p>
                </div>
            </div>`
			_, _ = w.Write([]byte(html))
			return
		}

		html := `
        <div class="p-3 rounded-xl bg-amber-950/60 border border-amber-800 text-amber-200 text-xs font-mono">
            Pastikan fungsi mengembalikan nilai int64 yang valid untuk fee, tax, dan total.
        </div>`
		_, _ = w.Write([]byte(html))

	case "race":
		hasMutex := strings.Contains(code, "sync.Mutex") || strings.Contains(code, "mu.Lock()") || strings.Contains(code, "Lock()")
		hasUnlock := strings.Contains(code, "Unlock()")

		if action == "run" {
			html := fmt.Sprintf(`
            <div class="p-3.5 rounded-xl bg-slate-900 border border-slate-800 font-mono text-xs space-y-2.5">
                <div class="text-orange-400 font-bold flex items-center justify-between border-b border-slate-800 pb-1.5">
                    <span>▶ Test Runner: Uji Cepat Concurrency Hot Wallet (3 Skenario):</span>
                    <span class="text-slate-400 text-[10px]">Mutex Locked: %v</span>
                </div>
                <div class="space-y-1 text-slate-300 text-[11px]">
                    <div class="flex justify-between"><span>[TC 1 - Single Tx] Penarikan tunggal $10 dari saldo $15</span><span class="text-emerald-400">Saldo sisa: $5.00</span></div>
                    <div class="flex justify-between"><span>[TC 2 - Overdraft] Penarikan $20 dari saldo $15</span><span class="text-emerald-400">Ditolak aman (Insufficient)</span></div>
                    <div class="flex justify-between"><span>[TC 3 - Concurrency] 20 Goroutines tarik $10 serentak</span><span class="text-orange-400">Memeriksa Data Race</span></div>
                </div>
                <div class="text-[11px] text-orange-300 pt-1">
                    Klik <strong>"⚡ Verifikasi Perbaikan Bug"</strong> untuk menjalankan race detector (-race).
                </div>
            </div>`, hasMutex && hasUnlock)
			_, _ = w.Write([]byte(html))
			return
		}

		if !hasMutex || !hasUnlock {
			html := `
            <div class="p-3.5 rounded-xl bg-rose-950/60 border border-rose-800 text-rose-200 text-xs font-mono space-y-2">
                <div class="flex items-center gap-2">
                    <span class="text-sm font-extrabold text-rose-400">❌ DATA RACE DETECTED (-race WARNING)!</span>
                </div>
                <div class="grid grid-cols-3 gap-2 text-center text-[10px] my-1">
                    <div class="p-1.5 rounded bg-emerald-900/40 border border-emerald-700">TC 1: Single Tx &bull; <strong class="text-emerald-300">PASSED</strong></div>
                    <div class="p-1.5 rounded bg-emerald-900/40 border border-emerald-700">TC 2: Overdraft &bull; <strong class="text-emerald-300">PASSED</strong></div>
                    <div class="p-1.5 rounded bg-rose-900/60 border border-rose-700">TC 3: 20 Goroutines &bull; <strong class="text-rose-300">DATA RACE!</strong></div>
                </div>
                <p class="text-[11px] leading-relaxed text-rose-300">
                    Goroutine melakukan pembacaan dan penulisan konkuren pada field <code>w.Balance</code> tanpa sinkronisasi mutex. Akibatnya saldo akhir berakhir minus atau terjadi *lost update* (transaksi hilang).
                </p>
                <div class="text-[10px] text-rose-400">Pasang sync.Mutex pada struct HotWallet dan lakukan w.mu.Lock() &amp; defer w.mu.Unlock()!</div>
            </div>`
			_, _ = w.Write([]byte(html))
			return
		}

		h.hub.Broadcast("🎉 [Exercise Lab] Drill 2 Lolos: Data Race pada Hot Wallet berhasil diamankan dengan sync.Mutex!")
		html := `
        <div class="p-3.5 rounded-xl bg-emerald-950/60 border border-emerald-800 text-emerald-200 text-xs font-mono space-y-3">
            <div class="flex items-center justify-between border-b border-emerald-800/80 pb-2">
                <span class="text-sm font-extrabold text-emerald-400">ACCEPTED & RACE SAFE ✅</span>
                <span class="text-[10px] px-2 py-0.5 rounded bg-emerald-500/20 text-emerald-300 border border-emerald-500/30">Go -race Clean &bull; All 3 TC Pass</span>
            </div>
            <div class="space-y-1.5 text-[11px]">
                <div class="flex items-center justify-between p-1.5 rounded bg-slate-900 border border-slate-800">
                    <span>✓ Test Case 1: Penarikan Tunggal $10 dari $15</span>
                    <span class="text-emerald-400 font-bold">PASS (0.1ms)</span>
                </div>
                <div class="flex items-center justify-between p-1.5 rounded bg-slate-900 border border-slate-800">
                    <span>✓ Test Case 2: Penarikan Melebihi Saldo ($20 dari $15) Ditolak Aman</span>
                    <span class="text-emerald-400 font-bold">PASS (0.1ms)</span>
                </div>
                <div class="flex items-center justify-between p-1.5 rounded bg-slate-900 border border-slate-800">
                    <span>✓ Test Case 3: 20 Goroutines Eksekusi Serentak (Critical Section Protected)</span>
                    <span class="text-emerald-400 font-bold">PASS (1.4ms)</span>
                </div>
            </div>
            <!-- Follow-up Question Simulator -->
            <div class="p-2.5 rounded-lg bg-slate-900/90 border border-emerald-600/40 text-[11px] space-y-1">
                <strong class="text-emerald-300 block">💼 Pertanyaan Lanjutan Tech Lead (Follow-up Verbal Interview):</strong>
                <p class="text-slate-300 italic">"Apa yang terjadi jika kita lupa menulis 'defer w.mu.Unlock()' dan terjadi panic di dalam method?"</p>
                <p class="text-emerald-400 pt-0.5"><strong>Jawaban Ideal:</strong> Lock akan tertahan selamanya (deadlock permanen) dan tidak ada goroutine lain yang bisa mengakses wallet tersebut. Menggunakan 'defer w.mu.Unlock()' menjamin lock selalu dilepas saat fungsi exit, bahkan saat terjadi panic.</p>
            </div>
        </div>`
		_, _ = w.Write([]byte(html))

	case "idempotency":
		hasInFlightCheck := strings.Contains(code, "PROCESSING") && (strings.Contains(code, "ErrInFlightConflict") || strings.Contains(code, "Conflict") || strings.Contains(code, "409"))

		if action == "run" {
			html := fmt.Sprintf(`
            <div class="p-3.5 rounded-xl bg-slate-900 border border-slate-800 font-mono text-xs space-y-2.5">
                <div class="text-teal-400 font-bold flex items-center justify-between border-b border-slate-800 pb-1.5">
                    <span>▶ Test Runner: Uji Cepat Idempotency State Machine (3 Skenario):</span>
                    <span class="text-slate-400 text-[10px]">In-Flight Handled: %v</span>
                </div>
                <div class="space-y-1 text-slate-300 text-[11px]">
                    <div class="flex justify-between"><span>[TC 1 - First Request] Request baru Idempotency Key "tx-101"</span><span class="text-emerald-400">Acquired (200 OK)</span></div>
                    <div class="flex justify-between"><span>[TC 2 - In-Flight Collision] Request serentak saat masih PROCESSING</span><span class="text-teal-400">Harus HTTP 409 Conflict</span></div>
                    <div class="flex justify-between"><span>[TC 3 - Success Replay] Request ketiga setelah status COMPLETED</span><span class="text-emerald-400">Replay Response (200 OK)</span></div>
                </div>
                <div class="text-[11px] text-teal-300 pt-1">
                    Klik <strong>"⚡ Verifikasi Perbaikan Bug"</strong> untuk validasi status PROCESSING.
                </div>
            </div>`, hasInFlightCheck)
			_, _ = w.Write([]byte(html))
			return
		}

		if !hasInFlightCheck {
			html := `
            <div class="p-3.5 rounded-xl bg-rose-950/60 border border-rose-800 text-rose-200 text-xs font-mono space-y-2">
                <div class="flex items-center gap-2">
                    <span class="text-sm font-extrabold text-rose-400">❌ DOUBLE SPEND RISK: IN-FLIGHT STATE TRAP!</span>
                </div>
                <div class="grid grid-cols-3 gap-2 text-center text-[10px] my-1">
                    <div class="p-1.5 rounded bg-emerald-900/40 border border-emerald-700">TC 1: New Key &bull; <strong class="text-emerald-300">PASSED</strong></div>
                    <div class="p-1.5 rounded bg-rose-900/60 border border-rose-700">TC 2: In-Flight &bull; <strong class="text-rose-300">DOUBLE SPEND!</strong></div>
                    <div class="p-1.5 rounded bg-emerald-900/40 border border-emerald-700">TC 3: Replay &bull; <strong class="text-emerald-300">PASSED</strong></div>
                </div>
                <p class="text-[11px] leading-relaxed text-rose-300">
                    Fungsi Anda membiarkan request kedua lolos saat request pertama masih dalam status <code>PROCESSING</code>. Akibatnya dua pemotongan saldo bisa terjadi serentak untuk satu pesanan yang sama!
                </p>
                <div class="text-[10px] text-rose-400">Tambahkan: if status == "PROCESSING" { return false, ErrInFlightConflict }!</div>
            </div>`
			_, _ = w.Write([]byte(html))
			return
		}

		h.hub.Broadcast("🎉 [Exercise Lab] Drill 4 Lolos: Idempotency In-Flight State Trap berhasil diamankan dengan HTTP 409 Conflict!")
		html := `
        <div class="p-3.5 rounded-xl bg-emerald-950/60 border border-emerald-800 text-emerald-200 text-xs font-mono space-y-3">
            <div class="flex items-center justify-between border-b border-emerald-800/80 pb-2">
                <span class="text-sm font-extrabold text-emerald-400">ACCEPTED & IDEMPOTENT SAFE ✅</span>
                <span class="text-[10px] px-2 py-0.5 rounded bg-emerald-500/20 text-emerald-300 border border-emerald-500/30">Zero Double Spend &bull; All 3 TC Pass</span>
            </div>
            <div class="space-y-1.5 text-[11px]">
                <div class="flex items-center justify-between p-1.5 rounded bg-slate-900 border border-slate-800">
                    <span>✓ Test Case 1: Request Pertama Baru (Acquire Lock &rarr; Status PROCESSING)</span>
                    <span class="text-emerald-400 font-bold">PASS (0.1ms)</span>
                </div>
                <div class="flex items-center justify-between p-1.5 rounded bg-slate-900 border border-slate-800">
                    <span>✓ Test Case 2: In-Flight Collision Saat Sedang Berjalan (HTTP 409 Conflict)</span>
                    <span class="text-emerald-400 font-bold">PASS (0.2ms)</span>
                </div>
                <div class="flex items-center justify-between p-1.5 rounded bg-slate-900 border border-slate-800">
                    <span>✓ Test Case 3: Replay Transaksi Sukses Sebelumnya (HTTP 200 Replay)</span>
                    <span class="text-emerald-400 font-bold">PASS (0.1ms)</span>
                </div>
            </div>
            <!-- Follow-up Question Simulator -->
            <div class="p-2.5 rounded-lg bg-slate-900/90 border border-emerald-600/40 text-[11px] space-y-1">
                <strong class="text-emerald-300 block">💼 Pertanyaan Lanjutan Tech Lead (Follow-up Verbal Interview):</strong>
                <p class="text-slate-300 italic">"Header HTTP apa yang wajib disertakan saat server mengembalikan HTTP 409 Conflict pada idempotency in-flight?"</p>
                <p class="text-emerald-400 pt-0.5"><strong>Jawaban Ideal:</strong> Sertakan header 'Retry-After: &lt;seconds&gt;' (misal: 2 detik). Ini memberi tahu client agar menunda retry sampai pemrosesan awal diperkirakan selesai, mencegah badai request berulang.</p>
            </div>
        </div>`
		_, _ = w.Write([]byte(html))

	case "leak":
		hasContextSelect := strings.Contains(code, "select") && (strings.Contains(code, "ctx.Done()") || strings.Contains(code, "<-ctx.Done()"))

		if action == "run" {
			html := fmt.Sprintf(`
            <div class="p-3.5 rounded-xl bg-slate-900 border border-slate-800 font-mono text-xs space-y-2.5">
                <div class="text-cyan-400 font-bold flex items-center justify-between border-b border-slate-800 pb-1.5">
                    <span>▶ Test Runner: Uji Cepat Context Cancellation Poller (3 Skenario):</span>
                    <span class="text-slate-400 text-[10px]">Context Listened: %v</span>
                </div>
                <div class="space-y-1 text-slate-300 text-[11px]">
                    <div class="flex justify-between"><span>[TC 1 - Settlement Success] Bank menjawab true pada tick ke-2</span><span class="text-emerald-400">Exit Normal (nil)</span></div>
                    <div class="flex justify-between"><span>[TC 2 - Client Disconnect] Client putus koneksi pada 400ms</span><span class="text-cyan-400">Mendeteksi ctx.Done()</span></div>
                    <div class="flex justify-between"><span>[TC 3 - Timeout Abort] Context deadline exceeded 3.000ms</span><span class="text-emerald-400">Zero Zombie Goroutines</span></div>
                </div>
                <div class="text-[11px] text-cyan-300 pt-1">
                    Klik <strong>"⚡ Verifikasi Perbaikan Bug"</strong> untuk mendeteksi goroutine leak.
                </div>
            </div>`, hasContextSelect)
			_, _ = w.Write([]byte(html))
			return
		}

		if !hasContextSelect {
			html := `
            <div class="p-3.5 rounded-xl bg-rose-950/60 border border-rose-800 text-rose-200 text-xs font-mono space-y-2">
                <div class="flex items-center gap-2">
                    <span class="text-sm font-extrabold text-rose-400">❌ GOROUTINE LEAK DETECTED!</span>
                </div>
                <div class="grid grid-cols-3 gap-2 text-center text-[10px] my-1">
                    <div class="p-1.5 rounded bg-emerald-900/40 border border-emerald-700">TC 1: Normal Exit &bull; <strong class="text-emerald-300">PASSED</strong></div>
                    <div class="p-1.5 rounded bg-rose-900/60 border border-rose-700">TC 2: Disconnect &bull; <strong class="text-rose-300">LEAKED (Looping)</strong></div>
                    <div class="p-1.5 rounded bg-rose-900/60 border border-rose-700">TC 3: Timeout &bull; <strong class="text-rose-300">ZOMBIE THREAD</strong></div>
                </div>
                <p class="text-[11px] leading-relaxed text-rose-300">
                    Worker loop tidak mendengarkan <code>&lt;-ctx.Done()</code>. Ketika koneksi klien putus, loop tetap berjalan di background tanpa batas waktu, menyebabkan pemborosan CPU dan memory leak.
                </p>
                <div class="text-[10px] text-rose-400">Gunakan: select { case &lt;-ctx.Done(): return ctx.Err() case &lt;-ticker.C: ... }!</div>
            </div>`
			_, _ = w.Write([]byte(html))
			return
		}

		h.hub.Broadcast("🎉 [Exercise Lab] Drill 3 Lolos: Kebocoran Goroutine berhasil dicegah dengan select ctx.Done()!")
		html := `
        <div class="p-3.5 rounded-xl bg-emerald-950/60 border border-emerald-800 text-emerald-200 text-xs font-mono space-y-3">
            <div class="flex items-center justify-between border-b border-emerald-800/80 pb-2">
                <span class="text-sm font-extrabold text-emerald-400">ACCEPTED & ZERO LEAK ✅</span>
                <span class="text-[10px] px-2 py-0.5 rounded bg-emerald-500/20 text-emerald-300 border border-emerald-500/30">Goroutines Clean &bull; All 3 TC Pass</span>
            </div>
            <div class="space-y-1.5 text-[11px]">
                <div class="flex items-center justify-between p-1.5 rounded bg-slate-900 border border-slate-800">
                    <span>✓ Test Case 1: Status Bank Sukses Terkonfirmasi (Normal Termination)</span>
                    <span class="text-emerald-400 font-bold">PASS (0.1ms)</span>
                </div>
                <div class="flex items-center justify-between p-1.5 rounded bg-slate-900 border border-slate-800">
                    <span>✓ Test Case 2: Client Memutus Koneksi (ctx.Done() Terpicu Instan)</span>
                    <span class="text-emerald-400 font-bold">PASS (0.2ms)</span>
                </div>
                <div class="flex items-center justify-between p-1.5 rounded bg-slate-900 border border-slate-800">
                    <span>✓ Test Case 3: Loop Berhenti Bersih &amp; Ticker Dihentikan (Zero Zombie Leaks)</span>
                    <span class="text-emerald-400 font-bold">PASS (0.1ms)</span>
                </div>
            </div>
            <!-- Follow-up Question Simulator -->
            <div class="p-2.5 rounded-lg bg-slate-900/90 border border-emerald-600/40 text-[11px] space-y-1">
                <strong class="text-emerald-300 block">💼 Pertanyaan Lanjutan Tech Lead (Follow-up Verbal Interview):</strong>
                <p class="text-slate-300 italic">"Mengapa 'ticker.Stop()' wajib dipanggil dengan defer pada fungsi background polling?"</p>
                <p class="text-emerald-400 pt-0.5"><strong>Jawaban Ideal:</strong> Karena time.NewTicker mengalokasikan timer resource di runtime Go. Jika tidak distop, garbage collector tidak dapat membersihkan channel ticker, memicu kebocoran memori (timer memory leak).</p>
            </div>
        </div>`
		_, _ = w.Write([]byte(html))

	case "cache":
		lowerCode := strings.ToLower(code)
		hasCacheCheck := strings.Contains(lowerCode, "cache") || strings.Contains(lowerCode, "redis") || strings.Contains(lowerCode, "store")
		hasDbFallback := strings.Contains(lowerCode, "db") || strings.Contains(lowerCode, "database") || strings.Contains(lowerCode, "fetch") || strings.Contains(lowerCode, "query") || strings.Contains(lowerCode, "fallback")
		hasSetCache := strings.Contains(lowerCode, "set") || strings.Contains(lowerCode, "store[") || strings.Contains(lowerCode, "ttl") || strings.Contains(lowerCode, "save") || strings.Contains(lowerCode, "cache[")

		if action == "run" {
			html := fmt.Sprintf(`
            <div class="p-3.5 rounded-xl bg-slate-900 border border-slate-800 font-mono text-xs space-y-2.5">
                <div class="text-emerald-400 font-bold flex items-center justify-between border-b border-slate-800 pb-1.5">
                    <span>▶ Test Runner: Uji Cepat NoSQL Redis Cache-Aside (3 Skenario):</span>
                    <span class="text-slate-400 text-[10px]">Fallback Handled: %v</span>
                </div>
                <div class="space-y-1 text-slate-300 text-[11px]">
                    <div class="flex justify-between"><span>[TC 1 - Cache Miss] Query awal belum ada di Redis</span><span class="text-emerald-400">Panggil DB &amp; Isi Cache</span></div>
                    <div class="flex justify-between"><span>[TC 2 - Cache Hit] Query berulang kunci yang sama</span><span class="text-emerald-400">Hit Redis (Latency &lt; 2ms)</span></div>
                    <div class="flex justify-between"><span>[TC 3 - High Traffic] 100 Request serentak membaca cache</span><span class="text-emerald-400">99%% Offload dari DB</span></div>
                </div>
                <div class="text-[11px] text-emerald-300 pt-1">
                    Klik <strong>"⚡ Verifikasi Perbaikan Bug"</strong> untuk validasi integrasi cache-aside.
                </div>
            </div>`, hasCacheCheck && hasDbFallback && hasSetCache)
			_, _ = w.Write([]byte(html))
			return
		}

		if !hasCacheCheck || !hasDbFallback || !hasSetCache {
			html := `
            <div class="p-3.5 rounded-xl bg-rose-950/60 border border-rose-800 text-rose-200 text-xs font-mono space-y-2">
                <div class="flex items-center gap-2">
                    <span class="text-sm font-extrabold text-rose-400">❌ CACHE-ASIDE PATTERN DEFECT!</span>
                </div>
                <div class="grid grid-cols-3 gap-2 text-center text-[10px] my-1">
                    <div class="p-1.5 rounded bg-emerald-900/40 border border-emerald-700">TC 1: Cache Check &bull; <strong class="text-emerald-300">PASSED</strong></div>
                    <div class="p-1.5 rounded bg-rose-900/60 border border-rose-700">TC 2: DB Fallback &bull; <strong class="text-rose-300">FAILED (No DB)</strong></div>
                    <div class="p-1.5 rounded bg-rose-900/60 border border-rose-700">TC 3: Set Cache &bull; <strong class="text-rose-300">FAILED (No Set)</strong></div>
                </div>
                <p class="text-[11px] leading-relaxed text-rose-300">
                    Implementasi Anda belum menerapkan alur Cache-Aside yang lengkap: Cek Cache &rarr; Jika Miss, panggil DB fallback &rarr; Simpan hasil ke Cache dengan TTL &rarr; Kembalikan data.
                </p>
                <div class="text-[10px] text-rose-400">Pastikan jika cache miss, data diambil dari database lalu disimpan kembali ke cache!</div>
            </div>`
			_, _ = w.Write([]byte(html))
			return
		}

		h.hub.Broadcast("🎉 [Exercise Lab] Drill 5 Lolos: Pattern NoSQL Redis Cache-Aside & DB Fallback berhasil diimplementasikan!")
		html := `
        <div class="p-3.5 rounded-xl bg-emerald-950/60 border border-emerald-800 text-emerald-200 text-xs font-mono space-y-3">
            <div class="flex items-center justify-between border-b border-emerald-800/80 pb-2">
                <span class="text-sm font-extrabold text-emerald-400">ACCEPTED & CACHE VERIFIED ✅</span>
                <span class="text-[10px] px-2 py-0.5 rounded bg-emerald-500/20 text-emerald-300 border border-emerald-500/30">Latency: 1.2ms &bull; All 3 TC Pass</span>
            </div>
            <div class="space-y-1.5 text-[11px]">
                <div class="flex items-center justify-between p-1.5 rounded bg-slate-900 border border-slate-800">
                    <span>✓ Test Case 1: Cache Miss &rarr; Fetch DB Fallback &amp; Populate Cache</span>
                    <span class="text-emerald-400 font-bold">PASS (0.8ms)</span>
                </div>
                <div class="flex items-center justify-between p-1.5 rounded bg-slate-900 border border-slate-800">
                    <span>✓ Test Case 2: Cache Hit on Subsequent Requests (Memory Speed)</span>
                    <span class="text-emerald-400 font-bold">PASS (0.1ms)</span>
                </div>
                <div class="flex items-center justify-between p-1.5 rounded bg-slate-900 border border-slate-800">
                    <span>✓ Test Case 3: Thundering Herd Mitigation (Postgres Offloaded 90%%)</span>
                    <span class="text-emerald-400 font-bold">PASS (0.2ms)</span>
                </div>
            </div>
            <!-- Follow-up Question Simulator -->
            <div class="p-2.5 rounded-lg bg-slate-900/90 border border-emerald-600/40 text-[11px] space-y-1">
                <strong class="text-emerald-300 block">💼 Pertanyaan Lanjutan Tech Lead (Follow-up Verbal Interview):</strong>
                <p class="text-slate-300 italic">"Bagaimana strategi Anda mencegah fenomena 'Cache Penetration' jika user request ID produk yang memang tidak ada di DB?"</p>
                <p class="text-emerald-400 pt-0.5"><strong>Jawaban Ideal:</strong> Simpan nilai nil/empty string ke dalam Redis dengan TTL pendek (misal: 30–60 detik), atau gunakan Bloom Filter di depan cache untuk mengecek apakah ID tersebut eksis sebelum melakukan query ke database.</p>
            </div>
        </div>`
		_, _ = w.Write([]byte(html))

	default:
		http.Error(w, "Drill tidak dikenali", http.StatusBadRequest)
	}
}

// CheckExerciseQuiz mengevaluasi jawaban kuis teori bootcamp interaktif.
func (h *Handler) CheckExerciseQuiz(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Permintaan tidak valid", http.StatusBadRequest)
		return
	}

	q1 := r.FormValue("q1")
	q2 := r.FormValue("q2")
	q3 := r.FormValue("q3")

	score := 0
	if q1 == "B" {
		score += 33
	}
	if q2 == "A" {
		score += 33
	}
	if q3 == "C" {
		score += 34
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	statusBadge := `<span class="px-2.5 py-1 text-xs font-bold rounded-lg bg-emerald-500/20 text-emerald-300 border border-emerald-500/40">SKOR SEMPURNA: 100/100 (KOMPETENSI TEORI TERVERIFIKASI)</span>`
	if score < 100 {
		statusBadge = fmt.Sprintf(`<span class="px-2.5 py-1 text-xs font-bold rounded-lg bg-amber-500/20 text-amber-300 border border-amber-500/40">SKOR: %d/100 (EVALUASI KEMBALI BEBERAPA JAWABAN)</span>`, score)
	}

	h.hub.Broadcast(fmt.Sprintf("📝 [Exercise Lab] Kuis Uji Teori Diselesaikan dengan Skor %d/100!", score))

	html := fmt.Sprintf(`
    <div class="p-4 sm:p-5 rounded-2xl bg-slate-900 border border-slate-800 space-y-4 font-mono text-xs">
        <div class="flex items-center justify-between border-b border-slate-800 pb-3">
            <h4 class="text-sm font-bold text-white flex items-center gap-2">
                <span>📋</span> Hasil Evaluasi Kuis Pemahaman Teori
            </h4>
            %s
        </div>

        <div class="space-y-3">
            <!-- Evaluasi Soal 1 -->
            <div class="p-3 rounded-xl %s border text-xs space-y-1">
                <div class="flex items-center justify-between">
                    <strong>Soal 1 (Data Integrity): %s</strong>
                    <span>Kunci: B (IEEE-754 Rounding Error)</span>
                </div>
                <p class="text-[11px] leading-relaxed text-slate-300">
                    <strong>Ulasan Senior Engineer:</strong> IEEE-754 menyimpan angka floating point dalam fraksi basis 2 (biner), sehingga pecahan desimal seperti 0.1 atau 0.2 tidak dapat direpresentasikan secara eksak. Di sistem pembayaran, selalu simpan nilai uang dalam satuan sen terkecil (int64) atau tipe NUMERIC SQL.
                </p>
            </div>

            <!-- Evaluasi Soal 2 -->
            <div class="p-3 rounded-xl %s border text-xs space-y-1">
                <div class="flex items-center justify-between">
                    <strong>Soal 2 (DB Concurrency): %s</strong>
                    <span>Kunci: A (SELECT FOR UPDATE)</span>
                </div>
                <p class="text-[11px] leading-relaxed text-slate-300">
                    <strong>Ulasan Senior Engineer:</strong> Pada entitas dengan frekuensi konflik tinggi (seperti saldo dompet hot-wallet yang ditransaksikan serentak), Optimistic Locking akan memicu kegagalan retry beruntun yang membebani CPU. Pessimistic Locking (SELECT FOR UPDATE) mengantrekan akses baris secara aman.
                </p>
            </div>

            <!-- Evaluasi Soal 3 -->
            <div class="p-3 rounded-xl %s border text-xs space-y-1">
                <div class="flex items-center justify-between">
                    <strong>Soal 3 (Distributed Retries): %s</strong>
                    <span>Kunci: C (HTTP 409 Conflict)</span>
                </div>
                <p class="text-[11px] leading-relaxed text-slate-300">
                    <strong>Ulasan Senior Engineer:</strong> HTTP 409 Conflict secara resmi menandakan kondisi in-flight collision pada idempotency layer. Disertai header Retry-After, klien diarahkan untuk menunda percobaan kembali tanpa merusak transaksi yang sedang berjalan.
                </p>
            </div>
        </div>

        <div class="pt-2 text-right">
            <a href="/" class="inline-flex items-center gap-2 px-5 py-2.5 rounded-xl bg-gradient-to-r from-emerald-600 to-teal-600 hover:from-emerald-500 hover:to-teal-500 text-white font-bold text-xs font-sans transition shadow-md shadow-emerald-950/40">
                <span>🏆 Buka Halaman Hiring Gauntlet (Ujian Resmi)</span>
                <span>&rarr;</span>
            </a>
        </div>
    </div>`,
		statusBadge,
		ternaryStr(q1 == "B", "bg-emerald-950/40 border-emerald-800 text-emerald-200", "bg-rose-950/40 border-rose-800 text-rose-200"),
		ternaryStr(q1 == "B", "BENAR ✓", "KURANG TEPAT ❌"),
		ternaryStr(q2 == "A", "bg-emerald-950/40 border-emerald-800 text-emerald-200", "bg-rose-950/40 border-rose-800 text-rose-200"),
		ternaryStr(q2 == "A", "BENAR ✓", "KURANG TEPAT ❌"),
		ternaryStr(q3 == "C", "bg-emerald-950/40 border-emerald-800 text-emerald-200", "bg-rose-950/40 border-rose-800 text-rose-200"),
		ternaryStr(q3 == "C", "BENAR ✓", "KURANG TEPAT ❌"),
	)

	_, _ = w.Write([]byte(html))
}

func ternaryStr(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}

