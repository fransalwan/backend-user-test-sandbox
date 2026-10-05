# Scenario 04: High Traffic & Flash Sales

## 🎯 The Challenge
Handle flash sales (e.g. 10,000 requests/sec competing for 100 limited vouchers/credits) without crashing the database or overselling.

## 💡 What We Practice
1. **Rate Limiting**:
   - Token Bucket / Sliding Window algorithm using Redis Lua scripts.
2. **Caching & Hotspot Mitigation**:
   - Cache-aside with atomic decrement (`DECRBY`) in Redis.
   - Reconciling Redis cache with PostgreSQL ledger asynchronously or batching commits.
3. **Queue-Based Decoupling**:
   - Absorbing peak bursts with message queues and worker pools.
