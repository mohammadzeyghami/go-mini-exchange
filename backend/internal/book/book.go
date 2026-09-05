// Package book implements the in-memory order book: two price-sorted sides,
// FIFO queues per price level, and O(1) cancel lookup by order id.
//
// The book is NOT safe for concurrent use — by design. Exactly one goroutine
// (the engine, chapter 4's single-writer pattern) owns it.
package book

import (
	"sort"

	"github.com/mohammadzeyghami/go-mini-exchange/internal/domain"
)

type Level struct {
	Price  int64
	Orders []*domain.Order // FIFO — index 0 is the oldest (time priority)
	Total  int64           // sum of Remaining at this price
}

type Book struct {
	bids []*Level // sorted descending — best bid first
	asks []*Level // sorted ascending — best ask first
	byID map[string]*domain.Order
}

func New() *Book {
	return &Book{byID: make(map[string]*domain.Order)}
}

func (b *Book) side(s domain.Side) *[]*Level {
	if s == domain.SideBuy {
		return &b.bids
	}
	return &b.asks
}

// levelIndex finds the insertion index for price on the given side.
func levelIndex(levels []*Level, s domain.Side, price int64) (int, bool) {
	i := sort.Search(len(levels), func(i int) bool {
		if s == domain.SideBuy {
			return levels[i].Price <= price
		}
		return levels[i].Price >= price
	})
	if i < len(levels) && levels[i].Price == price {
		return i, true
	}
	return i, false
}

// Insert rests an order in the book. The caller guarantees the id is unique.
func (b *Book) Insert(o *domain.Order) {
	levels := b.side(o.Side)
	i, ok := levelIndex(*levels, o.Side, o.Price)
	if !ok {
		lvl := &Level{Price: o.Price}
		*levels = append(*levels, nil)
		copy((*levels)[i+1:], (*levels)[i:])
		(*levels)[i] = lvl
	}
	lvl := (*levels)[i]
	lvl.Orders = append(lvl.Orders, o)
	lvl.Total += o.Remaining
	b.byID[o.ID] = o
}

// Best returns the top level of the opposite-facing side, or nil.
func (b *Book) Best(s domain.Side) *Level {
	levels := *b.side(s)
	if len(levels) == 0 {
		return nil
	}
	return levels[0]
}

// Get returns a resting order by id, or nil.
func (b *Book) Get(id string) *domain.Order { return b.byID[id] }

// Remove takes an order out of the book (cancel, or fully filled maker).
func (b *Book) Remove(o *domain.Order) {
	delete(b.byID, o.ID)
	levels := b.side(o.Side)
	i, ok := levelIndex(*levels, o.Side, o.Price)
	if !ok {
		return
	}
	lvl := (*levels)[i]
	for j, cur := range lvl.Orders {
		if cur.ID == o.ID {
			lvl.Orders = append(lvl.Orders[:j], lvl.Orders[j+1:]...)
			lvl.Total -= cur.Remaining
			break
		}
	}
	if len(lvl.Orders) == 0 {
		*levels = append((*levels)[:i], (*levels)[i+1:]...)
	}
}

// ReduceTop consumes qty from the oldest order at the top level of side s.
// It returns that maker order. When the maker is exhausted it leaves the book.
func (b *Book) ReduceTop(s domain.Side, qty int64) *domain.Order {
	levels := b.side(s)
	lvl := (*levels)[0]
	maker := lvl.Orders[0]
	maker.Remaining -= qty
	lvl.Total -= qty
	if maker.Remaining == 0 {
		maker.Status = domain.StatusFilled
		lvl.Orders = lvl.Orders[1:]
		delete(b.byID, maker.ID)
		if len(lvl.Orders) == 0 {
			*levels = append((*levels)[:0], (*levels)[1:]...)
		}
	}
	return maker
}

// Depth returns the top n levels of each side.
func (b *Book) Depth(n int) domain.Depth {
	d := domain.Depth{Bids: []domain.PriceLevel{}, Asks: []domain.PriceLevel{}}
	for i, lvl := range b.bids {
		if i == n {
			break
		}
		d.Bids = append(d.Bids, domain.PriceLevel{Price: lvl.Price, Qty: lvl.Total})
	}
	for i, lvl := range b.asks {
		if i == n {
			break
		}
		d.Asks = append(d.Asks, domain.PriceLevel{Price: lvl.Price, Qty: lvl.Total})
	}
	return d
}

// OpenOrders returns every resting order for a user (any order, if user == "").
func (b *Book) OpenOrders(userID string) []*domain.Order {
	var out []*domain.Order
	for _, o := range b.byID {
		if userID == "" || o.UserID == userID {
			out = append(out, o)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Seq < out[j].Seq })
	return out
}

// Validate checks the book's internal invariants (used by tests):
// sides sorted, level totals consistent, FIFO ordered by Seq.
func (b *Book) Validate() bool {
	check := func(levels []*Level, s domain.Side) bool {
		for i, lvl := range levels {
			if i > 0 {
				prev := levels[i-1].Price
				if s == domain.SideBuy && lvl.Price >= prev {
					return false
				}
				if s == domain.SideSell && lvl.Price <= prev {
					return false
				}
			}
			var sum int64
			var lastSeq uint64
			for _, o := range lvl.Orders {
				if o.Price != lvl.Price || o.Remaining <= 0 || o.Seq <= lastSeq {
					return false
				}
				lastSeq = o.Seq
				sum += o.Remaining
			}
			if sum != lvl.Total || len(lvl.Orders) == 0 {
				return false
			}
		}
		return true
	}
	return check(b.bids, domain.SideBuy) && check(b.asks, domain.SideSell)
}
