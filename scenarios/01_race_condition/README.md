# Scenario 01: Race Condition & Concurrency Control

## 🎯 The Challenge
Simulate high-concurrency wallet updates (e.g. concurrent top-ups or multiple checkouts targeting the same wallet balance at the exact same millisecond).

## 💡 What We Practice
1. **The Naive Approach (Race Condition)**:
   - `SELECT balance FROM wallets WHERE id = $1`
   - Calculate `new_balance = balance - amount` in Go
   - `UPDATE wallets SET balance = new_balance WHERE id = $1`
   - *Result*: Lost updates, phantom balances, or balance dipping below zero.

2. **Pessimistic Locking**:
   - `SELECT balance FROM wallets WHERE id = $1 FOR UPDATE`
   - Strict serialization at the database row level.
   - Trade-offs: Database lock contention, queueing, potential deadlocks under high load.

3. **Optimistic Locking**:
   - `UPDATE wallets SET balance = balance - amount, version = version + 1 WHERE id = $1 AND version = $2`
   - Check rows affected: If 0, conflict detected -> Retry with exponential backoff or abort.
   - Trade-offs: High throughput when contention is low; retry storms when contention is high.

4. **Redis Distributed Lock (Redlock / SETNX)**:
   - Offload locking overhead from PostgreSQL to Redis.
