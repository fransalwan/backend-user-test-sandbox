# Scenario 03: Distributed Transactions & Eventual Consistency

## 🎯 The Challenge
Execute cross-boundary financial workflows where 2PC (Two-Phase Commit) is impractical or anti-pattern (e.g. Wallet Service + External Payment Gateway + Notification Service).

## 💡 What We Practice
1. **Transactional Outbox Pattern**:
   - Write business state change + outbox event atomically in a single PostgreSQL transaction.
   - Outbox publisher with at-least-once delivery guarantee.
2. **Saga Pattern (Orchestrator vs Choreography)**:
   - Forward actions vs Compensating transactions (e.g., refund if third-party gateway fails).
3. **Consumer Idempotency & Message Deduplication**:
   - Safely process duplicate event messages without double-crediting balances.
