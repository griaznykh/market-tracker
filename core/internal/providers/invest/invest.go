package invest

import (
	"context"
	"market-service/internal/marketdata"
	"sync"

	"go.uber.org/zap"
	"opensource.tbank.ru/invest/invest-go/investgo"
)

type Provider struct {
	client   *investgo.Client
	mdStream *investgo.MarketDataStream

	logger *zap.SugaredLogger

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

type ProviderConfig struct {
	Token    string
	Endpoint string
}

func NewProvider(
	ctx context.Context,
	config ProviderConfig,
	logger *zap.SugaredLogger,
) (*Provider, error) {
	investConfig := investgo.Config{
		EndPoint:           config.Endpoint,
		Token:              config.Token,
		InsecureSkipVerify: true,
	}

	client, err := investgo.NewClient(ctx, investConfig, logger)
	if err != nil {
		return nil, err
	}

	mdClient := client.NewMarketDataStreamClient()

	stream, err := mdClient.MarketDataStream()
	if err != nil {
		return nil, err
	}

	return &Provider{
		client:   client,
		mdStream: stream,
		logger:   logger,
	}, nil
}

func (p *Provider) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)

	p.cancel = cancel

	p.wg.Add(1)

	go func() {
		defer p.wg.Done()

		err := p.mdStream.Listen()
		if err != nil {
			p.logger.Errorf("market data stream error: %v", err)
		}
	}()

	return nil
}

func (p *Provider) SubscribeTrades(
	ctx context.Context,
	req marketdata.TradeSubscription,
) (<-chan marketdata.Trade, error) {

	src, err := p.mdStream.SubscribeLastPrice(req.Tickers)
	if err != nil {
		return nil, err
	}

	out := make(chan marketdata.Trade)

	p.wg.Add(1)

	go func() {
		defer p.wg.Done()
		defer close(out)

		for {
			select {
			case <-ctx.Done():
				return

			case trade, ok := <-src:
				if !ok {
					return
				}

				out <- marketdata.Trade{
					Ticker: trade.GetTicker(),
					Price:  trade.GetPrice().ToFloat(),
					Time:   trade.GetTime().AsTime(),
				}
			}
		}
	}()

	return out, nil
}

func (p *Provider) Close() error {
	if p.cancel != nil {
		p.cancel()
	}

	p.mdStream.Stop()

	p.wg.Wait()

	return p.client.Stop()
}
