package service

import (
	"context"
	"errors"
	"fmt"
	"market-service/internal/db"
	"market-service/internal/marketdata/collector"

	"lib/auth"
	pb "lib/schema/gen/go/api/core/v1"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

type (
	// ExternalServiceConfig defines ExternalService configuration.
	ExternalServiceConfig struct {
		DB         *db.Client
		Collector  *collector.MarketDataCollector
		JwtManager auth.JwtManager
	}

	// ExternalService implements a customer-facing gRPC service.
	ExternalService struct {
		pb.UnimplementedCoreServiceServer
		db         *db.Client
		collector  *collector.MarketDataCollector
		jwtManager auth.JwtManager
	}
)

func (c *ExternalServiceConfig) validate() error {
	if c.DB == nil {
		return errors.New("database client is nil")
	}

	if c.Collector == nil {
		return errors.New("collector is nil")
	}

	return nil
}

func NewExternalService(config *ExternalServiceConfig) (*ExternalService, error) {
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	return &ExternalService{
		db:         config.DB,
		collector:  config.Collector,
		jwtManager: config.JwtManager,
	}, nil
}

func (s *ExternalService) RegisterGRPCServices(srv *grpc.Server) error {
	pb.RegisterCoreServiceServer(srv, s)

	return nil
}

func (s *ExternalService) RegisterGRPCGatewayHandlers(
	ctx context.Context,
	mux *runtime.ServeMux,
	endpoint string,
	dialOpts []grpc.DialOption,
) error {
	if err := pb.RegisterCoreServiceHandlerFromEndpoint(ctx, mux, endpoint, dialOpts); err != nil {
		return fmt.Errorf("register service handler: %w", err)
	}

	return nil
}
