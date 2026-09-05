// Package ledger is a double-entry ledger (chapter 7 of the book):
//
//   - every transaction is a set of entries whose deltas sum to zero per asset
//   - entries are append-only — mistakes are corrected by new transactions
//   - each user/asset pair has an available and a hold sub-account
//   - transactions are idempotent by key: the same event applied twice is a no-op
//
// The external world is the "treasury" account: deposits move value from the
// treasury (which goes negative — it represents everything outside the
// system), so the total of every asset is always exactly zero.
package ledger

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/mohammadzeyghami/go-mini-exchange/internal/domain"
)

const (
	AccTreasury = "system:treasury"
	AccFees     = "system:fees"
)

var (
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrUnknownHold         = errors.New("unknown hold")
)

// Bucket says which sub-account an entry touches.
type Bucket string

const (
	Available Bucket = "available"
	Hold      Bucket = "hold"
)

type Entry struct {
	Account string `json:"account"`
	Asset   string `json:"asset"`
	Bucket  Bucket `json:"bucket"`
	Delta   int64  `json:"delta"`
}

type Tx struct {
	ID      string    `json:"id"` // idempotency key
	Kind    string    `json:"kind"`
	Entries []Entry   `json:"entries"`
	At      time.Time `json:"at"`
}

type Balance struct {
	Available int64 `json:"available"`
	Hold      int64 `json:"hold"`
}

// holdState tracks one order's reservation so the leftover can be released.
type holdState struct {
	account  string
	asset    string
	reserved int64
	spent    int64
}

type Ledger struct {
	mu       sync.Mutex
	balances map[string]map[string]*Balance // account → asset → balance
	txs      []Tx                           // append-only journal
	applied  map[string]bool                // idempotency keys
	holds    map[string]*holdState          // orderID → reservation
}

func New() *Ledger {
	return &Ledger{
		balances: make(map[string]map[string]*Balance),
		applied:  make(map[string]bool),
		holds:    make(map[string]*holdState),
	}
}

func (l *Ledger) bal(account, asset string) *Balance {
	m, ok := l.balances[account]
	if !ok {
		m = make(map[string]*Balance)
		l.balances[account] = m
	}
	b, ok := m[asset]
	if !ok {
		b = &Balance{}
		m[asset] = b
	}
	return b
}

// apply commits a transaction atomically under the lock. It rejects any
// transaction that does not sum to zero per asset, and any entry that would
// take a non-treasury bucket negative. Caller holds l.mu.
func (l *Ledger) apply(tx Tx) error {
	if l.applied[tx.ID] {
		return nil // idempotent replay — chapter 19
	}
	sums := make(map[string]int64)
	for _, e := range tx.Entries {
		sums[e.Asset] += e.Delta
	}
	for asset, s := range sums {
		if s != 0 {
			return fmt.Errorf("unbalanced tx %s for %s: sum=%d", tx.ID, asset, s)
		}
	}
	// Check feasibility before mutating anything (all-or-nothing).
	next := make(map[string]int64)
	key := func(e Entry) string { return e.Account + "/" + e.Asset + "/" + string(e.Bucket) }
	for _, e := range tx.Entries {
		k := key(e)
		if _, ok := next[k]; !ok {
			b := l.bal(e.Account, e.Asset)
			if e.Bucket == Available {
				next[k] = b.Available
			} else {
				next[k] = b.Hold
			}
		}
		next[k] += e.Delta
		if e.Account != AccTreasury && next[k] < 0 {
			return fmt.Errorf("%w: %s %s %s would go to %d", ErrInsufficientBalance, e.Account, e.Asset, e.Bucket, next[k])
		}
	}
	for _, e := range tx.Entries {
		b := l.bal(e.Account, e.Asset)
		if e.Bucket == Available {
			b.Available += e.Delta
		} else {
			b.Hold += e.Delta
		}
	}
	tx.At = time.Now()
	l.txs = append(l.txs, tx)
	l.applied[tx.ID] = true
	return nil
}

// Deposit credits available funds from the treasury (paper money faucet).
func (l *Ledger) Deposit(idemKey, account, asset string, amount int64) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.apply(Tx{ID: idemKey, Kind: "deposit", Entries: []Entry{
		{Account: AccTreasury, Asset: asset, Bucket: Available, Delta: -amount},
		{Account: account, Asset: asset, Bucket: Available, Delta: amount},
	}})
}

// Reserve moves amount from available to hold for an order about to enter the
// engine. Fails (without side effects) when available is insufficient.
func (l *Ledger) Reserve(orderID, account, asset string, amount int64) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.applied["reserve:"+orderID] {
		return nil // replay — the hold state already exists
	}
	err := l.apply(Tx{ID: "reserve:" + orderID, Kind: "reserve", Entries: []Entry{
		{Account: account, Asset: asset, Bucket: Available, Delta: -amount},
		{Account: account, Asset: asset, Bucket: Hold, Delta: amount},
	}})
	if err != nil {
		return err
	}
	l.holds[orderID] = &holdState{account: account, asset: asset, reserved: amount}
	return nil
}

