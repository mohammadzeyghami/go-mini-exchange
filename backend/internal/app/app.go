// Package app wires the pieces together: validate → reserve funds in the
// ledger → hand the order to the engine → a single dispatcher goroutine
// consumes the engine's events in order and applies them to the ledger
// (idempotently), the trade history and the WebSocket hub.
//
// The dispatcher is the "queue consumer" of chapter 21 — in-process channels
// in v1, the same shape RabbitMQ would take in v2.
package app

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mohammadzeyghami/go-mini-exchange/internal/domain"
	"github.com/mohammadzeyghami/go-mini-exchange/internal/engine"
	"github.com/mohammadzeyghami/go-mini-exchange/internal/ledger"
)

var (
	ErrValidation = errors.New("validation")
)

// Broadcaster receives every public event (nil-safe: tests pass nil).
type Broadcaster interface {
	BroadcastTrade(t domain.Trade, lastPrice int64)
	BroadcastDepth(d domain.Depth)
	BroadcastOrder(o domain.Order)
}

type App struct {
	Ledger *ledger.Ledger
	Engine *engine.Engine

	hub      Broadcaster
	orderSeq atomic.Uint64

	mu        sync.Mutex
	trades    []domain.Trade // ring of recent trades, newest last
	lastPrice int64

	done chan struct{}
}

func New(hub Broadcaster) *App {
	a := &App{
		Ledger: ledger.New(),
		Engine: engine.New(),
		hub:    hub,
		done:   make(chan struct{}),
	}
	go a.Engine.Run()
	go a.dispatch()
	return a
}

// SeedUser gives an account its paper-trading starting balances.
func (a *App) SeedUser(userID string, usdtCents, btcSat int64) {
	_ = a.Ledger.Deposit("seed:"+userID+":usdt", userID, domain.AssetQuote, usdtCents)
	_ = a.Ledger.Deposit("seed:"+userID+":btc", userID, domain.AssetBase, btcSat)
}

// dispatch is the one consumer of engine events. Order matters: a trade is
// settled before the close event of the orders it filled, because the engine
// emits them in that order and this loop is sequential.
func (a *App) dispatch() {
	for ev := range a.Engine.Events() {
		switch e := ev.(type) {
		case engine.TradeEvent:
			// idempotent by trade id — a replay cannot double-settle (ch. 19/21)
			if err := a.Ledger.SettleTrade(e.Trade); err != nil {
				// A settle failure is a bug, not a user error: surface loudly.
				fmt.Println("LEDGER SETTLE FAILED:", err)
			}
			a.mu.Lock()
			a.trades = append(a.trades, e.Trade)
			if len(a.trades) > 200 {
				a.trades = a.trades[len(a.trades)-200:]
			}
			a.lastPrice = e.Trade.Price
			last := a.lastPrice
			a.mu.Unlock()
			if a.hub != nil {
				a.hub.BroadcastTrade(e.Trade, last)
			}
		case engine.OrderEvent:
			if e.Closed {
				_ = a.Ledger.ReleaseHold(e.Order.ID)
			}
			if a.hub != nil {
				a.hub.BroadcastOrder(e.Order)
			}
		case engine.BookEvent:
			if a.hub != nil {
				a.hub.BroadcastDepth(e.Depth)
			}
		case interface{ Resolve() }: // flush sentinel
			e.Resolve()
		}
	}
	close(a.done)
}

// Flush blocks until the dispatcher has consumed everything emitted so far —
// after it, ledger and history reflect every earlier command (tests).
func (a *App) Flush() { a.Engine.Flush() }

// Close shuts the engine and waits for the dispatcher to drain.
func (a *App) Close() {
	a.Engine.Close()
	<-a.done
}

type PlaceRequest struct {
	UserID string
	Side   domain.Side
	Type   domain.OrderType
	Price  int64 // cents/BTC; ignored for market
	Qty    int64 // satoshi
}

