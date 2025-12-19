package spot

import (
	"binance-spot/config"
	"binance-spot/tools"
	"context"
	"log"
	"os"
	"strconv"

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
func (s *Spot) getSymbolsInfo(keys []string, orders map[string]*order) {
	info, err := s.NewExchangeInfoService().Symbols(keys...).Do(context.Background())
	if err != nil {
		log.Printf("Failed to get exchange info: %v", err)
	}

	for _, symbol := range info.Symbols {
		for _, filter := range symbol.Filters {
			switch filter["filterType"].(binance.SymbolFilterType) {
			case binance.SymbolFilterTypePriceFilter:
				if i, ok := filter["tickSize"]; ok {
					orders[symbol.Symbol].tickSize = i.(string)
				}
			case binance.SymbolFilterTypeLotSize:
				if i, ok := filter["stepSize"]; ok {
					orders[symbol.Symbol].lotSize = i.(string)
				}
			case binance.SymbolFilterTypeNotional:
				if minNotional, ok := filter["minNotional"]; ok {
					orders[symbol.Symbol].minNotional = minNotional.(string)
				}
			}
		}
	}
}

// 获取交易对的最新买一价
func (s *Spot) getBidPrices(keys []string, orders map[string]*order) {
	info, err := s.NewListBookTickersService().Symbols(keys...).Do(context.Background())
	if err != nil {
		log.Printf("Failed to get book tickers: %v", err)
	}
	for _, ticker := range info {
		orders[ticker.Symbol].bidPrice = ticker.BidPrice
	}
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
	var (
		keys, orders = getSymbols(buyMap)
	)
	s.getSymbolsInfo(keys, orders)
	s.getBidPrices(keys, orders)

	c := make(chan struct{}, len(buyMap))
	for _, o := range orders {
		go func(_order *order) {
			s.doByCrypto(_order)
			c <- struct{}{}
		}(o)
	}

	for i := 0; i < len(buyMap); i++ {
		<-c
	}
}
