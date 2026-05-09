## dix

`dix` is a strongly typed, module-oriented application framework built on top of `do`.
It provides an immutable app spec, typed providers and invokes, lifecycle hooks, validation,
and a runtime model without forcing most users to deal with `do` directly.

## Current capabilities

- **Immutable spec**: `App` and `Module` are built as declarative specs.
- **Typed DI**: `ProviderN` registers typed constructors; `InvokeN` runs typed eager initialization.
- **Collection contributions**: `Into[T]` and `ContributeN[T]` collect distributed providers into typed slices, maps, and `collectionx` containers.
- **Lifecycle**: `OnStart` / `OnStop` hooks with priority, opt-in parallel execution, and `Runtime.Start/Stop/StopWithReport`.
- **Validation**: `app.Validate()` fails on graph errors; `app.ValidateReport()` also exposes validation warnings and missing-dependency suggestions for raw escape hatches.
- **Dependency graph**: `app.DependencyGraph()` / `app.Explain()` expose a static build graph, Graphviz DOT output, and topological order.
- **SubApps**: `SubApps(...)` builds child apps in child `do` scopes while inheriting parent services.
- **Runtime**: container access, health checks, lifecycle summaries, subapp summaries, and diagnostics.
- **Advanced features**: named services, alias binding, transient providers, overrides, scopes via `dix/advanced`.

## Package layout

- Default path: `github.com/arcgolabs/dix`
- Runtime metrics subpackage: `github.com/arcgolabs/dix/metrics`
- Advanced container features: `github.com/arcgolabs/dix/advanced`
- Test helpers: `github.com/arcgolabs/dix/testx`

## Documentation map

- Minimal module graph: [Getting Started](./getting-started)
- Runtime metrics and observability: [Metrics and observability](./metrics-and-observability)
- Health checks and HTTP handlers: [Health and lifecycle](./health-and-lifecycle)
- Fallible provider construction: [Error-returning providers](./error-providers)
- Release notes: [dix v0.5.0](./release-v0.5.0)
- Release notes: [dix v0.4.0](./release-v0.4.0)
- Release notes: [dix v0.3.0](./release-v0.3.0)
- Runnable example index: [dix examples](./examples)

## Install / Import

```bash
go get github.com/arcgolabs/dix@latest
```

## Key API surface (summary)

- `dix.New(name, ...)` / `dix.NewDefault(...)`
- `dix.NewModule(name, ...)`
- `dix.Modules(...)`, `dix.UseProfile(...)`, `dix.Version(...)`, `dix.UseLogger(...)`, `dix.LoggerFrom(...)`, `dix.UseLogger0/1(...)`
- `dix.UseEventLogger(...)`, `dix.UseEventLogger0/1(...)`
- `dix.WithObserver(...)` / `dix.WithObservers(...)`
- `dix.Providers(...)`, `dix.Hooks(...)`, `dix.Imports(...)`, `dix.Setups(...)`
- `dix.WithModules(...)`, `dix.WithProfile(...)`, `dix.WithVersion(...)`, `dix.WithLogger(...)`, `dix.WithLoggerFrom(...)`
- `dix.WithModuleProviders(...)`, `dix.WithModuleHooks(...)`, `dix.WithModuleImports(...)`
- `dix.WithModuleProvider(...)`, `dix.WithModuleHook(...)`, `dix.WithModuleImport(...)`
- `dix.Value(...)`, `dix.Invoke(...)`, `dix.ProviderN(...)`, `dix.OnStart(...)`, `dix.OnStop(...)`
- `dix.ResolveAs(...)`, `dix.ResolveAsContext(ctx, ...)`
- `dix.Eager(...)`
- `dix.LifecycleName(...)`, `dix.LifecyclePriority(...)`, `dix.LifecycleParallel(...)`, `dix.LifecycleTimeout(...)`, `dix.LifecycleConcurrency(...)`
- `dix.As[T]()`, `dix.Into[T](...)`, `dix.Key(...)`, `dix.Order(...)`, `dix.ContributeN[T](...)`
- `dix.SubApps(...)`, `dix.NewSubApp(...)`, `app.DependencyGraph()`, `app.Explain()`
- `app.Test(...)`, `dix.TestValue(...)`, `dix.TestProviders(...)`, `dix.TestDisableModules(...)`
- `rt.LifecycleSummary()`, `rt.SubAppSummaries()`, `rt.ScopePath()`
- `advanced.Named(...)`, `advanced.Alias(...)`, `advanced.NamedAlias(...)`, `advanced.Transient(...)`, `advanced.Override(...)`
- `testx.Validate(t, app)`, `testx.Build(t, app)`, `testx.Start(ctx, t, app)`
- `app.Validate()`, `app.ValidateReport()`, `app.Build()`, `app.Start(ctx)`, `app.RunContext(ctx)`
- `rt.Start(ctx)`, `rt.Stop(ctx)`, `rt.StopWithReport(ctx)`

## API style notes

