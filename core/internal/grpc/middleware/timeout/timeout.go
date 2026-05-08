package timeout

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

// UnaryServerInterceptor returns a new unary server interceptor for request timeout.
func UnaryServerInterceptor(timeout time.Duration) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
	) (interface{}, error) {
		newCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		return handler(newCtx, req)
	}
}
