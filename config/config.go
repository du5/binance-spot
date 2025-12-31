package config

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

type BuyCryptoMap map[string]float64

type OrderList struct {
	TimeAt string       `json:"time_at"`
	Order  BuyCryptoMap `json:"order"`
}

type config struct {
	MaxRetries    float64     `json:"max_retries"`
	TickSizePower float64     `json:"tick_size_power"`
	OrderLists    []OrderList `json:"order_list"`
}

func (o OrderList) time() time.Time {
	parsedTime, err := time.Parse("15:04:05", o.TimeAt)
	if err != nil {
		panic(err)
	}
	return parsedTime
}

//go:embed config.json
var _config []byte

var Config config

func init() {
	err := json.Unmarshal(_config, &Config)
	if err != nil {
		panic(err)
	}

	totalOrderAmount := 0.0
	orderAmount := map[string]float64{}
	for _, orderList := range Config.OrderLists {
		tt := orderList.time() // Validate time format
		orders := []string{}
		for k, v := range orderList.Order {
			orders = append(orders, fmt.Sprintf("%s -> %.2f", k, v))
			orderAmount[k] += v
			totalOrderAmount += v
		}
		log.Printf("Order at %s: %s", tt.Format("15:04:05"), strings.Join(orders, ", "))
	}
	log.Printf("per day total order amount: %.2f", totalOrderAmount)
	for k, v := range orderAmount {
		log.Printf("per day total order amount: %s -> %.2f", k, v)
	}
}
