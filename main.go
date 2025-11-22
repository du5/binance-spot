package main

import (
	"binance-spot/config"
	"binance-spot/spot"
	"log"
	"time"

	_ "github.com/joho/godotenv/autoload"
)

var (
	Spot = spot.NewSpotClient()
)

func init() {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		log.Fatalf("Failed to load timezone America/New_York: %v", err)
	}
	time.Local = loc
}

func main() {
	log.Printf("NewYork time is %s now.", time.Now().Format("2006-01-02 15:04:05"))
	tc := time.NewTicker(time.Second)
	for {
		<-tc.C
		now := time.Now()
		h, m, s := now.Hour(), now.Minute(), now.Second()
		if h == 13 && m == 0 && s == 0 {
			Spot.BuyCrypto(config.Config.PreOrder)
		}

		if h == 15 && m == 30 && s == 0 {
			Spot.BuyCrypto(config.Config.PostOrder)
		}
	}
}
