package app

import (
	"math/rand"
	"sync"
	"testing"

	"github.com/mohammadzeyghami/go-mini-exchange/internal/domain"
	"github.com/mohammadzeyghami/go-mini-exchange/internal/ledger"
)

// TestPropertyRandomOrders is the chapter-23 property test: a thousand random
// orders from concurrent users, then every invariant must hold:
//
//  1. every asset in the ledger sums to zero and no balance is negative
//  2. the book is internally consistent (sorted levels, FIFO, totals)
//  3. value is conserved: users + fee account own exactly what was deposited
//  4. nothing is stuck: with no open orders left, every hold is zero
//
// Run with -race — concurrency bugs in the single-writer path show up here.
func TestPropertyRandomOrders(t *testing.T) {
	a := New(nil)
	defer a.Close()

	users := []string{"u1", "u2", "u3", "u4"}
	const seedQuote = 10_000_000_00 // 10M USDT
	const seedBase = 100 * domain.SatPerBTC
	for _, u := range users {
		a.SeedUser(u, seedQuote, seedBase)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var placed []struct{ id, user string }

	for w := range users {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(worker) + 42))
			user := users[worker]
			for i := 0; i < 250; i++ {
				side := domain.SideBuy
				if rng.Intn(2) == 0 {
					side = domain.SideSell
				}
				typ := domain.TypeLimit
				price := int64(40_000_00 + rng.Intn(6_000_00)) // 40k–46k
				if rng.Intn(5) == 0 {
					typ = domain.TypeMarket
					price = 0
				}
				qty := int64((rng.Intn(50) + 1) * domain.MinLotSat)

				res, err := a.PlaceOrder(PlaceRequest{
					UserID: user, Side: side, Type: typ, Price: price, Qty: qty,
				})
				if err == nil && res.Order.Status == domain.StatusOpen {
					mu.Lock()
					placed = append(placed, struct{ id, user string }{res.Order.ID, user})
					mu.Unlock()
				}
				// occasionally cancel one of our resting orders
				if rng.Intn(4) == 0 {
					mu.Lock()
					if len(placed) > 0 {
						p := placed[rng.Intn(len(placed))]
						mu.Unlock()
						_ = a.CancelOrder(p.id, p.user) // may already be gone — fine
					} else {
						mu.Unlock()
					}
				}
			}
		}(w)
	}
	wg.Wait()
	a.Flush()

	// 1 + drift: the ledger's own three rules
	if err := a.Ledger.CheckInvariants(); err != nil {
		t.Fatalf("ledger invariants: %v", err)
	}
	// 2: the book's structural invariants
	if !a.Engine.ValidateBook() {
		t.Fatal("order book invariants broken")
	}
	// 3: conservation — everything deposited is still owned by someone
	var totalQ, totalB int64
	for _, owner := range append([]string{ledger.AccFees}, users...) {
		b := a.Ledger.Balances(owner)
		totalQ += b[domain.AssetQuote].Available + b[domain.AssetQuote].Hold
		totalB += b[domain.AssetBase].Available + b[domain.AssetBase].Hold
	}
	if totalQ != int64(len(users))*seedQuote {
		t.Fatalf("quote not conserved: have %d, want %d", totalQ, int64(len(users))*seedQuote)
	}
	if totalB != int64(len(users))*seedBase {
		t.Fatalf("base not conserved: have %d, want %d", totalB, int64(len(users))*seedBase)
	}
	// 4: cancel everything → all holds must return to zero
	for _, u := range users {
		for _, o := range a.Engine.Snapshot(u).OpenOrders {
			if err := a.CancelOrder(o.ID, u); err != nil {
				t.Fatalf("cancel %s: %v", o.ID, err)
			}
		}
	}
	a.Flush()
	for _, u := range users {
		b := a.Ledger.Balances(u)
		if b[domain.AssetQuote].Hold != 0 || b[domain.AssetBase].Hold != 0 {
			t.Fatalf("stuck hold for %s: %+v", u, b)
		}
	}
	if err := a.Ledger.CheckInvariants(); err != nil {
		t.Fatalf("ledger invariants after unwind: %v", err)
	}
}
