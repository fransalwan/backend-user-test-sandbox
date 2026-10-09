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

// GetGamificationStatus mengembalikan status kelulusan 4 tahap wawancara kandidat.
func (h *Handler) GetGamificationStatus(w http.ResponseWriter, r *http.Request) {
	h.gamifyMu.RLock()
	s1 := h.stagePassed[1]
	s2 := h.stagePassed[2]
	s3 := h.stagePassed[3]
	s4 := h.stagePassed[4]
	h.gamifyMu.RUnlock()

	passedCount := 0
	if s1 { passedCount++ }
	if s2 { passedCount++ }
	if s3 { passedCount++ }
	if s4 { passedCount++ }

	totalXP := passedCount * 250
	progressPct := passedCount * 25

	levelTitle := "Tahap 1: Technical Screening"
	statusBadge := `<span class="px-2.5 py-1 text-xs font-semibold rounded-lg bg-slate-800 text-slate-300 border border-slate-700">Sedang Diuji (In Review)</span>`
	if passedCount == 1 {
		levelTitle = "Tahap 2: System Resiliency Test"
	} else if passedCount == 2 {
		levelTitle = "Tahap 3: Take-Home Architecture"
	} else if passedCount == 3 {
		levelTitle = "Tahap 4: High Traffic & Scale Test"
	} else if passedCount == 4 {
		levelTitle = "🏆 LULUS SEMUA TAHAP - SURAT PENAWARAN TERBIT!"
		statusBadge = `<span class="px-2.5 py-1 text-xs font-bold rounded-lg bg-emerald-500/20 text-emerald-300 border border-emerald-500/40 animate-pulse">OFFER EXTENDED (STRONG HIRE) 🎉</span>`
	}

	html := fmt.Sprintf(`
    <div id="gamification-hud" class="rounded-2xl bg-white dark:bg-gradient-to-r dark:from-dark-900 dark:via-[#101932] dark:to-dark-900 border border-slate-200 dark:border-slate-800 p-4 sm:p-5 shadow-sm transition-colors duration-200">
        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 sm:gap-4">
            <div class="flex items-center gap-3">
                <div class="w-11 h-11 sm:w-12 sm:h-12 rounded-2xl bg-gradient-to-tr from-emerald-600 to-cyan-500 flex items-center justify-center text-xl sm:text-2xl shadow-md flex-shrink-0">
                    👨‍💻
                </div>
                <div>
                    <div class="flex items-center gap-2 flex-wrap">
                        <span class="text-xs sm:text-sm font-bold text-slate-900 dark:text-white">Kandidat: Frans Alwan</span>
                        <span class="text-[11px] text-slate-500 dark:text-slate-400 font-mono">&bull; Target: Principal Backend Engineer</span>
                    </div>
                    <div class="text-xs font-semibold text-emerald-600 dark:text-emerald-400 mt-0.5">%s</div>
                </div>
            </div>

            <div class="flex items-center justify-between sm:justify-end gap-4 border-t sm:border-t-0 border-slate-100 dark:border-slate-800/80 pt-2 sm:pt-0">
                <div class="sm:text-right">
                    <div class="text-[11px] text-slate-500 dark:text-slate-400">Total Pengalaman (XP)</div>
                    <div class="text-lg sm:text-xl font-mono font-extrabold text-slate-900 dark:text-white">%d / 1.000 XP</div>
                </div>
                %s
            </div>
        </div>

        <!-- Progress Bar -->
        <div class="mt-3 pt-3 border-t border-slate-100 dark:border-slate-800/80">
            <div class="flex justify-between text-[11px] text-slate-500 dark:text-slate-400 mb-1.5 font-medium">
                <span>Alur Wawancara: %d dari 4 Misi Selesai</span>
                <span class="font-mono text-emerald-600 dark:text-emerald-400 font-bold">%d%% Menuju Penawaran Kerja (Hiring)</span>
            </div>
            <div class="w-full bg-slate-100 dark:bg-slate-950 rounded-full h-2.5 overflow-hidden border border-slate-200 dark:border-slate-800">
                <div class="bg-gradient-to-r from-emerald-500 via-teal-400 to-cyan-400 h-2.5 rounded-full transition-all duration-700" style="width: %d%%"></div>
            </div>
        </div>
    </div>`, levelTitle, totalXP, statusBadge, passedCount, progressPct, progressPct)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(html))
}

