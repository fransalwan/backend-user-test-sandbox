package http

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	racecondition "github.com/fransalwan/backend-user-test-sandbox/scenarios/01_race_condition"
	"github.com/fransalwan/backend-user-test-sandbox/scenarios/02_idempotency"
)

//go:embed templates/*
var templateFS embed.FS

// Handler mengelola rute HTTP pada lapisan delivery.
type Handler struct {
	tmpl           *template.Template
	hub            *SSEHub
	raceSim        *racecondition.Simulator
	idempotencySim *idempotency.Simulator
}

// NewHandler menginisialisasi delivery handler beserta simulator dan SSE Hub.
func NewHandler() (*Handler, error) {
	tmpl, err := template.ParseFS(templateFS, "templates/index.html")
	if err != nil {
		return nil, fmt.Errorf("gagal mem-parsing template html: %w", err)
	}

	raceSim := racecondition.NewSimulator(100000) // Saldo awal $1,000.00 (100.000 cents)
	idemSim := idempotency.NewSimulator()
	hub := NewSSEHub()

	return &Handler{
		tmpl:           tmpl,
		hub:            hub,
		raceSim:        raceSim,
		idempotencySim: idemSim,
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

// GetWallets mengembalikan partial HTML kartu status saldo dompet (Bahasa Indonesia).
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
    <div id="wallet-cards" class="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div class="p-4 rounded-xl bg-slate-950 border border-slate-800">
            <div class="flex justify-between items-start">
                <div>
                    <div class="text-xs text-slate-400 font-mono">wallet-alice-001</div>
                    <div class="text-base font-semibold text-white flex items-center">
                        Alice (Hot Wallet) %s
                    </div>
                </div>
                <span class="px-2 py-0.5 text-[10px] font-mono rounded bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">Versi: %d</span>
            </div>
            <div class="mt-4 flex items-baseline justify-between">
                <span class="text-xs text-slate-500">Saldo Rekening</span>
                <span class="text-2xl font-mono %s">$%0.2f</span>
            </div>
            <div class="mt-2 text-[11px] text-slate-500 font-mono">%d sen (int64 - Non Float)</div>
        </div>

        <div class="p-4 rounded-xl bg-slate-950 border border-slate-800">
            <div class="flex justify-between items-start">
                <div>
                    <div class="text-xs text-slate-400 font-mono">wallet-bob-002</div>
                    <div class="text-base font-semibold text-white">Bob (Rekening Tujuan)</div>
                </div>
                <span class="px-2 py-0.5 text-[10px] font-mono rounded bg-slate-800 text-slate-400 border border-slate-700">Versi: 1</span>
            </div>
            <div class="mt-4 flex items-baseline justify-between">
                <span class="text-xs text-slate-500">Saldo Rekening</span>
                <span class="text-2xl font-mono font-bold text-white">$250.00</span>
            </div>
            <div class="mt-2 text-[11px] text-slate-500 font-mono">25.000 sen (int64)</div>
        </div>
    </div>
    `, anomalyBadge, version, statusColor, balanceDollars, balance)

	_, _ = w.Write([]byte(html))
}

// ResetWallets mengembalikan saldo Alice ke $1,000.00 (100.000 cents).
func (h *Handler) ResetWallets(w http.ResponseWriter, r *http.Request) {
	h.raceSim.Reset(100000)
	h.hub.Broadcast("🔄 Saldo Dompet Alice berhasil di-reset kembali ke $1.000,00 (100.000 sen)")
	h.GetWallets(w, r)
}

// RunScenario01 mengeksekusi simulasi Skenario 1 (Race Condition & Concurrency).
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
                <div class="text-sm font-bold text-emerald-400 font-mono">%d transaksi</div>
            </div>
            <div class="p-2.5 rounded-lg bg-slate-900 border border-slate-800">
                <div class="text-slate-500">Ditolak / Konflik</div>
                <div class="text-sm font-bold text-amber-400 font-mono">%d ditolak (%d retry)</div>
            </div>
        </div>
    </div>
    `, result.Strategy, statusBadge, result.DurationMilliseconds, alertBox,
		float64(result.InitialBalance)/100, float64(result.FinalBalance)/100,
		result.SuccessfulRequests, result.FailedRequests, result.ConflictRetries)

	_, _ = w.Write([]byte(html))
}

