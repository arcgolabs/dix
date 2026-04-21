---
title: 'dix examples'
linkTitle: 'examples'
description: 'Runnable examples for dix'
weight: 10
---

## dix Examples

This page collects the runnable `examples/dix` programs and maps them to the API surface they demonstrate.

## Run Locally

Run from the `examples/dix` module:

```bash
cd examples/dix
go run ./basic
go run ./collection_contributions
go run ./runtime_scope
go run ./inspect
```

## Core Examples

| Example | Focus | Directory |
| --- | --- | --- |
| `basic` | immutable app spec, `app.Start(ctx)`, health checks, `logx` integration | [examples/dix/basic](https://github.com/DaiYuANg/arcgo/tree/main/examples/dix/basic) |
| `aggregate_params` | provider graph composition with multiple typed dependencies | [examples/dix/aggregate_params](https://github.com/DaiYuANg/arcgo/tree/main/examples/dix/aggregate_params) |
| `collection_contributions` | distributed providers collected into slices, maps, and `collectionx` containers | [examples/dix/collection_contributions](https://github.com/DaiYuANg/arcgo/tree/main/examples/dix/collection_contributions) |
| `build_runtime` | explicit `Build()` to `Runtime` flow | [examples/dix/build_runtime](https://github.com/DaiYuANg/arcgo/tree/main/examples/dix/build_runtime) |
| `build_failure` | validation/build failure behavior | [examples/dix/build_failure](https://github.com/DaiYuANg/arcgo/tree/main/examples/dix/build_failure) |
| `metrics` | `dix/metrics`, external Prometheus handler, runtime metrics | [examples/dix/metrics](https://github.com/DaiYuANg/arcgo/tree/main/examples/dix/metrics) |

## Advanced Examples

| Example | Focus | Directory |
| --- | --- | --- |
| `advanced_do_bridge` | explicit `do` bridge setup | [examples/dix/advanced_do_bridge](https://github.com/DaiYuANg/arcgo/tree/main/examples/dix/advanced_do_bridge) |
| `named_alias` | named services and typed alias binding | [examples/dix/named_alias](https://github.com/DaiYuANg/arcgo/tree/main/examples/dix/named_alias) |
| `runtime_scope` | request-like runtime scope and scoped providers | [examples/dix/runtime_scope](https://github.com/DaiYuANg/arcgo/tree/main/examples/dix/runtime_scope) |
| `transient` | transient provider semantics | [examples/dix/transient](https://github.com/DaiYuANg/arcgo/tree/main/examples/dix/transient) |
| `override` | structured overrides | [examples/dix/override](https://github.com/DaiYuANg/arcgo/tree/main/examples/dix/override) |
| `inspect` | runtime inspection and diagnostics | [examples/dix/inspect](https://github.com/DaiYuANg/arcgo/tree/main/examples/dix/inspect) |

## Full-stack backend

End-to-end service shape: **configx** (dotenv → file → env), **logx**, **eventx**, **httpx** (chi + Huma), **dix**, **dbx** (SQLite). Layout: `config`, `domain`, `repo`, `service`, `api`, `event`, `db`.

| Example | Focus | Directory |
| --- | --- | --- |
| `backend` | integrated HTTP API + DI + SQLite persistence | [examples/dix/backend](https://github.com/DaiYuANg/arcgo/tree/main/examples/dix/backend) |

```bash
cd examples/dix
go run ./backend
# Optional: APP_SERVER_PORT=3000 APP_DB_DSN=file:app.db
# Open http://localhost:8080/docs
```

From the repository root:

```bash
go run ./examples/dix/backend
```

## Example: Collection Contributions

```go
endpointModule := dix.NewModule("endpoints",
    dix.Providers(
        dix.Provider0(func() *HealthEndpoint {
            return &HealthEndpoint{}
        }, dix.As[Endpoint](), dix.Into[Endpoint](dix.Key("health"), dix.Order(-100))),
        dix.Provider1(func(cfg Config) *UserEndpoint {
            return &UserEndpoint{Config: cfg}
        }, dix.Into[Endpoint](dix.Key("users"), dix.Order(10))),
        dix.Contribute0[Endpoint](func() Endpoint {
            return &MetricsEndpoint{}
        }, dix.Key("metrics"), dix.Order(20)),
    ),
)

serverModule := dix.NewModule("server",
    dix.Imports(endpointModule),
    dix.Providers(
        dix.Provider2(func(routes []Endpoint, byName collectionx.OrderedMap[string, Endpoint]) *Server {
            return &Server{Routes: routes, RoutesByName: byName}
        }),
    ),
)
```

Use `Into[T]` when a normal provider should remain injectable as its concrete output and also join a collection role such as `Endpoint`.
Use `ContributeN[T]` when the provider only exists for that collection role.
Consumers can depend directly on `[]T`, `collectionx.List[T]`, `map[string]T`, `collectionx.Map[string, T]`, or `collectionx.OrderedMap[string, T]`.
`Order(...)` controls slice/list order, and `Key(...)` is required when resolving map-like collections.
`As[T]` is still a unique alias binding, not a multi-binding collection contribution.

## Example: Basic App Composition

```go
app := dix.New(
    "basic",
    dix.UseLogger(logger),
    dix.WithModule(
        dix.NewModule("config",
            dix.Providers(dix.Provider0(func() Config { return Config{Port: 8080} })),
        ),
    ),
)

if err := app.Validate(); err != nil {
    panic(err)
}

rt, err := app.Start(context.Background())
if err != nil {
    panic(err)
}
defer func() {
    _, _ = rt.StopWithReport(context.Background())
}()
```

## Example: Validation Report For Raw Bridges

```go
report := app.ValidateReport()
if err := report.Err(); err != nil {
    panic(err)
}
for _, warning := range report.Warnings {
    logger.Warn("validation warning", "kind", warning.Kind, "module", warning.Module, "label", warning.Label)
}
```

Use this path when the module graph intentionally includes raw bridges.

## Example: Declared Raw Metadata

```go
module := dix.NewModule("bridge",
    dix.Providers(
        dix.Provider0(func() Config { return Config{Port: 8080} }),
        dix.RawProviderWithMetadata(func(c *dix.Container) {
            dix.ProvideValueT(c, &Server{})
        }, dix.ProviderMetadata{
            Label:        "RawServerProvider",
            Output:       dix.TypedService[*Server](),
            Dependencies: []dix.ServiceRef{dix.TypedService[Config]()},
        }),
    ),
    dix.Setups(
        advanced.DoSetupWithMetadata(func(raw do.Injector) error {
            _ = raw
            return nil
        }, dix.SetupMetadata{
            Label:         "RawBridgeSetup",
            Dependencies:  []dix.ServiceRef{dix.TypedService[Config]()},
            GraphMutation: true,
        }),
    ),
)
```

Declaring metadata keeps raw integration possible without making validation completely blind.

## Example: Runtime Scope

```go
requestScope := advanced.Scope(rt, "request-42", func(injector do.Injector) {
    advanced.ProvideScopedValue(injector, RequestContext{RequestID: "req-42"})
    advanced.ProvideScoped2(injector, func(cfg AppConfig, req RequestContext) ScopedService {
        return ScopedService{Config: cfg, Request: req}
    })
})

svc, err := advanced.ResolveScopedAs[ScopedService](requestScope)
if err != nil {
    panic(err)
}
fmt.Println(svc.Request.RequestID)
```

## Example: Error-returning Providers

```go
app := dix.NewApp("errors",
    dix.NewModule("errors",
        dix.Providers(
            dix.ProviderErr0(func() (*Config, error) {
                return loadConfig()
            }),
        ),
        dix.Setups(
            advanced.OverrideErr0(func() (*Config, error) {
                return loadConfigFromFixture()
            }),
        ),
    ),
)

requestScope := advanced.Scope(rt, "request-42", func(injector do.Injector) {
    advanced.ProvideScopedNamedErr0(injector, "tenant.default", func() (string, error) {
        return resolveTenantFromRequest()
    })
})
```

Use the `Err` suffixed helpers when construction can fail and the failure should flow through normal resolution.

## Example: Shortcut APIs

```go
module := dix.NewModule("shortcuts",
    dix.Providers(
        dix.Value(Config{Port: 8080}),
        advanced.Named("locale.default", "en-US"),
        advanced.Transient(func() int { return nextID() }),
    ),
    dix.Invokes(
        dix.Invoke(func() {
            fmt.Println("warmup")
        }),
    ),
    dix.Setups(
        advanced.Alias[*englishGreeter, greeter](),
        advanced.Override(func() string { return "override" }),
    ),
)
```

Use these shortcuts when the registration has no dependencies and the longer explicit names add noise without extra information.

## Example: Fine-Grained Inspection

```go
provided := advanced.ListProvidedServices(rt)
deps := advanced.ExplainNamedDependencies(rt, "tenant.default")

fmt.Println("provided services:", len(provided))
fmt.Println("tenant graph known:", deps["tenant.default"] != "")
```

Use the fine-grained inspection helpers when you only need one diagnostic view.
`InspectRuntime(...)` remains convenient, but it is the heavier aggregation path.

For caller-controlled shutdown, the app-level shortcut is:

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

if err := app.RunContext(ctx); err != nil {
    panic(err)
}
```
