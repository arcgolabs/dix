---
title: 'dix Health and Lifecycle'
linkTitle: 'health-lifecycle'
description: 'Register health checks and expose health endpoints'
weight: 3
---

## Health and lifecycle

`dix` runtime can execute:

- general health checks (`CheckHealth`)
- liveness checks (`CheckLiveness`)
- readiness checks (`CheckReadiness`)

You register checks on `*dix.Container` (typically in `Setups(...)`, `SetupContainer(...)`, or `WithModuleSetup(...)`). For HTTP, `Runtime` exposes ready-to-use handlers:

- `rt.HealthHandler()` → `/healthz`
- `rt.LivenessHandler()` → `/livez`
- `rt.ReadinessHandler()` → `/readyz`

## 1) Install

```bash
go get github.com/arcgolabs/dix@latest
go get github.com/arcgolabs/logx@latest
```

## 2) Create `main.go`

This example registers a liveness check that always passes, and a readiness check that depends on a resolved `*Server`.

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/arcgolabs/dix"
	"github.com/arcgolabs/logx"
)

type Config struct {
	Port int
}

type Server struct {
	Logger *slog.Logger
	Config Config
}

func main() {
	configModule := dix.NewModule("config",
		dix.Providers(dix.Provider0(func() Config { return Config{Port: 8080} })),
	)

	logger, err := logx.NewDevelopment()
	if err != nil {
		panic(err)
	}

	serverModule := dix.NewModule("server",
		dix.Imports(configModule),
		dix.Providers(
			dix.Provider2(func(logger *slog.Logger, cfg Config) *Server {
				return &Server{Logger: logger, Config: cfg}
			}),
		),
		dix.Setups(dix.SetupContainer(func(c *dix.Container) error {
			c.RegisterLivenessCheck("process", func(context.Context) error { return nil })
			c.RegisterReadinessCheck("bootstrap", func(context.Context) error {
				server, ok := dix.ResolveOptional[*Server](c)
				if !ok || server == nil {
					return errors.New("server not ready")
				}
				return nil
			})
			return nil
		})),
	)

	app := dix.NewDefault(
		dix.UseProfile(dix.ProfileDev),
		dix.Version("0.1.0"),
		dix.Modules(serverModule),
		dix.UseLogger(logger),
	)

	rt, err := app.Start(context.Background())
	if err != nil {
		panic(err)
	}
	defer func() { _ = rt.Stop(context.Background()) }()

	fmt.Println("health:", rt.CheckHealth(context.Background()).Healthy())
	fmt.Println("liveness:", rt.CheckLiveness(context.Background()).Healthy())
	fmt.Println("readiness:", rt.CheckReadiness(context.Background()).Healthy())

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", rt.HealthHandler())
	mux.HandleFunc("/livez", rt.LivenessHandler())
	mux.HandleFunc("/readyz", rt.ReadinessHandler())

_ = mux
}
```

## Optional: use DI logger as framework logger (`logx`)

If your health/lifecycle modules also own logger wiring, you can let `dix` internal logs use the DI-produced logger:

```go
type LogBundle struct {
	Logger *slog.Logger
}

logModule := dix.NewModule("logx",
	dix.Providers(
		dix.Provider0(func() *LogBundle {
			return &LogBundle{
				Logger: logx.MustNew(logx.WithConsole(true), logx.WithDebugLevel()),
			}
		}),
		dix.Provider1(func(logs *LogBundle) *slog.Logger {
			return logs.Logger
		}),
	),
	dix.Hooks(
		dix.OnStop(func(_ context.Context, logs *LogBundle) error {
			return logx.Close(logs.Logger)
		}),
	),
)

app := dix.NewDefault(
	dix.Modules(logModule, serverModule),
)
```

This keeps logger lifecycle in modules while replacing the framework default logger before build logging starts.

## Related

- [Getting Started](./getting-started)
- Example guide: [dix examples](./examples)
