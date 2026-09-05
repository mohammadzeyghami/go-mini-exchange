package ledger

import (
	"errors"
	"testing"
	"time"

	"github.com/mohammadzeyghami/go-mini-exchange/internal/domain"
)

func TestDepositReserveRelease(t *testing.T) {
	l := New()
	if err := l.Deposit("d1", "alice", domain.AssetQuote, 10_000); err != nil {
		t.Fatal(err)
	}
	if err := l.Reserve("o1", "alice", domain.AssetQuote, 4_000); err != nil {
		t.Fatal(err)
	}
	b := l.Balances("alice")[domain.AssetQuote]
	if b.Available != 6_000 || b.Hold != 4_000 {
		t.Fatalf("after reserve: %+v", b)
	}
	// insufficient available → error, no side effects
	if err := l.Reserve("o2", "alice", domain.AssetQuote, 7_000); !errors.Is(err, ErrInsufficientBalance) {
		t.Fatalf("want ErrInsufficientBalance, got %v", err)
	}
	if err := l.ReleaseHold("o1"); err != nil {
		t.Fatal(err)
	}
	b = l.Balances("alice")[domain.AssetQuote]
	if b.Available != 10_000 || b.Hold != 0 {
		t.Fatalf("after release: %+v", b)
	}
	if err := l.CheckInvariants(); err != nil {
		t.Fatal(err)
	}
}

func testTrade() domain.Trade {
	return domain.Trade{
		ID: "t1", Price: 40_000_00, Qty: domain.SatPerBTC, // 1 BTC @ 40,000.00
		BuyOrderID: "ob", SellOrderID: "os",
		BuyerID: "alice", SellerID: "bob",
		TakerSide: domain.SideBuy, At: time.Now(),
	}
}

func setupTradeLedger(t *testing.T) *Ledger {
	t.Helper()
	l := New()
	must := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
	must(l.Deposit("da", "alice", domain.AssetQuote, 100_000_00))
	must(l.Deposit("db", "bob", domain.AssetBase, 2*domain.SatPerBTC))
	must(l.Reserve("ob", "alice", domain.AssetQuote, 40_000_00))
	must(l.Reserve("os", "bob", domain.AssetBase, domain.SatPerBTC))
	return l
}

func TestSettleTradeMovesHoldsAndFees(t *testing.T) {
	l := setupTradeLedger(t)
	if err := l.SettleTrade(testTrade()); err != nil {
		t.Fatal(err)
	}
	// buyer (taker, 20 bps on BTC): receives 1 BTC − 0.002 BTC
	ab := l.Balances("alice")[domain.AssetBase]
	wantBTC := domain.SatPerBTC - domain.Fee(domain.SatPerBTC, domain.TakerFeeBps)
	if ab.Available != wantBTC {
		t.Fatalf("alice BTC = %d, want %d", ab.Available, wantBTC)
	}
	// seller (maker, 10 bps on USDT): receives 40,000.00 − 40.00
	bq := l.Balances("bob")[domain.AssetQuote]
	wantQ := int64(40_000_00) - domain.Fee(40_000_00, domain.MakerFeeBps)
	if bq.Available != wantQ {
		t.Fatalf("bob USDT = %d, want %d", bq.Available, wantQ)
	}
	// holds fully consumed
	if h := l.Balances("alice")[domain.AssetQuote].Hold; h != 0 {
		t.Fatalf("alice quote hold = %d", h)
	}
	if h := l.Balances("bob")[domain.AssetBase].Hold; h != 0 {
		t.Fatalf("bob base hold = %d", h)
	}
	if err := l.CheckInvariants(); err != nil {
		t.Fatal(err)
	}
}

func TestSettleTradeIsIdempotent(t *testing.T) {
	l := setupTradeLedger(t)
	tr := testTrade()
	if err := l.SettleTrade(tr); err != nil {
		t.Fatal(err)
	}
	before := l.TxCount()
	if err := l.SettleTrade(tr); err != nil { // exact replay — must be a no-op
		t.Fatal(err)
	}
	if l.TxCount() != before {
		t.Fatal("replayed trade appended a transaction")
	}
	if err := l.CheckInvariants(); err != nil {
		t.Fatal(err)
	}
}

func TestUnbalancedTxRejected(t *testing.T) {
	l := New()
	l.mu.Lock()
	err := l.apply(Tx{ID: "bad", Kind: "test", Entries: []Entry{
		{Account: "alice", Asset: domain.AssetQuote, Bucket: Available, Delta: 10},
	}})
	l.mu.Unlock()
	if err == nil {
		t.Fatal("unbalanced transaction was accepted")
	}
}
