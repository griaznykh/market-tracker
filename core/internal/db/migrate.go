package db

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"

	"github.com/jackc/tern/v2/migrate"
)

func (c *Client) Migrate(ctx context.Context, migrations embed.FS) error {
	dbConn, err := c.pgConnPool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire a database connection: %w", err)
	}
	defer dbConn.Release()

	migrator, err := migrate.NewMigrator(ctx, dbConn.Conn(), "schema_version")
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}

	subdir, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("migrations not found")
	}

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

	log.Printf("migration's done %d", ver)

	return nil
}
