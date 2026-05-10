package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"lib/signal"
	"log"
	config "market-service/internal/configs"
	"market-service/internal/db"
	"market-service/internal/grpc/server"
	"market-service/internal/grpc/service"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
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

	config := config.NewConfig()

	dbPool, err := pgxpool.New(ctx, config.DSN)
	if err != nil {
		e := fmt.Errorf("connect to db: %w", err)
		log.Println(e.Error())
		return
	}
	defer dbPool.Close()

	dbClient := db.NewClient(dbPool)
	err = dbClient.Migrate(ctx, migrations)
	if err != nil {
		e := fmt.Errorf("migrate db: %w", err)
		log.Println(e.Error())
		return
	}

	serverConfig := server.Config{
		GRPC:     &server.GRPCServer{Port: config.GRPC_PORT, ReflectionEnabled: config.GRPC_REFLECTION},
		HTTP:     &server.HTTPServer{Port: config.HTTP_PORT},
		Services: []service.SelfRegisteringService{
			// TODO add services
		},
	}

	server, err := server.New(serverConfig)
	if err != nil {
		e := fmt.Errorf("init gRPC server: %w", err)
		log.Println(e.Error())
		return
	}

	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return server.Run(gCtx)
	})

	if err = g.Wait(); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, http.ErrServerClosed) {
		e := fmt.Errorf("an un-recoverable error occurred, exiting: %w", err)
		log.Println(e.Error())
		return
	}
}
