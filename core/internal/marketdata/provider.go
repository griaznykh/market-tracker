package marketdata

import (
	"context"
	"sync"
	"time"
)

type (
	SubscriptionRequest struct {
		Provider string
		Tickers  []string
	}

	UnsubscriptionRequest struct {
		Provider string
	}

	Data struct {
		Provider string
		Ticker   string
		Price    float64
		Time     time.Time
	}

	Subscription struct {
		Channel <-chan Data
		Cancel  func()
		Wg      *sync.WaitGroup
	}

	MarketDataProvider interface {
		Subscribe(ctx context.Context, req SubscriptionRequest) (*Subscription, error)
		Unsubscribe(s *Subscription)
	}
)
