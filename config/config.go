package config

import (
	_ "embed"
	"encoding/json"
)

type BuyCryptoMap map[string]float64

type config struct {
	PostOrder     BuyCryptoMap `json:"post_order"`
	PreOrder      BuyCryptoMap `json:"pre_order"`
	MaxRetries    float64      `json:"max_retries"`
	TickSizePower float64      `json:"tick_size_power"`
}

//go:embed config.json
var _config []byte

var Config config

func init() {
	err := json.Unmarshal(_config, &Config)
	if err != nil {
		panic(err)
	}
}
