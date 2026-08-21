## dix

`dix` is a strongly typed, module-oriented application framework built on top of `do`.
It provides an immutable app spec, typed providers and invokes, lifecycle hooks, validation,
and a runtime model without forcing most users to deal with `do` directly.

## Current capabilities

- **Immutable spec**: `App` and `Module` are built as declarative specs.
- **Typed DI**: `ProviderN` and `ModuleBuilder` register typed constructors; `Conditional[T]` models optional runtime services; `InvokeN` runs typed eager initialization.
- **Collection contributions**: `Into[T]` and `ContributeN[T]` collect distributed providers into typed slices, maps, and `collectionx` containers.
- **Lifecycle**: `OnStart` / `OnStop` hooks with priority, opt-in parallel execution, and `Runtime.Start/Stop/StopWithReport`.
- **Validation**: `app.Validate()` fails on graph errors; `app.ValidateReport()` also exposes validation warnings, warning counts, service declaration counts, and missing-dependency suggestions for raw escape hatches.
- **Dependency graph**: `app.DependencyGraph()` / `app.Explain()` expose a static build graph, Graphviz DOT output, topological order, service node indexes, and module-service relation tables.
- **SubApps**: `SubApps(...)` builds child apps in child `do` scopes while inheriting parent services.
- **Runtime**: container access, health checks, lifecycle summaries, subapp summaries, diagnostics, and optional recent-event buffers.
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
- `dix.NewModuleBuilder(name).Provide(...).Contribute(...).Build()`
- `dix.Modules(...)`, `dix.UseProfile(...)`, `dix.Version(...)`, `dix.UseLogger(...)`, `dix.LoggerFrom(...)`, `dix.UseLogger0/1(...)`
- `dix.UseEventLogger(...)`, `dix.UseEventLogger0/1(...)`
- `dix.WithObserver(...)` / `dix.WithObservers(...)`
- `dix.RecentEvents(capacity)`, `dix.NewEventRecorder(capacity)`
- `dix.Providers(...)`, `dix.Hooks(...)`, `dix.Imports(...)`, `dix.Setups(...)`
- `dix.WithModules(...)`, `dix.WithProfile(...)`, `dix.WithVersion(...)`, `dix.WithLogger(...)`, `dix.WithLoggerFrom(...)`
- `dix.WithModuleProviders(...)`, `dix.WithModuleHooks(...)`, `dix.WithModuleImports(...)`
- `dix.WithModuleProvider(...)`, `dix.WithModuleHook(...)`, `dix.WithModuleImport(...)`
- `dix.Value(...)`, `dix.Provider(...)`, `dix.ProviderErr(...)`, `dix.Invoke(...)`, `dix.ProviderN(...)`, `dix.ConditionalProviderN(...)`, `dix.OnStart(...)`, `dix.OnStop(...)`
- `dix.ResolveAs(...)`, `dix.ResolveKey(...)`, `dix.ResolveAsContext(ctx, ...)`, `dix.ResolveKeyContext(ctx, ...)`
- `container.Resolve[T]()`, `container.ResolveKey(key)`, `container.MustResolve[T]()`, `container.ResolveOptional[T]()`
- `dix.Eager(...)`, `dix.When(...)`, `dix.Unless(...)`, `dix.ProfileIs(...)`, `dix.EnvEquals(...)`, `dix.EnvSet(...)`
- `dix.LifecycleName(...)`, `dix.LifecycleAfter(...)`, `dix.LifecycleBefore(...)`, `dix.LifecyclePriority(...)`, `dix.LifecycleParallel(...)`, `dix.LifecycleTimeout(...)`, `dix.LifecycleConcurrency(...)`
- `dix.As[T]()`, `dix.Into[T](...)`, `dix.Key(...)`, `dix.Order(...)`, `dix.ContributeN[T](...)`
- `dix.SubApps(...)`, `dix.NewSubApp(...)`, `app.DependencyGraph()`, `app.Explain()`
- `rt.Scope(...)`, `dix.ScopeFunc(...)`, `dix.NamedService[T](...)`, `dix.ProvideKeyValue(...)`, `dix.ProvideKey(...)`, `dix.ProvideKey1(...)`
- `app.Test(...)`, `dix.TestValue(...)`, `dix.TestProviders(...)`, `dix.TestDisableModules(...)`
- `rt.LifecycleSummary()`, `rt.SubAppSummaries()`, `rt.ScopePath()`
- `rt.RecentEvents()`, `rt.EventRecorder()`, `dix.RuntimeEventRecordsOf[T](...)`, `dix.EventValuesOf[T](...)`
- `report.WarningKindCounts()`, `report.DeclaredServiceCounts()`, `graph.RelationTable()`, `graph.ServiceNodeIndex()`, `graph.ModuleNodeIndex()`
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
- Use `NamedService[T](name)` to create a typed key for named services, then call `key.Resolve(container)`, `key.Provide(container, ...)`, or the corresponding `Container` methods. Public named-service registration and resolution APIs intentionally avoid bare `string` names.
- Prefer `container.Resolve[T]()` and `ServiceKey[T]` methods in new code. The older top-level resolve and registration helpers remain available as low-level compatibility entry points during the v1 migration.
- Use `Conditional[T]` for optional runtime services. It is an alias of `mo.Option[T]`; a provider can return `mo.Some(value)` or `mo.None[T]()` after inspecting injected dependencies, and downstream consumers must explicitly accept `Conditional[T]`.
- Use `When(...)` / `Unless(...)` for build-time provider pruning based on static inputs such as bools, zero-argument functions, profile functions, condition contexts, or env helpers. Do not use `When` for DI-dependent decisions; return `Conditional[T]` instead.
- Use `As[T]` for a unique typed alias, and `Into[T]` / `ContributeN[T]` for multi-binding collection roles. Collection consumers can depend directly on `[]T`, `collectionx.List[T]`, `map[string]T`, `collectionx.Map[string, T]`, or `collectionx.OrderedMap[string, T]`.
- Use `SubApps(...)` when a child app should share parent services but keep its own modules, lifecycle hooks, and child `do` scope.
- Lifecycle hooks start by ascending priority and stop by descending priority. Use `LifecycleAfter(...)` / `LifecycleBefore(...)` for named hook dependencies; priority and declaration order remain the tie-breakers. Hooks remain serial unless `LifecycleParallel()` is set, in which case adjacent unconstrained hooks with the same priority can run through the configured lifecycle worker pool. Use `LifecycleTimeout(...)` to pass an individual hook a deadline-aware context.
- In `dix/advanced`, the shorter aliases such as `Named(...)`, `Alias(...)`, `Transient(...)`, and `Override(...)` keep the same semantics as the older explicit names.
- When you want the common build-then-start flow, prefer `app.Start(ctx)`; use `app.Build()` when you need an explicit pre-start runtime handle.
- When the caller owns cancellation or shutdown timing, prefer `app.RunContext(ctx)` over `app.Run()`.

