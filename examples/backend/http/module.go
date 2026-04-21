// Package http wires the backend example HTTP server.
package http

import (
	"log/slog"

	"github.com/DaiYuANg/arcgo/examples/dix/backend/api"
	"github.com/DaiYuANg/arcgo/examples/dix/backend/config"
	"github.com/DaiYuANg/arcgo/examples/dix/backend/service"
	"github.com/DaiYuANg/arcgo/httpx"
	"github.com/DaiYuANg/arcgo/httpx/adapter"
	"github.com/DaiYuANg/arcgo/httpx/adapter/std"
	httpxdix "github.com/DaiYuANg/arcgo/httpx/dix"
	"github.com/arcgolabs/arcgo/dix"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
)

// Module wires the backend example HTTP API server.
var Module = httpxdix.NewModule("http",
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
	httpxdix.WithImports(config.Module, service.Module),
	httpxdix.WithListenPort1(func(cfg config.AppConfig) int { return cfg.Server.Port }),
)
