package collector

import (
	"context"
	"errors"
	"fmt"
	"market-service/internal/marketdata"
	"market-service/internal/marketdata/broker"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type (
	Task struct {
		Provider string
		Tickers  []string
	}

	Config struct {
		Logger    *zap.Logger
		Providers map[string]marketdata.MarketDataProvider
		Tasks     []Task
	}

	MarketDataCollector struct {
		logger    *zap.Logger
		providers map[string]marketdata.MarketDataProvider
		tasks     []Task
		broker    *broker.Broker
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

	broker := broker.NewBroker()

	return &MarketDataCollector{
		logger:    config.Logger,
		providers: config.Providers,
		tasks:     config.Tasks,
		broker:    broker,
	}, nil
}

func (mdc *MarketDataCollector) Run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	// start tasks
	for _, task := range mdc.tasks {
		provider, ok := mdc.providers[task.Provider]
		if !ok {
			return fmt.Errorf("get provider error")
		}

		for _, ticker := range task.Tickers {
			subscription, err := provider.Subscribe(
				ctx,
				marketdata.SubscriptionRequest{
					Tickers: []string{ticker},
				},
			)
			if err != nil {
				return fmt.Errorf("subscribe to ticker: %w", err)
			}

			topic := fmt.Sprintf("%s:%s", task.Provider, ticker)

			g.Go(func() error {
				for trade := range subscription.Channel() {
					mdc.broker.Publish(topic, trade)
				}

				return nil
			})

			g.Go(func() error {
				sub := mdc.broker.Subscribe(topic)
				for trade := range sub {
					// TODO save to db
					fmt.Println(
						trade.Ticker,
						trade.Price,
					)
				}

				return nil
			})

		}
	}

	<-ctx.Done()

	mdc.broker.Close()

	return g.Wait()
}
