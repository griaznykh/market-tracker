package marketdata

import (
	"context"
	"time"
)

type (
	SubscriptionRequest struct {
		Provider string
		Tickers  []string
	}

	Data struct {
		Provider string
		Ticker   string
		Price    float64
		Time     time.Time
	}

	MarketDataProvider interface {
		Subscribe(ctx context.Context, req SubscriptionRequest) (*Subscription, error)
	}
)
