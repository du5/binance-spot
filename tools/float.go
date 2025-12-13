package tools

import (
	"binance-spot/config"
	"fmt"
	"math"
)

// 根据购买总价amount，当前买一价price，价格最小变动单位tickSize，数量最小变动单位lotSize，和重试次数n，计算出调整后的价格和数量
func RoundPriceAndQuantity(amount, price, tickSize, lotSize, MinNotional, n float64) (roundedPrice, roundedQuantity string) {
	precision := int(math.Round(-math.Log10(tickSize)))       // 价格精度
	lotSizePrecision := int(math.Round(-math.Log10(lotSize))) // 数量精度

	rawPrice := price * (1.0 - (math.Pow(config.Config.TickSizePower, n)-1.0)/100.0)
	flooredPrice := math.Floor(rawPrice/tickSize) * tickSize

	if flooredPrice <= 0 {
		flooredPrice = lotSize
	}

	priceFormat := fmt.Sprintf("%%.%df", precision)           // 根据精度格式化字符串
	quantityFormat := fmt.Sprintf("%%.%df", lotSizePrecision) // ...

	rawQty := amount / flooredPrice
	qty := math.Floor(rawQty/lotSize) * lotSize
	if qty <= 0 {
		qty = lotSize
	}

	for flooredPrice*qty < MinNotional {
		qty += lotSize
	}

	roundedPrice = fmt.Sprintf(priceFormat, flooredPrice)
	roundedQuantity = fmt.Sprintf(quantityFormat, qty)
	return
}
