package server

import (
	"context"
	"errors"
	"fmt"
	"lib/auth"
	grpc_protovalidate "lib/middleware/protovalidate"
	"lib/middleware/ratelimit"
	grpc_timeout "lib/middleware/timeout"
	ratelimiter "lib/rate-limiter"
	"log"
	"market-service/internal/grpc/service"
	"net"
	"net/http"
	"time"

	"buf.build/go/protovalidate"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

type (
	GRPCServer struct {
		Port              uint
		ReflectionEnabled bool
	}

	HTTPServer struct {
		Port uint
	}

	Config struct {
		GRPC               *GRPCServer
		HTTP               *HTTPServer
		Services           []service.SelfRegisteringService
		RateLimiterManager *ratelimiter.RateLimiterManager
		JwtManager         auth.JwtManager
	}

	Server struct {
		grpc               *GRPCServer
		http               *HTTPServer
		services           []service.SelfRegisteringService
		rateLimiterManager *ratelimiter.RateLimiterManager
		jwtManager         auth.JwtManager
	}
)

func (c *Config) validate() error {
	switch {
	case c.GRPC == nil:
		return errors.New("grpc server settings may not be nil")

	case c.HTTP == nil:
		return errors.New("http server settings may not be nil")

	case c.JwtManager == nil:
		return errors.New("jwt manager settings may not be nil")

	case c.RateLimiterManager == nil:
		return errors.New("rate limiter manager settings may not be nil")

	default:
		return nil
	}
}

// New creates a new gRPC server.
func New(config Config) (*Server, error) {
	if err := config.validate(); err != nil {
		return nil, err
	}

	return &Server{
		grpc:               config.GRPC,
		http:               config.HTTP,
		services:           config.Services,
		jwtManager:         config.JwtManager,
		rateLimiterManager: config.RateLimiterManager,
	}, nil
}

func (server *Server) Run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	grpcServerAddress := fmt.Sprintf(":%d", server.grpc.Port)
	httpServerAddress := fmt.Sprintf(":%d", server.http.Port)

	authInterceptor := auth.NewAuthMiddleware(server.jwtManager)

	grpcSrv := getGRPCServer(nil,
		authInterceptor.UnaryServerInterceptor(map[string]bool{
			"/api.core.v1.CoreService/Register":       true,
			"/api.core.v1.CoreService/CreateApiToken": true,
		}), ratelimit.UnaryServerInterceptor(func(ctx context.Context) (string, bool) {
			user, ok := auth.UserFromContext(ctx)
			if ok {
				return user.Id, ok
			}
			return "", ok
		}, server.rateLimiterManager),
	)

	httpSrv, httpMux := getHTTPServer(httpServerAddress)

	grpcDialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	for _, svc := range server.services {
		if err := svc.RegisterGRPCServices(grpcSrv); err != nil {
			return fmt.Errorf("register grpc services: %w", err)
		}
		if err := svc.RegisterGRPCGatewayHandlers(ctx, httpMux, grpcServerAddress, grpcDialOpts); err != nil {
			return fmt.Errorf("register grpc gateway handlers: %w", err)
		}
	}

	// enable server reflection protocol if requested
	if server.grpc.ReflectionEnabled {
		reflection.Register(grpcSrv)
	}

	g.Go(func() error {
		// start listening on grpc port

		lc := net.ListenConfig{}
		listener, err := lc.Listen(ctx, "tcp", grpcServerAddress)
		if err != nil {
			return fmt.Errorf("listen port: %w", err)
		}

		log.Println("starting grpc server on address", grpcServerAddress)

		return grpcSrv.Serve(listener)
	})

	g.Go(func() error {
		// start listening on http port

		log.Println("starting http server on address", httpServerAddress)

		err := httpSrv.ListenAndServe()

		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}

		return err
	})

	<-ctx.Done()

	log.Println("application is shutting down, gracefully stopping all the servers")

	if err := httpSrv.Shutdown(ctx); err != nil {
		log.Println("failed to shutdown http server", err)
	}

	grpcSrv.Stop()

	return g.Wait()
}

func getGRPCServer(inOpts []grpc.ServerOption, inInterceptors ...grpc.UnaryServerInterceptor) *grpc.Server {
	validator, err := protovalidate.New()
	if err != nil {
		panic(fmt.Errorf("initialize protovalidate: %w", err))
	}

	interceptors := []grpc.UnaryServerInterceptor{
		grpc_recovery.UnaryServerInterceptor(),
		grpc_timeout.UnaryServerInterceptor(30 * time.Second),
	}

	// external middleware
	interceptors = append(
		interceptors,
		inInterceptors...,
	)

	interceptors = append(
		interceptors,
		grpc_protovalidate.UnaryServerInterceptor(
			validator,
		),
	)

	srvOpts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			interceptors...,
		),
	}

	// append our server opts to what they gave us
	srvOpts = append(inOpts, srvOpts...)

	// create new grpc server
	grpcSrv := grpc.NewServer(srvOpts...)

	return grpcSrv
}

func getHTTPServer(httpAddr string) (*http.Server, *runtime.ServeMux) {
	mux := runtime.NewServeMux()

	httpSrv := &http.Server{
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       120 * time.Second,
		Addr:              httpAddr,
		Handler:           mux,
	}

	return httpSrv, mux
}
