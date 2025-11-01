module binance-spot

go 1.25.3

require (
	github.com/adshao/go-binance/v2 v2.8.7
	github.com/joho/godotenv v1.5.1
)

require (
	github.com/bitly/go-simplejson v0.5.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/jpillora/backoff v1.0.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
)

replace github.com/adshao/go-binance/v2 => github.com/du5/go-binance/v2 v2.8.8-0.20251031094802-af4bf3fa4ca1
