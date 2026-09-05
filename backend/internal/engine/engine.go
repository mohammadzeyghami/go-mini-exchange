// Package engine is the matching engine: one goroutine owns the order book
// (chapter 4's single-writer pattern), commands come in over a channel one at
// a time, and every effect leaves as an event. No locks, no I/O in the hot
// loop — persistence and fan-out happen in the consumers (chapter 9).
package engine

import (
	"errors"
	"fmt"
	"time"

	"github.com/mohammadzeyghami/go-mini-exchange/internal/book"
	"github.com/mohammadzeyghami/go-mini-exchange/internal/domain"
)

var (
	ErrNotFound    = errors.New("order not found")
	ErrNotOwner    = errors.New("order belongs to another user")
	ErrNoLiquidity = errors.New("no liquidity for market order")
)

// ---- events ----------------------------------------------------------------

// TradeEvent is one fill; consumers settle it in the ledger (idempotent by
// trade id), append it to history and broadcast it.
type TradeEvent struct{ Trade domain.Trade }

// OrderEvent reports an order reaching a resting or terminal state. Closed
// is true when the order left the engine for good — the consumer releases
// the order's unspent hold then.
type OrderEvent struct {
	Order  domain.Order
	Closed bool
}

// BookEvent carries a fresh depth snapshot after a command changed the book.
type BookEvent struct{ Depth domain.Depth }

// flushEvent lets tests and callers wait until consumers saw everything
// emitted before it.
type flushEvent struct{ done chan struct{} }

// ---- commands --------------------------------------------------------------

type PlaceResult struct {
	Order  domain.Order
	Trades []domain.Trade
	Err    error
}

type placeCmd struct {
	order *domain.Order
	reply chan PlaceResult
}

type cancelCmd struct {
	orderID string
	userID  string
	reply   chan error
}

type snapshotCmd struct {
	userID string
	reply  chan Snapshot
}

// Snapshot is a consistent read of the book taken inside the engine loop.
type Snapshot struct {
	Depth      domain.Depth
	BestBid    int64 // 0 when empty
	BestAsk    int64
	OpenOrders []domain.Order
}

// ---- engine ----------------------------------------------------------------

const DepthLevels = 15

type Engine struct {
	commands chan any
	events   chan any
	book     *book.Book
	seq      uint64
	tradeSeq uint64
}

func New() *Engine {
	return &Engine{
		commands: make(chan any, 1024),
		events:   make(chan any, 8192),
		book:     book.New(),
	}
}

// Events is consumed by exactly one dispatcher goroutine.
func (e *Engine) Events() <-chan any { return e.events }

// Run is the single writer. It owns e.book; nothing else touches it.
func (e *Engine) Run() {
	for cmd := range e.commands {
		switch c := cmd.(type) {
		case placeCmd:
			c.reply <- e.place(c.order)
			e.events <- BookEvent{Depth: e.book.Depth(DepthLevels)}
		case cancelCmd:
			c.reply <- e.cancel(c.orderID, c.userID)
			e.events <- BookEvent{Depth: e.book.Depth(DepthLevels)}
		case snapshotCmd:
			c.reply <- e.snapshot(c.userID)
		case validateCmd:
			c.reply <- e.book.Validate()
		case flushEvent:
			e.events <- c
		}
	}
	close(e.events)
}

// Close stops the engine loop (tests).
func (e *Engine) Close() { close(e.commands) }

// Place submits an order and waits for the engine's verdict.
func (e *Engine) Place(o *domain.Order) PlaceResult {
	reply := make(chan PlaceResult, 1)
	e.commands <- placeCmd{order: o, reply: reply}
	return <-reply
}

// Cancel removes a resting order.
func (e *Engine) Cancel(orderID, userID string) error {
	reply := make(chan error, 1)
	e.commands <- cancelCmd{orderID: orderID, userID: userID, reply: reply}
	return <-reply
}