## Validation model

- Use `app.Validate()` when you only care about hard failures.
- Use `app.ValidateReport()` when you also want to inspect warnings from raw providers, raw invokes, raw hooks, or raw setups.
- Typed `ProviderN` / `InvokeN` / `OnStart` / `OnStop` stay on the strict validation path.
- Raw escape hatches are still supported, but you should prefer the metadata-aware forms such as `RawProviderWithMetadata(...)`, `RawInvokeWithMetadata(...)`, `RawHookWithMetadata(...)`, `RawSetupWithMetadata(...)`, and `advanced.DoSetupWithMetadata(...)` so the validator can keep reasoning about dependencies and graph mutations.
- Missing dependency errors include nearby available service names to make split-module and renamed-service issues easier to diagnose.
- `ValidationReport.WarningKindCounts()` returns a `collectionx/set.MultiSet` keyed by warning kind.
- `ValidationReport.DeclaredServiceCounts()` returns a `collectionx/set.MultiSet` keyed by service name, which helps diagnose duplicate declarations without parsing error strings.

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

relations := explanation.Graph.RelationTable()
serviceIndex := explanation.Graph.ServiceNodeIndex()
moduleIndex := explanation.Graph.ModuleNodeIndex()
_, _ = relations, serviceIndex, moduleIndex
```

`DependencyGraph.Directed()` returns a clone of the underlying `collectionx/graph` directed graph for callers that need lower-level traversal.
`DependencyGraph.RelationTable()` returns a `collectionx/mapping.Table` from `{app,module}` to `{app,service}` to relation kinds such as `consumes`, `provides`, `aliases`, `contributes`, and `overrides`.
`DependencyGraph.ServiceNodeIndex()` returns a `collectionx/mapping.BiMap` between service keys and graph node IDs.
`DependencyGraph.ModuleNodeIndex()` returns a `collectionx/mapping.BiMap` between module keys and graph node IDs.
`advanced.InspectRuntime(...)` also includes the dependency graph, lifecycle summary, and subapp summaries.

## Lifecycle scheduling and build logs

```go
app := dix.New("worker",
	dix.LifecycleConcurrency(4),
	dix.Modules(dix.NewModule("runtime",
		dix.Hooks(
			dix.OnStart0(startCache, dix.LifecycleName("cache"), dix.LifecyclePriority(10), dix.LifecycleParallel()),
			dix.OnStart0(startQueue, dix.LifecycleName("queue"), dix.LifecycleAfter("cache"), dix.LifecyclePriority(10), dix.LifecycleParallel(), dix.LifecycleTimeout(10 * time.Second)),
			dix.OnStop0(stopQueue, dix.LifecycleName("queue"), dix.LifecyclePriority(10), dix.LifecycleParallel()),
			dix.OnStop0(stopCache, dix.LifecycleName("cache"), dix.LifecycleAfter("queue"), dix.LifecyclePriority(10), dix.LifecycleParallel()),
		),
	)),
)
```

Build, start, stop, setup, invoke, provider construction, and explicit `ResolveAsContext` diagnostics include duration fields when debug logging is enabled. Observers can also implement `ProviderObserver`, `ResolveObserver`, or `LifecycleHookObserver` for structured diagnostic events suitable for metrics and tracing.
Use `dix.RecentEvents(capacity)` when a runtime should keep the last N framework events in memory. The runtime stores them in a `collectionx/list.ConcurrentRingBuffer`; `rt.RecentEvents()` returns a FIFO `collectionx` list snapshot, and `rt.RecentEventSnapshot()` exposes the underlying ring-buffer snapshot. Use `RuntimeEventRecordsOf[dix.ResolveEvent](rt)` or `EventValuesOf[dix.BuildEvent](records)` when callers need typed event subsets without manual type switches.

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