// SettleTrade applies one fill: quote moves from the buyer's hold to the
// seller, base moves from the seller's hold to the buyer, and each side pays
// a fee on the asset it receives (maker/taker rates from domain).
func (l *Ledger) SettleTrade(t domain.Trade) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.applied["trade:"+t.ID] {
		return nil // replay — holds were already advanced once
	}

	notional := domain.Notional(t.Price, t.Qty)
	buyerBps, sellerBps := int64(domain.MakerFeeBps), int64(domain.TakerFeeBps)
	if t.TakerSide == domain.SideBuy {
		buyerBps, sellerBps = domain.TakerFeeBps, domain.MakerFeeBps
	}
	feeBase := domain.Fee(t.Qty, buyerBps)      // buyer receives base
	feeQuote := domain.Fee(notional, sellerBps) // seller receives quote

	err := l.apply(Tx{ID: "trade:" + t.ID, Kind: "trade", Entries: []Entry{
		// quote: buyer hold → seller available (minus fee) + fee account
		{Account: t.BuyerID, Asset: domain.AssetQuote, Bucket: Hold, Delta: -notional},
		{Account: t.SellerID, Asset: domain.AssetQuote, Bucket: Available, Delta: notional - feeQuote},
		{Account: AccFees, Asset: domain.AssetQuote, Bucket: Available, Delta: feeQuote},
		// base: seller hold → buyer available (minus fee) + fee account
		{Account: t.SellerID, Asset: domain.AssetBase, Bucket: Hold, Delta: -t.Qty},
		{Account: t.BuyerID, Asset: domain.AssetBase, Bucket: Available, Delta: t.Qty - feeBase},
		{Account: AccFees, Asset: domain.AssetBase, Bucket: Available, Delta: feeBase},
	}})
	if err != nil {
		return err
	}
	if h, ok := l.holds[t.BuyOrderID]; ok {
		h.spent += notional
	}
	if h, ok := l.holds[t.SellOrderID]; ok {
		h.spent += t.Qty
	}
	return nil
}

// ReleaseHold returns an order's unspent reservation to available and drops
// the hold. Safe to call twice (the hold is gone the second time).
func (l *Ledger) ReleaseHold(orderID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	h, ok := l.holds[orderID]
	if !ok {
		return nil
	}
	leftover := h.reserved - h.spent
	if leftover > 0 {
		err := l.apply(Tx{ID: "release:" + orderID, Kind: "release", Entries: []Entry{
			{Account: h.account, Asset: h.asset, Bucket: Hold, Delta: -leftover},
			{Account: h.account, Asset: h.asset, Bucket: Available, Delta: leftover},
		}})
		if err != nil {
			return err
		}
	}
	delete(l.holds, orderID)
	return nil
}

// Balances returns a copy of one account's balances.
func (l *Ledger) Balances(account string) map[string]Balance {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make(map[string]Balance)
	for asset, b := range l.balances[account] {
		out[asset] = *b
	}
	return out
}

// CheckInvariants recomputes everything from the journal and verifies the
// three unbreakable rules (chapter 7). It returns nil when the books are
// clean. Run in tests and, in main, periodically at runtime.
func (l *Ledger) CheckInvariants() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 1. Every asset sums to zero across all entries.
	sums := make(map[string]int64)
	// 3. The balance projection equals the journal replay.
	replay := make(map[string]int64) // account/asset/bucket → sum
	for _, tx := range l.txs {
		for _, e := range tx.Entries {
			sums[e.Asset] += e.Delta
			replay[e.Account+"/"+e.Asset+"/"+string(e.Bucket)] += e.Delta
		}
	}
	for asset, s := range sums {
		if s != 0 {
			return fmt.Errorf("asset %s does not sum to zero: %d", asset, s)
		}
	}
	// 2. No non-treasury bucket is negative.
	for account, assets := range l.balances {
		for asset, b := range assets {
			if account != AccTreasury && (b.Available < 0 || b.Hold < 0) {
				return fmt.Errorf("negative balance: %s %s %+v", account, asset, b)
			}
			if replay[account+"/"+asset+"/available"] != b.Available ||
				replay[account+"/"+asset+"/hold"] != b.Hold {
				return fmt.Errorf("projection drift: %s %s", account, asset)
			}
		}
	}
	return nil
}

// TxCount returns the journal length (used by tests).
func (l *Ledger) TxCount() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.txs)
}
