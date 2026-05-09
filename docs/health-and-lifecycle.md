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

## Runtime lifecycle summaries

After `Build()` or `Start()`, a runtime can report its bound lifecycle hooks and built subapps without exposing internal fields:

```go
summary := rt.LifecycleSummary()
fmt.Println("start hooks:", summary.StartHooks)
fmt.Println("stop hooks:", summary.StopHooks)

subapps := rt.SubAppSummaries()
for _, subapp := range subapps.Values() {
	fmt.Println("subapp:", subapp.Name, "parent:", subapp.ParentName, "state:", subapp.State)
}
```

For nested apps, `rt.ScopePath()` returns app names from root to the current runtime. `rt.IsSubApp()` and `rt.ParentName()` are useful when lifecycle code needs to log whether it is running in a child app.

## Lifecycle ordering and parallel hooks

Lifecycle hooks are serial by default. `start` hooks run by ascending priority, and `stop` hooks run by descending priority so teardown mirrors startup.

```go
runtimeModule := dix.NewModule("runtime",
	dix.Hooks(
		dix.OnStart0(startCache,
			dix.LifecycleName("cache"),
			dix.LifecyclePriority(10),
			dix.LifecycleParallel(),
		),
		dix.OnStart0(startQueue,
			dix.LifecycleName("queue"),
			dix.LifecycleAfter("cache"),
			dix.LifecyclePriority(10),
			dix.LifecycleParallel(),
			dix.LifecycleTimeout(10 * time.Second),
		),
		dix.OnStop0(stopQueue,
			dix.LifecycleName("queue"),
			dix.LifecyclePriority(10),
			dix.LifecycleParallel(),
		),
		dix.OnStop0(stopCache,
			dix.LifecycleName("cache"),
			dix.LifecycleAfter("queue"),
			dix.LifecyclePriority(10),
			dix.LifecycleParallel(),
		),
	),
)

app := dix.NewDefault(
	dix.LifecycleConcurrency(4),
	dix.Modules(runtimeModule),
)
```

Use `LifecycleAfter(...)` and `LifecycleBefore(...)` when named hooks have ordering dependencies. The scheduler performs a topological sort within each lifecycle phase; priority and declaration order are still used as tie-breakers for hooks that are otherwise independent.

Only adjacent unconstrained hooks with the same priority and `LifecycleParallel()` are run together. Any serial hook or named ordering constraint remains a barrier. `LifecycleTimeout(...)` passes a deadline-aware child context to the hook; hooks should respect `ctx.Done()` for cooperative cancellation. `LifecycleSummary()` includes hook names, priorities, ordering constraints, parallel flags, timeouts, and the resolved lifecycle concurrency.

When debug logging is enabled, lifecycle hooks log duration fields. Build diagnostics also include build duration, setup duration, invoke duration, provider construction duration, and explicit `ResolveAsContext` duration after framework logging has been configured. Observers can implement `ProviderObserver`, `ResolveObserver`, or `LifecycleHookObserver` to receive structured diagnostic events for metrics and tracing.

## Eager provider warmup

Most providers stay lazy and are constructed on first resolve. For critical infrastructure, mark the provider eager so build performs warmup before invokes and runtime start:

```go
dbModule := dix.NewModule("db",
	dix.Providers(
		dix.Provider1(NewDatabase, dix.Eager()),
	),
)
```

Eager provider failures fail `Build()` with module, provider label, and service context. Raw eager providers must declare `ProviderMetadata.Output`.

## Test overrides

For tests, derive a test-only app with provider replacements and disabled modules:

```go
rt := testx.BuildWith(t, app,
	dix.TestDisableModules("db"),
	dix.TestValue[Store](mockStore),
)
```

`app.Test(...)` does not mutate the original app. Replacement providers remove matching outputs or aliases from the cloned module graph before validation, so mocks still participate in normal dependency checks.

## Related

- [Getting Started](./getting-started)
- Example guide: [dix examples](./examples)
