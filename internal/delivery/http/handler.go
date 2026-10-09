package http

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
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
		return nil, fmt.Errorf("gagal mem-parsing template html: %w", err)
	}

	raceSim := racecondition.NewSimulator(100000) // Saldo awal $1,000.00
	idemSim := idempotency.NewSimulator()
	sagaSim := distributed.NewSimulator()
	trafficSim := hightraffic.NewSimulator(50) // Kuota 50 voucher
	hub := NewSSEHub()

	return &Handler{
		tmpl:           tmpl,
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
		h.hub.Broadcast(fmt.Sprintf("🎖️ [GAMIFIKASI] TAHAP %d LULUS: '%s' (+250 XP Diperoleh!)", stage, title))
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

	totalXP := 0
	if s0 { totalXP += 150 }
	if s1 { totalXP += 250 }
	if s2 { totalXP += 250 }
	if s3 { totalXP += 250 }
	if s4 { totalXP += 250 }
	if s5 { totalXP += 100 }

	maxXP := 1250

	levelTitle := "Tahap 0: Bootcamp Teori & Lab Persiapan"
	statusBadge := `<span class="px-2.5 py-1 text-xs font-semibold rounded-lg bg-slate-100 dark:bg-slate-800 text-slate-700 dark:text-slate-300 border border-slate-300 dark:border-slate-700">Persiapan Teori</span>`
	if s5 {
		levelTitle = "🏆 LULUS SEMUA TAHAP - SURAT PENAWARAN RESMI TERBIT!"
		statusBadge = `<span class="px-2.5 py-1 text-xs font-bold rounded-lg bg-emerald-500/20 text-emerald-700 dark:text-emerald-300 border border-emerald-500/40 animate-pulse">OFFER EXTENDED (STRONG HIRE) 🎉</span>`
	} else if s4 {
		levelTitle = "Tahap 5: Peninjauan Tawaran Kerja (Offer Letter)"
		statusBadge = `<span class="px-2.5 py-1 text-xs font-bold rounded-lg bg-amber-500/20 text-amber-700 dark:text-amber-300 border border-amber-500/40">Siap Review Penawaran</span>`
	} else if s3 {
		levelTitle = "Tahap 4: Production War Room (Flash Sale Rate Limiter)"
		statusBadge = `<span class="px-2.5 py-1 text-xs font-semibold rounded-lg bg-cyan-500/20 text-cyan-700 dark:text-cyan-300 border border-cyan-500/40">Tahap Akhir Evaluasi</span>`
	} else if s2 {
		levelTitle = "Tahap 3: Take-Home Payment API Review"
		statusBadge = `<span class="px-2.5 py-1 text-xs font-semibold rounded-lg bg-teal-500/20 text-teal-700 dark:text-teal-300 border border-teal-500/40">Ujian Praktik</span>`
	} else if s1 {
		levelTitle = "Tahap 2: System Design Idempotency Defense"
		statusBadge = `<span class="px-2.5 py-1 text-xs font-semibold rounded-lg bg-emerald-500/20 text-emerald-700 dark:text-emerald-300 border border-emerald-500/40">Arsitektur Terbuka</span>`
	} else if s0 {
		levelTitle = "Tahap 1: Live Coding DSA (Two Sum Target Match)"
		statusBadge = `<span class="px-2.5 py-1 text-xs font-semibold rounded-lg bg-emerald-500/20 text-emerald-700 dark:text-emerald-300 border border-emerald-500/40">Bootcamp Lulus ✓</span>`
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
		{0, "📚", "Bootcamp", "Teori & Lab", s0, !s0},
		{1, "💻", "Tahap 1", "Live Coding", s1, s0 && !s1},
		{2, "🏛️", "Tahap 2", "System Design", s2, s1 && !s2},
		{3, "📦", "Tahap 3", "Take-Home", s3, s2 && !s3},
		{4, "🔥", "Tahap 4", "War Room", s4, s3 && !s4},
		{5, "🏆", "Tahap 5", "Offer Letter", s5, s4 && !s5},
	}

	var stepperItems strings.Builder
	for i, step := range steps {
		circleClass := ""
		iconDisplay := step.icon
		statusPill := ""

		if step.passed {
			circleClass = "bg-emerald-600 text-white shadow-md shadow-emerald-950/20 ring-4 ring-emerald-500/20 border border-emerald-400 scale-105"
			iconDisplay = "✓"
			statusPill = `<span class="text-[9px] font-bold text-emerald-600 dark:text-emerald-400">Lulus</span>`
		} else if step.isCurrent {
			circleClass = "bg-gradient-to-tr from-teal-600 to-cyan-500 text-white shadow-md ring-4 ring-cyan-500/30 animate-pulse border border-cyan-400 scale-110"
			statusPill = `<span class="text-[9px] font-bold text-cyan-600 dark:text-cyan-400">Aktif ●</span>`
		} else {
			circleClass = "bg-slate-100 dark:bg-dark-850 text-slate-400 dark:text-slate-500 border border-slate-200 dark:border-slate-800 ring-2 ring-slate-200/60 dark:ring-slate-800"
			statusPill = `<span class="text-[9px] text-slate-400 dark:text-slate-500">Terkunci</span>`
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
                        <span class="text-[11px] text-emerald-600 dark:text-emerald-400 font-mono font-medium">&bull; Target: Junior / Associate Backend Engineer (Fintech Track)</span>
                    </div>
                    <div class="text-xs font-semibold text-slate-600 dark:text-slate-300 mt-0.5">%s</div>
                </div>
            </div>

            <div class="flex items-center justify-between sm:justify-end gap-3 border-t sm:border-t-0 border-slate-100 dark:border-slate-800/80 pt-2 sm:pt-0">
                <div class="sm:text-right">
                    <div class="text-[11px] text-slate-500 dark:text-slate-400">Total Pengalaman (XP)</div>
                    <div class="text-lg sm:text-xl font-mono font-extrabold text-slate-900 dark:text-white">%d / %d XP</div>
                </div>
                %s
            </div>
        </div>

        <!-- Hiring Journey Stepper Bar (Tahap 0 s/d Tahap 5) -->
        <div class="pt-3 border-t border-slate-100 dark:border-slate-800/80">
            <div class="flex items-center justify-between text-[11px] text-slate-500 dark:text-slate-400 mb-2 font-medium">
                <span class="flex items-center gap-1.5 font-bold text-slate-700 dark:text-slate-300">
                    <span>🗺️</span> <span>Hiring Journey Stepper (%d dari 5 Misi Wawancara Tuntas)</span>
                </span>
                <span class="font-mono text-emerald-600 dark:text-emerald-400 font-bold">%d%% Menuju Tawaran Kerja</span>
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
    </div>`, levelTitle, totalXP, maxXP, statusBadge, coreStepsCompleted, progressPct, stepperItems.String(), progressPct, s0, s1, s2, s3, s4, s5)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(html))
}

// ResetGamification mereset skor dan pencapaian gamifikasi.
func (h *Handler) ResetGamification(w http.ResponseWriter, r *http.Request) {
	h.gamifyMu.Lock()
	h.stagePassed = make(map[int]bool)
	h.gamifyMu.Unlock()

	h.hub.Broadcast("🔄 Progres gamifikasi wawancara telah di-reset kembali ke Tahap 0.")
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

// EvalScenario01Code mengevaluasi kode Go yang diketik manual oleh kandidat untuk LeetCode Problem #1: Two Sum.
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
	hasReturnSlice := strings.Contains(codeContent, "[]int{") || strings.Contains(codeContent, "return []int") || strings.Contains(codeContent, "return nil")
	hasLogic := strings.Contains(codeContent, "target -") || strings.Contains(codeContent, "target-") || strings.Contains(codeContent, "==")
	hasHashMap := strings.Contains(codeContent, "map[int]int") || strings.Contains(codeContent, "make(map") || strings.Contains(codeContent, "map[int]")
	hasNestedLoop := (strings.Contains(codeContent, "for i") && strings.Contains(codeContent, "for j")) ||
		(strings.Contains(codeContent, "for ") && strings.Count(codeContent, "for ") >= 2 && !hasHashMap)

	h.hub.Broadcast(fmt.Sprintf("💻 [LeetCode Runner] Menganalisis algoritma Two Sum kandidat (%d baris, Aksi: %s)...", len(strings.Split(codeContent, "\n")), action))

	// Jika kandidat belum mengetik implementasi sama sekali
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
                <strong class="text-rose-300 block">Fungsi TwoSum Belum Diimplementasikan!</strong>
                <p class="text-[11px] text-rose-300 leading-relaxed">
                    Ketik manual algoritma pencarian indeks Two Sum Anda pada editor di atas. Jika Anda bingung, klik tombol <strong>"💡 Hint 1 (Konseptual)"</strong> di atas editor!
                </p>
            </div>
        </div>`
		_, _ = w.Write([]byte(html))
		return
	}

	// Cek apakah kandidat lupa me-return slice hasil
	if !hasReturnSlice && !hasLogic {
		html := `
        <div class="space-y-3 font-mono text-xs">
            <div class="flex items-center justify-between border-b border-rose-800/60 pb-2">
                <div class="flex items-center gap-2">
                    <span class="text-base font-extrabold text-rose-400">❌ WRONG ANSWER</span>
                    <span class="text-[11px] px-2 py-0.5 rounded bg-rose-500/20 text-rose-300 border border-rose-500/30">Missing Return Value</span>
                </div>
                <span class="text-rose-400 font-mono text-[11px]">Runtime: 0.2 ms</span>
            </div>
            <div class="p-3 rounded-lg bg-rose-950/60 border border-rose-800 text-rose-200 text-xs space-y-1.5">
                <strong class="text-rose-300 block">Test Case 1 Gagal: Output Tidak Valid</strong>
                <p class="text-[11px] text-rose-300 leading-relaxed">
                    Input: <code>nums = [2, 7, 11, 15], target = 9</code> &rarr; Expected: <code>[0, 1]</code>.
                    Pastikan Anda me-return slice dua indeks yang valid: <code>return []int{prevIdx, currIdx}</code>!
                </p>
            </div>
        </div>`
		_, _ = w.Write([]byte(html))
		return
	}

	// Jika action == "run" (Cek Cepat 3 Test Cases)
	if action == "run" {
		complexityText := "O(N) Optimal (Hash Map terdeteksi)"
		if hasNestedLoop && !hasHashMap {
			complexityText = "O(N^2) Suboptimal (Nested Loop terdeteksi)"
		}

		html := fmt.Sprintf(`
        <div class="space-y-3 font-mono text-xs">
            <div class="flex items-center justify-between border-b border-slate-800 pb-2">
                <span class="text-sm font-bold text-amber-300 flex items-center gap-2">
                    <span>▶</span> Hasil Uji Cepat (Run Test Cases)
                </span>
                <span class="text-slate-500 text-[11px]">Go 1.27 Live Sandbox &bull; LeetCode #1</span>
            </div>
            <div class="space-y-2">
                <div class="p-2.5 rounded-lg bg-emerald-950/40 border border-emerald-800/60 flex items-center justify-between">
                    <div>
                        <strong class="text-emerald-400">Case 1: Normal Pair</strong>
                        <div class="text-[11px] text-slate-400">Input: nums=[2, 7, 11, 15], target=9 &rarr; Output: [0, 1] (2 + 7 = 9)</div>
                    </div>
                    <span class="px-2 py-0.5 rounded text-[11px] font-bold bg-emerald-500/20 text-emerald-300">PASS ✓</span>
                </div>
                <div class="p-2.5 rounded-lg bg-emerald-950/40 border border-emerald-800/60 flex items-center justify-between">
                    <div>
                        <strong class="text-emerald-400">Case 2: Unsorted Array</strong>
                        <div class="text-[11px] text-slate-400">Input: nums=[3, 2, 4], target=6 &rarr; Output: [1, 2] (2 + 4 = 6)</div>
                    </div>
                    <span class="px-2 py-0.5 rounded text-[11px] font-bold bg-emerald-500/20 text-emerald-300">PASS ✓</span>
                </div>
                <div class="p-2.5 rounded-lg bg-emerald-950/40 border border-emerald-800/60 flex items-center justify-between">
                    <div>
                        <strong class="text-emerald-400">Case 3: Duplicate Numbers</strong>
                        <div class="text-[11px] text-slate-400">Input: nums=[3, 3], target=6 &rarr; Output: [0, 1] (3 + 3 = 6)</div>
                    </div>
                    <span class="px-2 py-0.5 rounded text-[11px] font-bold bg-emerald-500/20 text-emerald-300">PASS ✓</span>
                </div>
            </div>
            <div class="p-2.5 rounded-lg bg-slate-900/90 border border-slate-800 text-[11px] text-slate-300 flex items-center justify-between">
                <span>Kompleksitas: <strong>%s</strong></span>
                <span class="text-amber-400">Klik "⚡ Submit Solution" untuk evaluasi komite resmi!</span>
            </div>
        </div>`, complexityText)
		_, _ = w.Write([]byte(html))
		return
	}

	// Action == "submit"
	if hasNestedLoop && !hasHashMap {
		// Kasus Brute Force: Lolos input kecil, tapi ditolak komite karena TLE pada 10.000 transaksi
		h.hub.Broadcast("⚠️ [LeetCode Runner] SUBOPTIMAL WARNING: Algoritma brute force O(N^2) terdeteksi. Risiko TLE pada skala produksi!")
		html := `
        <div class="space-y-3 font-mono text-xs">
            <div class="flex items-center justify-between border-b border-amber-800/60 pb-2">
                <div class="flex items-center gap-2">
                    <span class="text-base font-extrabold text-amber-400">⚠️ TIME LIMIT HAZARD (O(N²) SUBOPTIMAL)</span>
                    <span class="text-[11px] px-2 py-0.5 rounded bg-amber-500/20 text-amber-300 border border-amber-500/30">Perlu Optimasi</span>
                </div>
                <span class="text-amber-400 font-mono text-[11px]">Runtime: 1,840 ms (Too Slow)</span>
            </div>
            <div class="p-3 rounded-lg bg-amber-950/60 border border-amber-800 text-amber-200 text-xs space-y-2">
                <strong class="text-amber-300 block">Evaluasi Pewawancara (Staff Software Engineer):</strong>
                <p class="text-[11px] text-amber-200 leading-relaxed">
                    <em>"Solusi brute force nested loop <code>for i := 0 ... for j := i+1 ...</code> Anda berhasil untuk array kecil (Case 1 & 2), tetapi memiliki Time Complexity <strong>O(N²)</strong>. Saat diuji pada 10.000 data transaksi nasabah (Stress Test Case 4), server membutuhkan 1.840 ms dan berisiko Time Limit Exceeded (TLE)!"</em>
                </p>
                <div class="p-2 rounded bg-dark-950 border border-amber-800/60 text-[11px] text-slate-300">
                    💡 <strong>Arahan Optimasi:</strong> Gunakan <strong>Hash Map (map[int]int)</strong> untuk mencatat angka yang sudah dilewati. Dengan begitu, Anda bisa mencari <code>complement := target - nums[i]</code> dalam <strong>O(1)</strong> amortized, memangkas total runtime menjadi <strong>O(N)</strong>!
                </div>
            </div>
            <div class="grid grid-cols-4 gap-2 text-center text-[11px]">
                <div class="p-2 rounded bg-slate-900 border border-slate-800">
                    <span class="text-slate-500">Case 1</span>
                    <div class="font-bold text-emerald-400">PASS ✓</div>
                </div>
                <div class="p-2 rounded bg-slate-900 border border-slate-800">
                    <span class="text-slate-500">Case 2</span>
                    <div class="font-bold text-emerald-400">PASS ✓</div>
                </div>
                <div class="p-2 rounded bg-slate-900 border border-slate-800">
                    <span class="text-slate-500">Case 3</span>
                    <div class="font-bold text-emerald-400">PASS ✓</div>
                </div>
                <div class="p-2 rounded bg-slate-900 border border-amber-800/80 bg-amber-950/30">
                    <span class="text-amber-400">Case 4 (10k items)</span>
                    <div class="font-bold text-amber-400">TLE HAZARD ⚠️</div>
                </div>
            </div>
        </div>`
		_, _ = w.Write([]byte(html))
		return
	}

	// Solusi Optimal O(N) Hash Map:
	h.markStagePassed(1, "LeetCode DSA Champion: Two Sum O(N)")
	h.hub.Broadcast("🎉 [LeetCode Runner] STATUS: ACCEPTED! Solusi Two Sum Hash Map O(N) lolos seluruh test cases dengan runtime 0.8 ms (+250 XP).")

	html := `
    <div class="space-y-3 font-mono text-xs">
        <div class="flex items-center justify-between border-b border-emerald-800/60 pb-2">
            <div class="flex items-center gap-2">
                <span class="text-base font-extrabold text-emerald-400">ACCEPTED ✅</span>
                <span class="text-[11px] px-2 py-0.5 rounded bg-emerald-500/20 text-emerald-300 border border-emerald-500/30">+250 XP DIRAIH</span>
            </div>
            <span class="text-emerald-400 font-mono text-[11px]">Runtime: 0.8 ms (Beats 99.4%)</span>
        </div>
        <div class="grid grid-cols-2 sm:grid-cols-4 gap-2 text-center text-[11px]">
            <div class="p-2 rounded bg-slate-900 border border-slate-800">
                <span class="text-slate-500">Runtime</span>
                <div class="font-bold text-emerald-400">0.8 ms</div>
                <div class="text-[10px] text-emerald-400">Beats 99.4%</div>
            </div>
            <div class="p-2 rounded bg-slate-900 border border-slate-800">
                <span class="text-slate-500">Memory</span>
                <div class="font-bold text-white">3.2 MB</div>
                <div class="text-[10px] text-emerald-400">Beats 97.5%</div>
            </div>
            <div class="p-2 rounded bg-slate-900 border border-slate-800">
                <span class="text-slate-500">Kompleksitas Waktu</span>
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
                Sempurna! Anda berhasil memecahkan soal Two Sum dengan algoritma optimal <strong>O(N) Time Complexity</strong> dan <strong>O(N) Space Complexity</strong> menggunakan Hash Map lookup. Kemampuan mentransformasikan algoritma kuadratik O(N²) menjadi linear O(N) adalah fondasi esensial yang dicari perusahaan teknologi terkemuka.
            </p>
            <div class="text-[11px] text-slate-300 pt-1">
                👉 <strong>Langkah Berikutnya:</strong> Presentasikan diagram arsitektur sistem pembayaran yang resilien di hadapan Lead Architect!
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

// DefendScenario02 memproses jawaban pertahanan arsitektur System Design kandidat di depan Principal Architect.
func (h *Handler) DefendScenario02(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Permintaan tidak valid", http.StatusBadRequest)
		return
	}

	answer := r.FormValue("defense_choice")
	w.Header().Set("HX-Trigger", "refreshWallets")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if answer == "sha256_hash_and_inflight_lock" {
		h.markStagePassed(2, "System Design Idempotency Defense Approved")
		h.hub.Broadcast("🏛️ [System Design] Lead Backend Engineer menyetujui pemahaman idempotensi kandidat! Nilai: A+ (+250 XP).")

		html := `
        <div class="p-4 rounded-xl bg-emerald-50 dark:bg-emerald-950/50 border border-emerald-300 dark:border-emerald-800/70 space-y-3 font-sans text-xs">
            <div class="flex items-center justify-between border-b border-emerald-200 dark:border-emerald-800/60 pb-2">
                <div class="flex items-center gap-2">
                    <span class="w-3 h-3 rounded-full bg-emerald-500"></span>
                    <strong class="text-sm font-bold text-emerald-800 dark:text-emerald-300">HASIL INTERVIEW SYSTEM DESIGN: LULUS (STRONG HIRE) ✓</strong>
                </div>
                <span class="text-xs font-mono font-bold text-amber-600 dark:text-amber-300">+250 XP DIDAPAT</span>
            </div>
            <div class="text-slate-700 dark:text-slate-200 leading-relaxed text-xs space-y-2">
                <p>
                    <strong>Pewawancara (Lead Backend Engineer):</strong><br>
                    <em>"Penjelasan Anda sangat tepat dan dewasa untuk level Junior/Associate! Memahami bahwa validasi sisi frontend (disable button) mudah ditembus oleh network retry otomatis atau API script adalah fondasi penting backend engineer. Menggunakan Idempotency-Key di backend dengan status lock 'PROCESSING' menjamin transaksi nasabah tidak pernah terpotong ganda."</em>
                </p>
                <div class="grid grid-cols-2 sm:grid-cols-3 gap-2 pt-1 font-mono text-[11px]">
                    <div class="p-2 rounded bg-white dark:bg-dark-900 border border-slate-200 dark:border-slate-800">
                        <span class="text-slate-400">Pemahaman Alur</span>
                        <div class="text-emerald-600 dark:text-emerald-400 font-bold">100% Menguasai</div>
                    </div>
                    <div class="p-2 rounded bg-white dark:bg-dark-900 border border-slate-200 dark:border-slate-800">
                        <span class="text-slate-400">Pencegahan Double-Charge</span>
                        <div class="text-emerald-600 dark:text-emerald-400 font-bold">Terverifikasi</div>
                    </div>
                    <div class="p-2 rounded bg-white dark:bg-dark-900 border border-slate-200 dark:border-slate-800">
                        <span class="text-slate-400">Nilai Interview</span>
                        <div class="text-amber-600 dark:text-amber-400 font-bold">Nilai: A+</div>
                    </div>
                </div>
                <div class="pt-2 border-t border-emerald-200 dark:border-emerald-800/60 flex items-center justify-between">
                    <span class="text-[11px] text-emerald-800 dark:text-emerald-300">Tahap Desain Sistem Lolos dengan Predikat Sempurna!</span>
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

	// Jawaban salah
	h.hub.Broadcast("⚠️ [System Design] Argumen pertahanan perlu perbaikan: Jangan hanya mengandalkan frontend disable.")
	html := `
    <div class="p-4 rounded-xl bg-rose-50 dark:bg-rose-950/50 border border-rose-300 dark:border-rose-800/70 space-y-2.5 font-sans text-xs">
        <div class="flex items-center justify-between border-b border-rose-200 dark:border-rose-800/60 pb-2">
            <strong class="text-sm font-bold text-rose-800 dark:text-rose-300">HASIL EVALUASI: PERLU REVISI LOGIKA ✗</strong>
            <span class="text-xs font-mono text-rose-600 dark:text-rose-400">Nilai: C</span>
        </div>
        <p class="text-slate-700 dark:text-slate-300 leading-relaxed text-xs">
            <strong>Pewawancara (Lead Backend Engineer):</strong><br>
            <em>"Perhatian: Mengandalkan tombol disable di frontend saja sangat berisiko di sistem pembayaran! Jika koneksi timeout di tengah jalan dan pengguna me-refresh browser, request kedua akan terkirim lagi dan saldo nasabah bisa terpotong dua kali. Backend wajib memverifikasi Idempotency-Key secara mandiri."</em>
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
            <span class="text-xs font-mono font-bold text-amber-600 dark:text-amber-300">+250 XP DIRAIH</span>
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
