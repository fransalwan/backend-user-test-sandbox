# ⚡ Backend User Test Sandbox (Fintech Dojo)

> A deliberate practice arena ("Dojo") designed specifically for backend engineers to prepare, simulate, and excel in rigorous Fintech technical interviews, live coding sessions, system design assessments, and take-home assignments.

---

## 🏛️ Architecture: Modular Monolith (Clean / Hexagonal)

The sandbox strictly follows Clean Architecture / Hexagonal Architecture separation of concerns:

```text
backend-user-test-sandbox/
├── .agents/                    # Agent guidance, rules & workflow skills
│   ├── rules/                  # Personal context, project rules, working rules
│   └── skills/                 # Agent CLI skills suite
├── cmd/
│   └── sandbox/                # Application entrypoint & HTTP server
├── internal/
│   ├── core/                   # DOMAIN LAYER (Pure Go, ZERO external dependencies)
│   │   ├── entity/             # Base domain models (Wallet, LedgerEntry, Transaction)
│   │   ├── port/               # Ports: Repository, Cache, Broker, PaymentGateway
│   │   └── service/            # Business service use-case interfaces
│   ├── adapter/                # INFRASTRUCTURE LAYER
│   │   ├── postgres/           # PostgreSQL strict ACID persistence
│   │   ├── redis/              # Distributed locking & cache engine
│   │   └── mock/               # Mock external gateways (Payment, Notification)
│   └── delivery/               # PRESENTATION LAYER
│       └── http/               # HTTP handlers, SSE Hub, embedded templates
│           └── templates/      # HTMX + Tailwind CSS dashboard UI
├── scenarios/                  # CORE TRAINING MODULES
│   ├── 01_race_condition/      # Implemented: Pessimistic vs Optimistic Locking
│   ├── 02_idempotency/         # Roadmap: Idempotency Keys, Handling Retries
│   ├── 03_distributed/         # Roadmap: Transactional Outbox & Saga Pattern
│   └── 04_high_traffic/        # Roadmap: Rate Limiting, Caching, Flash Sales
├── chaos/                      # Fault injection tools (latency, DB connection drops)
├── load_tests/                 # Concurrency stress tests with k6
├── docs/
│   └── interview_qa.md         # Fintech interview questions & principal-level answers
├── migrations/                 # PostgreSQL DDL migrations with strict constraints
├── Makefile                    # Automation shortcuts (setup, run, test, lint)
├── go.mod
└── README.md
```

---

## 🛠️ Complete Tech Stack

| Layer | Technology | Details |
| :--- | :--- | :--- |
| **Language** | **Go (Golang 1.27+)** | Concurrency native (*goroutines* & *channels*), zero-allocation design. |
| **Database** | **PostgreSQL** | Strict ACID compliance, check constraints (`balance >= 0`), row locks. |
| **Cache & Lock** | **Redis** | Distributed locks (`SETNX`), atomic operations, sliding-window rate limiters. |
| **Testing** | **Testify + Testcontainers-Go** | Automated integration tests against real Docker containers. |
| **Load Testing** | **k6** | Concurrency stress testing with 50-500 virtual users. |
| **Frontend** | **Go `html/template` + HTMX** | Embedded with `//go:embed`, zero Node.js / `node_modules` overhead. |
| **Styling** | **Tailwind CSS** | Dark-mode fintech styling via lightweight CDN. |
| **Live Stream** | **Server-Sent Events (SSE)** | Real-time browser streaming of lock acquisition, contention, and audits. |

---

## 🥊 The 4 Training Scenarios

### ✅ Scenario 01: Concurrency Control & Race Conditions (`scenarios/01_race_condition`)
- **Fintech Problem**: Concurrent transactions targeting the same wallet at the exact same millisecond.
- **Interactive Demonstrations**:
  1. **Naive Read-Modify-Write (Race Hazard ⚠️)**: Demonstrates phantom balance updates and negative balance overdraft.
  2. **Pessimistic Locking (`SELECT FOR UPDATE` 🔒)**: Strict database row serialization, zero overdraft, guaranteed consistency.
  3. **Optimistic Locking (Version Check 🔄)**: Atomic compare-and-swap (`WHERE version = v`), exponential backoff retries.
  4. **Redis Distributed Lock (SETNX 🔑)**: Offloading lock contention from PostgreSQL to Redis.

### 🎯 Scenario 02: Idempotency & Network Retries (`scenarios/02_idempotency`)
- **Fintech Problem**: Network drops, gateway timeouts, and duplicate client retries leading to double-spending.
- **Key Concepts**: `Idempotency-Key` header, atomic deduplication, in-flight locking (HTTP 409 Conflict), cached response replays.

### 🎯 Scenario 03: Distributed Transactions & Eventual Consistency (`scenarios/03_distributed`)
- **Fintech Problem**: Coordinating money movement with external third-party payment gateways without 2PC.
- **Key Concepts**: Transactional Outbox pattern, Saga Orchestration, compensating transactions (automated refunds), and message deduplication.

### 🎯 Scenario 04: High Traffic & Flash Sales (`scenarios/04_high_traffic`)
- **Fintech Problem**: Extreme traffic spikes (e.g., 10,000 requests/sec competing for 100 limited vouchers/cashbacks).
- **Key Concepts**: Token Bucket / Sliding Window rate limiting (Redis Lua), hot-account caching with `DECRBY`, and asynchronous ledger reconciliation.

---

## 🗺️ Roadmap Pengembangan Selanjutnya

1. **Fase 2**: Integrasi Real Database (PostgreSQL + Redis + Testcontainers-Go hermetic testing).
2. **Fase 3**: Scenario 02 (`02_idempotency`) — Idempotency Key Engine, Request Lifecycle, & UI Simulation.
3. **Fase 4**: Scenario 03 (`03_distributed`) — Transactional Outbox & Saga Orchestrator dengan Compensating Transactions.
4. **Fase 5**: Scenario 04 (`04_high_traffic`) — Redis Lua Rate Limiting & Hot-Account Caching.
5. **Fase 6**: Chaos Injection (`chaos/`) & k6 Concurrency Benchmarks (`load_tests/`).
6. **Fase 7**: Interview Defense & System Design Guide (`docs/interview_qa.md`).

---

## 🚀 Getting Started

### 1. Prerequisites
- Go 1.27+ installed

### 2. Run the Sandbox
```bash
# Setup dependencies
make setup

# Run the server & interactive dashboard
make run
```

Buka di browser:
👉 **`http://localhost:8080`**

### 3. Run Automated Tests
```bash
# Run unit & scenario tests
make test

# Run tests with Go Race Detector
make test-race

# Run linter
make lint
```

---

## 📜 Financial Invariants & Rules

1. **Integer Money Representation**: Balances and amounts are strictly stored as `int64` (in cents / smallest currency units). Floating-point arithmetic is forbidden.
2. **Double-Entry Bookkeeping**: Every fund movement consists of balanced debit and credit entries.
3. **Immutable Ledgers**: Ledger rows are append-only. Past records are never updated or deleted.
