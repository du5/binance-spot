package spot

import (
	"testing"

	_ "github.com/joho/godotenv/autoload"
)

func TestBuyCrypto(t *testing.T) {
	spotClient := NewSpotClient()
	spotClient.BuyCrypto(BuyCryptoMap{
		// 山寨币买入
		"SUIFDUSD":  5,
		"LINKFDUSD": 5,
		"HBARFDUSD": 5,
		"AVAXFDUSD": 5,
		"XRPFDUSD":  5,
		// 主流币买入
		"BTCFDUSD": 24,
		"ETHFDUSD": 16,
		"BNBFDUSD": 8,
		"SOLFDUSD": 8,
		// 稳定资产买入
		"PAXGUSDT": 12,
	})
}
