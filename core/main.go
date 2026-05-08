package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"lib/signal"
	"log"
	config "market-service/internal/configs"
	"market-service/internal/grpc/server"
	"market-service/internal/grpc/service"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/tern/v2/migrate"
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

	dbPool, err := pgxpool.New(ctx, config.POSTGRES_DSN)
	if err != nil {
		e := fmt.Errorf("connect to db: %w", err)
		log.Println(e.Error())
		return
	}
	defer dbPool.Close()

	dbConn, err := dbPool.Acquire(ctx)
	if err != nil {
		e := fmt.Errorf("acquire a database connection: %w", err)
		log.Println(e.Error())
		return
	}
	defer dbConn.Release()

	if err := migrateDatabase(ctx, dbConn.Conn()); err != nil {
		e := fmt.Errorf("failed migration: %w", err)
		log.Println(e.Error())
		return
	}

	serverConfig := server.Config{
		GRPC:     &server.GRPCServer{Port: 8080, ReflectionEnabled: true}, // TODO remove hardcode. port should be received from config
		HTTP:     &server.HTTPServer{Port: 8081},                          // TODO remove hardcode. port should be received from config
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

func migrateDatabase(ctx context.Context, conn *pgx.Conn) error {
	migrator, err := migrate.NewMigrator(ctx, conn, "schema_version")
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}

	subdir, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("migrations not found")
	}

	log.Println(subdir)

	if err = migrator.LoadMigrations(subdir); err != nil {
		return fmt.Errorf("load migrations: %w", err)
	}

	if err = migrator.Migrate(ctx); err != nil {
		return fmt.Errorf("failed migration: %w", err)

	}

	ver, err := migrator.GetCurrentVersion(ctx)
	if err != nil {
		return fmt.Errorf("get current schema version: %w", err)
	}

	log.Println("migration's done %i", ver)
	return nil
}