- `dix` keeps the existing `WithModule*` option family for compatibility.
- `dix` also keeps the existing `WithProfile` / `WithVersion` / `WithLogger` / `WithModules` app option family for compatibility.
- New code can prefer the shorter module option aliases such as `Providers(...)`, `Hooks(...)`, `Imports(...)`, `Invokes(...)`, `Setups(...)`, `Description(...)`, and `Tags(...)`.
- Framework logger priority is: internal default, module-provided or resolver-provided `*slog.Logger`, then direct `UseLogger(...)` / `WithLogger(...)`. `UseEventLogger...` can still replace the internal event logger.
- `WithLoggerFrom...` remains supported for custom resolver flows, but a normal logger should live in the module graph.
- `Observers(...)` remain the extension path for sidecar consumers such as metrics, not the primary framework logger hook.
- For zero-dependency registrations, `Value(...)` and `Invoke(...)` reduce the remaining boilerplate on the core path.
- Use `As[T]` for a unique typed alias, and `Into[T]` / `ContributeN[T]` for multi-binding collection roles. Collection consumers can depend directly on `[]T`, `collectionx.List[T]`, `map[string]T`, `collectionx.Map[string, T]`, or `collectionx.OrderedMap[string, T]`.
- Use `SubApps(...)` when a child app should share parent services but keep its own modules, lifecycle hooks, and child `do` scope.
- Lifecycle hooks start by ascending priority and stop by descending priority. Hooks remain serial unless `LifecycleParallel()` is set, in which case adjacent hooks with the same priority can run through the configured lifecycle worker pool. Use `LifecycleTimeout(...)` to pass an individual hook a deadline-aware context.
- In `dix/advanced`, the shorter aliases such as `Named(...)`, `Alias(...)`, `Transient(...)`, and `Override(...)` keep the same semantics as the older explicit names.
- When you want the common build-then-start flow, prefer `app.Start(ctx)`; use `app.Build()` when you need an explicit pre-start runtime handle.
- When the caller owns cancellation or shutdown timing, prefer `app.RunContext(ctx)` over `app.Run()`.

## Validation model

- Use `app.Validate()` when you only care about hard failures.
- Use `app.ValidateReport()` when you also want to inspect warnings from raw providers, raw invokes, raw hooks, or raw setups.
- Typed `ProviderN` / `InvokeN` / `OnStart` / `OnStop` stay on the strict validation path.
- Raw escape hatches are still supported, but you should prefer the metadata-aware forms such as `RawProviderWithMetadata(...)`, `RawInvokeWithMetadata(...)`, `RawHookWithMetadata(...)`, `RawSetupWithMetadata(...)`, and `advanced.DoSetupWithMetadata(...)` so the validator can keep reasoning about dependencies and graph mutations.
- Missing dependency errors include nearby available service names to make split-module and renamed-service issues easier to diagnose.

## Dependency graph and inspection

```go
explanation, err := app.Explain()
if err != nil {
	panic(err)
}

fmt.Println(explanation.String())
fmt.Println(explanation.Graph.DOT())

order, err := explanation.Graph.TopologicalOrder()
if err != nil {
	panic(err)
}
fmt.Println("nodes:", order.Len())
```

`DependencyGraph.Directed()` returns a clone of the underlying `collectionx/graph` directed graph for callers that need lower-level traversal.
`advanced.InspectRuntime(...)` also includes the dependency graph, lifecycle summary, and subapp summaries.

## Lifecycle scheduling and build logs

```go
app := dix.New("worker",
	dix.LifecycleConcurrency(4),
	dix.Modules(dix.NewModule("runtime",
		dix.Hooks(
			dix.OnStart0(startCache, dix.LifecycleName("cache"), dix.LifecyclePriority(10), dix.LifecycleParallel()),
			dix.OnStart0(startQueue, dix.LifecycleName("queue"), dix.LifecyclePriority(10), dix.LifecycleParallel(), dix.LifecycleTimeout(10 * time.Second)),
			dix.OnStop0(stopQueue, dix.LifecycleName("queue"), dix.LifecyclePriority(10), dix.LifecycleParallel()),
			dix.OnStop0(stopCache, dix.LifecycleName("cache"), dix.LifecyclePriority(10), dix.LifecycleParallel()),
		),
	)),
)
```

Build, start, stop, setup, invoke, provider construction, and explicit `ResolveAsContext` diagnostics include duration fields when debug logging is enabled.

## Eager providers

Providers stay lazy by default. Use `Eager()` when build should fail fast and construct a service before runtime start:

```go
dbModule := dix.NewModule("db",
	dix.Providers(
		dix.Provider1(NewDatabase, dix.Eager()),
	),
)
```

Eager warmup runs after provider registration and setup, before invokes and subapp build. Raw eager providers must declare `ProviderMetadata.Output` so `dix` knows which service to resolve.

## Integration guide

- **configx**: load typed config once, then provide it as dependencies in modules.
- **logx**: initialize one process logger and inject into service modules.
- **observabilityx**: use `dix/metrics` to emit build/start/stop/health/state-transition metrics to Prometheus or OpenTelemetry.
- **httpx**: do HTTP bootstrap in setup/hook stages; keep route registration in dedicated modules.
- **dbx / kvx**: isolate persistence setup into infra modules.

## Testing and benchmarks

```bash
go test ./...
go test -run ^$ -bench . -benchmem
```

For package tests, `dix/testx` provides small helpers that fail `testing.TB` on validation, build, start, or cleanup errors.

```go
runtime := testx.BuildWith(t, app,
	dix.TestDisableModules("db"),
	dix.TestValue[Store](mockStore),
)
```

`app.Test(...)` derives a test-only App without mutating the original spec. Test providers replace matching provider outputs or aliases before validation, so disabled modules can be replaced with mocks while keeping the normal static dependency checks.

## Production notes

- Keep module boundaries domain-driven; avoid large all-in-one modules.
- Fail fast on validate/build errors before runtime start.
- Use `StopWithReport` when teardown visibility matters.
