package spot

import (
	"binance-spot/config"
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/adshao/go-binance/v2"
)

const rebalanceQuoteAsset = "USDT"

type rebalanceAsset struct {
	symbol                     string
	baseAsset                  string
	targetWeight               float64
	free                       float64
	locked                     float64
	bid                        float64
	ask                        float64
	lotStep                    float64
	lotPrecision               int
	minQuantity                float64
	maxQuantity                float64
	minNotional                float64
	quotePrecision             int
	quoteOrderQtyMarketAllowed bool
}

type rebalancePlan struct {
	asset       *rebalanceAsset
	quoteAmount float64
}

// Rebalance adjusts only the assets represented by targets. The target values
// are relative weights (the summed order_list amounts), not amounts to trade.
func (s *Spot) Rebalance(targets config.BuyCryptoMap) {
	s.tradeMu.Lock()
	defer s.tradeMu.Unlock()

	assets, usdtBefore, err := s.loadRebalanceAssets(targets)
	if err != nil {
		log.Printf("Skip portfolio rebalance: %v", err)
		return
	}

	sells, buys, totalValue := buildRebalancePlan(assets)
	if totalValue <= 0 {
		log.Printf("Skip portfolio rebalance: configured crypto holdings have no value")
		return
	}

	log.Printf("Portfolio rebalance started: configured crypto value %.8f %s (USDT excluded)", totalValue, rebalanceQuoteAsset)
	for _, asset := range assets {
		currentValue := (asset.free + asset.locked) * asset.bid
		log.Printf("Position %s: value %.8f %s, current %.4f%%, target %.4f%%", asset.baseAsset, currentValue, rebalanceQuoteAsset, currentValue/totalValue*100, asset.targetWeight/totalTargetWeight(assets)*100)
	}

	// Complete the entire sell phase before calculating how much USDT is
	// actually available for the buy phase.
	for _, plan := range sells {
		s.executeRebalanceSell(plan)
	}

	usdtAfter, err := s.getFreeBalance(rebalanceQuoteAsset)
	if err != nil {
		log.Printf("Skip portfolio rebalance buys: get post-sell %s balance: %v", rebalanceQuoteAsset, err)
		return
	}
	buyBudget := math.Max(0, usdtAfter-usdtBefore)
	totalBuyNeed := totalPlanAmount(buys)
	if buyBudget > totalBuyNeed {
		buyBudget = totalBuyNeed
	}
	if buyBudget <= 0 || totalBuyNeed <= 0 {
		log.Printf("Portfolio rebalance finished after sells: no new %s proceeds available for buys", rebalanceQuoteAsset)
		return
	}

	for _, plan := range buys {
		allocated := buyBudget * plan.quoteAmount / totalBuyNeed
		s.executeRebalanceBuy(plan.asset, allocated)
	}
	log.Printf("Portfolio rebalance finished: buy budget %.8f %s", buyBudget, rebalanceQuoteAsset)
}

