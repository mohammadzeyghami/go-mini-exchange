package app

import (
	"math/rand"
	"time"

	"github.com/mohammadzeyghami/go-mini-exchange/internal/domain"
)

// StartDemoBot runs a toy market maker so the book and the trade feed are
// alive without human traders: it quotes both sides around a random-walking
// mid price and occasionally crosses the spread. Demo only — real market
// making (inventory, skew, adverse selection) is out of scope by design.
func (a *App) StartDemoBot(userID string) {
	a.SeedUser(userID, 500_000_000_00, 10_000*domain.SatPerBTC) // deep pockets

	go func() {
		rng := rand.New(rand.NewSource(time.Now().UnixNano()))
		mid := int64(43_000_00) // 43,000.00 USDT
		var open []string

		for {
			select {
			case <-a.done:
				return
			case <-time.After(time.Duration(250+rng.Intn(500)) * time.Millisecond):
			}

			// random walk the mid price, ±0.05% per tick
			mid += int64(rng.Intn(5)-2) * (mid / 2000)

			side := domain.SideBuy
			offset := -int64(rng.Intn(30)+1) * (mid / 10_000) // up to −0.3%
			if rng.Intn(2) == 0 {
				side = domain.SideSell
				offset = -offset
			}
			qty := int64((rng.Intn(40) + 2) * domain.MinLotSat) // 0.0002–0.0084 BTC

			typ := domain.TypeLimit
			price := mid + offset
			if rng.Intn(6) == 0 { // sometimes take liquidity → trades happen
				typ = domain.TypeMarket
				price = 0
			}

			res, err := a.PlaceOrder(PlaceRequest{
				UserID: userID, Side: side, Type: typ, Price: price, Qty: qty,
			})
			if err == nil && res.Order.Status == domain.StatusOpen {
				open = append(open, res.Order.ID)
			}

			// keep the book tidy: cancel old quotes now and then
			if len(open) > 40 {
				i := rng.Intn(len(open))
				_ = a.CancelOrder(open[i], userID)
				open = append(open[:i], open[i+1:]...)
			}
		}
	}()
}
