package collector

import (
	"context"
	"errors"
	"fmt"
	"market-service/internal/marketdata"
	"market-service/internal/marketdata/broker"
	"sync"

	"go.uber.org/zap"
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
		mu            sync.RWMutex
		logger        *zap.Logger
		providers     map[string]marketdata.MarketDataProvider
		subscriptions map[string]*marketdata.Subscription
		tasks         []Task
		broker        *broker.Broker
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
		logger:        config.Logger,
		providers:     config.Providers,
		tasks:         config.Tasks,
		broker:        broker,
		subscriptions: make(map[string]*marketdata.Subscription),
	}, nil
}

func (c *MarketDataCollector) Run(ctx context.Context) error {
	for _, task := range c.tasks {
		for _, ticker := range task.Tickers {
			ch, err := c.Subscribe(
				ctx,
				task.Provider,
				ticker,
			)
			if err != nil {
				return err
			}

			go func(ctx context.Context, ch <-chan marketdata.Data) {
				for {
					select {
					case <-ctx.Done():
						return
					case trade := <-ch:
						fmt.Println(trade.Ticker, trade.Price)
					}
				}
			}(ctx, ch)
		}
	}

	go func(ctx context.Context) {
		events := c.broker.Event()

		for {
			select {
			case <-ctx.Done():
				return
			case event := <-events:
				switch event.Event {
				case broker.Subscribe:
					provider, ok := c.providers[event.Provider]
					if !ok {
						c.logger.Error("no provider to subscribe error")
						continue
					}

					subscription, err := provider.Subscribe(ctx, marketdata.SubscriptionRequest{Tickers: []string{event.Ticker}})
					if err != nil {
						c.logger.Error("provider subscription error", zap.Error(err))
						continue
					}

					key := fmt.Sprintf("%s:%s", event.Provider, event.Ticker)

					c.mu.Lock()
					c.subscriptions[key] = subscription
					c.mu.Unlock()

					go func(sub *marketdata.Subscription) {
						select {
						case <-sub.Done():
							return
						case data := <-sub.Channel():
							c.broker.Publish(data)
						}
					}(subscription)

				case broker.Unsubscribe:
					key := fmt.Sprintf("%s:%s", event.Provider, event.Ticker)

					c.mu.Lock()
					subscription, ok := c.subscriptions[key]
					if ok {
						subscription.Close()
						delete(c.subscriptions, key)
					}
					c.mu.Unlock()

					if !ok {
						c.logger.Error("no subscription to unsubscribe error")
					}
				}
			}
		}
	}(ctx)

	<-ctx.Done()

	c.broker.Close()

	return nil
}

func (c *MarketDataCollector) Subscribe(
	ctx context.Context,
	provider string,
	ticker string,
) (<-chan marketdata.Data, error) {
	_, ok := c.providers[provider]
	if !ok {
		return nil, fmt.Errorf("provider not found")
	}

	return c.broker.Subscribe(provider, ticker), nil
}

func (c *MarketDataCollector) Unsubscribe(
	provider string,
	ticker string,
	ch <-chan marketdata.Data,
) {
	c.broker.Unsubscribe(provider, ticker, ch)
}
