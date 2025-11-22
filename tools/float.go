package tools

import (
	"fmt"
	"math"
	"strconv"
)

// 根据购买总价amount，当前买一价price，价格最小变动单位tickSize，数量最小变动单位lotSize，和重试次数n，计算出调整后的价格和数量
func RoundPriceAndQuantity(amount float64, price, tickSize, lotSize string, n float64) (roundedPrice, roundedQuantity string) {
	priceFloat, _ := strconv.ParseFloat(price, 64)
	tickSizeFloat, _ := strconv.ParseFloat(tickSize, 64)
	lotSizeFloat, _ := strconv.ParseFloat(lotSize, 64)

	precision := int(math.Round(-math.Log10(tickSizeFloat)))       // 价格精度
	lotSizePrecision := int(math.Round(-math.Log10(lotSizeFloat))) // 数量精度

	rawPrice := priceFloat - n*tickSizeFloat
	flooredPrice := math.Floor(rawPrice/tickSizeFloat) * tickSizeFloat

	if flooredPrice <= 0 {
		flooredPrice = tickSizeFloat
	}

	priceFormat := fmt.Sprintf("%%.%df", precision)           // 根据精度格式化字符串
	quantityFormat := fmt.Sprintf("%%.%df", lotSizePrecision) // ...

	rawQty := amount / flooredPrice
	qty := math.Floor(rawQty/lotSizeFloat) * lotSizeFloat
	if qty <= 0 {
		qty = lotSizeFloat
	}

	for flooredPrice*qty < 5 {
		qty += lotSizeFloat
	}

	roundedPrice = fmt.Sprintf(priceFormat, flooredPrice)
	roundedQuantity = fmt.Sprintf(quantityFormat, qty)
	return
}
