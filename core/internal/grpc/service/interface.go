package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

type SelfRegisteringService interface {
	RegisterGRPCServices(server *grpc.Server) error

	RegisterGRPCGatewayHandlers(
		ctx context.Context,
		mux *runtime.ServeMux,
		endpoint string,
		opts []grpc.DialOption,
	) error
}
