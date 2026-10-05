# Scenario 02: Idempotency & Network Retries

## 🎯 The Challenge
Simulate network timeouts, client retry storms, and double-submit scenarios on financial payment and transfer endpoints.

## 💡 What We Practice
1. **Idempotency Key Lifecycle**:
   - Header: `Idempotency-Key: <UUID>`
   - States: `STARTED` / `PROCESSING`, `COMPLETED`, `FAILED`
2. **Atomic Lock & Deduplication**:
   - Redis or PostgreSQL atomic insert (`INSERT ... ON CONFLICT DO NOTHING`)
   - Returning cached responses for identical requests without re-executing ledger operations.
3. **Concurrent In-Flight Requests**:
   - Handling when duplicate request B arrives while request A is still processing (HTTP 409 Conflict / Request In Progress).
