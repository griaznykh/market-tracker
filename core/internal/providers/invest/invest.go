package invest

import (
	"context"
	"market-service/internal/marketdata"
	"strings"
	"sync"

	"go.uber.org/zap"
	"opensource.tbank.ru/invest/invest-go/investgo"
)

type InvestProvider struct {
	logger   *zap.SugaredLogger
	client   *investgo.Client
	mdClient *investgo.MarketDataStreamClient
}

type InvestProviderConfig struct {
	Token    string
	Endpoint string
}

func NewInvestProvider(
	ctx context.Context,
	config InvestProviderConfig,
	logger *zap.SugaredLogger,
) (*InvestProvider, error) {
	investConfig := investgo.Config{
		EndPoint:           config.Endpoint,
		Token:              config.Token,
		InsecureSkipVerify: true,
	}

	client, err := investgo.NewClient(ctx, investConfig, logger)
	if err != nil {
		return nil, err
	}

	MDClient := client.NewMarketDataStreamClient()

	return &InvestProvider{
		client:   client,
		mdClient: MDClient,
		logger:   logger,
	}, nil
}

func (p *InvestProvider) Subscribe(ctx context.Context, req marketdata.SubscriptionRequest) (*marketdata.Subscription, error) {
	var wg sync.WaitGroup

	MDStream, err := p.mdClient.MarketDataStream()
	if err != nil {
		return nil, err
	}

	src, err := MDStream.SubscribeLastPrice(req.Tickers)
	if err != nil {
		return nil, err
	}

	out := make(chan marketdata.Data)

	wg.Go(func() {
		err := MDStream.Listen()
		if err != nil {
			p.logger.Errorf("market data stream error: %v", err)
		}
	})

	wg.Go(func() {
		defer close(out)

		for {
			select {
			case <-ctx.Done():
				p.logger.Infof("stop listening %s channels", strings.Join(req.Tickers, ", "))
				return
			case data, ok := <-src:
				if !ok {
					return
				}

				out <- marketdata.Data{
					Provider: "tbank",
					Ticker:   data.GetTicker(),
					Price:    data.GetPrice().ToFloat(),
					Time:     data.GetTime().AsTime(),
				}
			}
		}

	})

	return &marketdata.Subscription{
		Channel: out,
		Cancel:  MDStream.Stop,
		Wg:      &wg,
	}, nil
}

func (p *InvestProvider) Unsubscribe(s *marketdata.Subscription) {
	s.Cancel()
	s.Wg.Wait()
	p.logger.Info("Unsubscribe finish")
}
