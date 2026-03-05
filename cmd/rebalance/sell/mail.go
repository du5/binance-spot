package main

import (
	"binance-spot/spot"
	"context"
	"log"
	"time"

	"github.com/adshao/go-binance/v2"
)

func init() {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		log.Fatalf("Failed to load timezone America/New_York: %v", err)
	}
	time.Local = loc

	Spot.TestClient()
}

var (
	Spot = spot.NewSpotClient()
)

func main() {
	for {
		<-time.After(50 * time.Millisecond)
		if time.Now().Unix() >= 1772715599 {
			_, err := Spot.NewCreateOrderService().
				Symbol("OPNUSDT").
				Side(binance.SideTypeBuy).
				Type(binance.OrderTypeMarket).
				Quantity("146.5").Do(context.Background())
			if err != nil {
				log.Printf("Failed to create order: %v", err)
			} else {
				log.Println("Order created successfully")
				break
			}
		}
	}
}
