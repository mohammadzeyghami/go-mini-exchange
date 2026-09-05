# How this was built

This repo was built with **Claude Code** (Anthropic's agentic CLI) doing the
typing, and me doing the engineering: the design was written down *before*
the code, and every generated line had to pass through quality gates I set.

## The process

1. **Design first, in prose.** The matching engine, ledger and event flow
   were specified in a written document (a Persian-language study book I
   built for myself covering exchange internals: order books, price-time
   priority, double-entry accounting, hold/release, idempotent settlement).
   The AI implemented against that spec — it never invented the design.

2. **Structural prompts, not "write me an exchange".** Each package was
   requested with its contract spelled out: *"the engine is one goroutine
   owning the book; commands over a channel; no I/O in the loop; effects
   leave as events"* — so the architecture is a decision, not an accident.

3. **Quality gates the AI's output had to clear:**
   - `go build`, `go vet`, and `go test ./... -race` after every step —
     the race detector is non-negotiable for concurrent money code;
   - a **property-based test** (1,000 random concurrent orders → ledger
     sums to zero, value conserved, no stuck holds) written alongside the
     features, not after;
   - red-flag review of generated Go: floats near money, swallowed `err`,
     value receivers where pointer semantics matter, missing hold release
     paths, double-close on channels;
   - the frontend had to type-check (`tsc --noEmit`) and `next build` clean.

4. **Honest scope.** When a corner was cut (market orders as banded IOC
   limits, in-process queue instead of RabbitMQ), it was written into the
   README's trade-offs table instead of being hidden.

## What the human decided vs. what the AI typed

| decided by me (the design) | typed by the AI |
|---|---|
| single-writer engine, events-out architecture | the Go implementation of it |
| integer money, floor-once, release-the-remainder | the arithmetic and its tests |
| reserve → match → settle → release lifecycle | handlers, dispatcher, ledger code |
| what the property test must prove | the test code |
| what is out of scope and why | — (that's the point: scope is a human call) |

The interesting part of working this way isn't speed — it's that the
review skills matter *more*, not less: the bugs an AI writes in concurrent
financial code are exactly the subtle ones (`-race`, idempotency, hold
leaks), so the gates above are where the engineering actually lives.
