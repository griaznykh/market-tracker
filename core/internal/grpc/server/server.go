package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	grpc_protovalidate "market-service/internal/grpc/middleware/protovalidate"
	grpc_timeout "market-service/internal/grpc/middleware/timeout"
	"market-service/internal/grpc/service"
	"market-service/internal/registry"
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
		GRPC        *GRPCServer
		HTTP        *HTTPServer
		Services    []service.SelfRegisteringService
		JwtRegistry registry.JwtRegistry
	}

	Server struct {
		grpc        *GRPCServer
		http        *HTTPServer
		services    []service.SelfRegisteringService
		jwtRegistry registry.JwtRegistry
	}
)

func (c *Config) validate() error {
	switch {
	case c.GRPC == nil:
		return errors.New("grpc server settings may not be nil")

	case c.HTTP == nil:
		return errors.New("http server settings may not be nil")

	case c.JwtRegistry == nil:
		return errors.New("jwt registry settings may not be nil")

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
		grpc:        config.GRPC,
		http:        config.HTTP,
		services:    config.Services,
		jwtRegistry: config.JwtRegistry,
	}, nil
}

func (server *Server) Run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	grpcServerAddress := fmt.Sprintf(":%d", server.grpc.Port)
	httpServerAddress := fmt.Sprintf(":%d", server.http.Port)

	grpcSrv := getGRPCServer()
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

func getGRPCServer(inOpts ...grpc.ServerOption) *grpc.Server {
	validator, err := protovalidate.New()
	if err != nil {
		panic(fmt.Errorf("initialize protovalidate: %w", err))
	}

	srvOpts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			grpc_timeout.UnaryServerInterceptor(30*time.Second),
			grpc_recovery.UnaryServerInterceptor(),
			grpc_protovalidate.UnaryServerInterceptor(
				validator,
			),
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
