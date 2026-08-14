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

	Spot.TestClient()
}

func main() {
	log.Printf("NewYork time is %s now.", time.Now().Format("2006-01-02 15:04:05"))
	tc := time.NewTicker(time.Second)
	for {
		<-tc.C
		now := time.Now().Format("15:04:05")
		for _, orderList := range config.Config.OrderLists {
			if orderList.TimeAt == now {
				// 异步执行,避免下单重试阻塞 ticker 而错过后续时间点
				go Spot.BuyCrypto(orderList.Order)
			}
		}
	}
}
