package advanced_test

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/arcgolabs/dix"
	"github.com/stretchr/testify/require"
)

type Config struct {
	DSN  string
	Port int
}

func ProvideConfig() Config {
	return Config{
		DSN:  "sqlite://test.db",
		Port: 8080,
	}
}

type Database struct {
	dsn string
}

func NewDatabase(dsn string) *Database {
	return &Database{dsn: dsn}
}

func (d *Database) Connect() error { return nil }
func (d *Database) Close() error   { return nil }

func ProvideDatabase(cfg Config) *Database {
	return NewDatabase(cfg.DSN)
}

type Server struct {
	addr string
}

func ProvideServer(cfg Config) *Server {
	return &Server{
		addr: fmt.Sprintf(":%d", cfg.Port),
	}
}

func (s *Server) Start() error                 { return nil }
func (s *Server) Stop(_ context.Context) error { return nil }

type testGreeter interface {
	Greet() string
}

type testGreeterImpl struct{}

func (g *testGreeterImpl) Greet() string {
	return "hello"
}

func buildRuntime(t *testing.T, app *dix.App) *dix.Runtime {
	t.Helper()
	rt, err := app.Build()
	require.NoError(t, err)
	require.NotNil(t, rt)
	return rt
}

func loggerModule(logger *slog.Logger) dix.Module {
	return dix.NewModule("logger",
		dix.WithModuleProvider(dix.Value(logger)),
	)
}
