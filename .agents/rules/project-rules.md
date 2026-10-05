# Project Rules & Architecture Standards

## 🏛️ Architecture: Modular Monolith (Clean / Hexagonal)
The sandbox follows a strict Clean Architecture / Hexagonal Architecture separation of concerns:

```
internal/
├── core/                # DOMAIN LAYER (Pure Go, ZERO external dependencies)
│   ├── entity/          # Base models, value objects, domain invariants
│   ├── port/            # Outbound and inbound interfaces (Repository, Cache, Gateway)
│   └── service/         # Domain business rules & use case interfaces
├── adapter/             # INFRASTRUCTURE LAYER (Implements ports)
│   ├── postgres/        # PostgreSQL SQL queries, transactions, connection pool
│   ├── redis/           # Redis caching, distributed locks
│   └── mock/            # Fake external implementations for tests
└── delivery/            # PRESENTATION LAYER
    └── http/            # HTTP handlers, routing, request/response DTOs
```

### 1. Domain Inviolability
- `internal/core` must **never** import `database/sql`, `github.com/lib/pq`, `github.com/redis/go-redis`, or any third-party frameworks.
- Domain models and business logic must be testable with pure Go unit tests without external processes.

### 2. Financial System Invariants
- **Never use floating-point types (`float32`, `float64`) for money.** Always use `int64` (representing the smallest currency unit, e.g., cents/satoshis) or an exact decimal type.
- **Double-Entry Principle**: Any movement of money requires balanced debit and credit entries.
- **Append-Only Ledger**: Ledger entries are immutable. Balances can be adjusted only by adding new ledger records, never by mutating past records.
- **Strict Isolation & Concurrency**:
  - Always enforce appropriate transaction isolation (`READ COMMITTED`, `REPEATABLE READ`, or `SERIALIZABLE`).
  - Protect critical financial resources using explicit locking (`SELECT ... FOR UPDATE` or Optimistic concurrency control via `version = version + 1`).
- **Idempotency**: All mutating operations (transfers, debits, credits, refunds) MUST accept an `Idempotency-Key` and guarantee that duplicate calls return identical results without double-processing.

### 3. Scenario Isolation
- Each scenario in `scenarios/` is an independent module with:
  - Its own runnable demonstration / CLI / tests.
  - Clear benchmarking and failure injection.
  - Detailed README or documentation explaining the problem, the naive approach, the failure mode, and the robust solution.

### 4. Presentation & Visualization (Frontend: HTMX + Tailwind CSS + SSE)
- Zero JavaScript build overhead: No Node.js / `node_modules` required.
- Go `html/template` embedded with `//go:embed` inside `internal/delivery/http` for single-binary portability.
- **HTMX**: Powers AJAX form submissions and real-time partial DOM swaps without page reloads.
- **Tailwind CSS**: Modern utility-first styling loaded via lightweight CDN.
- **Server-Sent Events (SSE)**: Streams live goroutine concurrency events, database lock acquisition timings, and ledger audit mutations directly to the browser dashboard.

