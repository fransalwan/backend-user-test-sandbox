# 🏛️ Fintech Junior Backend Assessment Sandbox

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
│   └── interview_qa.md         # Panduan tanya-jawab wawancara teknis backend entry-level
├── docker-compose.yml          # Local container stack: PostgreSQL 16 & Redis 7
├── Makefile                    # Perintah otomatisasi lint, test, dan run
├── go.mod
└── README.md
```

---

## 📋 Pipeline Evaluasi Kompetensi Kandidat

Sandbox ini membagi alur penilaian teknis kandidat ke dalam 5 tahapan berbobot standar rekrutmen engineering fintech:

| Tahap | Modul Evaluasi | Bobot | Fokus Penilaian & Deliverables |
| :---: | :--- | :---: | :--- |
| **0** | **Bootcamp Teori & Lab** | Persiapan | Fondasi invarian uang sen (`int64`), *double-entry bookkeeping*, dan lab coding interaktif. |
| **1** | **Live Coding Algoritma DSA** | **20%** | Implementasi deteksi transaksi duplikat dalam kompleksitas waktu optimal $\mathcal{O}(N)$ Hash Map. |
| **2** | **System Design Architecture Defense** | **20%** | Mempertahankan arsitektur *Idempotency Storage*, *Double-Entry Ledger*, dan *External Gateway Resiliency*. |
| **3** | **Clean Architecture Take-Home Submission** | **20%** | Menyerahkan repositori terstruktur, skema SQL Migrations, Docker Compose lokal, dan *table-driven tests*. |
| **4** | **Production War Room Incident Resiliency** | **20%** | Mengaktifkan *Token Bucket Rate Limiter* (40 RPS) untuk mencegah *thundering herd* dan *database pool exhaustion*. |
| **5** | **Surat Penawaran Resmi (Offer Letter)** | **20%** | Penerbitan formal Surat Penawaran Kerja A4 resmi (*Junior Backend Engineer*) dengan verifikasi kelulusan 100%. |

---

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

## 💼 Panduan Wawancara Teknis (Interview Q&A)

Tersedia dokumen panduan tanya-jawab mendalam untuk persiapan wawancara teknis di:
👉 **[`docs/interview_qa.md`](file:///c:/Users/Admin/Desktop/portfolio-opensource/backend-user-test-sandbox/docs/interview_qa.md)**

Topik esensial yang dibahas mencakup:
1. Alasan mutlak larangan `float64` untuk nominal uang dan representasi `int64` sen.
2. Perbedaan *Pessimistic Row-Level Lock* (`SELECT FOR UPDATE`) vs *Optimistic Lock* (`version` check).
3. Siklus *Idempotency Key* (`HTTP 409 Conflict` vs `HTTP 422 Unprocessable Entity`).
4. Perilaku *Slices vs Arrays* di Go dan filosofi *Explicit Error Handling* (`if err != nil`).
5. Batas tanggung jawab Clean Architecture (*Handler*, *Usecase/Service*, *Repository*).
6. Algoritma *Token Bucket* vs *Leaky Bucket* dalam menangani *Thundering Herd*.

---

## 📄 Format Dokumen Cetak Surat Penawaran Kerja (Tahap 5)
Ketika kandidat menyelesaikan 5 tahap evaluasi, sistem secara otomatis menerbitkan dokumen legal A4:
- Standar tipografi resmi korporasi: **Times New Roman 12pt** dengan spasi teratur.
- Kop surat formal, nomor surat keputusan rekrutmen, dan rincian kompensasi bulanan (*IDR 13.500.000* + tunjangan).
- Fitur cetak langsung (`window.print()`) yang otomatis menyembunyikan navigasi web untuk hasil cetak PDF/kertas A4 bersih.

---

## 📜 Lisensi & Kontribusi
Proyek ini bersifat open-source dan ditujukan untuk memajukan standar kesiapan teknis para kandidat *Junior Backend Engineer* di ekosistem rekayasa perangkat lunak finansial. Pull requests dan penyempurnaan skenario dipersilakan!
