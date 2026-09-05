// Package domain holds the core types shared by the book, engine and ledger.
//
// Money is integer only (see chapter 7 of the book — never float):
//   - BTC quantities are in satoshi (1 BTC = 1e8 sat).
//   - USDT amounts and prices are in cents (1 USDT = 100 cents).
//   - Price is "cents per 1 BTC"; the notional of a fill is
//     floor(price * qty / 1e8) cents.
package domain

import "time"

const (
	AssetBase  = "BTC"
	AssetQuote = "USDT"

	// SatPerBTC is the number of base units in one BTC.
	SatPerBTC = 100_000_000

	// MinLotSat is the minimum order quantity (0.0001 BTC).
	MinLotSat = 10_000

	// MaxQtySat / MaxPriceCents bound int64 math: price*qty stays far below
	// the int64 range (1e9 * 1e11 / 1e8 = 1e12).
	MaxQtySat     = 100_000_000_000 // 1000 BTC
	MaxPriceCents = 1_000_000_000   // 10,000,000 USDT

	// MakerFeeBps / TakerFeeBps are charged on the asset each side receives.
	MakerFeeBps = 10
	TakerFeeBps = 20
)

type Side string

const (
	SideBuy  Side = "buy"
	SideSell Side = "sell"
)

func (s Side) Opposite() Side {
	if s == SideBuy {
		return SideSell
	}
	return SideBuy
}

type OrderType string

const (
	TypeLimit  OrderType = "limit"
	TypeMarket OrderType = "market"
)

// TimeInForce: GTC rests in the book, IOC cancels the unfilled remainder.
type TimeInForce string

const (
	GTC TimeInForce = "gtc"
	IOC TimeInForce = "ioc"
)

type OrderStatus string

const (
	StatusOpen      OrderStatus = "open"
	StatusFilled    OrderStatus = "filled"
	StatusCancelled OrderStatus = "cancelled"
)

type Order struct {
	ID        string      `json:"id"`
	UserID    string      `json:"userId"`
	Side      Side        `json:"side"`
	Type      OrderType   `json:"type"`
	TIF       TimeInForce `json:"tif"`
	Price     int64       `json:"price"` // cents per BTC (limit price; band price for market)
	Qty       int64       `json:"qty"`   // satoshi
	Remaining int64       `json:"remaining"`
	Status    OrderStatus `json:"status"`
	Seq       uint64      `json:"seq"` // engine arrival order — time priority
	CreatedAt time.Time   `json:"createdAt"`
}

type Trade struct {
	ID          string    `json:"id"`
	Price       int64     `json:"price"` // maker's price — always
	Qty         int64     `json:"qty"`
	BuyOrderID  string    `json:"buyOrderId"`
	SellOrderID string    `json:"sellOrderId"`
	BuyerID     string    `json:"buyerId"`
	SellerID    string    `json:"sellerId"`
	TakerSide   Side      `json:"takerSide"`
	At          time.Time `json:"at"`
}

// Notional returns the quote cost of qty satoshi at price cents/BTC, floored.
// The flooring remainder is handled by hold release (chapter 10: rounding is
// defined in one place, deterministically).
func Notional(price, qty int64) int64 {
	return price * qty / SatPerBTC
}

// Fee returns the fee for an amount at the given basis points, floored.
func Fee(amount int64, bps int64) int64 {
	return amount * bps / 10_000
}

// PriceLevel is one row of a depth snapshot.
type PriceLevel struct {
	Price int64 `json:"price"`
	Qty   int64 `json:"qty"`
}

// Depth is a point-in-time view of the top of the book.
type Depth struct {
	Bids []PriceLevel `json:"bids"` // best (highest) first
	Asks []PriceLevel `json:"asks"` // best (lowest) first
}