func (s *Spot) loadRebalanceAssets(targets config.BuyCryptoMap) ([]*rebalanceAsset, float64, error) {
	if len(targets) == 0 {
		return nil, 0, fmt.Errorf("order_list does not contain any symbols")
	}

	keys := make([]string, 0, len(targets))
	for symbol, weight := range targets {
		if weight <= 0 {
			return nil, 0, fmt.Errorf("target weight for %s must be positive", symbol)
		}
		keys = append(keys, symbol)
	}
	sort.Strings(keys)

	exchangeInfo, err := s.NewExchangeInfoService().Symbols(keys...).Do(context.Background())
	if err != nil {
		return nil, 0, fmt.Errorf("get exchange info: %w", err)
	}

	assetsBySymbol := make(map[string]*rebalanceAsset, len(keys))
	for _, symbol := range exchangeInfo.Symbols {
		weight, ok := targets[symbol.Symbol]
		if !ok {
			continue
		}
		if symbol.QuoteAsset != rebalanceQuoteAsset {
			return nil, 0, fmt.Errorf("%s uses quote asset %s; rebalancing requires %s", symbol.Symbol, symbol.QuoteAsset, rebalanceQuoteAsset)
		}

		asset := &rebalanceAsset{
			symbol:                     symbol.Symbol,
			baseAsset:                  symbol.BaseAsset,
			targetWeight:               weight,
			quotePrecision:             symbol.QuoteAssetPrecision,
			quoteOrderQtyMarketAllowed: symbol.QuoteOrderQtyMarketAllowed,
		}
		setRebalanceFilters(asset, symbol.Filters)
		if asset.lotStep <= 0 {
			return nil, 0, fmt.Errorf("%s has no usable market lot step", symbol.Symbol)
		}
		assetsBySymbol[symbol.Symbol] = asset
	}

	assets := make([]*rebalanceAsset, 0, len(keys))
	for _, symbol := range keys {
		asset, ok := assetsBySymbol[symbol]
		if !ok {
			return nil, 0, fmt.Errorf("exchange info did not return configured symbol %s", symbol)
		}
		assets = append(assets, asset)
	}

	tickers, err := s.NewListBookTickersService().Symbols(keys...).Do(context.Background())
	if err != nil {
		return nil, 0, fmt.Errorf("get book tickers: %w", err)
	}
	for _, ticker := range tickers {
		asset, ok := assetsBySymbol[ticker.Symbol]
		if !ok {
			continue
		}
		asset.bid, err = parsePositiveFloat(ticker.BidPrice)
		if err != nil {
			return nil, 0, fmt.Errorf("parse %s bid price: %w", ticker.Symbol, err)
		}
		asset.ask, err = parsePositiveFloat(ticker.AskPrice)
		if err != nil {
			return nil, 0, fmt.Errorf("parse %s ask price: %w", ticker.Symbol, err)
		}
	}

	account, err := s.NewGetAccountService().Do(context.Background())
	if err != nil {
		return nil, 0, fmt.Errorf("get account balances: %w", err)
	}
	balances := make(map[string]binance.Balance, len(account.Balances))
	for _, balance := range account.Balances {
		balances[balance.Asset] = balance
	}

	for _, asset := range assets {
		if asset.bid <= 0 || asset.ask <= 0 {
			return nil, 0, fmt.Errorf("book ticker did not return usable prices for %s", asset.symbol)
		}
		balance := balances[asset.baseAsset]
		asset.free, err = parseNonNegativeFloat(balance.Free)
		if err != nil {
			return nil, 0, fmt.Errorf("parse %s free balance: %w", asset.baseAsset, err)
		}
		asset.locked, err = parseNonNegativeFloat(balance.Locked)
		if err != nil {
			return nil, 0, fmt.Errorf("parse %s locked balance: %w", asset.baseAsset, err)
		}
	}

	usdtBalance := balances[rebalanceQuoteAsset]
	usdtFree, err := parseNonNegativeFloat(usdtBalance.Free)
	if err != nil {
		return nil, 0, fmt.Errorf("parse %s free balance: %w", rebalanceQuoteAsset, err)
	}
	return assets, usdtFree, nil
}

func setRebalanceFilters(asset *rebalanceAsset, filters []map[string]any) {
	var marketStep, marketMin, marketMax float64
	for _, filter := range filters {
		filterType, _ := filter["filterType"].(string)
		switch filterType {
		case string(binance.SymbolFilterTypeLotSize):
			asset.lotStep = filterFloat(filter, "stepSize")
			asset.minQuantity = filterFloat(filter, "minQty")
			asset.maxQuantity = filterFloat(filter, "maxQty")
		case string(binance.SymbolFilterTypeMarketLotSize):
			marketStep = filterFloat(filter, "stepSize")
			marketMin = filterFloat(filter, "minQty")
			marketMax = filterFloat(filter, "maxQty")
		case string(binance.SymbolFilterTypeNotional), string(binance.SymbolFilterTypeMinNotional):
			if minNotional := filterFloat(filter, "minNotional"); minNotional > asset.minNotional {
				asset.minNotional = minNotional
			}
		}
	}
	if marketStep > 0 {
		asset.lotStep = marketStep
		asset.minQuantity = marketMin
		asset.maxQuantity = marketMax
	}
	asset.lotPrecision = incrementPrecision(asset.lotStep)
}

func buildRebalancePlan(assets []*rebalanceAsset) (sells, buys []rebalancePlan, totalValue float64) {
	weightTotal := totalTargetWeight(assets)
	for _, asset := range assets {
		totalValue += (asset.free + asset.locked) * asset.bid
	}
	if totalValue <= 0 || weightTotal <= 0 {
		return nil, nil, totalValue
	}

	for _, asset := range assets {
		currentValue := (asset.free + asset.locked) * asset.bid
		targetValue := totalValue * asset.targetWeight / weightTotal
		difference := currentValue - targetValue
		if difference > 0 {
			sells = append(sells, rebalancePlan{asset: asset, quoteAmount: difference})
		} else if difference < 0 {
			buys = append(buys, rebalancePlan{asset: asset, quoteAmount: -difference})
		}
	}
	return sells, buys, totalValue
}

