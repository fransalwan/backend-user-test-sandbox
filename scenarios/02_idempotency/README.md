# Skenario 02: Idempotensi & Penanganan Retry Jaringan (Anti Double-Spending)

## 🎯 Masalah Finansial
Di sistem perbankan dan *payment gateway*, timeout jaringan adalah hal yang pasti terjadi (*network is unreliable*).
Ketika nasabah mengklik tombol **"Transfer $50"**:
1. Server menerima request dan memotong saldo nasabah di database.
2. Saat server hendak mengirim respons balik ke aplikasi mobile nasabah, koneksi internet seluler nasabah terputus (*timeout*).
3. Aplikasi nasabah menganggap transaksi gagal, lalu otomatis melakukan **retry** 3-5 kali, atau nasabah panik mengklik tombol "Kirim" berkali-kali.
4. **Bencana Tanpa Idempotensi**: Saldo nasabah terpotong 5 kali ($250 lenyap) padahal maksud user hanya 1 kali transaksi!

---

## 💡 Mekanisme & Solusi Idempotensi
1. **Idempotency Key (UUID)**:
   - Klien mengirim header unik: `Idempotency-Key: <UUID>`.
2. **Siklus Hidup Status Kunci**:
   - `PROCESSING`: Dikunci sesaat ketika transaksi pertama sedang dieksekusi.
   - `COMPLETED`: Transaksi selesai, response payload (HTTP status, JSON body) disimpan di cache/database.
   - `FAILED`: Transaksi gagal diproses.
3. **Pencegahan In-Flight Collision (HTTP 409 Conflict)**:
   - Jika request kedua tiba saat request pertama masih dalam status `PROCESSING`, server menolak request kedua dengan **HTTP 409 Conflict** untuk mencegah race condition.
4. **Cached Response Replay**:
   - Jika request duplikat tiba setelah transaksi selesai (`COMPLETED`), server **TIDAK** memotong saldo lagi dan langsung mengembalikan respons tersimpan dari cache.
5. **Verifikasi Hash Payload (HTTP 422 Unprocessable Entity)**:
   - Jika request kedua menggunakan kunci yang sama namun nominal/parameter diubah, server menggagalkan request untuk mencegah manipulasi data.

---

## 🧪 Cara Menjalankan Tes Otomatis
```bash
go test -v ./scenarios/02_idempotency
```
