// Package app wires the backend example application entrypoint.
package app

import (
	"log/slog"
	"os"

	"github.com/arcgolabs/dix/examples/backend/config"
	"github.com/arcgolabs/dix/examples/backend/db"
	"github.com/arcgolabs/dix/examples/backend/event"
	"github.com/arcgolabs/dix/examples/backend/http"
	"github.com/arcgolabs/dix/examples/backend/repo"
	"github.com/arcgolabs/dix/examples/backend/service"
	"github.com/DaiYuANg/arcgo/logx"
	"github.com/arcgolabs/dix"
)

// Run starts the backend example application.
func Run() {
	logger := logx.MustNew(logx.WithConsole(true), logx.WithDebugLevel())

	a := dix.New(
		"backend",
		dix.Version("0.1.0"),
		dix.UseLogger(logger),
		dix.Modules(
			config.Module,
			event.Module,
			db.Module,
			repo.Module,
			service.Module,
			http.Module,
		),
	)

	if err := a.Run(); err != nil {
		logger.Error("backend exited", slog.String("error", err.Error()))
		closeLogger(logger)
		os.Exit(1)
	}
	closeLogger(logger)
}

func closeLogger(logger *slog.Logger) {
	if err := logx.Close(logger); err != nil {
		slog.Default().Error("close logger failed", slog.String("error", err.Error()))
	}
}
