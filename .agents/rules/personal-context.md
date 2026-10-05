# Personal Context & Developer Profile

## 👤 Profile & Objective
- **Engineer**: Backend Engineer preparing for senior & principal engineering technical assessments in Fintech / Payment / Banking domains.
- **Goal**: Master high-concurrency patterns, strict financial consistency, idempotency, distributed transactions, and system resiliency under chaos conditions.
- **Mindset**:
  - Zero tolerance for data corruption, double-spending, or balance anomalies.
  - Pragmatic and rigorous: Understand deeply *why* a particular locking strategy or architectural pattern is chosen over another, including performance and operational trade-offs.
  - Interview / Take-Home Focus: Code must be clean, modular, self-documenting, and accompanied by automated tests verifying failure modes and concurrency safety.

## 🎯 Target Competencies
1. **Concurrency Control**: Pessimistic Locking (`SELECT FOR UPDATE`), Optimistic Locking (Version numbers), Distributed Locks (Redis Redlock).
2. **Idempotency & Deduplication**: Idempotency keys, atomic deduplication checks, two-phase commit patterns, handling network retries.
3. **Data Integrity & Ledgers**: Immutable append-only double-entry bookkeeping, strict balance checks, database constraints, transaction isolation levels.
4. **Distributed Systems & Messaging**: Outbox pattern, Saga pattern (orchestrated vs choreographed), message deduplication, at-least-once delivery handling.
5. **Resilience & High Traffic**: Rate limiting (token bucket / leaky bucket / sliding window), caching strategies (cache-aside, write-through, stale-while-revalidate), circuit breakers, graceful degradation.
6. **Live Coding & System Design Presentation**: Ability to articulate tradeoffs, bottlenecks, failure modes, and recovery strategies under interview conditions.
