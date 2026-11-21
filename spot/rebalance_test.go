package spot

import (
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"binance-spot/config"

	"github.com/adshao/go-binance/v2"
)

func TestBuildRebalancePlanUsesConfiguredAssetsOnly(t *testing.T) {
	assets := []*rebalanceAsset{
		{symbol: "BNBUSDT", baseAsset: "BNB", targetWeight: 15, free: 20, locked: 10, bid: 1},
		{symbol: "SOLUSDT", baseAsset: "SOL", targetWeight: 15, free: 10, bid: 1},
		{symbol: "BTCUSDT", baseAsset: "BTC", targetWeight: 35, free: 30, bid: 1},
		{symbol: "ETHUSDT", baseAsset: "ETH", targetWeight: 25, free: 20, bid: 1},
	}

	sells, buys, total := buildRebalancePlan(assets)
	if total != 90 {
		t.Fatalf("total value = %v, want 90", total)
	}
	if len(sells) != 1 || sells[0].asset.baseAsset != "BNB" || sells[0].quoteAmount != 15 {
		t.Fatalf("sells = %#v, want BNB value 15", sells)
	}

	wantBuys := map[string]float64{"SOL": 5, "BTC": 5, "ETH": 5}
	if len(buys) != len(wantBuys) {
		t.Fatalf("buy count = %d, want %d", len(buys), len(wantBuys))
	}
	for _, buy := range buys {
		if buy.quoteAmount != wantBuys[buy.asset.baseAsset] {
			t.Errorf("buy %s = %v, want %v", buy.asset.baseAsset, buy.quoteAmount, wantBuys[buy.asset.baseAsset])
		}
	}
}

func TestFloorToIncrement(t *testing.T) {
	got := floorToIncrement(1.234567, 0.001)
	if math.Abs(got-1.234) > 1e-12 {
		t.Fatalf("floorToIncrement = %.12f, want 1.234", got)
	}
}

func TestRebalanceSellsBeforeBuyingAndIgnoresUnconfiguredAssets(t *testing.T) {
	var (
		mu     sync.Mutex
		orders []string
		sold   bool
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v3/exchangeInfo":
			fmt.Fprint(w, `{"timezone":"UTC","serverTime":1,"symbols":[
				{"symbol":"BNBUSDT","baseAsset":"BNB","quoteAsset":"USDT","quoteAssetPrecision":8,"quoteOrderQtyMarketAllowed":true,"filters":[{"filterType":"LOT_SIZE","minQty":"0.001","maxQty":"100000","stepSize":"0.001"},{"filterType":"NOTIONAL","minNotional":"1"}]},
				{"symbol":"BTCUSDT","baseAsset":"BTC","quoteAsset":"USDT","quoteAssetPrecision":8,"quoteOrderQtyMarketAllowed":true,"filters":[{"filterType":"LOT_SIZE","minQty":"0.001","maxQty":"100000","stepSize":"0.001"},{"filterType":"NOTIONAL","minNotional":"1"}]},
				{"symbol":"ETHUSDT","baseAsset":"ETH","quoteAsset":"USDT","quoteAssetPrecision":8,"quoteOrderQtyMarketAllowed":true,"filters":[{"filterType":"LOT_SIZE","minQty":"0.001","maxQty":"100000","stepSize":"0.001"},{"filterType":"NOTIONAL","minNotional":"1"}]},
				{"symbol":"SOLUSDT","baseAsset":"SOL","quoteAsset":"USDT","quoteAssetPrecision":8,"quoteOrderQtyMarketAllowed":true,"filters":[{"filterType":"LOT_SIZE","minQty":"0.001","maxQty":"100000","stepSize":"0.001"},{"filterType":"NOTIONAL","minNotional":"1"}]}
			]}`)
		case "/api/v3/ticker/bookTicker":
			fmt.Fprint(w, `[
				{"symbol":"BNBUSDT","bidPrice":"1","askPrice":"1"},
				{"symbol":"BTCUSDT","bidPrice":"1","askPrice":"1"},
				{"symbol":"ETHUSDT","bidPrice":"1","askPrice":"1"},
				{"symbol":"SOLUSDT","bidPrice":"1","askPrice":"1"}
			]`)
		case "/api/v3/account":
			mu.Lock()
			usdt := "100"
			if sold {
				usdt = "115"
			}
			mu.Unlock()
			fmt.Fprintf(w, `{"balances":[
				{"asset":"BNB","free":"30","locked":"0"},
				{"asset":"BTC","free":"30","locked":"0"},
				{"asset":"ETH","free":"20","locked":"0"},
				{"asset":"SOL","free":"10","locked":"0"},
				{"asset":"DOGE","free":"1000000","locked":"0"},
				{"asset":"USDT","free":"%s","locked":"0"}
			]}`, usdt)
		case "/api/v3/order":
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse order form: %v", err)
			}
			side := r.Form.Get("side")
			symbol := r.Form.Get("symbol")
			mu.Lock()
			orders = append(orders, side+" "+symbol)
			if side == string(binance.SideTypeSell) {
				sold = true
			}
			mu.Unlock()
			fmt.Fprintf(w, `{"symbol":"%s","orderId":1,"status":"FILLED","executedQty":"1","cummulativeQuoteQty":"5"}`, symbol)
		default:
			http.Error(w, "unexpected endpoint", http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := binance.NewClient("test", "test")
	client.BaseURL = server.URL
	client.HTTPClient = server.Client()
	spotClient := &Spot{Client: client}
	spotClient.Rebalance(config.BuyCryptoMap{
		"BNBUSDT": 15,
		"SOLUSDT": 15,
		"BTCUSDT": 35,
		"ETHUSDT": 25,
	})

	mu.Lock()
	defer mu.Unlock()
	if len(orders) != 4 {
		t.Fatalf("orders = %v, want one sell followed by three buys", orders)
	}
	if orders[0] != "SELL BNBUSDT" {
		t.Fatalf("first order = %q, want SELL BNBUSDT", orders[0])
	}
	for _, order := range orders[1:] {
		if !strings.HasPrefix(order, "BUY ") {
			t.Fatalf("orders = %v, found a non-buy after sell phase", orders)
		}
	}
}
