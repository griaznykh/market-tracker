package ratelimit

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type RateLimiter interface {
	Request(string) bool
}

// UnaryServerInterceptor returns a new unary server interceptor that control rate limits
func UnaryServerInterceptor(getId func(ctx context.Context) (string, bool), ratelimiter RateLimiter) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		id, ok := getId(ctx)
		if !ok {
			return handler(ctx, req)
		}

		if ratelimiter.Request(id) {
			return handler(ctx, req)
		}

		return nil, status.Error(codes.ResourceExhausted, "rate limit exeeded")
	}
}
