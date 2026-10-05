# Fintech Backend Interview Questions & Reference Guide

This document captures expected technical questions, failure modes, trade-off analysis, and principal-level answers for backend fintech roles.

---

## 🏦 Topic 1: Concurrency & Race Conditions
### Q1: Why can't we just use `UPDATE wallets SET balance = balance - 100 WHERE id = 1` directly?
**Answer**:
While an atomic `UPDATE` avoids lost updates for simple arithmetic, it has crucial shortcomings in fintech:
1. **Balance Invariant Validation**: You cannot easily prevent balance underflow (negative balance) unless you have a `CHECK (balance >= 0)` constraint or `WHERE balance >= 100`.
2. **Double-Entry Ledger Integrity**: A wallet balance is a *derived snapshot*. In double-entry bookkeeping, you must simultaneously insert an immutable ledger entry and link it to an audit trail inside the same transaction.
3. **Complex Multi-Account Transactions**: In account transfers (Alice -> Bob), you need to update two rows. Without proper locking or ordering (e.g. always lock lower ID first), you risk deadlocks.

### Q2: Compare Pessimistic Locking (`SELECT FOR UPDATE`) vs Optimistic Locking (`version = version + 1`).
| Dimension | Pessimistic Locking (`SELECT FOR UPDATE`) | Optimistic Locking (`version = version + 1`) |
| :--- | :--- | :--- |
| **Mechanism** | Acquires exclusive row-level lock in DB until transaction commits/aborts. | Assumes conflicts are rare; checks version number at commit time. |
| **High Contention** | Predictable queueing; no wasted work retrying. | Severe retry storms, high CPU & DB churn, declining throughput. |
| **Low Contention** | Minor overhead of acquiring row locks. | Very lightweight, fast, no blocking locks held. |
| **Best Used For** | Wallet deductions, limited stock flash sales, hot account balances. | User profile updates, low-frequency product edits, batch reconciliations. |

---

## 🔑 Topic 2: Idempotency
### Q1: What makes an API endpoint truly idempotent in a payment gateway?
**Answer**:
1. **Idempotency Key**: The client submits a unique token (typically UUIDv4) in the `Idempotency-Key` header.
2. **Atomicity**: The server stores the key atomically (e.g., in a Redis key or PostgreSQL table with a `UNIQUE` constraint).
3. **Request State Tracking**:
   - `STARTED`: If a concurrent request with the same key arrives while the first is running, return `409 Conflict` or queue.
   - `COMPLETED`: Store the full response payload (status code, headers, body). Replaying the request immediately returns the cached response without re-triggering funds movement.
4. **TTL / Expiry**: Keys typically expire after 24-72 hours.
