# Skenario 04: High Traffic & Flash Sales (Pengendalian Serbuan Kuota)

## 🎯 Masalah Finansial: Bahaya Overselling & Thundering Herd
Pada event promosi tengah malam (*Midnight Flash Sale* / Cashback Terbatas):
Tersedia **50 kuota voucher**, namun diserbu oleh **250 - 10.000 request konkuren** dalam waktu 1 detik.

### ⚠️ Bahaya Tanpa Rate Limiter & Atomic Caching (Naif):
- Semua goroutine membaca snapshot stok yang sama dari database.
- Database PostgreSQL mengalami *thundering herd / CPU spike 100%*.
- Kuota voucher jebol menjadi minus (misal: `-15`), artinya perusahaan membagikan 65 voucher padahal kuota hanya 50 (Overselling)!

---

## 🛡️ Solusi Standar Industri Fintech
1. **Token Bucket Rate Limiter**:
   - Membatasi laju request masuk (misal maksimal 40 RPS).
   - Request yang melebihi batas langsung dicegat dengan **HTTP 429 Too Many Requests** tanpa menyentuh database.
2. **Redis Atomic Decrement (`DECRBY`)**:
   - Stok dikurangi secara atomik di memori Redis menggunakan atomic CAS / Lua script.
   - Menghentikan pemotongan tepat saat kuota menyentuh 0.
3. **Queue-Based Worker Smoothing (Leaky Bucket)**:
   - Request diserap ke antrean buffer dan diproses secara teratur oleh worker pool.

---

## 🧪 Cara Menjalankan Tes Otomatis
```bash
go test -v ./scenarios/04_high_traffic
```
