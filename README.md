# go-mini-exchange

A tiny but honest **spot crypto exchange**: a Go matching engine with
price-time priority, a double-entry ledger with available/hold accounting,
a REST + WebSocket API, and a live Next.js trading dashboard.

Built as a learning milestone — the goal is to demonstrate *mechanism*, not
production scale: every design decision below is the same one a real CEX
makes, implemented at the smallest size that still teaches it.

![CI](https://github.com/mohammadzeyghami/go-mini-exchange/actions/workflows/ci.yml/badge.svg)

## Architecture

```
                Next.js dashboard (:3300)
                REST + WebSocket
                       │
                       ▼
            ┌─────────────────────┐
            │  API (Go, :8140)    │  validation, error mapping, CORS
            └─────────┬───────────┘
             reserve  │  place/cancel (channel)
                      ▼
   ┌──────────┐   ┌──────────────────────┐
   │  Ledger  │◄──│  Matching Engine     │  ONE goroutine owns the book
   │ (double- │   │  (single writer,     │  price-time priority
   │  entry)  │   │   in-memory book)    │  no locks, no I/O in hot loop
   └────▲─────┘   └─────────┬────────────┘
        │                   │ events (trades, order updates, depth)
        │         ┌─────────▼────────────┐
        └─────────│  Dispatcher          │  single consumer, in order:
   settle (idem)  │  (in-process queue)  │  ledger → history → WS fan-out
                  └─────────┬────────────┘
                            ▼
                     WebSocket hub → browsers
```

## Quick start

```bash
docker compose up --build
# dashboard → http://localhost:3300   API → http://localhost:8140
```

or natively:

```bash
cd backend && go run ./cmd/api          # API on :8140 (+ demo market-maker bot)
cd frontend && npm i && npm run dev -- --port 3300
```

Place an order with curl (paper users `alice` / `bob` are pre-funded):

```bash
curl -s -X POST localhost:8140/api/orders \
  -H 'Content-Type: application/json' -H 'X-User: alice' \
  -d '{"side":"buy","type":"limit","price":4300000,"qty":100000}'
# price = cents per BTC (43,000.00), qty = satoshi (0.001 BTC)

curl -s localhost:8140/api/trades | head
curl -s localhost:8140/api/balances -H 'X-User: alice'
```

Tests (race detector always on — a data race in an exchange is a money bug):

```bash
cd backend && go test ./... -race
```

## What it does

- **Matching engine** — limit / market / cancel, price–time priority, fills
  always at the maker's price. One goroutine owns the order book; commands
  arrive over a channel, effects leave as events (the *single-writer*
  pattern). No mutex touches the book, and `-race` proves it.
- **Double-entry ledger** — every fill is one balanced transaction
  (buyer hold → seller, seller hold → buyer, fees to the fee account; every
  asset sums to zero across the whole system, always). Funds are **reserved**
  (available → hold) before an order reaches the engine, and the unspent
  hold is released when the order dies. Idempotent by transaction key: a
  replayed event settles zero times.
- **Events, then persistence** — the engine never does I/O; a single
  dispatcher consumes its events in order and applies them to the ledger,
  trade history and the WebSocket hub. This is the RabbitMQ shape
  (at-least-once + idempotent consumer) with an in-process channel as v1.
- **Live dashboard** — Next.js + TypeScript + TanStack Query + axios +
  react-hook-form + Tailwind + shadcn/ui + framer-motion: animated depth
  ladder, live trade tape, order form, balances with holds, open orders.
- **Property-based test** — 1,000 random concurrent orders, then: ledger
  sums to zero, no negative balance, book sorted/FIFO-consistent, total value
  conserved to the cent, and after cancelling everything no hold is stuck.
- **Invariant checker at runtime** — the ledger re-derives itself from the
  journal every 5s; imbalance panics on purpose. Books that don't balance
  should be loud.

## Decisions & trade-offs

| decision | why |
|---|---|
| Money is `int64` (cents / satoshi), never float | `0.1 + 0.2 != 0.3`; flooring is defined in exactly one place |
| Single-writer engine instead of a locked book | deterministic ordering (fairness + replayability) and zero lock contention; this is how real matching engines work |
| Market orders become IOC limits at a ±5% protective band | bounds what must be reserved up-front *and* doubles as crude price protection; documented, not hidden |
| Reserve-before-engine, release-on-close | the ledger, not the engine, owns solvency; the engine can stay pure matching |
| In-process channel instead of RabbitMQ | same shape (ordered consumer + idempotency key), zero infra; v2 swaps the transport, not the logic |
| In-memory state, no database | the book *is* in-memory in real exchanges; persistence here would be event-log replay, which the journal already models |
| `X-User` header instead of auth | paper trading demo; auth is a solved problem orthogonal to the engine |

## Deliberately out of scope

Margin/liquidation, multiple markets, real blockchain deposits/withdrawals,
order types beyond limit/market/IOC, persistence across restarts, and
industrial matching performance. Each is a layer on top of what's here —
and a smaller finished project beats a bigger unfinished one.

## How it was built

With AI tooling (Claude Code) driving the implementation against a written
design, with quality gates — see [HOW-IT-WAS-BUILT.md](HOW-IT-WAS-BUILT.md).

## License

[MIT](LICENSE)
