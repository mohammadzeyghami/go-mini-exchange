// REST handlers (chapter 5): thin — decode, validate, call the app, encode.
// Identity is the X-User header (paper trading, no auth — out of scope, see
// README). Error mapping: validation → 422, insufficient balance → 422,
// unknown order → 404, someone else's order → 403.
package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/mohammadzeyghami/go-mini-exchange/internal/app"
	"github.com/mohammadzeyghami/go-mini-exchange/internal/domain"
	"github.com/mohammadzeyghami/go-mini-exchange/internal/engine"
	"github.com/mohammadzeyghami/go-mini-exchange/internal/ledger"
)

type Server struct {
	App *app.App
	Hub *Hub
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("GET /api/orderbook", s.orderbook)
	mux.HandleFunc("GET /api/trades", s.trades)
	mux.HandleFunc("GET /api/balances", s.balances)
	mux.HandleFunc("GET /api/orders", s.openOrders)
	mux.HandleFunc("POST /api/orders", s.placeOrder)
	mux.HandleFunc("POST /api/faucet", s.faucet)
	mux.HandleFunc("DELETE /api/orders/{id}", s.cancelOrder)
	mux.HandleFunc("GET /ws", s.Hub.Serve)
	return withCORS(mux)
}

// withCORS: the dashboard runs on another port in dev; this is a public demo
// API with no credentials, so a blanket allow is fine here.
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-User")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func user(r *http.Request) string {
	if u := r.Header.Get("X-User"); u != "" {
		return u
	}
	return "alice"
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, app.ErrValidation),
		errors.Is(err, ledger.ErrInsufficientBalance),
		errors.Is(err, engine.ErrNoLiquidity):
		status = http.StatusUnprocessableEntity
	case errors.Is(err, engine.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, engine.ErrNotOwner):
		status = http.StatusForbidden
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) orderbook(w http.ResponseWriter, r *http.Request) {
	snap := s.App.Engine.Snapshot("")
	writeJSON(w, http.StatusOK, map[string]any{
		"depth":   snap.Depth,
		"bestBid": snap.BestBid,
		"bestAsk": snap.BestAsk,
		"last":    s.App.LastPrice(),
	})
}

func (s *Server) trades(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if q := r.URL.Query().Get("limit"); q != "" {
		if n, err := strconv.Atoi(q); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	writeJSON(w, http.StatusOK, s.App.RecentTrades(limit))
}

func (s *Server) balances(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.App.Ledger.Balances(user(r)))
}

func (s *Server) openOrders(w http.ResponseWriter, r *http.Request) {
	snap := s.App.Engine.Snapshot(user(r))
	if snap.OpenOrders == nil {
		snap.OpenOrders = []domain.Order{}
	}
	writeJSON(w, http.StatusOK, snap.OpenOrders)
}

type placeOrderReq struct {
	Side  string `json:"side"`
	Type  string `json:"type"`
	Price int64  `json:"price"` // cents/BTC
	Qty   int64  `json:"qty"`   // satoshi
}

func (s *Server) placeOrder(w http.ResponseWriter, r *http.Request) {
	var req placeOrderReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	res, err := s.App.PlaceOrder(app.PlaceRequest{
		UserID: user(r),
		Side:   domain.Side(req.Side),
		Type:   domain.OrderType(req.Type),
		Price:  req.Price,
		Qty:    req.Qty,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	trades := res.Trades
	if trades == nil {
		trades = []domain.Trade{}
	}
	writeJSON(w, http.StatusCreated, map[string]any{"order": res.Order, "trades": trades})
}

// faucet seeds the calling user's paper-trading balances. Idempotent: the
// ledger's seed transaction ids make a second call a no-op, so any client
// (e.g. the Telegram mini-app) can call it on every login.
func (s *Server) faucet(w http.ResponseWriter, r *http.Request) {
	u := user(r)
	s.App.SeedUser(u, 100_000_000, 50*domain.SatPerBTC) // 1,000,000.00 USDT + 50 BTC
	writeJSON(w, http.StatusOK, s.App.Ledger.Balances(u))
}

func (s *Server) cancelOrder(w http.ResponseWriter, r *http.Request) {
	if err := s.App.CancelOrder(r.PathValue("id"), user(r)); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}
