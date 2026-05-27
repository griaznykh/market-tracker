package tbank

import (
	"context"
	"market-service/internal/marketdata"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"opensource.tbank.ru/invest/invest-go/investgo"
)

type TbankProvider struct {
	logger   *zap.SugaredLogger
	client   *investgo.Client
	mdClient *investgo.MarketDataStreamClient
	mdStream *investgo.MarketDataStream
}

type TbankProviderConfig struct {
	Token    string
	Endpoint string
}

func NewTbankProvider(
	ctx context.Context,
	config TbankProviderConfig,
	logger *zap.SugaredLogger,
) (*TbankProvider, error) {
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

	return &TbankProvider{
		logger:   logger,
		client:   client,
		mdClient: MDClient,
	}, nil
}

func (p *TbankProvider) Subscribe(ctx context.Context, req marketdata.SubscriptionRequest) (*marketdata.Subscription, error) {
	p.logger.Infof("Subscribe called for tickers: %v", req.Tickers)

	ctx, cancel := context.WithCancel(ctx)

	var wg sync.WaitGroup

	MDStream, err := p.mdClient.MarketDataStream()
	if err != nil {
		cancel()
		return nil, err
	}

	src, err := MDStream.SubscribeLastPrice(req.Tickers)
	if err != nil {
		cancel()
		return nil, err
	}

	out := make(chan marketdata.Data)
	finished := make(chan struct{})

	wg.Go(func() {
		<-ctx.Done()
		MDStream.Stop()
	})

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

				p.logger.Infof(
					"ticker=%s price=%f time=%v",
					data.GetTicker(),
					data.GetPrice().ToFloat(),
					data.GetTime().AsTime(),
				)

				out <- marketdata.Data{
					Provider: "tbank",
					Ticker:   data.GetTicker(),
					Price:    data.GetPrice().ToFloat(),
					Time:     data.GetTime().AsTime(),
				}
			}
		}

	})

	go func() {
		wg.Wait()
		close(finished)
	}()

	cancelFunc := func() {
		cancel()

		select {
		case <-finished:
			p.logger.Infof(
				"subscription closed: %s",
				strings.Join(req.Tickers, ", "),
			)

		case <-time.After(5 * time.Second):
			p.logger.Warnf(
				"timeout waiting subscription shutdown: %s",
				strings.Join(req.Tickers, ", "),
			)
		}
	}

	return marketdata.NewSubscription(out, cancelFunc), nil
}
