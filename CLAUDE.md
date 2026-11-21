# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A Binance spot trading bot in Go. It does two things, both with **real market orders**:

1. **DCA buys** — at times listed in `config.json` (`order_list`), buys each symbol with a fixed USDT amount.
2. **Portfolio rebalancing** — on cron schedules, sells over-weight assets, then buys under-weight assets using only the realized USDT proceeds.

Everything is a real order against Binance using `API_KEY` / `SECRET_KEY` — no paper trading.

## Commands

```bash
go build ./...                # main binary: binance-spot
go run .                      # run the bot (needs API_KEY/SECRET_KEY, .env is autoloaded)
go test ./...                 # run all tests
go test ./spot -run TestFloorToIncrement -v   # single test
go vet ./...
```

- `config/config.json.example` is compiled in via `go:embed`, so config changes require a rebuild.
- `spot/spot_test.go` (`TestBuyCrypto`) places **real market orders** against the live API. Do not run it casually.

## Architecture

- [main.go](main.go) — entry point. `init()` sets `time.Local = America/New_York` (both cron and the ticker depend on this). Starts a cron for `Spot.Rebalance` and a 1-second ticker that fires `Spot.BuyCrypto` when `order_list[].time_at` matches the current second (`15:04:05` format). Buy jobs run in goroutines so retries don't block the ticker.
- [config/](config/) — embeds `config.json`, validates it in `init()`. `TargetAllocations()` sums each symbol's amount across all daily orders; these sums are the **relative target weights** for rebalancing, not absolute trade amounts. Config fields: `max_retries` (buy retries), `tick_size_power` (per-retry price discount), `schedules` (cron, NY time), `keep` (base assets held long-term — must not appear in order_list, validated at startup; rebalance never touches them), `order_list` (time + symbol→USDT).
- [spot/](spot/) — `Spot` wraps `*binance.Client` with a `tradeMu` mutex so rebalancing and scheduled buys never trade concurrently. `BuyCrypto` fetches exchangeInfo + book tickers, then market-buys each symbol with retries; `tools.RoundPriceAndQuantity` rounds price/quantity to exchange tick/lot sizes. `Rebalance` (spot/rebalance.go) loads positions and filters, plans sells/buys, executes the full sell phase first, then distributes the actual post-sell USDT balance across buys.
- [tools/](tools/) — price/quantity rounding helpers shared by the buy path.
- [cmd/dust/](cmd/dust/) — standalone utility for dust logs; not part of the main binary.

## Testing pattern

`spot/rebalance_test.go` mocks the Binance API with an `httptest` server by overriding `client.BaseURL` and `client.HTTPClient` — follow this pattern for new tests that touch the API. Pure-logic tests (`buildRebalancePlan`, `floorToIncrement`, etc.) need no mock.

## Conventions

- Code comments and commit messages in this repo are written in Chinese; match that style.
