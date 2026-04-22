package db

import (
	"context"
	"log/slog"

	"github.com/DaiYuANg/arcgo/dbx"
	schemax "github.com/DaiYuANg/arcgo/dbx/schema"
	"github.com/DaiYuANg/arcgo/dbx/schemamigrate"
	"github.com/arcgolabs/dix"
	"github.com/arcgolabs/dix/examples/backend/config"
	"github.com/arcgolabs/dix/examples/backend/schema"
)

// Module wires the backend example database and schema services.
var Module = dix.NewModule("db",
	dix.Imports(config.Module),
	dix.Providers(
		dix.Provider2(func(cfg config.AppConfig, log *slog.Logger) *dbx.DB {
			database, err := OpenSQLite(cfg.DB.DSN, DefaultOpts(log)...)
			if err != nil {
				panic(err)
			}
			userSchema := schema.UserSchema{}
			users := schemax.MustSchema("users", userSchema)
			if _, err := schemamigrate.AutoMigrate(context.Background(), database, users); err != nil {
				panic(err)
			}
			return database
		}),
		dix.Provider0(func() schema.UserSchema {
			s := schema.UserSchema{}
			return schemax.MustSchema("users", s)
		}),
	),
	dix.Hooks(
		dix.OnStop(func(_ context.Context, database *dbx.DB) error {
			return database.Close()
		}),
	),
)
