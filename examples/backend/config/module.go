package config

import (
	"fmt"
	"log/slog"

	"github.com/arcgolabs/configx"
	"github.com/DaiYuANg/arcgo/pkg/randomport"
	"github.com/arcgolabs/dix"
)

// Module loads the backend example configuration and startup logs.
var Module = dix.NewModule("config",
	dix.Providers(
		dix.Provider1(func(log *slog.Logger) AppConfig {
			var cfg AppConfig
			loader := configx.New(
				configx.WithDefaults(map[string]any{
					"server.port": 8080,
					"db.dsn":      "file:backend?mode=memory&cache=shared",
				}),
				configx.WithEnvPrefix("APP_"),
			)
			if err := loader.Load(&cfg); err != nil {
				log.Error("config load failed", slog.String("error", err.Error()))
				panic(err)
			}
			if cfg.Server.Port == 0 {
				cfg.Server.Port = randomport.MustFind()
			}
			return cfg
		}),
	),
	dix.Invokes(
		dix.Invoke2(func(cfg AppConfig, log *slog.Logger) {
			addr := fmt.Sprintf(":%d", cfg.Server.Port)
			log.Info("backend starting",
				slog.String("address", addr),
				slog.String("docs", "http://localhost"+addr+"/docs"),
				slog.String("openapi", "http://localhost"+addr+"/openapi.json"),
			)
		}),
	),
)
