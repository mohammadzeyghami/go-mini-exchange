package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/mohammadzeyghami/go-mini-exchange/internal/api"
	"github.com/mohammadzeyghami/go-mini-exchange/internal/app"
	"github.com/mohammadzeyghami/go-mini-exchange/internal/domain"
)

func main() {
	hub := api.NewHub()
	a := app.New(hub)

	// Paper-trading accounts: 1,000,000 USDT + 50 BTC each.
	for _, u := range []string{"alice", "bob"} {
		a.SeedUser(u, 100_000_000, 50*domain.SatPerBTC) // 1,000,000.00 USDT + 50 BTC
	}

	a.StartInvariantChecker(5 * time.Second)

	if os.Getenv("DEMO_BOT") != "false" {
		a.StartDemoBot("mm-bot")
	}

	srv := &api.Server{App: a, Hub: hub}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8140"
	}
	log.Printf("go-mini-exchange listening on :%s (market %s/%s)", port, domain.AssetBase, domain.AssetQuote)
	log.Fatal(http.ListenAndServe("0.0.0.0:"+port, srv.Routes()))
}
