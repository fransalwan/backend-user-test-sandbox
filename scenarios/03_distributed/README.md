# Skenario 03: Transactional Outbox & Saga Pattern (Pencairan Dana Antar Sistem)

## 🎯 Masalah Finansial: Bahaya Dual-Write
Ketika sebuah sistem finansial berinteraksi dengan pihak ketiga (Bank Mitra / Payment Gateway / Microservice lain):
Nasabah meminta pencairan dana (*Withdrawal / Payout*) sebesar **$100**.

### ⚠️ Anti-Pattern Dual-Write yang Sering Dibuat Developer:
```go
// 1. Potong saldo di database lokal
wallet.Balance -= 100
db.Save(wallet)

// 2. Panggil API Bank Mitra / Payment Gateway
resp, err := bankClient.Transfer(100)
if err != nil {
    // APA YANG TERJADI JIKA KONEKSI TIMEOUT ATAU SERVER RESTART DI SINI?
    // Saldo di DB lokal sudah berkurang $100, tetapi uang tidak terkirim ke bank!
    // Nasabah dirugikan, uang lenyap tanpa jejak!
}
```

---

## 🛡️ Solusi Standar Industri Fintech

### 1. Pola Transactional Outbox
Menyimpan mutasi lokal dan event outbox dalam **1 database transaction atomik** yang sama:
```sql
BEGIN TRANSACTION;
  UPDATE wallets SET balance = balance - 100 WHERE id = 'alice';
  INSERT INTO transactions (id, amount, status) VALUES ('tx-1', 100, 'PENDING');
  INSERT INTO outbox_events (id, event_type, payload, status) VALUES ('evt-1', 'WITHDRAWAL_INITIATED', '...', 'PENDING');
COMMIT;
```
Jika server mati sebelum pesan terkirim, event tetap tersimpan aman di database dan akan diproses ulang oleh **Outbox Relay Worker** (*At-least-once delivery*).

### 2. Saga Pattern & Compensating Transactions
Karena Two-Phase Commit (2PC) lambat dan tidak didukung oleh API Bank luar, kita menggunakan **Saga**:
- **Forward Action**: Potong saldo lokal ➡️ Panggil gateway bank eksternal.
- **Compensating Action (Rollback Finansial)**: Jika Bank merespons error 5xx atau timeout yang tidak dapat dipulihkan, Saga Orchestrator otomatis memicu **Compensating Transaction**:
  - Melakukan *Credit Refund* atomik mengembalikan saldo $100 ke dompet nasabah.
  - Mencatat mutasi ledger kompensasi.
  - Memperbarui status transaksi menjadi `COMPENSATED_AUTO_REFUNDED`.
  - **Hasil**: Zero data loss, saldo nasabah aman kembali utuh!

---

## 🧪 Cara Menjalankan Tes Otomatis
```bash
go test -v ./scenarios/03_distributed
```
