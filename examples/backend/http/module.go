// Package http wires the backend example HTTP server.
package http

import (
	"context"
	"log/slog"

	"github.com/arcgolabs/dix"
	"github.com/arcgolabs/dix/examples/backend/api"
	"github.com/arcgolabs/dix/examples/backend/config"
	"github.com/arcgolabs/dix/examples/backend/service"
	"github.com/arcgolabs/httpx"
	"github.com/arcgolabs/httpx/adapter"
	"github.com/arcgolabs/httpx/adapter/std"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
)

// Module wires the backend example HTTP API server.
var Module = dix.NewModule("http",
	dix.Imports(config.Module, service.Module),
	dix.Providers(
		dix.Provider2(func(svc service.UserService, log *slog.Logger) httpx.ServerRuntime {
			router := chi.NewMux()
			router.Use(middleware.Logger, middleware.Recoverer, middleware.RequestID)
			ad := std.New(router, adapter.HumaOptions{
				Title:       "ArcGo Backend API",
				Version:     "1.0.0",
				Description: "configx + logx + eventx + httpx + dix + dbx",
				DocsPath:    "/docs",
				OpenAPIPath: "/openapi.json",
			})
			server := httpx.New(
				httpx.WithAdapter(ad),
				httpx.WithLogger(log),
				httpx.WithPrintRoutes(true),
				httpx.WithValidator(validator.New(validator.WithRequiredStructEnabled())),
				httpx.WithValidation(),
			)
			api.RegisterRoutes(server, svc)
			return server
		}),
	),
	dix.Hooks(
		dix.OnStart2(func(_ context.Context, server httpx.ServerRuntime, cfg config.AppConfig) error {
			port := cfg.Server.Port
			logger := server.Logger()
			if logger != nil {
				logger.Info("http listen scheduled", "port", port)
			}
			go func() {
				if err := server.ListenPort(port); err != nil && logger != nil {
					logger.Error("http server stopped", "port", port, "error", err.Error())
				}
			}()
			return nil
		}),
		dix.OnStop(func(_ context.Context, server httpx.ServerRuntime) error {
			return server.Shutdown()
		}),
	),
)
