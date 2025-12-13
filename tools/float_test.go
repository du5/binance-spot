package tools

import (
	"math"
	"testing"
)

func TestPrice(t *testing.T) {
	price := 100.0
	for n := 0.0; n < 15; n++ {
		tmpPrice := price * (1.0 - (math.Pow(1.05, n)-1.0)/100.0)
		t.Logf("n=%.0f, rawPrice=%.8f, flooredPrice=%.8f", n, tmpPrice, math.Floor(tmpPrice/0.01)*0.01)
	}

	/*
		=== RUN   TestPrice
		./binance-spot/tools/float_test.go:12: n=0, rawPrice=100.00000000, flooredPrice=100.00000000
		./binance-spot/tools/float_test.go:12: n=1, rawPrice=99.95000000, flooredPrice=99.95000000
		./binance-spot/tools/float_test.go:12: n=2, rawPrice=99.89750000, flooredPrice=99.89000000
		./binance-spot/tools/float_test.go:12: n=3, rawPrice=99.84237500, flooredPrice=99.84000000
		./binance-spot/tools/float_test.go:12: n=4, rawPrice=99.78449375, flooredPrice=99.78000000
		./binance-spot/tools/float_test.go:12: n=5, rawPrice=99.72371844, flooredPrice=99.72000000
		./binance-spot/tools/float_test.go:12: n=6, rawPrice=99.65990436, flooredPrice=99.65000000
		./binance-spot/tools/float_test.go:12: n=7, rawPrice=99.59289958, flooredPrice=99.59000000
		./binance-spot/tools/float_test.go:12: n=8, rawPrice=99.52254456, flooredPrice=99.52000000
		./binance-spot/tools/float_test.go:12: n=9, rawPrice=99.44867178, flooredPrice=99.44000000
		./binance-spot/tools/float_test.go:12: n=10, rawPrice=99.37110537, flooredPrice=99.37000000
		./binance-spot/tools/float_test.go:12: n=11, rawPrice=99.28966064, flooredPrice=99.28000000
		./binance-spot/tools/float_test.go:12: n=12, rawPrice=99.20414367, flooredPrice=99.20000000
		./binance-spot/tools/float_test.go:12: n=13, rawPrice=99.11435086, flooredPrice=99.11000000
		./binance-spot/tools/float_test.go:12: n=14, rawPrice=99.02006840, flooredPrice=99.02000000
	*/
}
