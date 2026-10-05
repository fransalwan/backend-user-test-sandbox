# Working Rules & Collaboration Guidelines

## 🛠️ Engineering Execution Rules

### 1. Verification First (No Faith-Based Coding)
- Every solution must be verified against real concurrency conditions:
  - Unit tests with Go's race detector enabled (`go test -race ./...`).
  - Integration tests with real PostgreSQL and Redis instances (via `testcontainers-go`).
  - Concurrency simulations testing high contention (e.g., 50-100 parallel goroutines competing for a single wallet balance).
  - Load testing scripts using k6 for throughput, latency, and error rate profiling.

### 2. Idiomatic Go Standards
- Explicit error handling: Always wrap errors with context (`fmt.Errorf("failed to debit wallet %s: %w", walletID, err)`).
- Context propagation: Pass `ctx context.Context` as the first argument in all I/O or network calls.
- Resource management: Always `defer` closes/rollbacks/releases immediately after acquisition.
- Struct field tags: Explicit tags for JSON and database mapping (`json:"id" db:"id"`).

### 3. Fintech Interview & Dojo Mindset
- Always document the "Trade-Off Analysis":
  - Why Pessimistic Locking over Optimistic Locking (or vice-versa)?
  - What are the throughput, latency, and deadlock risks?
  - How does Redis distributed locking handle lock expiry / network partitions?
- Maintain `docs/interview_qa.md` with:
  - Potential interviewer questions for each scenario.
  - Common pitfalls candidates make.
  - Senior/Principal level answers explaining trade-offs, database internal locks (row locks vs page locks vs predicate locks), and system design implications.
