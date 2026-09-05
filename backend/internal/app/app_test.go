package app

import (
	"testing"

	"github.com/mohammadzeyghami/go-mini-exchange/internal/domain"
)

func newTestApp(t *testing.T) *App {
	t.Helper()
	a := New(nil)
	t.Cleanup(a.Close)
	a.SeedUser("alice", 1_000_000_00, 10*domain.SatPerBTC)
	a.SeedUser("bob", 1_000_000_00, 10*domain.SatPerBTC)
	return a
}

func place(t *testing.T, a *App, user string, side domain.Side, typ domain.OrderType, price, qty int64) PlaceRequest {
	t.Helper()
	return PlaceRequest{UserID: user, Side: side, Type: typ, Price: price, Qty: qty}
}

func TestLimitOrdersMatchAtMakerPrice(t *testing.T) {
	a := newTestApp(t)

	// bob rests a sell at 43,000.00 for 0.5 BTC
	res, err := a.PlaceOrder(place(t, a, "bob", domain.SideSell, domain.TypeLimit, 43_000_00, domain.SatPerBTC/2))
	if err != nil {
		t.Fatal(err)
	}
	if res.Order.Status != domain.StatusOpen || len(res.Trades) != 0 {
		t.Fatalf("maker should rest: %+v", res.Order)
	}

	// alice crosses with a buy limit at 43,100.00 for 0.3 BTC
	res, err = a.PlaceOrder(place(t, a, "alice", domain.SideBuy, domain.TypeLimit, 43_100_00, 3*domain.SatPerBTC/10))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Trades) != 1 {
		t.Fatalf("want 1 trade, got %d", len(res.Trades))
	}
	// price-time priority: the trade happens at the MAKER's price
	if res.Trades[0].Price != 43_000_00 {
		t.Fatalf("trade price = %d, want maker's 4300000", res.Trades[0].Price)
	}
	if res.Order.Status != domain.StatusFilled {
		t.Fatalf("taker should be filled: %+v", res.Order)
	}

	a.Flush()
	if err := a.Ledger.CheckInvariants(); err != nil {
		t.Fatal(err)
	}
	// alice paid at 43,000 not 43,100 — surplus hold was released
	b := a.Ledger.Balances("alice")[domain.AssetQuote]
	spent := domain.Notional(43_000_00, 3*domain.SatPerBTC/10)
	if b.Available != 1_000_000_00-spent || b.Hold != 0 {
		t.Fatalf("alice quote after fill: %+v (spent=%d)", b, spent)
	}
}

func TestTimePriorityWithinLevel(t *testing.T) {
	a := newTestApp(t)
	r1, _ := a.PlaceOrder(place(t, a, "bob", domain.SideSell, domain.TypeLimit, 43_000_00, domain.MinLotSat))
	r2, _ := a.PlaceOrder(place(t, a, "bob", domain.SideSell, domain.TypeLimit, 43_000_00, domain.MinLotSat))

	res, err := a.PlaceOrder(place(t, a, "alice", domain.SideBuy, domain.TypeLimit, 43_000_00, domain.MinLotSat))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Trades) != 1 || res.Trades[0].SellOrderID != r1.Order.ID {
		t.Fatalf("first-in should fill first: got %+v", res.Trades)
	}
	_ = r2
}

func TestPartialFillRestsRemainder(t *testing.T) {
	a := newTestApp(t)
	_, _ = a.PlaceOrder(place(t, a, "bob", domain.SideSell, domain.TypeLimit, 43_000_00, domain.MinLotSat))
	res, err := a.PlaceOrder(place(t, a, "alice", domain.SideBuy, domain.TypeLimit, 43_000_00, 3*domain.MinLotSat))
	if err != nil {
		t.Fatal(err)
	}
	if res.Order.Status != domain.StatusOpen || res.Order.Remaining != 2*domain.MinLotSat {
		t.Fatalf("remainder should rest: %+v", res.Order)
	}
	snap := a.Engine.Snapshot("alice")
	if len(snap.OpenOrders) != 1 || snap.BestBid != 43_000_00 {
		t.Fatalf("book should hold the remainder: %+v", snap)
	}
}

func TestMarketOrderIsIOCWithBand(t *testing.T) {
	a := newTestApp(t)
	_, _ = a.PlaceOrder(place(t, a, "bob", domain.SideSell, domain.TypeLimit, 43_000_00, domain.MinLotSat))

	// market buy for far more than the book holds → fills what exists, cancels the rest
	res, err := a.PlaceOrder(place(t, a, "alice", domain.SideBuy, domain.TypeMarket, 0, 10*domain.MinLotSat))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Trades) != 1 || res.Order.Status != domain.StatusCancelled {
		t.Fatalf("IOC: want 1 fill + cancelled remainder, got %+v %+v", res.Trades, res.Order)
	}
	a.Flush()
	if b := a.Ledger.Balances("alice")[domain.AssetQuote]; b.Hold != 0 {
		t.Fatalf("cancelled remainder must release its hold: %+v", b)
	}

	// empty book → market order rejected outright
	a2 := newTestApp(t)
	if _, err := a2.PlaceOrder(place(t, a2, "alice", domain.SideBuy, domain.TypeMarket, 0, domain.MinLotSat)); err == nil {
		t.Fatal("market order on empty book should fail")
	}
}

func TestCancelReleasesHold(t *testing.T) {
	a := newTestApp(t)
	res, err := a.PlaceOrder(place(t, a, "alice", domain.SideBuy, domain.TypeLimit, 40_000_00, domain.MinLotSat))
	if err != nil {
		t.Fatal(err)
	}
	if err := a.CancelOrder(res.Order.ID, "bob"); err == nil {
		t.Fatal("cancelling someone else's order should fail")
	}
	if err := a.CancelOrder(res.Order.ID, "alice"); err != nil {
		t.Fatal(err)
	}
	a.Flush()
	b := a.Ledger.Balances("alice")[domain.AssetQuote]
	if b.Available != 1_000_000_00 || b.Hold != 0 {
		t.Fatalf("cancel must fully release: %+v", b)
	}
}

func TestInsufficientBalanceRejected(t *testing.T) {
	a := newTestApp(t)
	// alice has 10 BTC — selling 11 must fail before reaching the engine
	if _, err := a.PlaceOrder(place(t, a, "alice", domain.SideSell, domain.TypeLimit, 43_000_00, 11*domain.SatPerBTC)); err == nil {
		t.Fatal("oversell was accepted")
	}
	snap := a.Engine.Snapshot("")
	if len(snap.Depth.Asks) != 0 {
		t.Fatal("rejected order reached the book")
	}
}
