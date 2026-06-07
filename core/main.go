package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"lib/auth"
	ratelimiter "lib/rate-limiter"
	"lib/signal"
	"log"
	config "market-service/internal/configs"
	"market-service/internal/db"
	"market-service/internal/grpc/server"
	"market-service/internal/grpc/service"
	"market-service/internal/marketdata"
	"market-service/internal/marketdata/collector"
	"market-service/internal/providers/tbank"
	"net/http"
	"time"

	"lib/grpcx"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

//go:embed migrations/*.sql
var migrations embed.FS

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("Error loading .env file")
	}
}

func main() {
	ctx := signal.Context()

	logger, err := zap.NewProduction()
	if err != nil {
		log.Println(fmt.Errorf("create new logger error: %w", err))
		return
	}

	config := config.NewConfig()

	dbPool, err := pgxpool.New(ctx, config.DSN)
	if err != nil {
		e := fmt.Errorf("connect to db: %w", err)
		logger.Error(e.Error())
		return
	}
	defer dbPool.Close()

	dbClient := db.NewClient(dbPool)
	err = dbClient.Migrate(ctx, migrations)
	if err != nil {
		e := fmt.Errorf("migrate db: %w", err)
		logger.Error(e.Error())
		return
	}

	investConfig := tbank.TbankProviderConfig{
		Token: config.INVEST_API_TOKEN,
	}

	investProvider, err := tbank.NewTbankProvider(
		ctx,
		investConfig,
		logger.Sugar(),
	)

	if err != nil {
		e := fmt.Errorf("init tbank provider err: %w", err)
		logger.Error(e.Error())
		return
	}

	collectorConfig := collector.Config{
		Logger: logger,
		Providers: map[string]marketdata.MarketDataProvider{
			"tbank": investProvider,
		},
		Tasks: []collector.Task{
			{Provider: "tbank", Tickers: config.TICKERS},
		},
	}

	mdCollector, err := collector.New(collectorConfig)
	if err != nil {
		e := fmt.Errorf("init market data collector err: %w", err)
		logger.Error(e.Error())
		return
	}

	// 10 request per minute allowed for auth guard api
	rateLimitManager := ratelimiter.NewRateLimiterManager(ctx, 10, time.Second*300, time.Second*60)
	jwtManager := auth.NewJwtManager(config.JWT_SECRET, config.JWT_DURATION)

	externalService, err := service.NewExternalService(&service.ExternalServiceConfig{
		DB:         dbClient,
		Collector:  mdCollector,
		JwtManager: jwtManager,
	})

	if err != nil {
		e := fmt.Errorf("init external gRPC service: %w", err)
		logger.Error(e.Error())
		return
	}

	serverConfig := server.Config{
		Logger: logger,
		GRPC:   &server.GRPCServer{Port: config.GRPC_PORT, ReflectionEnabled: config.GRPC_REFLECTION},
		HTTP:   &server.HTTPServer{Port: config.HTTP_PORT},
		Services: []grpcx.SelfRegisteringService{
			externalService,
		},
		JwtManager:         jwtManager,
		RateLimiterManager: rateLimitManager,
	}

	server, err := server.New(serverConfig)
	if err != nil {
		e := fmt.Errorf("init gRPC server: %w", err)
		logger.Error(e.Error())
		return
	}

	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return server.Run(gCtx)
	})
	g.Go(func() error {
		return mdCollector.Run(ctx)
	})

	if err = g.Wait(); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, http.ErrServerClosed) {
		e := fmt.Errorf("an un-recoverable error occurred, exiting: %w", err)
		logger.Error(e.Error())
		return
	}
}
