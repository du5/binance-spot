package tools

import (
	"fmt"
	"math"
	"strconv"
)

// 根据购买总价amount，当前买一价price，价格最小变动单位tickSize，数量最小变动单位lotSize，和重试次数n，计算出调整后的价格和数量
func RoundPriceAndQuantity(amount float64, price, tickSize, lotSize string, n float64) (string, string) {
	priceFloat, _ := strconv.ParseFloat(price, 64)
	tickSizeFloat, _ := strconv.ParseFloat(tickSize, 64)
	lotSizeFloat, _ := strconv.ParseFloat(lotSize, 64)

	precision := int(math.Round(-math.Log10(tickSizeFloat)))       // 价格精度
	lotSizePrecision := int(math.Round(-math.Log10(lotSizeFloat))) // 数量精度

	result := priceFloat - tickSizeFloat*n   // 每次重试降低价格
	quantity := amount/result + lotSizeFloat // 每次多买一点以防止不足 5u 订单被拒绝

	format := fmt.Sprintf("%%.%df", precision)                // 根据精度格式化字符串
	quantityFormat := fmt.Sprintf("%%.%df", lotSizePrecision) // ...

	return fmt.Sprintf(format, result), fmt.Sprintf(quantityFormat, quantity)
}
