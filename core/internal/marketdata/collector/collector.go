package collector

import (
	"context"
	"errors"
	"fmt"
	"market-service/internal/marketdata"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type (
	Config struct {
		Logger    *zap.Logger
		Providers map[string]marketdata.MarketDataProvider
	}

	MarketDataCollector struct {
		logger    *zap.Logger
		providers map[string]marketdata.MarketDataProvider
	}
)

func (c *Config) validate() error {
	switch {
	case c.Logger == nil:
		return errors.New("logger settings may not be nil")

	case c.Providers == nil:
		return errors.New("providers settings may not be nil")

	default:
		return nil
	}
}

func New(config Config) (*MarketDataCollector, error) {
	if err := config.validate(); err != nil {
		return nil, err
	}

	return &MarketDataCollector{
		logger:    config.Logger,
		providers: config.Providers,
	}, nil
}

func (mdc *MarketDataCollector) Run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	provider, ok := mdc.providers["tbank"]
	if !ok {
		return fmt.Errorf("get tbank provider error")
	}

	subscription, err := provider.Subscribe(
		ctx,
		marketdata.SubscriptionRequest{
			Tickers: []string{
				"SBER_TQBR",
			},
		},
	)
	if err != nil {
		return fmt.Errorf("subscribe to ticker: %w", err)
	}

	time.Sleep(10 * time.Second)

	provider.Unsubscribe(subscription)

	for trade := range subscription.Channel {
		fmt.Println(
			trade.Ticker,
			trade.Price,
		)
	}

	<-ctx.Done()

	return g.Wait()
}