// PlaceOrder validates, reserves funds, and submits to the engine.
//
// A market order is turned into an IOC limit at a protective band around the
// current best price (±5%) — that bounds what must be reserved up front and
// doubles as crude price protection (README: decisions & trade-offs).
func (a *App) PlaceOrder(req PlaceRequest) (engine.PlaceResult, error) {
	if req.Side != domain.SideBuy && req.Side != domain.SideSell {
		return engine.PlaceResult{}, fmt.Errorf("%w: side must be buy or sell", ErrValidation)
	}
	if req.Type != domain.TypeLimit && req.Type != domain.TypeMarket {
		return engine.PlaceResult{}, fmt.Errorf("%w: type must be limit or market", ErrValidation)
	}
	if req.Qty < domain.MinLotSat || req.Qty > domain.MaxQtySat {
		return engine.PlaceResult{}, fmt.Errorf("%w: qty must be between %d and %d sat", ErrValidation, domain.MinLotSat, domain.MaxQtySat)
	}

	tif := domain.GTC
	price := req.Price
	if req.Type == domain.TypeMarket {
		snap := a.Engine.Snapshot("")
		var ref int64
		if req.Side == domain.SideBuy {
			ref = snap.BestAsk
		} else {
			ref = snap.BestBid
		}
		if ref == 0 {
			return engine.PlaceResult{}, engine.ErrNoLiquidity
		}
		if req.Side == domain.SideBuy {
			price = ref + ref/20 // +5% band
		} else {
			price = ref - ref/20 // -5% band
		}
		tif = domain.IOC
	}
	if price <= 0 || price > domain.MaxPriceCents {
		return engine.PlaceResult{}, fmt.Errorf("%w: price must be between 1 and %d cents", ErrValidation, domain.MaxPriceCents)
	}

	o := &domain.Order{
		ID:     fmt.Sprintf("o-%d", a.orderSeq.Add(1)),
		UserID: req.UserID,
		Side:   req.Side,
		Type:   req.Type,
		TIF:    tif,
		Price:  price,
		Qty:    req.Qty,
	}

	// Reserve before the engine ever sees the order (chapter 7: hold).
	if req.Side == domain.SideBuy {
		notional := domain.Notional(price, req.Qty)
		if notional <= 0 {
			return engine.PlaceResult{}, fmt.Errorf("%w: order notional rounds to zero", ErrValidation)
		}
		if err := a.Ledger.Reserve(o.ID, req.UserID, domain.AssetQuote, notional); err != nil {
			return engine.PlaceResult{}, err
		}
	} else {
		if err := a.Ledger.Reserve(o.ID, req.UserID, domain.AssetBase, req.Qty); err != nil {
			return engine.PlaceResult{}, err
		}
	}

	res := a.Engine.Place(o)
	return res, res.Err
}

// CancelOrder cancels a resting order owned by userID.
func (a *App) CancelOrder(orderID, userID string) error {
	return a.Engine.Cancel(orderID, userID)
}

// RecentTrades returns up to limit most-recent trades, newest first.
func (a *App) RecentTrades(limit int) []domain.Trade {
	a.mu.Lock()
	defer a.mu.Unlock()
	n := len(a.trades)
	if limit > n {
		limit = n
	}
	out := make([]domain.Trade, 0, limit)
	for i := n - 1; i >= n-limit; i-- {
		out = append(out, a.trades[i])
	}
	return out
}

// LastPrice returns the most recent trade price (0 before the first trade).
func (a *App) LastPrice() int64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.lastPrice
}

// StartInvariantChecker runs the ledger's invariant check periodically —
// chapter 7's rule 5: imbalance is a red alert, not a warning log.
func (a *App) StartInvariantChecker(every time.Duration) {
	go func() {
		t := time.NewTicker(every)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				if err := a.Ledger.CheckInvariants(); err != nil {
					panic(fmt.Sprintf("LEDGER INVARIANT BROKEN: %v", err))
				}
			case <-a.done:
				return
			}
		}
	}()
}
