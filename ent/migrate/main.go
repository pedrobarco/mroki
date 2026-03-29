//go:build ignore

package main

import (
	"context"
	"database/sql"
	"log"
	"os"

	"github.com/pedrobarco/mroki/ent/migrate"

	atlas "ariga.io/atlas/sql/migrate"
	_ "ariga.io/atlas/sql/postgres"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql/schema"
	"github.com/jackc/pgx/v5/stdlib"
)

func init() {
	// Atlas's Postgres driver calls sql.Open("postgres", ...) but pgx/v5/stdlib
	// registers itself as "pgx". Register the pgx driver under "postgres" so
	// Atlas can find it.
	sql.Register("postgres", stdlib.GetDefaultDriver())
}

func main() {
	ctx := context.Background()

	// Create a local migration directory able to understand Atlas migration file format for replay.
	dir, err := atlas.NewLocalDir("ent/migrate/migrations")
	if err != nil {
		log.Fatalf("failed creating atlas migration directory: %v", err)
	}

	// Migrate diff options.
	opts := []schema.MigrateOption{
		schema.WithDir(dir),
		schema.WithMigrationMode(schema.ModeReplay),
		schema.WithDialect(dialect.Postgres),
		schema.WithFormatter(atlas.DefaultFormatter),
		schema.WithDropColumn(true),
		schema.WithDropIndex(true),
	}

	if len(os.Args) != 2 {
		log.Fatalln("migration name is required. Use: 'go run -mod=mod ent/migrate/main.go <name>'")
	}

	// Generate migrations using Atlas support for PostgreSQL.
	// The dev-url should point to a clean, empty PostgreSQL database.
	devURL := os.Getenv("ATLAS_DEV_URL")
	if devURL == "" {
		devURL = "postgres://postgres:pass@localhost:5433/test?search_path=public&sslmode=disable"
	}

	err = migrate.NamedDiff(ctx, devURL, os.Args[1], opts...)
	if err != nil {
		log.Fatalf("failed generating migration file: %v", err)
	}
}
