package http

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
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
	statusColor := "text-white"
	anomalyBadge := ""
	if balance < 0 {
		statusColor = "text-rose-400 font-bold"
		anomalyBadge = `<span class="ml-2 px-1.5 py-0.5 text-[10px] rounded bg-rose-500/20 text-rose-300 border border-rose-500/40">OVERDRAFT!</span>`
	}

	html := fmt.Sprintf(`
    <div id="wallet-cards" class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div class="rounded-xl bg-gradient-to-b from-[#0f172a] to-[#0b1120] border border-slate-800 p-5 shadow-md">
            <div class="flex justify-between items-start">
                <div>
                    <div class="flex items-center gap-2">
                        <span class="text-xs text-slate-400 font-mono">wallet-alice-001</span>
                        <span class="px-2 py-0.5 text-[10px] rounded-full bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 font-medium">Rekening Utama</span>
                    </div>
                    <div class="text-base font-bold text-white mt-1 flex items-center">
                        Alice (Hot Wallet Target) %s
                    </div>
                </div>
                <span class="px-2.5 py-1 text-xs font-mono font-semibold rounded-lg bg-emerald-500/10 text-emerald-400 border border-emerald-500/30">
                    Versi: %d
                </span>
            </div>
            <div class="mt-5 flex items-baseline justify-between border-t border-slate-800/60 pt-3">
                <span class="text-xs text-slate-400 font-medium">Saldo Tersedia</span>
                <div class="text-right">
                    <span class="text-3xl font-mono font-extrabold %s tracking-tight">$%0.2f</span>
                    <div class="text-[11px] text-slate-500 font-mono mt-0.5">%d sen (int64)</div>
                </div>
            </div>
        </div>

        <div class="rounded-xl bg-gradient-to-b from-[#0f172a] to-[#0b1120] border border-slate-800 p-5 shadow-md">
            <div class="flex justify-between items-start">
                <div>
                    <div class="flex items-center gap-2">
                        <span class="text-xs text-slate-400 font-mono">wallet-bob-002</span>
                        <span class="px-2 py-0.5 text-[10px] rounded-full bg-slate-800 text-slate-400 border border-slate-700 font-medium">Rekening Penerima</span>
                    </div>
                    <div class="text-base font-bold text-white mt-1">Bob (Rekening Tujuan)</div>
                </div>
                <span class="px-2.5 py-1 text-xs font-mono font-semibold rounded-lg bg-slate-800 text-slate-300 border border-slate-700">
                    Versi: 1
                </span>
            </div>
            <div class="mt-5 flex items-baseline justify-between border-t border-slate-800/60 pt-3">
                <span class="text-xs text-slate-400 font-medium">Saldo Tersedia</span>
                <div class="text-right">
                    <span class="text-3xl font-mono font-extrabold text-white tracking-tight">$250.00</span>
                    <div class="text-[11px] text-slate-500 font-mono mt-0.5">25.000 sen (int64)</div>
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
    <div id="gamification-hud" class="rounded-2xl bg-gradient-to-r from-dark-900 via-[#101932] to-dark-900 border border-slate-800 p-5 shadow-lg">
        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
            <div class="flex items-center gap-3">
                <div class="w-12 h-12 rounded-2xl bg-gradient-to-tr from-emerald-600 to-cyan-500 flex items-center justify-center text-2xl shadow-md">
                    👨‍💻
                </div>
                <div>
                    <div class="flex items-center gap-2">
                        <span class="text-sm font-bold text-white">Kandidat: Frans Alwan</span>
                        <span class="text-xs text-slate-400 font-mono">&bull; Target: Principal Backend Engineer</span>
                    </div>
                    <div class="text-xs font-semibold text-emerald-400 mt-0.5">%s</div>
                </div>
            </div>

            <div class="flex items-center gap-4">
                <div class="text-right">
                    <div class="text-xs text-slate-400">Total Pengalaman (XP)</div>
                    <div class="text-xl font-mono font-extrabold text-white">%d / 1.000 XP</div>
                </div>
                %s
            </div>
        </div>

        <!-- Progress Bar -->
        <div class="mt-4 pt-3 border-t border-slate-800/80">
            <div class="flex justify-between text-[11px] text-slate-400 mb-1.5 font-medium">
                <span>Alur Wawancara: %d dari 4 Misi Selesai</span>
                <span class="font-mono text-emerald-400">%d%% Menuju Penawaran Kerja (Hiring)</span>
            </div>
            <div class="w-full bg-slate-950 rounded-full h-2.5 overflow-hidden border border-slate-800">
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
