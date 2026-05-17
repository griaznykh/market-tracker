package marketdata

import (
	"context"
	"time"
)

type (
	TradeSubscription struct {
		Tickers []string
	}

	Trade struct {
		Ticker string
		Price  float64
		Time   time.Time
	}

	MarketDataProvider interface {
		Start(ctx context.Context) error
		Close() error

		SubscribeTrades(
			ctx context.Context,
			req TradeSubscription,
		) (<-chan Trade, error)
	}
)