// RunScenario02 mengeksekusi simulasi Skenario 2 (Idempotensi & Network Retries).
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

	w.Header().Set("HX-Trigger", "refreshWallets")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	statusBadge := `<span class="px-2 py-0.5 text-xs font-semibold rounded bg-emerald-500/20 text-emerald-400 border border-emerald-500/30">IDEMPOTEN &bull; ZERO DOUBLE-SPENDING</span>`
	alertBox := ""

	if result.HasDoubleSpending {
		statusBadge = `<span class="px-2 py-0.5 text-xs font-semibold rounded bg-rose-500/20 text-rose-400 border border-rose-500/30 animate-pulse">DOUBLE-SPENDING TERJADI! 💸💸💸</span>`
		alertBox = fmt.Sprintf(`
        <div class="p-3 rounded-lg bg-rose-950/60 border border-rose-800 text-rose-200 text-xs">
            <strong>🚨 Bencana Finansial:</strong> Permintaan transfer duplikat tanpa kunci idempotensi memotong saldo sebanyak <strong>%d kali</strong>!<br>
            Total saldo yang lenyap: <strong>$%0.2f</strong> padahal user hanya berniat transfer 1 kali. Inilah mengapa Idempotency Key wajib diimplementasikan di semua API pembayaran.
        </div>`, result.ProcessedCount, float64(result.TotalDeducted)/100)
	} else if result.InFlightConflicts > 0 {
		statusBadge = `<span class="px-2 py-0.5 text-xs font-semibold rounded bg-amber-500/20 text-amber-300 border border-amber-500/30">IN-FLIGHT CONFLICT (HTTP 409)</span>`
		alertBox = fmt.Sprintf(`
        <div class="p-3 rounded-lg bg-amber-950/60 border border-amber-800 text-amber-200 text-xs">
            <strong>🛡️ Proteksi In-Flight Aktif:</strong> %d permintaan bersamaan dicegat dengan status <strong>HTTP 409 Conflict</strong> karena transaksi pertama masih berstatus PROCESSING.<br>
            Ini mencegah dua proses/worker memproses pembayaran yang sama secara serentak.
        </div>`, result.InFlightConflicts)
	} else if result.PayloadMismatches > 0 {
		statusBadge = `<span class="px-2 py-0.5 text-xs font-semibold rounded bg-purple-500/20 text-purple-300 border border-purple-500/30">PAYLOAD MISMATCH (HTTP 422)</span>`
		alertBox = `
        <div class="p-3 rounded-lg bg-purple-950/60 border border-purple-800 text-purple-200 text-xs">
            <strong>🛑 Manipulasi Terdeteksi:</strong> Kunci idempotensi sama persis, namun hash payload berbeda. Server menolak eksekusi dengan <strong>HTTP 422 Unprocessable Entity</strong> untuk mencegah manipulasi nominal transfer.
        </div>`
	} else {
		alertBox = fmt.Sprintf(`
        <div class="p-3 rounded-lg bg-emerald-950/60 border border-emerald-800 text-emerald-200 text-xs">
            <strong>✅ Perlindungan Idempotensi Sempurna:</strong> Dari %d kali percobaan permintaan, saldo hanya dipotong <strong>1 kali ($%0.2f)</strong>.<br>
            %d permintaan sisanya langsung menerima respons dari cache tanpa memotong saldo lagi.
        </div>`, result.TotalRequests, float64(cfg.AmountCents)/100, result.CachedReplays)
	}

	html := fmt.Sprintf(`
    <div class="space-y-3">
        <div class="flex items-center justify-between">
            <div class="flex items-center gap-2">
                <span class="text-sm font-bold text-white uppercase tracking-wider">Hasil Uji Idempotensi</span>
                %s
            </div>
            <span class="text-xs text-slate-400 font-mono">Waktu Eksekusi: %d ms</span>
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
    </div>
    `, statusBadge, result.DurationMs, alertBox,
		result.TotalRequests, result.ProcessedCount, float64(result.TotalDeducted)/100,
		result.CachedReplays, result.InFlightConflicts)

	_, _ = w.Write([]byte(html))
}

// ClearIdempotencyRecords menghapus catatan kunci idempotensi.
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