func (s *Spot) executeRebalanceSell(plan rebalancePlan) {
	asset := plan.asset
	quantity := plan.quoteAmount / asset.bid
	if quantity > asset.free {
		quantity = asset.free
	}
	if asset.maxQuantity > 0 && quantity > asset.maxQuantity {
		quantity = asset.maxQuantity
	}
	quantity = floorToIncrement(quantity, asset.lotStep)
	notional := quantity * asset.bid
	if quantity <= 0 || quantity < asset.minQuantity || notional < asset.minNotional {
		log.Printf("Skip rebalance sell %s: quantity %.12f, value %.8f %s is below exchange minimum", asset.symbol, quantity, notional, rebalanceQuoteAsset)
		return
	}

	response, err := s.NewCreateOrderService().
		Symbol(asset.symbol).
		Side(binance.SideTypeSell).
		Type(binance.OrderTypeMarket).
		Quantity(strconv.FormatFloat(quantity, 'f', asset.lotPrecision, 64)).
		NewOrderRespType(binance.NewOrderRespTypeFULL).
		Do(context.Background())
	if err != nil {
		log.Printf("Failed to rebalance sell %s quantity %.12f: %v", asset.symbol, quantity, err)
		return
	}
	log.Printf("Rebalance sold %s quantity %s, quote received %s, order ID: %d", asset.symbol, response.ExecutedQuantity, response.CummulativeQuoteQuantity, response.OrderID)
}

func (s *Spot) executeRebalanceBuy(asset *rebalanceAsset, quoteAmount float64) {
	quoteAmount = floorToPrecision(quoteAmount, asset.quotePrecision)
	if quoteAmount <= 0 || quoteAmount < asset.minNotional {
		log.Printf("Skip rebalance buy %s: value %.8f %s is below exchange minimum", asset.symbol, quoteAmount, rebalanceQuoteAsset)
		return
	}

	service := s.NewCreateOrderService().
		Symbol(asset.symbol).
		Side(binance.SideTypeBuy).
		Type(binance.OrderTypeMarket).
		NewOrderRespType(binance.NewOrderRespTypeFULL)
	if asset.quoteOrderQtyMarketAllowed {
		service = service.QuoteOrderQty(strconv.FormatFloat(quoteAmount, 'f', asset.quotePrecision, 64))
	} else {
		quantity := floorToIncrement(quoteAmount/asset.ask, asset.lotStep)
		if asset.maxQuantity > 0 && quantity > asset.maxQuantity {
			quantity = floorToIncrement(asset.maxQuantity, asset.lotStep)
		}
		if quantity <= 0 || quantity < asset.minQuantity || quantity*asset.ask < asset.minNotional {
			log.Printf("Skip rebalance buy %s: calculated quantity %.12f is below exchange minimum", asset.symbol, quantity)
			return
		}
		service = service.Quantity(strconv.FormatFloat(quantity, 'f', asset.lotPrecision, 64))
	}

	response, err := service.Do(context.Background())
	if err != nil {
		log.Printf("Failed to rebalance buy %s value %.8f %s: %v", asset.symbol, quoteAmount, rebalanceQuoteAsset, err)
		return
	}
	log.Printf("Rebalance bought %s quantity %s, quote spent %s, order ID: %d", asset.symbol, response.ExecutedQuantity, response.CummulativeQuoteQuantity, response.OrderID)
}

func (s *Spot) getFreeBalance(asset string) (float64, error) {
	account, err := s.NewGetAccountService().Do(context.Background())
	if err != nil {
		return 0, err
	}
	for _, balance := range account.Balances {
		if balance.Asset == asset {
			return parseNonNegativeFloat(balance.Free)
		}
	}
	return 0, nil
}

func totalTargetWeight(assets []*rebalanceAsset) float64 {
	var total float64
	for _, asset := range assets {
		total += asset.targetWeight
	}
	return total
}

func totalPlanAmount(plans []rebalancePlan) float64 {
	var total float64
	for _, plan := range plans {
		total += plan.quoteAmount
	}
	return total
}

func filterFloat(filter map[string]any, key string) float64 {
	value, _ := filter[key].(string)
	parsed, _ := strconv.ParseFloat(value, 64)
	return parsed
}

func parsePositiveFloat(value string) (float64, error) {
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("invalid positive number %q", value)
	}
	return parsed, nil
}

func parseNonNegativeFloat(value string) (float64, error) {
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil || parsed < 0 {
		return 0, fmt.Errorf("invalid non-negative number %q", value)
	}
	return parsed, nil
}

func incrementPrecision(increment float64) int {
	formatted := strconv.FormatFloat(increment, 'f', -1, 64)
	if dot := strings.IndexByte(formatted, '.'); dot >= 0 {
		return len(strings.TrimRight(formatted[dot+1:], "0"))
	}
	return 0
}

func floorToIncrement(value, increment float64) float64 {
	if increment <= 0 {
		return 0
	}
	return math.Floor((value+increment*1e-9)/increment) * increment
}

func floorToPrecision(value float64, precision int) float64 {
	if precision < 0 {
		return 0
	}
	factor := math.Pow10(precision)
	return math.Floor(value*factor) / factor
}
