package spot

import (
	"binance-spot/config"
	"testing"

	_ "github.com/joho/godotenv/autoload"
)

func TestBuyCrypto(t *testing.T) {
	spotClient := NewSpotClient()
	spotClient.BuyCrypto(config.Config.PreOrder)
}
