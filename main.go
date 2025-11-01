package main

import (
	"binance-spot/spot"
	"log"
	"maps"
	"time"

	_ "github.com/joho/godotenv/autoload"
)

var (
	Spot = spot.NewSpotClient()

	postOrder = spot.BuyCryptoMap{
		// 主流币买入
		"BTCFDUSD": 24,
		"ETHFDUSD": 16,
		"BNBFDUSD": 8,
		"SOLFDUSD": 8,
		// 稳定资产买入
		"PAXGUSDT": 12,
	}
	preOrder = spot.BuyCryptoMap{
		// 山寨币买入
		"SUIFDUSD":  5,
		"LINKFDUSD": 5,
		"HBARFDUSD": 5,
		"AVAXFDUSD": 5,
		"XRPFDUSD":  5,
	}
)

func init() {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		log.Fatalf("Failed to load timezone America/New_York: %v", err)
	}
	time.Local = loc

	maps.Copy(preOrder, postOrder)
}

func main() {
	log.Printf("NewYork time is %s now.", time.Now().Format("2006-01-02 15:04:05"))
	tc := time.NewTicker(time.Second)
	for {
		<-tc.C
		now := time.Now()
		h, m, s := now.Hour(), now.Minute(), now.Second()
		if h == 13 && m == 0 && s == 0 {
			Spot.BuyCrypto(preOrder)
		}

		if h == 15 && m == 30 && s == 0 {
			Spot.BuyCrypto(postOrder)
		}
	}
}
