package auth

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"lib/auth"
)

type AuthMiddleware struct {
	jwtManager auth.JwtManager
}

func NewAuthMiddleware(jwtManager auth.JwtManager) *AuthMiddleware {
	return &AuthMiddleware{
		jwtManager: jwtManager,
	}
}

func (m *AuthMiddleware) UnaryServerInterceptor(
	publicMethods map[string]bool,
) grpc.UnaryServerInterceptor {

	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {

		// public endpoint
		if publicMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(
				codes.Unauthenticated,
				"metadata is not provided",
			)
		}

		values := md.Get("authorization")
		if len(values) == 0 {
			return nil, status.Error(
				codes.Unauthenticated,
				"authorization token is not provided",
			)
		}

		authHeader := values[0]

		const bearerPrefix = "Bearer "

		if !strings.HasPrefix(authHeader, bearerPrefix) {
			return nil, status.Error(
				codes.Unauthenticated,
				"invalid authorization format",
			)
		}

		accessToken := strings.TrimPrefix(authHeader, bearerPrefix)

		claims, err := m.jwtManager.Verify(ctx, accessToken)
		if err != nil {
			return nil, status.Error(
				codes.Unauthenticated,
				"invalid access token",
			)
		}

		user := &auth.User{
			Id: claims.Id,
		}

		ctx = auth.ContextWithUser(ctx, user)

		return handler(ctx, req)
	}
}
