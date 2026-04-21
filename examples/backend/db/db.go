// Package db configures the database used by the backend example.
package db

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/DaiYuANg/arcgo/dbx"
	sqlitedialect "github.com/DaiYuANg/arcgo/dbx/dialect/sqlite"
	// Register the pure-Go SQLite driver used by the example.
	_ "modernc.org/sqlite"
)

// OpenSQLite opens the backend example SQLite database.
func OpenSQLite(dsn string, opts ...dbx.Option) (*dbx.DB, error) {
	if dsn == "" {
		dsn = "file:backend?mode=memory&cache=shared"
	}
	db, err := dbx.Open(
		dbx.WithDriver("sqlite"),
		dbx.WithDSN(dsn),
		dbx.WithDialect(sqlitedialect.New()),
		dbx.ApplyOptions(opts...),
	)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if _, err := db.ExecContext(context.Background(), `PRAGMA foreign_keys = ON`); err != nil {
		closeErr := db.Close()
		return nil, fmt.Errorf("enable sqlite foreign keys: %w", errors.Join(err, closeErr))
	}
	return db, nil
}

// DefaultOpts returns the default dbx options for the backend example.
func DefaultOpts(logger *slog.Logger) []dbx.Option {
	if logger == nil {
		return nil
	}
	return []dbx.Option{
		dbx.WithLogger(logger),
		dbx.WithDebug(false),
	}
}