// Snapshot reads the book consistently from inside the loop.
func (e *Engine) Snapshot(userID string) Snapshot {
	reply := make(chan Snapshot, 1)
	e.commands <- snapshotCmd{userID: userID, reply: reply}
	return <-reply
}

// Flush resolves when every event emitted before it has been consumed.
func (e *Engine) Flush() {
	done := make(chan struct{})
	e.commands <- flushEvent{done: done}
	<-done
}

// FlushDone is called by the dispatcher when it meets the sentinel.
func (ev flushEvent) Resolve() { close(ev.done) }

// ---- matching --------------------------------------------------------------

func crosses(taker *domain.Order, makerPrice int64) bool {
	if taker.Side == domain.SideBuy {
		return makerPrice <= taker.Price
	}
	return makerPrice >= taker.Price
}

// place implements price-time priority (chapter 9): walk the opposite side
// best-first, fill FIFO within a level, always at the maker's price.
func (e *Engine) place(o *domain.Order) PlaceResult {
	e.seq++
	o.Seq = e.seq
	o.Status = domain.StatusOpen
	o.Remaining = o.Qty
	o.CreatedAt = time.Now()

	var trades []domain.Trade
	for o.Remaining > 0 {
		level := e.book.Best(o.Side.Opposite())
		if level == nil || !crosses(o, level.Price) {
			break
		}
		maker := level.Orders[0]
		qty := min(o.Remaining, maker.Remaining)
		e.tradeSeq++
		t := domain.Trade{
			ID:        fmt.Sprintf("t-%d", e.tradeSeq),
			Price:     maker.Price, // the resting order keeps its price
			Qty:       qty,
			TakerSide: o.Side,
			At:        time.Now(),
		}
		if o.Side == domain.SideBuy {
			t.BuyOrderID, t.BuyerID = o.ID, o.UserID
			t.SellOrderID, t.SellerID = maker.ID, maker.UserID
		} else {
			t.SellOrderID, t.SellerID = o.ID, o.UserID
			t.BuyOrderID, t.BuyerID = maker.ID, maker.UserID
		}
		o.Remaining -= qty
		makerDone := e.book.ReduceTop(o.Side.Opposite(), qty) // mutates maker
		trades = append(trades, t)
		e.events <- TradeEvent{Trade: t}
		if makerDone.Remaining == 0 {
			e.events <- OrderEvent{Order: *makerDone, Closed: true}
		}
	}

	switch {
	case o.Remaining == 0:
		o.Status = domain.StatusFilled
		e.events <- OrderEvent{Order: *o, Closed: true}
	case o.TIF == domain.IOC:
		o.Status = domain.StatusCancelled
		e.events <- OrderEvent{Order: *o, Closed: true}
	default:
		e.book.Insert(o)
		e.events <- OrderEvent{Order: *o, Closed: false}
	}
	return PlaceResult{Order: *o, Trades: trades}
}

func (e *Engine) cancel(orderID, userID string) error {
	o := e.book.Get(orderID)
	if o == nil {
		return ErrNotFound
	}
	if o.UserID != userID {
		return ErrNotOwner
	}
	e.book.Remove(o)
	o.Status = domain.StatusCancelled
	e.events <- OrderEvent{Order: *o, Closed: true}
	return nil
}

func (e *Engine) snapshot(userID string) Snapshot {
	s := Snapshot{Depth: e.book.Depth(DepthLevels)}
	if lvl := e.book.Best(domain.SideBuy); lvl != nil {
		s.BestBid = lvl.Price
	}
	if lvl := e.book.Best(domain.SideSell); lvl != nil {
		s.BestAsk = lvl.Price
	}
	for _, o := range e.book.OpenOrders(userID) {
		s.OpenOrders = append(s.OpenOrders, *o)
	}
	return s
}

// ValidateBook is a test hook: it runs the book's own invariant check from
// inside the loop, so it is safe under -race.
func (e *Engine) ValidateBook() bool {
	reply := make(chan bool, 1)
	e.commands <- validateCmd{reply: reply}
	return <-reply
}

type validateCmd struct{ reply chan bool }