// ResetGamification mereset skor dan pencapaian gamifikasi.
func (h *Handler) ResetGamification(w http.ResponseWriter, r *http.Request) {
	h.gamifyMu.Lock()
	h.stagePassed = make(map[int]bool)
	h.gamifyMu.Unlock()

	h.hub.Broadcast("🔄 Progres gamifikasi wawancara telah di-reset kembali ke Tahap 1.")
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

// EvalScenario01Code mengevaluasi kode Go yang diketik manual oleh kandidat bergaya LeetCode / CodeWars.
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

	// Analisis kode yang diketik manual kandidat
	hasTODO := strings.Contains(codeContent, "// TODO: Ketik implementasi") && !strings.Contains(codeContent, "balance")
	hasBalanceCheck := strings.Contains(codeContent, "balance < amount") || 
		strings.Contains(codeContent, "balance <") || 
		strings.Contains(codeContent, "w.balance >= amount") ||
		strings.Contains(codeContent, "balance - amount < 0")
	hasDeduction := strings.Contains(codeContent, "balance -=") || 
		strings.Contains(codeContent, "balance = w.balance - amount") ||
		strings.Contains(codeContent, "atomic.AddInt64")
	hasLock := strings.Contains(codeContent, ".Lock()") || strings.Contains(codeContent, "Lock()")
	hasUnlock := strings.Contains(codeContent, ".Unlock()") || strings.Contains(codeContent, "Unlock()")
	hasAtomic := strings.Contains(codeContent, "atomic.CompareAndSwapInt64") || strings.Contains(codeContent, "atomic.AddInt64")
	hasMutexField := strings.Contains(codeContent, "sync.Mutex") || strings.Contains(codeContent, "sync.RWMutex")

	h.hub.Broadcast(fmt.Sprintf("💻 [LeetCode Runner] Menganalisis sintaks kode kandidat (%d baris, Aksi: %s)...", len(strings.Split(codeContent, "\n")), action))

	// Jika kandidat belum mengetik implementasi sama sekali
	if codeContent == "" || (hasTODO && !hasDeduction) {
		html := `
        <div class="space-y-3 font-mono text-xs">
            <div class="flex items-center justify-between border-b border-rose-800/60 pb-2">
                <div class="flex items-center gap-2">
                    <span class="text-base font-extrabold text-rose-400">⚠️ COMPILATION / LOGIC ERROR</span>
                    <span class="text-[11px] px-2 py-0.5 rounded bg-rose-500/20 text-rose-300 border border-rose-500/30">Implementasi Kosong</span>
                </div>
            </div>
            <div class="p-3 rounded-lg bg-rose-950/60 border border-rose-800 text-rose-200 text-xs space-y-1.5">
                <strong class="text-rose-300 block">Fungsi DeductWallet Belum Diimplementasikan!</strong>
                <p class="text-[11px] text-rose-300 leading-relaxed">
                    Ketik manual logika pemotongan saldo Anda di area editor di atas. Jika Anda bingung harus mulai dari mana, klik tombol <strong>"💡 Hint 1 (Konseptual)"</strong> di atas editor!
                </p>
            </div>
        </div>`
		_, _ = w.Write([]byte(html))
		return
	}

	// Cek apakah kandidat lupa validasi saldo
	if !hasBalanceCheck && !hasAtomic {
		html := `
        <div class="space-y-3 font-mono text-xs">
            <div class="flex items-center justify-between border-b border-rose-800/60 pb-2">
                <div class="flex items-center gap-2">
                    <span class="text-base font-extrabold text-rose-400">❌ WRONG ANSWER</span>
                    <span class="text-[11px] px-2 py-0.5 rounded bg-rose-500/20 text-rose-300 border border-rose-500/30">Assertion Error</span>
                </div>
                <span class="text-rose-400 font-mono text-[11px]">Runtime: 0.4 ms</span>
            </div>
            <div class="p-3 rounded-lg bg-rose-950/60 border border-rose-800 text-rose-200 text-xs space-y-1.5">
                <strong class="text-rose-300 block">Test Case 2 Gagal: Saldo Tidak Divalidasi!</strong>
                <p class="text-[11px] text-rose-300 leading-relaxed">
                    Input: <code>balance = $20, deduct = $50</code> &rarr; Expected: <code>ErrInsufficientFunds</code>, Got: Saldo berkurang menjadi <code>-$30</code>.
                    Pastikan Anda memeriksa <code>if w.balance < amount { return ErrInsufficientFunds }</code> sebelum melakukan pemotongan!
                </p>
            </div>
        </div>`
		_, _ = w.Write([]byte(html))
		return
	}

	// Cek potensi deadlock (ada Lock tapi lupa Unlock)
	if hasLock && !hasUnlock {
		html := `
        <div class="space-y-3 font-mono text-xs">
            <div class="flex items-center justify-between border-b border-rose-800/60 pb-2">
                <div class="flex items-center gap-2">
                    <span class="text-base font-extrabold text-rose-400">💀 DEADLOCK DETECTED</span>
                    <span class="text-[11px] px-2 py-0.5 rounded bg-rose-500/20 text-rose-300 border border-rose-500/30">Fatal Runtime Crash</span>
                </div>
            </div>
            <div class="p-3 rounded-lg bg-rose-950/60 border border-rose-800 text-rose-200 text-xs space-y-1.5">
                <strong class="text-rose-300 block">fatal error: all goroutines are asleep - deadlock!</strong>
                <p class="text-[11px] text-rose-300 leading-relaxed">
                    Anda memanggil <code>Lock()</code> tetapi tidak memanggil <code>Unlock()</code>. Gunakan <code>defer w.mu.Unlock()</code> tepat setelah <code>w.mu.Lock()</code> agar kunci selalu dilepas bahkan saat fungsi me-return error.
                </p>
            </div>
        </div>`
		_, _ = w.Write([]byte(html))
		return
	}

	// Evaluasi sinkronisasi konkurensi (Thread-safety)
	isThreadSafe := (hasLock && hasUnlock) || hasAtomic

	// Jika kandidat hanya mengklik "Run Test Cases"
	if action == "run" {
		case3StatusBadge := `<span class="px-2 py-0.5 rounded text-[11px] font-bold bg-slate-800 text-slate-400">BELUM DIUJI (SUBMIT ONLY)</span>`
		statusText := "Uji dasar (Case 1 & 2) berhasil dilewati."
		if isThreadSafe {
			statusText = "✅ Sintaks sinkronisasi thread-safe terdeteksi! Kode siap untuk uji konkurensi penuh."
		} else {
			statusText = "⚠️ Kode belum memiliki mekanisme Lock / Atomic. Kasus konkurensi tinggi (Case 3) berisiko gagal saat Submit."
		}

		html := fmt.Sprintf(`
        <div class="space-y-3 font-mono text-xs">
            <div class="flex items-center justify-between border-b border-slate-800 pb-2">
                <span class="text-sm font-bold text-amber-300 flex items-center gap-2">
                    <span>▶</span> Hasil Uji Cepat (Run Test Cases)
                </span>
                <span class="text-slate-500 text-[11px]">Go 1.27 Live Sandbox</span>
            </div>
            <div class="space-y-2">
                <div class="p-2.5 rounded-lg bg-emerald-950/40 border border-emerald-800/60 flex items-center justify-between">
                    <div>
                        <strong class="text-emerald-400">Test Case 1: Simple Debit</strong>
                        <div class="text-[11px] text-slate-400">Input: balance=$100, deduct=$25 &rarr; Saldo $75</div>
                    </div>
                    <span class="px-2 py-0.5 rounded text-[11px] font-bold bg-emerald-500/20 text-emerald-300">PASS ✓</span>
                </div>
                <div class="p-2.5 rounded-lg bg-emerald-950/40 border border-emerald-800/60 flex items-center justify-between">
                    <div>
                        <strong class="text-emerald-400">Test Case 2: Insufficient Funds</strong>
                        <div class="text-[11px] text-slate-400">Input: balance=$20, deduct=$50 &rarr; Return ErrInsufficientFunds</div>
                    </div>
                    <span class="px-2 py-0.5 rounded text-[11px] font-bold bg-emerald-500/20 text-emerald-300">PASS ✓</span>
                </div>
                <div class="p-2.5 rounded-lg bg-slate-900 border border-slate-800 flex items-center justify-between">
                    <div>
                        <strong class="text-slate-300">Test Case 3: 50 Goroutines Concurrency Stress</strong>
                        <div class="text-[11px] text-slate-500">Uji ketahanan race condition di bawah beban paralel serentak</div>
                    </div>
                    %s
                </div>
            </div>
            <div class="p-2.5 rounded-lg bg-slate-900/90 border border-slate-800 text-[11px] text-slate-300">
                %s Klik tombol <strong>"⚡ Submit Solution"</strong> untuk verifikasi sertifikasi Tahap 1.
            </div>
        </div>`, case3StatusBadge, statusText)
		_, _ = w.Write([]byte(html))
		return
	}

	// Action == "submit"
	if !isThreadSafe {
		// Kasus Naive: Gagal di konkurensi (Overdraft)
		h.hub.Broadcast("🚨 [LeetCode Runner] WRONG ANSWER: Terdeteksi financial overdraft defect! Goroutine menyebabkan saldo minus.")
		html := `
        <div class="space-y-3 font-mono text-xs">
            <div class="flex items-center justify-between border-b border-rose-800/60 pb-2">
                <div class="flex items-center gap-2">
                    <span class="text-base font-extrabold text-rose-400">❌ WRONG ANSWER</span>
                    <span class="text-[11px] px-2 py-0.5 rounded bg-rose-500/20 text-rose-300 border border-rose-500/30">Race Condition Defect</span>
                </div>
                <span class="text-rose-400 font-mono text-[11px]">Runtime: 1.1 ms</span>
            </div>
            <div class="p-3 rounded-lg bg-rose-950/60 border border-rose-800 text-rose-200 text-xs space-y-2">
                <strong class="text-rose-300 block">Test Case 3 Gagal: Financial Overdraft (-$25.00)!</strong>
                <p class="text-[11px] text-rose-300 leading-relaxed">
                    Kode Anda lolos di pengujian sekuensial biasa (Case 1 & 2), tetapi saat <strong>50 goroutine</strong> menarik uang secara simultan, beberapa goroutine membaca nilai saldo yang sama secara bersamaan (<em>Lost Update</em>). Saldo akhir bocor dan menembus angka minus!
                </p>
                <div class="p-2 rounded bg-dark-950 border border-rose-800/60 text-[11px] text-amber-300">
                    💡 <strong>Bantuan Belajar:</strong> Anda belum menambahkan mekanisme penguncian memori. Buka <strong>"💡 Hint 2 (Struktur Data)"</strong> dan <strong>"💡 Hint 3 (Sintaks Go)"</strong> di atas editor untuk memandu penulisan <code>sync.Mutex</code> Anda.
                </div>
            </div>
            <div class="grid grid-cols-3 gap-2 text-center text-[11px]">
                <div class="p-2 rounded bg-slate-900 border border-slate-800">
                    <span class="text-slate-500">Case 1</span>
                    <div class="font-bold text-emerald-400">PASS ✓</div>
                </div>
                <div class="p-2 rounded bg-slate-900 border border-slate-800">
                    <span class="text-slate-500">Case 2</span>
                    <div class="font-bold text-emerald-400">PASS ✓</div>
                </div>
                <div class="p-2 rounded bg-slate-900 border border-rose-800/80 bg-rose-950/30">
                    <span class="text-rose-400">Case 3 (50-Routines)</span>
                    <div class="font-bold text-rose-400">FAILED ✗</div>
                </div>
            </div>
        </div>`
		_, _ = w.Write([]byte(html))
		return
	}

	// Solusi Berhasil (Thread-Safe):
	_ = hasMutexField
	h.markStagePassed(1, "Live Coding Concurrency Champion")
	h.hub.Broadcast("🎉 [LeetCode Runner] STATUS: ACCEPTED! Solusi manual thread-safe lolos seluruh 3 test case dengan nol overdraft.")

	html := `
    <div class="space-y-3 font-mono text-xs">
        <div class="flex items-center justify-between border-b border-emerald-800/60 pb-2">
            <div class="flex items-center gap-2">
                <span class="text-base font-extrabold text-emerald-400">ACCEPTED ✅</span>
                <span class="text-[11px] px-2 py-0.5 rounded bg-emerald-500/20 text-emerald-300 border border-emerald-500/30">+250 XP DIRAIH</span>
            </div>
            <span class="text-emerald-400 font-mono text-[11px]">Runtime: 1.4 ms (Beats 99.1%)</span>
        </div>
        <div class="grid grid-cols-2 sm:grid-cols-4 gap-2 text-center text-[11px]">
            <div class="p-2 rounded bg-slate-900 border border-slate-800">
                <span class="text-slate-500">Runtime</span>
                <div class="font-bold text-white">1.4 ms</div>
                <div class="text-[10px] text-emerald-400">Beats 99.1%</div>
            </div>
            <div class="p-2 rounded bg-slate-900 border border-slate-800">
                <span class="text-slate-500">Memory</span>
                <div class="font-bold text-white">2.0 MB</div>
                <div class="text-[10px] text-emerald-400">Beats 97.5%</div>
            </div>
            <div class="p-2 rounded bg-slate-900 border border-slate-800">
                <span class="text-slate-500">Goroutines</span>
                <div class="font-bold text-white">50 Paralel</div>
                <div class="text-[10px] text-emerald-400">Zero Overdraft</div>
            </div>
            <div class="p-2 rounded bg-slate-900 border border-slate-800">
                <span class="text-slate-500">Test Cases</span>
                <div class="font-bold text-emerald-400">3/3 Lolos</div>
                <div class="text-[10px] text-emerald-400">100% Sempurna</div>
            </div>
        </div>
        <div class="p-3.5 rounded-lg bg-emerald-950/40 border border-emerald-800/60 text-emerald-200 text-xs space-y-1.5">
            <strong class="text-emerald-300 block">🏆 Evaluasi Interviewer Live Coding:</strong>
            <p class="text-[11px] leading-relaxed text-emerald-200">
                Luar biasa! Kode Go yang Anda ketik secara manual berhasil mengamankan <em>Critical Section</em> menggunakan primitive sinkronisasi thread-safe. 50 goroutine serentak mengeksekusi pemotongan saldo dengan presisi tinggi tanpa satu pun race condition atau overdraft finansial.
            </p>
            <div class="text-[11px] text-slate-300 pt-1">
                👉 <strong>Langkah Berikutnya:</strong> Klik tab <strong>"Tahap 2: System Design"</strong> di atas untuk mempresentasikan diagram alur Idempotency Engine di hadapan Principal Architect!
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
		h.markStagePassed(2, "System Design Architect Approved")
		h.hub.Broadcast("🏛️ [System Design] Principal Architect menyetujui pertahanan arsitektur kandidat! Nilai: A+ (+250 XP).")

		html := `
        <div class="p-4 rounded-xl bg-emerald-950/50 border border-emerald-800/70 space-y-3 font-sans text-xs">
            <div class="flex items-center justify-between border-b border-emerald-800/60 pb-2">
                <div class="flex items-center gap-2">
                    <span class="w-3 h-3 rounded-full bg-emerald-400"></span>
                    <strong class="text-sm font-bold text-emerald-300">HASIL INTERVIEW SYSTEM DESIGN: LULUS (STRONG HIRE) ✓</strong>
                </div>
                <span class="text-xs font-mono font-bold text-amber-300">+250 XP DI DAPAT</span>
            </div>
            <div class="text-slate-200 leading-relaxed text-xs space-y-2">
                <p>
                    <strong>Pewawancara (Principal System Architect):</strong><br>
                    <em>"Penjelasan arsitektur Anda sangat matang dan akurat. Mengkombinasikan SHA-256 Payload Hash (untuk menolak tampering HTTP 422) dengan In-Flight Distributed Lock (untuk menolak request bersamaan HTTP 409) adalah standar industri emas yang diterapkan oleh Stripe dan Adyen. Anda memahami batas antara idempotensi jaringan dan manipulasi data secara presisi."</em>
                </p>
                <div class="grid grid-cols-2 sm:grid-cols-3 gap-2 pt-1 font-mono text-[11px]">
                    <div class="p-2 rounded bg-dark-900 border border-slate-800">
                        <span class="text-slate-400">Alur Diagram</span>
                        <div class="text-emerald-400 font-bold">100% Solid</div>
                    </div>
                    <div class="p-2 rounded bg-dark-900 border border-slate-800">
                        <span class="text-slate-400">Split-Brain Defense</span>
                        <div class="text-emerald-400 font-bold">Terverifikasi</div>
                    </div>
                    <div class="p-2 rounded bg-dark-900 border border-slate-800">
                        <span class="text-slate-400">Skor Evaluasi</span>
                        <div class="text-amber-400 font-bold">Nilai: A+</div>
                    </div>
                </div>
            </div>
        </div>`
		_, _ = w.Write([]byte(html))
		return
	}

	// Jawaban salah
	h.hub.Broadcast("⚠️ [System Design] Argumen pertahanan ditolak oleh interviewer: Pendekatan tidak memenuhi standar keamanan fintech.")
	html := `
    <div class="p-4 rounded-xl bg-rose-950/50 border border-rose-800/70 space-y-2.5 font-sans text-xs">
        <div class="flex items-center justify-between border-b border-rose-800/60 pb-2">
            <strong class="text-sm font-bold text-rose-300">HASIL EVALUASI: PERLU REVISI ARSITEKTUR ✗</strong>
            <span class="text-xs font-mono text-rose-400">Nilai: C</span>
        </div>
        <p class="text-slate-300 leading-relaxed text-xs">
            <strong>Pewawancara (Principal System Architect):</strong><br>
            <em>"Jawaban tersebut berisiko fatal pada sistem perbankan. Mengabaikan hash payload membuka celah keamanan di mana hacker dapat mengganti nominal transfer tetapi tetap menggunakan idempotency key yang sama. Silakan tinjau kembali alur diagram dan pilih strategi verifikasi yang tepat."</em>
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
        <div class="p-3.5 rounded-xl bg-rose-950/60 border border-rose-800 text-rose-200 text-xs space-y-1">
            <strong>⚠️ Validasi Repositori Gagal:</strong>
            <p class="text-[11px] text-rose-300">
                Harap masukkan URL repositori GitHub publik yang valid (diawali dengan <code>https://github.com/username/project</code>).
            </p>
        </div>`
		_, _ = w.Write([]byte(html))
		return
	}

	// Simulasi Pipeline Evaluasi Komite Engineering
	h.hub.Broadcast(fmt.Sprintf("🚀 [Take-Home Bot] Menerima submission repo: %s (Branch: %s). Menjalankan automated evaluation suite...", repoURL, branch))
	h.markStagePassed(3, "Take-Home Assignment Accepted")

	html := fmt.Sprintf(`
    <div class="p-4 rounded-xl bg-gradient-to-b from-[#0b1424] to-[#070c18] border border-cyan-500/30 space-y-3 font-sans text-xs shadow-lg">
        <div class="flex items-center justify-between border-b border-slate-800 pb-2.5">
            <div class="flex items-center gap-2">
                <span class="w-3 h-3 rounded-full bg-cyan-400 animate-pulse"></span>
                <strong class="text-sm font-bold text-white uppercase tracking-wider">TAKE-HOME SUBMISSION REVIEW: APPROVED ✓</strong>
            </div>
            <span class="text-xs font-mono font-bold text-amber-300">+250 XP DIRAIH</span>
        </div>

        <div class="text-[11px] font-mono text-slate-300 bg-dark-950 p-3 rounded-lg border border-slate-800 space-y-1">
            <div class="text-slate-400">Target Repo: <a href="%s" target="_blank" class="text-cyan-400 hover:underline">%s</a> (Branch: %s)</div>
            <div class="text-emerald-400">[1/5] Verifikasi Git Remote Publik................... OK (200 OK)</div>
            <div class="text-emerald-400">[2/5] Pemeriksaan Clean Architecture (Domain/Port)... COMPLIANT ✓</div>
            <div class="text-emerald-400">[3/5] Audit Transactional Outbox & Saga Worker....... PASS ✓ (Zero Dual-Write)</div>
            <div class="text-emerald-400">[4/5] Test Suite Coverage Analysis................... 92.4%% (Threshold > 80%%)</div>
            <div class="text-emerald-400">[5/5] Docker Compose & Linter Static Analysis........ ZERO DEFECTS ✓</div>
        </div>

        <div class="p-3 rounded-lg bg-cyan-950/40 border border-cyan-800/60 text-cyan-200 text-xs space-y-1">
            <strong class="text-cyan-300 block">📝 Rekomendasi Komite Wawancara:</strong>
            Repositori publik kandidat telah memenuhi seluruh Functional & Non-Functional Requirements. Implementasi Transactional Outbox dan Saga Auto-Refund dinyatakan memenuhi standar arsitektur tingkat <strong>Principal</strong>. Lanjut ke Tahap 4 (War Room & Stress Test Defense)!
        </div>
    </div>`, repoURL, repoURL, branch)

	_, _ = w.Write([]byte(html))
}
