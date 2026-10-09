# 📋 Panduan Aksi Kandidat di Luar Sandbox
### *Langkah Nyata Persiapan Portofolio, Take-Home Test, & Live Interview (Junior Backend Engineer)*

> **Dokumen ini adalah pegangan praktis untuk Anda gunakan di luar aplikasi sandbox ini.** Panduan ini mencakup cara memamerkan proyek ini di CV/LinkedIn, cara memanfaatkan aset kode saat mengerjakan take-home test dari perusahaan nyata, serta skrip latihan verbal untuk menghadapi pertanyaan wawancara teknis.

---

## 📑 Daftar Isi
1. [Optimasi Profil GitHub & CV / Resume (Standar ATS)](#1-optimasi-profil-github--cv--resume-standar-ats)
2. [SOP Menghadapi Tugas "Take-Home Test" dari Perusahaan](#2-sop-menghadapi-tugas-take-home-test-dari-perusahaan)
3. [Skrip & Kisi-Kisi Latihan Respon Verbal Live Interview](#3-skrip--kisi-kisi-latihan-respon-verbal-live-interview)
4. [Checklist Hari-H Wawancara Teknis](#4-checklist-hari-h-wawancara-teknis)

---

## 1. Optimasi Profil GitHub & CV / Resume (Standar ATS)

### A. Pin Repositori di Profil GitHub
1. Pastikan repositori ini (`backend-user-test-sandbox`) bersifat **Public**.
2. Masuk ke profil GitHub Anda $\rightarrow$ klik **Customize your pins** $\rightarrow$ centang repositori ini agar tampil di bagian paling atas profil Anda.
3. Pastikan badge GitHub Actions **`build: passing`** menyala hijau di file `README.md`.

### B. Template Penulisan di CV / Resume (Bagian *Projects*)
Gunakan format bullet-points berstandar industri (Action Verb + Context + Measurable Result):

```markdown
**Fintech Core Backend Sandbox & Assessment Engine** | *Go, PostgreSQL, Redis, Docker*
• Merancang arsitektur modular Core Payment Service menggunakan Clean Architecture (Handler, Usecase, Repository) tanpa dependensi sirkular.
• Mengimplementasikan transaksi ACID perbankan dengan Pessimistic Row Locking (`SELECT FOR UPDATE`) pada PostgreSQL untuk mengeliminasi race conditions dan negative balance overdraft.
• Membangun Idempotency Engine berstandar industri dengan mitigasi in-flight collision (HTTP 409) dan deteksi mutasi payload (HTTP 422).
• Mengembangkan simulasi proteksi lonjakan traffic Flash Sale menggunakan Token Bucket Rate Limiter (40 RPS) via Redis atomic counter untuk menjamin zero overselling.
• Menyusun table-driven unit tests komprehensif dengan test coverage >85% serta automasi CI/CD via GitHub Actions.
```

### C. Postingan LinkedIn (Portfolio Showcase)
Bagikan progres belajar Anda di LinkedIn untuk menarik perhatian tech recruiter:
> *"Baru saja menyelesaikan pendalaman backend engineering fundamentals pada sistem fintech/core banking menggunakan Go (Golang), PostgreSQL, dan Redis.*
> 
> *Fokus utama bukan sekadar membuat CRUD, melainkan mengamankan transaksi finansial berkonkurensi tinggi: membedakan Pessimistic Locking (`SELECT FOR UPDATE`) vs Optimistic Locking, menjamin idempotensi request transfer dana dengan hash payload verification, serta meredam lonjakan traffic flash sale dengan Token Bucket Rate Limiter.*
> 
> *Kode sumber, skema migrasi SQL, dan table-driven tests sudah saya dokumentasikan di GitHub: [link repo]*
> 
> *#Golang #BackendEngineering #Fintech #CleanArchitecture #SoftwareEngineering"*

---

## 2. SOP Menghadapi Tugas "Take-Home Test" dari Perusahaan

Ketika Anda melamar ke sebuah perusahaan dan diberikan tugas take-home (misal: *"Buat API Dompet Digital / Transfer Saldo"* dengan tenggat waktu 3–5 hari), **jangan mulai dari nol**. Gunakan aset yang sudah Anda pelajari di sandbox ini:

### Langkah Eksekusi Cepat:
1. **Inisialisasi Proyek Baru**:
   - Buat repo GitHub baru dengan nama tugas (misal: `fintech-transfer-service`).
   - Salin struktur folder Clean Architecture dari Tahap 3 sandbox ini:
     ```text
     cmd/api/main.go
     internal/delivery/http/
     internal/domain/
     internal/repository/postgres/
     internal/usecase/
     migrations/
     docker-compose.yml
     .github/workflows/ci.yml
     ```
2. **Gunakan Docker Compose & SQL Migrations**:
   - Salin file `docker-compose.yml` (PostgreSQL 16 + Redis) dari sandbox ini.
   - Salin `migrations/000001_init_schema.up.sql` dan sesuaikan tabelnya.
   - **Poin Plus**: Sertakan perintah `docker compose up -d` di `README.md` tugas Anda agar reviewer bisa menjalankan proyek Anda dalam 10 detik!
3. **Sertakan File Postman Collection**:
   - Sertakan file `docs/fintech_api.postman_collection.json` di repo tugas Anda.
   - Reviewer akan sangat mengapresiasi karena mereka tidak perlu mengetik perintah cURL manual.
4. **Tuliskan Unit Test Berbasis Tabel (Table-Driven Tests)**:
   - Salin pola dari `internal/delivery/http/handler_test.go`. Uji skenario sukses, saldo tidak cukup, dan duplikasi request.

---

## 3. Skrip & Kisi-Kisi Latihan Respon Verbal Live Interview

Sering kali kegagalan pelamar junior adalah **gugup saat menjelaskan konsep secara lisan**. Latihlah pelafalan jawaban di bawah ini di depan cermin atau rekaman suara smartphone Anda:

---

### ❓ Pertanyaan 1: "Mengapa nilai uang tidak boleh disimpan menggunakan tipe Float atau Double?"
> **🗣️ Jawaban Verbal Standar Tech Lead**:
> *"Nilai uang tidak boleh disimpan sebagai floating-point karena standar IEEE-754 merepresentasikan pecahan dalam basis biner 2, bukan desimal 10. Ini menyebabkan ketidakpresisian biner, seperti `0.1 + 0.2 = 0.30000000000000004`.*
> *Dalam sistem finansial, kesalahan pembulatan 1 sen sekalipun bisa menyebabkan diskrepansi audit buku besar jutaan rupiah.*
> *Solusinya: Di kode Go, kita wajib merepresentasikan uang sebagai `int64` dalam satuan terkecil (sen/cents), atau di PostgreSQL menggunakan tipe data berpresisi eksak `NUMERIC(18, 4)`."*

---

### ❓ Pertanyaan 2: "Kapan kita memakai Pessimistic Locking (`SELECT FOR UPDATE`) dan kapan Optimistic Locking?"
> **🗣️ Jawaban Verbal Standar Tech Lead**:
> *"Keduanya digunakan untuk mencegah race condition, tetapi dengan asumsi konflik yang berbeda:*
> *• **Pessimistic Locking (`SELECT FOR UPDATE`)** kita gunakan saat potensi konflik tinggi dan resiko finansialnya fatal, contohnya pemotongan saldo dompet. Transaksi langsung mengunci baris database secara eksklusif sehingga proses lain wajib mengantre.*
> *• **Optimistic Locking (kolom version)** kita gunakan saat transaksi jarang bertabrakan (low contention), misalnya update profil pengguna. Kita mengecek `WHERE version = old_version`. Jika bentrok, aplikasi melakukan retry.*
> *Untuk mutasi saldo rekening nasabah, industri fintech lebih memilih Pessimistic Locking demi kepastian integritas saldo."*

---

### ❓ Pertanyaan 3: "Bagaimana cara Anda merancang Idempotency Key pada API Transfer?"
> **🗣️ Jawaban Verbal Standar Tech Lead**:
> *"Klien wajib mengirimkan header unik, misalnya `Idempotency-Key: UUIDv4`. Siklus penanganannya:*
> *1. Saat request masuk, kita cek di penyimpanan (Redis/DB). Jika kunci belum ada, kita simpan dengan status `PROCESSING`.*
> *2. Jika ada request kedua masuk dengan kunci sama saat request pertama belum selesai, kita tolak dengan **HTTP 409 Conflict** (In-Flight collision).*
> *3. Setelah transaksi sukses, status diperbarui ke `COMPLETED` beserta respons payload-nya.*
> *4. Jika klien mengirim ulang request yang sama, kita sajikan respons cache tersebut tanpa memotong saldo ulang.*
> *5. Jika klien mengirim kunci yang sama tapi jumlah uang atau rekening tujuan diubah, kita tolak dengan **HTTP 422 Unprocessable Entity** (Payload Tampering)."*

---

### ❓ Pertanyaan 4: "Mengapa Go tidak memiliki exception try-catch seperti Java atau Python?"
> **🗣️ Jawaban Verbal Standar Tech Lead**:
> *"Go memperlakukan error sebagai nilai biasa (first-class values). Pembuat Go mendesain ini secara sengaja agar:*
> *1. Tidak ada aliran kontrol program tersembunyi yang melompat ke blok exception di luar fungsi.*
> *2. Memaksa developer secara sadar dan eksplisit memikirkan setiap skenario kegagalan (`if err != nil`).*
> *Di sistem perbankan dan fintech, pendekatan eksplisit ini sangat menguntungkan karena mencegah kegagalan sistem yang tak terduga (*silent failures*)."*

---

### ❓ Pertanyaan 5: "Apa perbedaan pembagian layer di Clean Architecture?"
> **🗣️ Jawaban Verbal Standar Tech Lead**:
> *"Clean Architecture memisahkan tanggung jawab menjadi 3 lapisan utama:*
> *• **Delivery / Handler**: Hanya bertugas menerima HTTP request, validasi format JSON, dan mengembalikan HTTP response. Lapisan ini tidak boleh tahu detail query SQL.*
> *• **Usecase / Service**: Pusat logika bisnis. Di sini aturan seperti validasi saldo cukup, pembuatan jurnal pembukuan berpasangan, dan orkestrasi transaksi dijalankan.*
> *• **Repository**: Abstraksi akses data ke database (PostgreSQL, Redis). Lapisan bisnis hanya memanggil interface repository, sehingga database bisa diganti tanpa merombak logika bisnis."*

---

## 4. Checklist Hari-H Wawancara Teknis

Lakukan persiapan ini 1 jam sebelum jadwal wawancara dimulai:
- [ ] Buka repositori GitHub Anda di tab browser, siap untuk screen-sharing jika diminta menjelaskan struktur kode.
- [ ] Buka terminal dan pastikan `docker compose up -d` sudah siap jalan jika interviewer meminta demo lokal.
- [ ] Baca kembali ringkasan jawaban verbal di atas untuk melancarkan artikulasi.
- [ ] Ingat aturan emas: **Jujur jika tidak tahu**, lalu jelaskan bagaimana cara Anda mencari solusinya (*"Saya belum pernah mengonfigurasi fitur X secara mendalam di production, tapi berdasarkan dokumentasinya cara kerjanya adalah Y..."*).
