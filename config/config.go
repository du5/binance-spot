package config

import (
	_ "embed"
	"encoding/json"
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
	parsedTime, err := time.Parse("15:04:00", o.TimeAt)
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

	for _, orderList := range Config.OrderLists {
		orderList.time() // Validate time format
	}
}
