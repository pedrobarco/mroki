package ent

import (
	"database/sql"

	entsql "entgo.io/ent/dialect/sql"
	genent "github.com/pedrobarco/mroki/ent"
)

// NewPostgresClient creates an ent client from a Postgres *sql.DB.
func NewPostgresClient(db *sql.DB) *genent.Client {
	drv := entsql.OpenDB("postgres", db)
	return genent.NewClient(genent.Driver(drv))
}

// NewSQLiteClient creates an ent client from a SQLite *sql.DB.
func NewSQLiteClient(db *sql.DB) *genent.Client {
	drv := entsql.OpenDB("sqlite3", db)
	return genent.NewClient(genent.Driver(drv))
}
