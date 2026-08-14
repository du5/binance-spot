package spot

import (
	"binance-spot/config"
	"binance-spot/tools"
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"

	"github.com/adshao/go-binance/v2"
)

var (
	API_KEY     = os.Getenv("API_KEY")
	SECRET_KEY  = os.Getenv("SECRET_KEY")
	MAX_RETRIES = config.Config.MaxRetries
)

func init() {
	if API_KEY == "" || SECRET_KEY == "" {
		log.Fatal("API_KEY or SECRET_KEY is not set in environment variables")
	}

}

type Spot struct {
	*binance.Client
}

func NewSpotClient() *Spot {
	return &Spot{binance.NewClient(API_KEY, SECRET_KEY)}
}

func (s *Spot) TestClient() {
	_, err := s.NewWalletBalanceService().QuoteAsset("USDT").Do(context.Background())
	if err != nil {
		log.Fatalf("Failed to connect to Binance API: %v", err)
	}
	log.Println("Successfully connected to Binance API")

	timeOffset, err := s.NewSetServerTimeService().Do(context.Background())
	if err != nil {
		log.Fatalf("Failed to synchronize server time: %v", err)
	}
	log.Printf("Synchronized server time with offset: %d ms", timeOffset)
}

func getSymbols(buyMap config.BuyCryptoMap) (keys []string, orders map[string]*order) {
	orders = make(map[string]*order)
	for symbol, amount := range buyMap {
		orders[symbol] = &order{symbol: symbol, amount: amount}
		keys = append(keys, symbol)
	}

	return keys, orders
}

// 获取交易对的最小变动单位
func (s *Spot) getSymbolsInfo(keys []string, orders map[string]*order) error {
	info, err := s.NewExchangeInfoService().Symbols(keys...).Do(context.Background())
	if err != nil {
		return fmt.Errorf("get exchange info: %w", err)
	}

	for _, symbol := range info.Symbols {
		o, ok := orders[symbol.Symbol]
		if !ok {
			continue
		}
		for _, filter := range symbol.Filters {
			filterType, _ := filter["filterType"].(string)
			switch filterType {
			case string(binance.SymbolFilterTypePriceFilter):
				if v, ok := filter["tickSize"].(string); ok {
					o.tickSize = v
				}
			case string(binance.SymbolFilterTypeLotSize):
				if v, ok := filter["stepSize"].(string); ok {
					o.lotSize = v
				}
			case string(binance.SymbolFilterTypeNotional):
				if v, ok := filter["minNotional"].(string); ok {
					o.minNotional = v
				}
			}
		}
	}
	return nil
}

// 获取交易对的最新买一价
func (s *Spot) getBidPrices(keys []string, orders map[string]*order) error {
	info, err := s.NewListBookTickersService().Symbols(keys...).Do(context.Background())
	if err != nil {
		return fmt.Errorf("get book tickers: %w", err)
	}
	for _, ticker := range info {
		if o, ok := orders[ticker.Symbol]; ok {
			o.bidPrice = ticker.BidPrice
		}
	}
	return nil
}

func (s *Spot) doByCrypto(o *order) {
	retries := 0.0
	for {
		// 下单逻辑
		roundedPrice, roundedQuantity := tools.RoundPriceAndQuantity(o.amount, o.BidPrice(), o.TickSize(), o.LotSize(), o.MinNotional(), retries)
		border, err := s.NewCreateOrderService().
			Symbol(o.symbol).
			Side(binance.SideTypeBuy).
			Type(binance.OrderTypeLimitMaker).
			Price(roundedPrice).
			Quantity(roundedQuantity).
			Do(context.Background())
		retries++

		if err != nil {
			log.Printf("Round %.0f: Failed to buy %s at price %s, quantity %s: %v", retries, o.symbol, roundedPrice, roundedQuantity, err)
		} else {
			log.Printf("Round %.0f: Successfully buy %s at price %s, quantity %s, order ID: %d", retries, o.symbol, roundedPrice, roundedQuantity, border.OrderID)
			break
		}

		if retries >= MAX_RETRIES {
			log.Printf("Max retries reached for %s", o.symbol)
			break
		}
	}
}

type order struct {
	symbol      string
	amount      float64
	tickSize    string
	lotSize     string
	bidPrice    string
	minNotional string
}

func (o order) MinNotional() float64 {
	return o.ParseFloat(o.minNotional)
}
func (o order) BidPrice() float64 {
	return o.ParseFloat(o.bidPrice)
}
func (o order) TickSize() float64 {
	return o.ParseFloat(o.tickSize)
}
func (o order) LotSize() float64 {
	return o.ParseFloat(o.lotSize)
}

func (o order) ParseFloat(fs string) float64 {
	f, _ := strconv.ParseFloat(fs, 64)
	return f
}

func (s *Spot) BuyCrypto(buyMap config.BuyCryptoMap) {
	keys, orders := getSymbols(buyMap)
	if err := s.getSymbolsInfo(keys, orders); err != nil {
		log.Printf("Skip buying: %v", err)
		return
	}
	if err := s.getBidPrices(keys, orders); err != nil {
		log.Printf("Skip buying: %v", err)
		return
	}

	var wg sync.WaitGroup
	for _, o := range orders {
		// 行情数据不完整时下单参数会产生除零/死循环,直接跳过
		if o.TickSize() <= 0 || o.LotSize() <= 0 || o.BidPrice() <= 0 {
			log.Printf("Skip %s: incomplete market data (tickSize=%q, lotSize=%q, bidPrice=%q)", o.symbol, o.tickSize, o.lotSize, o.bidPrice)
			continue
		}
		wg.Add(1)
		go func(o *order) {
			defer wg.Done()
			s.doByCrypto(o)
		}(o)
	}
	wg.Wait()
}
