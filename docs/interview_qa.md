# 📘 Junior Backend Engineer Technical Interview Guide (Fintech Track)

> Panduan komprehensif persiapan wawancara teknis backend level **Junior / Entry-Level / Associate**. Dokumen ini merangkum pertanyaan fundamental, analogi membumi, failure mode, dan jawaban standar industri yang dicari oleh *Engineering Manager* dan *Lead Engineer*.

---

## 🏦 1. Fundamental Moneter & Konkurensi

### Q1: Mengapa tipe data `float` dilarang keras untuk menyimpan saldo uang?
**Jawaban**:
Format `float32` dan `float64` menggunakan representasi biner pecahan IEEE 754 yang **tidak memiliki presisi eksak** untuk pecahan desimal.
- Contoh: `0.1 + 0.2` pada float menghasilkan `0.30000000000000004`.
- Dalam sistem perbankan dengan jutaan transaksi per hari, selisih pembulatan (*rounding error*) ini akan menyebabkan akumulasi selisih kas (*leakage*) hingga jutaan rupiah.
- **Standar Industri**: Selalu gunakan integer murni (`int64` atau `BIGINT` di database) dalam satuan terkecil mata uang (misal: sen USD atau Rupiah penuh).

### Q2: Mengapa kita tidak boleh hanya mengeksekusi `UPDATE wallets SET balance = balance - 100 WHERE id = 1` tanpa validasi transaksi?
**Jawaban**:
1. **Pencegahan Saldo Negatif (Underflow)**: Query UPDATE langsung tanpa klausa `WHERE balance >= 100` atau constraint database `CHECK (balance >= 0)` akan membuat saldo menjadi minus jika dua request masuk bersamaan.
2. **Prinsip Double-Entry Bookkeeping**: Saldo dompet hanyalah *snapshot turunan*. Setiap mutasi wajib mencatat 2 baris ledger berpasangan (DEBIT dan KREDIT) di dalam satu transaksi database atomik (`BEGIN ... COMMIT`).
3. **Deadlock Hazard pada Transfer Antar-Akun**: Jika User A mentransfer ke User B di saat yang sama User B mentransfer ke User A, kedua transaksi bisa saling menunggu kunci baris (*deadlock*) jika tidak ada standarisasi urutan penguncian (misal: selalu kunci ID yang lebih kecil terlebih dahulu).

### Q3: Kapan kita memilih Pessimistic Locking (`SELECT FOR UPDATE`) dibanding Optimistic Locking?
| Dimensi | Pessimistic Locking (`SELECT FOR UPDATE`) | Optimistic Locking (`WHERE version = v`) |
| :--- | :--- | :--- |
| **Mekanisme** | Mengunci baris database secara eksklusif hingga transaksi selesai. | Mengecek versi tanpa lock; gagal/retry jika versi berubah. |
| **Beban Konflik Tinggi** | Sangat teratur (antrean FIFO baris), tidak ada komputasi yang terbuang sia-sia untuk retry. | Terjadi *retry storm*, memboroskan CPU dan koneksi database. |
| **Use Case Terbaik** | Pemotongan saldo dompet (*wallet deduction*), flash sale kuota terbatas. | Update profil pengguna, edit deskripsi katalog, sinkronisasi batch berkala. |

---

## 🔑 2. Idempotensi & Jaringan Tidak Andal (*Unreliable Network*)

### Q1: Apa yang dimaksud dengan Idempotency dan mengapa krusial di Payment Gateway?
**Jawaban**:
Idempotensi menjamin bahwa memanggil endpoint yang sama berkali-kali dengan parameter yang sama **menghasilkan efek saldo yang sama seperti hanya dipanggil satu kali**.
- Di dunia nyata, koneksi internet ponsel nasabah sering *timeout* tepat setelah mengirim dana. SDK aplikasi nasabah akan secara otomatis mencoba lagi (*retry*).
- Tanpa idempotensi, percobaan ulang tersebut akan memotong saldo nasabah dua kali (*double-spending*).

### Q2: Kapan API harus mengembalikan status HTTP 409 Conflict vs 422 Unprocessable Entity?
- **HTTP 409 Conflict**: Request kedua dengan `Idempotency-Key` yang sama datang saat request pertama masih dalam proses (*in-flight/processing*). Server menolak untuk mencegah tabrakan eksekusi simultan.
- **HTTP 422 Unprocessable Entity**: Request datang dengan `Idempotency-Key` yang sama, namun payload (misal: nominal uang atau penerima) berbeda dari request awal. Ini mengindikasikan ketidakcocokan parameter (*payload mismatch/tampering*).

---

## 🐹 3. Fundamental Pemrograman Golang

