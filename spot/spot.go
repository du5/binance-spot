package spot

import (
	"binance-spot/tools"
	"context"
	"log"
	"os"

	"github.com/adshao/go-binance/v2"
)

var (
	API_KEY     = os.Getenv("API_KEY")
	SECRET_KEY  = os.Getenv("SECRET_KEY")
	MAX_RETRIES = 15.0
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

type BuyCryptoMap map[string]float64

func getSymbols(buyMap BuyCryptoMap) []string {
	symbols := make([]string, 0, len(buyMap))
	for k := range buyMap {
		symbols = append(symbols, k)
	}

	return symbols
}

// 获取交易对的最小变动单位
func (s *Spot) getSymbolsInfo(symbols ...string) (map[string]string, map[string]string) {
	tickSizes := map[string]string{}
	lotSizes := map[string]string{}
	info, err := s.NewExchangeInfoService().Symbols(symbols...).Do(context.Background())
	if err != nil {
		log.Printf("Failed to get exchange info: %v", err)
	}

	for _, symbol := range info.Symbols {
		for _, filter := range symbol.Filters {
			switch filter["filterType"].(string) {
			case string(binance.SymbolFilterTypePriceFilter):
				if i, ok := filter["tickSize"]; ok {
					tickSizes[symbol.Symbol] = i.(string)
				}
			case string(binance.SymbolFilterTypeLotSize):
				if i, ok := filter["stepSize"]; ok {
					lotSizes[symbol.Symbol] = i.(string)
				}
			}
		}
	}
	return tickSizes, lotSizes
}

// 获取交易对的最新买一价
func (s *Spot) getBidPrices(symbols ...string) map[string]string {
	bidPrices := map[string]string{}
	info, err := s.NewListBookTickersService().Symbols(symbols...).Do(context.Background())
	if err != nil {
		log.Printf("Failed to get book tickers: %v", err)
	}
	for _, ticker := range info {
		bidPrices[ticker.Symbol] = ticker.BidPrice
	}
	return bidPrices
}

func (s *Spot) doByCrypto(symbol, tickSize, lotSize, bidPrice string, amount float64) {
	retries := 0.0
	for {
		// 下单逻辑
		roundedPrice, roundedQuantity := tools.RoundPriceAndQuantity(amount, bidPrice, tickSize, lotSize, retries)
		order, err := s.NewCreateOrderService().
			Symbol(symbol).
			Side(binance.SideTypeBuy).
			Type(binance.OrderTypeLimitMaker).
			Price(roundedPrice).
			Quantity(roundedQuantity).
			Do(context.Background())
		retries++

		if err != nil {
			log.Printf("Round %.0f: Failed to buy %s at price %s, quantity %s: %v", retries, symbol, roundedPrice, roundedQuantity, err)
		} else {
			log.Printf("Round %.0f: Successfully buy %s at price %s, quantity %s, order ID: %d", retries, symbol, roundedPrice, roundedQuantity, order.OrderID)
			break
		}

		if retries >= MAX_RETRIES {
			log.Printf("Max retries reached for %s", symbol)
			break
		}
	}
}

func (s *Spot) BuyCrypto(buyMap BuyCryptoMap) {
	var (
		symbols             = getSymbols(buyMap)
		tickSizes, lotSizes = s.getSymbolsInfo(symbols...)
		bidPrices           = s.getBidPrices(symbols...)
	)

	c := make(chan struct{}, len(buyMap))
	for symbol, amount := range buyMap {
		go func(symbol string, amount float64) {
			s.doByCrypto(symbol, tickSizes[symbol], lotSizes[symbol], bidPrices[symbol], amount)
			c <- struct{}{}
		}(symbol, amount)
	}

	for i := 0; i < len(buyMap); i++ {
		<-c
	}
}
