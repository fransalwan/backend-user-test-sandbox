# 🏛️ Fintech Junior Backend Assessment Sandbox

[![Go CI](https://github.com/fransalwan/backend-user-test-sandbox/actions/workflows/ci.yml/badge.svg)](https://github.com/fransalwan/backend-user-test-sandbox/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/fransalwan/backend-user-test-sandbox)](https://goreportcard.com/report/github.com/fransalwan/backend-user-test-sandbox)

> **A Production-Grade Technical Interview Sandbox & Competency Pipeline tailored for Junior & Entry-Level Backend Engineers (Fintech & Core Banking Track).**

---

## 📌 Gambaran Umum & Fokus Entry-Level

Sandbox ini dirancang khusus untuk memvalidasi dan melatih kompetensi rekayasa perangkat lunak backend tingkat pemula (*Junior / Associate Backend Engineer*) dengan standar industri finansial nyata. Menghindari abstraksi teoritis semata, sandbox ini menyediakan pipeline evaluasi 5 tahap yang dapat dijalankan secara lokal dengan tumpukan teknologi modern: **Go (Golang)**, **PostgreSQL 16**, **Redis 7**, dan **Docker Compose**.

### 🎯 4 Pilar Fundamental yang Diuji:
1. **Zero-Float Money Invariant**: Representasi saldo dan mutasi finansial wajib menggunakan `int64` (satuan sen terkecil) untuk mengeliminasi kesalahan pembulatan biner IEEE-754.
2. **ACID Transaction & Concurrency Locking**: Penguncian tingkat baris (*Row-Level Lock*: `SELECT FOR UPDATE`) pada PostgreSQL untuk mencegah *race condition* dan saldo minus (*negative balance overdraft*).
3. **Idempotency Engine**: Penanganan request berulang dengan status *in-flight collision* (`HTTP 409 Conflict`) dan deteksi anomali mutasi parameter (`HTTP 422 Unprocessable Entity`).
4. **Clean Architecture & Table-Driven Tests**: Pemisahan tanggung jawab yang terisolasi (*Handler* $\rightarrow$ *Service/Usecase* $\rightarrow$ *Repository*) serta pengujian unit berulang berbasis tabel (*table-driven unit testing*).

---

## 🏛️ Arsitektur Direktori Proyek

```text
backend-user-test-sandbox/
├── .github/
│   └── workflows/              # GitHub Actions CI automated testing pipeline
├── cmd/
│   └── sandbox/                # Application entrypoint & HTTP server
├── internal/
│   ├── core/                   # DOMAIN LAYER (Pure Go, tanpa dependensi eksternal)
│   │   ├── entity/             # Model domain: Wallet, LedgerEntry, Transaction
│   │   ├── port/               # Interface Port: Repository, Cache, Idempotency
│   │   └── service/            # Logika bisnis transfer & pembukuan berpasangan
│   ├── adapter/                # INFRASTRUCTURE LAYER
│   │   ├── postgres/           # Implementasi persistensi PostgreSQL ACID
│   │   ├── redis/              # Distributed lock & token bucket engine
│   │   └── mock/               # Mock external banking gateways
│   └── delivery/               # PRESENTATION LAYER
│       └── http/               # HTTP REST Handlers, SSE Event Stream, Template Views
│           ├── handler.go      # Handlers & Simulation Business Logic
│           ├── handler_test.go # Comprehensive Table-Driven Unit Tests
│           └── templates/      # Dashboard antarmuka HTMX + Tailwind CSS
├── migrations/                 # Skema DDL Database Terstruktur
│   ├── 000001_init_schema.up.sql    # DDL Wallets, Transactions, & Ledger
│   └── 000001_init_schema.down.sql  # Clean rollback script
├── docs/
│   ├── candidate_action_guide.md       # Panduan aksi kandidat di luar app (CV, take-home, verbal interview)
│   ├── fintech_api.postman_collection.json # Koleksi Postman resmi siap impor
│   └── interview_qa.md                 # Panduan tanya-jawab wawancara teknis backend entry-level
├── docker-compose.yml          # Local container stack: PostgreSQL 16 & Redis 7
├── Makefile                    # Perintah otomatisasi lint, test, dan run
├── go.mod
└── README.md
```

---

## 🌐 3 Pilar Ekosistem Sandbox (Pemisahan Pembelajaran, Latihan, & Ujian)

Untuk memberikan pengalaman belajar yang terstruktur dan memisahkan proses seleksi formal dari tempat berlatih bebas tekanan, sandbox ini membagi fitur ke dalam **3 Halaman Mandiri**:

```text
┌─────────────────────────┐     ┌─────────────────────────────┐     ┌─────────────────────────┐
│     📚 BOOTCAMP         │     │     ⚡ EXERCISE LAB         │     │   🏆 HIRING GAUNTLET    │
│       (/bootcamp)       │────▶│       (/exercise)           │────▶│      (/ atau /hiring)       │
│  Fondasi Teori & Konsep │     │  Latihan Mental & Debugging │     │ 5 Tahap Seleksi Formal  │
│  Active Recall Flashcard│     │  Repetisi Bebas Tekanan     │     │ Offer Letter Cetak A4   │
└─────────────────────────┘     └─────────────────────────────┘     └─────────────────────────┘
```

1. **📚 Bootcamp (`/bootcamp`) — Tempat Mempelajari Teori Fundamental**:
   - Modul Algoritma: Analisis Big-O $\mathcal{O}(1)$ vs $\mathcal{O}(N)$ vs $\mathcal{O}(N^2)$, pola Target Complement, dan trade-off memori Slice vs Hash Map di Go.
   - Modul System Design: Mengapa disable button di frontend tidak cukup (API Idempotency), Double-Entry Ledger atomik, dan proteksi Token Bucket Rate Limiting.
   - **Active Recall Flashcard Simulator**: 5 pertanyaan verbal wawancara Tech Lead untuk melatih daya ingat aktif.

2. **⚡ Exercise Lab (`/exercise`) — Melatih Mental Live Coding & Repetisi Debugging**:
   - **Tujuan Khusus**: Mengasah ketenangan mental kandidat menghadapi sesi live coding tanpa rasa takut gagal atau konsekuensi penolakan rekrutmen.
   - **4 Live Debugging Drills (Repetisi Kasus Nyata)**:
     1. *Drill 1: Floating-Point Fee Disaster* &mdash; Refactor kalkulasi fee desimal rentan bocor menjadi `int64` (sen).
     2. *Drill 2: Unprotected Concurrent Hot Wallet* &mdash; Amankan operasi debit multi-goroutine menggunakan `sync.Mutex` (lolos Go `-race` detector).
     3. *Drill 3: In-Flight Idempotency State Trap* &mdash; Tangani status `PROCESSING` untuk mengembalikan `HTTP 409 Conflict` dan mencegah *double billing*.
     4. *Drill 4: Goroutine Context Leak* &mdash; Tambahkan `select { case <-ctx.Done(): ... }` pada background poller pihak ketiga untuk mengeliminasi zombie goroutines.
   - **Rapid-Fire Theory Mastery Quiz**: Kuis evaluasi pemahaman teori dengan umpan balik teknis industri instan.
   - **Stopwatch Mental Timer**: Fitur pengukur waktu opsional untuk melatih ritme mengetik di bawah simulasi batas waktu.

3. **🏆 Hiring Gauntlet (`/` atau `/hiring`) — Simulasi Ujian Seleksi Resmi 5 Tahap**:
   - Pipeline evaluasi teknis formal berbobot 20% per tahap:
     * **Tahap 1: Live Coding DSA** (Financial Transaction Deduplication $\mathcal{O}(N)$).
     * **Tahap 2: System Design Architecture Defense** (Idempotency, Double-Entry Ledger, Circuit Breaker).
     * **Tahap 3: Take-Home Code Review** (Clean Architecture, SQL Migration, Docker Compose, Postman).
     * **Tahap 4: Incident Resiliency War Room** (Token Bucket Rate Limiting 40 RPS).
     * **Tahap 5: Surat Penawaran Kerja Resmi (Offer Letter)** (Format cetak resmi legal A4 2 Halaman Pas).

## 🛠️ Persyaratan Sistem & Tech Stack

- **Go (Golang)**: Versi 1.23+ (Disarankan 1.27)
- **Docker & Docker Compose**: Untuk menjalankan PostgreSQL 16 & Redis 7 secara lokal
- **Frontend Dashboard**: Go `html/template` + HTMX (tanpa perlu Node.js atau `npm`)
- **Real-time Event**: Server-Sent Events (SSE) bawaan Go HTTP standard library

---

## 🚀 Panduan Menjalankan Sandbox

### 1. Menjalankan Database & Cache Lokal (Docker Compose)
Jalankan container PostgreSQL 16 dan Redis 7 dengan satu perintah:

```bash
docker compose up -d
```

- **PostgreSQL**: Port `5432` (`postgres:postgres@localhost:5432/fintech_db?sslmode=disable`)
- **Redis**: Port `6379` (`redis://localhost:6379`)

Untuk mematikan container:
```bash
docker compose down
```

### 2. Menjalankan Aplikasi Sandbox
Jalankan server HTTP lokal:

```bash
# Melalui Go CLI
go run ./cmd/sandbox

# Atau jika menggunakan binary yang sudah dikompilasi (Windows)
.\sandbox.exe
```

Buka peramban (browser) di alamat:
👉 **[http://localhost:8080](http://localhost:8080)**

---

## 🧪 Menjalankan Pengujian Otomatis (Unit & Race Tests)

Seluruh pengujian unit mengadopsi pola **Table-Driven Tests** standar Go untuk menguji *edge cases*:

```bash
# Menjalankan seluruh pengujian unit
go test -v ./...

# Menjalankan pengujian paket HTTP handler
go test -v ./internal/delivery/http

# Menjalankan pengujian dengan Go Race Detector
go test -race ./...
```

Contoh keluaran pengujian:
```text
=== RUN   TestFindDuplicateTransactions_TableDriven
=== RUN   TestFindDuplicateTransactions_TableDriven/empty_slice
=== RUN   TestFindDuplicateTransactions_TableDriven/no_duplicates_distinct_hashes
=== RUN   TestFindDuplicateTransactions_TableDriven/duplicate_found_within_window
--- PASS: TestFindDuplicateTransactions_TableDriven (0.00s)
=== RUN   TestHTTP_Endpoints_TableDriven
=== RUN   TestHTTP_Endpoints_TableDriven/root_dashboard_GET
=== RUN   TestHTTP_Endpoints_TableDriven/gamification_status_GET
=== RUN   TestHTTP_Endpoints_TableDriven/takehome_submission_valid_repo_POST
--- PASS: TestHTTP_Endpoints_TableDriven (0.01s)
PASS
ok      github.com/your-username/backend-user-test-sandbox/internal/delivery/http       0.045s
```

---

## 💼 Panduan Wawancara & Aksi Nyata Kandidat di Luar Sandbox

Tersedia dua panduan komprehensif bagi kandidat:
1. 👉 **[`docs/candidate_action_guide.md`](file:///c:/Users/Admin/Desktop/portfolio-opensource/backend-user-test-sandbox/docs/candidate_action_guide.md)**:
   - Template penulisan resume/CV standar ATS untuk memamerkan proyek ini.
   - SOP menghadapi tugas *Take-Home Test* nyata dari perusahaan (memanfaatkan boilerplate, docker, dan skema SQL dari sandbox ini).
   - Skrip latihan verbal untuk menjawab pertanyaan wawancara teknis secara lancar.
2. 👉 **[`docs/interview_qa.md`](file:///c:/Users/Admin/Desktop/portfolio-opensource/backend-user-test-sandbox/docs/interview_qa.md)**:
   - Pendalaman teknis: Kenapa dilarang `float64`, Row Lock vs Optimistic Lock, HTTP 409 vs 422, Filosofi `if err != nil`, dan Token Bucket Rate Limiting.
3. 👉 **[`docs/fintech_api.postman_collection.json`](file:///c:/Users/Admin/Desktop/portfolio-opensource/backend-user-test-sandbox/docs/fintech_api.postman_collection.json)**:
   - Koleksi Postman resmi siap impor (atau unduh langsung via tombol di web dashboard).

---

## 📄 Format Dokumen Cetak Surat Penawaran Kerja (Tahap 5)
Ketika kandidat menyelesaikan 5 tahap evaluasi, sistem secara otomatis menerbitkan dokumen legal A4:
- Standar tipografi resmi korporasi: **Times New Roman 12pt** dengan spasi teratur.
- Kop surat formal, nomor surat keputusan rekrutmen, dan rincian kompensasi bulanan (*IDR 13.500.000* + tunjangan).
- Fitur cetak langsung (`window.print()`) yang otomatis menyembunyikan navigasi web untuk hasil cetak PDF/kertas A4 bersih.

---

## 📜 Lisensi & Kontribusi
Proyek ini bersifat open-source dan ditujukan untuk memajukan standar kesiapan teknis para kandidat *Junior Backend Engineer* di ekosistem rekayasa perangkat lunak finansial. Pull requests dan penyempurnaan skenario dipersilakan!