### Q1: Apa perbedaan `Slice` dan `Array` di Go, dan mengapa slice bisa memicu race condition?
**Jawaban**:
- **Array**: Memiliki ukuran tetap yang ditentukan saat kompilasi (misal: `[5]int`), nilainya disimpan secara *value-copy*.
- **Slice**: Merupakan struct header kecil yang membungkus pointer ke *underlying array*, panjang (`len`), dan kapasitas (`cap`).
- **Bahaya Concurrency**: Karena slice berbagi pointer ke *underlying array* yang sama, jika dua goroutine melakukan `append` atau memodifikasi indeks tanpa penguncian (`sync.Mutex`), akan terjadi penimpaan data (*data race*) atau memori rusak (*memory corruption*).

### Q2: Mengapa Go tidak menggunakan `try-catch`, melainkan pengembalian error eksplisit `if err != nil`?
**Jawaban**:
1. **Kejelasan Alur Kendali (Predictability)**: `try-catch` sering kali menyembunyikan alur loncatan eksekusi (*hidden control flow*) yang membuat kode sulit di-trace.
2. **Error sebagai Nilai Kelas Satu (First-Class Value)**: Di Go, error adalah tipe data biasa yang harus ditangani langsung pada titik kegagalan, memaksa engineer memikirkan penanganan kegagalan secara sadar.
3. **Auditability**: Dalam sistem keuangan, setiap langkah yang gagal harus segera di-log atau di-rollback secara eksplisit, bukan dilempar ke exception handler global yang tidak jelas konteksnya.

### Q3: Kapan kita menggunakan `sync.Mutex` vs `Channel`?
- **Gunakan `sync.Mutex`**: Untuk melindungi state atau variabel bersama di dalam memori (*shared in-memory state*), seperti peta (`map`), saldo dompet, atau counter cache.
- **Gunakan `Channel`**: Untuk mengoordinasikan alur kerja (*orchestration*), transfer kepemilikan data antar-goroutine, atau membatasi konkurensi (worker pool).
- *Prinsip Go: "Do not communicate by sharing memory; instead, share memory by communicating."*

---

## 🏛️ 4. Clean Architecture & Database Persistence

### Q1: Apa pemisahan tanggung jawab antara Handler, Service (Usecase), dan Repository?
- **Handler (Delivery Layer)**: Menerima request HTTP, mem-parse JSON/Form, memvalidasi format input mendasar, dan mengembalikan status code HTTP (200, 400, 500). Tidak boleh berisi logika bisnis perbankan.
- **Service / Usecase (Business Logic)**: Berisi aturan bisnis (apakah saldo cukup, apakah pengirim mentransfer ke diri sendiri, orkestrasi mutasi debit/kredit).
- **Repository (Infrastructure Layer)**: Berhubungan langsung dengan database SQL atau Redis. Mengeksekusi query database dan mengonversi baris tabel menjadi domain model.

### Q2: Mengapa koneksi jaringan eksternal (seperti panggil API Bank) DILARANG ditaruh di dalam blok Database Transaction (`BEGIN ... COMMIT`)?
**Jawaban**:
1. **Penyanderaan Connection Pool**: Transaksi database menahan satu slot koneksi fisik dari pool. Jika panggilan ke API bank eksternal memakan waktu 5 detik (atau timeout), koneksi database tersebut tersandera selama 5 detik.
2. **Kelelahan Koneksi (Pool Exhaustion)**: Beberapa request lambat saja sudah cukup untuk menghabiskan seluruh kuota koneksi database server, menyebabkan seluruh aplikasi crash dengan error `too many connections`.
3. **Ketidakkonsistenan Dual-Write**: Jika panggilan API Bank sukses tetapi commit database lokal gagal (misal koneksi DB drop tepat sebelum `COMMIT`), uang nasabah di bank luar sudah terkirim namun catatan mutasi di DB lokal tidak ada.

---

## 🚦 5. High Traffic & Rate Limiting

### Q1: Apa perbedaan Token Bucket dan Leaky Bucket dengan analogi sederhana?
- **Token Bucket (Analogi Kasir & Tiket Masuk)**:
  - Token diisi secara teratur ke dalam wadah (misal: 40 token per detik).
  - Setiap request mengambil 1 token. Jika wadah masih punya token sisa, sistem mengizinkan lonjakan request (*burst traffic*). Jika kosong, request langsung ditolak dengan HTTP 429.
  - Sangat cocok untuk API publik yang memperbolehkan lonjakan sesaat selama rata-rata beban terkendali.
- **Leaky Bucket (Analogi Ember Bocor Teratur)**:
  - Request masuk ke dalam ember antrean, dan bocor keluar ke worker pemroses dengan laju konstan dan stabil tanpa toleransi burst.
  - Sangat cocok untuk meratakan beban (*traffic shaping*) menuju sistem hilir yang rapuh.
