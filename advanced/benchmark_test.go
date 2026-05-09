//revive:disable:file-length-limit Advanced benchmark coverage is grouped to keep scenario setup shared.

package advanced_test

import (
	"context"
	"log/slog"
	"strconv"
	"testing"

	"github.com/arcgolabs/dix"
	dixadvanced "github.com/arcgolabs/dix/advanced"
)

type benchConfig struct {
	DSN      string
	CacheURL string
	Port     int
}

type benchDatabase struct {
	dsn string
}

type benchCache struct {
	url string
}

type benchRepository struct {
	db    *benchDatabase
	cache *benchCache
}

type benchService struct {
	repo   *benchRepository
	config benchConfig
	logger *slog.Logger
}

type benchHandler struct {
	service *benchService
}

type benchGreeter interface {
	Greet() string
}

type benchEnglishGreeter struct {
	logger *slog.Logger
}

func (g *benchEnglishGreeter) Greet() string { return "hello" }

type benchTransientToken struct {
	service *benchService
}

type benchRequestContext struct {
	RequestID string
}

type benchScopedHandler struct {
	service *benchService
	req     benchRequestContext
}

func (s *benchService) Start(context.Context) error { return nil }
func (s *benchService) Stop(context.Context) error  { return nil }

func benchmarkLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func newBenchmarkModules() (dix.Module, dix.Module, dix.Module) {
	coreModule := dix.NewModule("core",
		dix.WithModuleProviders(
			dix.Provider0(func() benchConfig {
				return benchConfig{
					DSN:      "postgres://bench",
					CacheURL: "redis://bench",
					Port:     8080,
				}
			}),
			dix.Provider1(func(cfg benchConfig) *benchDatabase {
				return &benchDatabase{dsn: cfg.DSN}
			}),
			dix.Provider1(func(cfg benchConfig) *benchCache {
				return &benchCache{url: cfg.CacheURL}
			}),
		),
	)

	domainModule := dix.NewModule("domain",
		dix.WithModuleImports(coreModule),
		dix.WithModuleProviders(
			dix.Provider2(func(db *benchDatabase, cache *benchCache) *benchRepository {
				return &benchRepository{db: db, cache: cache}
			}),
			dix.Provider3(func(repo *benchRepository, cfg benchConfig, logger *slog.Logger) *benchService {
				return &benchService{repo: repo, config: cfg, logger: logger}
			}),
		),
	)

	transportModule := dix.NewModule("transport",
		dix.WithModuleImports(domainModule),
		dix.WithModuleProviders(
			dix.Provider1(func(service *benchService) *benchHandler {
				return &benchHandler{service: service}
			}),
		),
		dix.WithModuleHooks(
			dix.OnStart(func(ctx context.Context, service *benchService) error {
				return service.Start(ctx)
			}),
			dix.OnStop(func(ctx context.Context, service *benchService) error {
				return service.Stop(ctx)
			}),
		),
		dix.WithModuleSetup(func(c *dix.Container, _ dix.Lifecycle) error {
			c.RegisterReadinessCheck("service", func(_ context.Context) error {
				_, err := dix.ResolveAs[*benchService](c)
				return err
			})
			return nil
		}),
	)

	return coreModule, domainModule, transportModule
}

func newAdvancedBenchmarkApp() *dix.App {
	coreModule, domainModule, transportModule := newBenchmarkModules()
	advancedModule := dix.NewModule("advanced",
		dix.WithModuleImports(transportModule),
		dix.WithModuleProviders(
			dixadvanced.NamedValue(dix.NamedService[string]("locale.default"), "en-US"),
			dix.Provider1(func(logger *slog.Logger) *benchEnglishGreeter {
				return &benchEnglishGreeter{logger: logger}
			}),
			dixadvanced.NamedProvider1(dix.NamedService[*benchEnglishGreeter]("greeter.en"), func(logger *slog.Logger) *benchEnglishGreeter {
				return &benchEnglishGreeter{logger: logger}
			}),
			dixadvanced.TransientProvider1[*benchTransientToken, *benchService](func(service *benchService) *benchTransientToken {
				return &benchTransientToken{service: service}
			}),
		),
		dix.WithModuleSetups(
			dixadvanced.BindAlias[*benchEnglishGreeter, benchGreeter](),
			dixadvanced.BindNamedAlias(
				dix.NamedService[*benchEnglishGreeter]("greeter.en"),
				dix.NamedService[benchGreeter]("greeter.en.alias"),
			),
		),
	)

	return dix.New(
		"advanced-benchmark",
		dix.WithModules(loggerModule(benchmarkLogger()), coreModule, domainModule, transportModule, advancedModule),
	)
}

func buildAdvancedBenchmarkRuntime(b *testing.B) *dix.Runtime {
	b.Helper()

	rt, err := newAdvancedBenchmarkApp().Build()
	if err != nil {
		b.Fatal(err)
	}

	return rt
}

var (
	benchGreeterSink        benchGreeter
	benchNamedValueSink     string
	benchTransientTokenSink *benchTransientToken
	benchScopedHandlerSink  benchScopedHandler
	benchInspectionSink     dixadvanced.Inspection
)

func BenchmarkAdvancedResolveNamed(b *testing.B) {
	rt := buildAdvancedBenchmarkRuntime(b)

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		value, err := dix.ResolveKey(rt.Container(), dix.NamedService[string]("locale.default"))
		if err != nil {
			b.Fatal(err)
		}
		benchNamedValueSink = value
	}
}

func BenchmarkAdvancedResolveAssignable(b *testing.B) {
	rt := buildAdvancedBenchmarkRuntime(b)

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		greeter, err := dixadvanced.ResolveAssignableAs[benchGreeter](rt.Container())
		if err != nil {
			b.Fatal(err)
		}
		benchGreeterSink = greeter
	}
}

func BenchmarkAdvancedResolveTypedAlias(b *testing.B) {
	rt := buildAdvancedBenchmarkRuntime(b)

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		greeter, err := dix.ResolveAs[benchGreeter](rt.Container())
		if err != nil {
			b.Fatal(err)
		}
		benchGreeterSink = greeter
	}
}

func BenchmarkAdvancedResolveTransient(b *testing.B) {
	rt := buildAdvancedBenchmarkRuntime(b)

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		token, err := dix.ResolveAs[*benchTransientToken](rt.Container())
		if err != nil {
			b.Fatal(err)
		}
		benchTransientTokenSink = token
	}
}

func BenchmarkAdvancedInspectRuntime(b *testing.B) {
	rt := buildAdvancedBenchmarkRuntime(b)

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		benchInspectionSink = dixadvanced.InspectRuntime(rt, "locale.default", "greeter.en.alias")
	}
}

func BenchmarkAdvancedExplainScopeTree(b *testing.B) {
	rt := buildAdvancedBenchmarkRuntime(b)

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		benchInspectionSink.ScopeTree = dixadvanced.ExplainScopeTree(rt)
	}
}

func BenchmarkAdvancedListProvidedServices(b *testing.B) {
	rt := buildAdvancedBenchmarkRuntime(b)

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		benchInspectionSink.ProvidedServices = dixadvanced.ListProvidedServices(rt)
	}
}

func BenchmarkAdvancedExplainNamedDependencies(b *testing.B) {
	rt := buildAdvancedBenchmarkRuntime(b)

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		benchInspectionSink.NamedDependencies = dixadvanced.ExplainNamedDependencies(rt, "locale.default", "greeter.en.alias")
	}
}

func BenchmarkAdvancedInspectRuntimeLight(b *testing.B) {
	rt := buildAdvancedBenchmarkRuntime(b)
	opts := dixadvanced.InspectOptions{
		IncludeNamedDeps: true,
	}

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		benchInspectionSink = dixadvanced.InspectRuntimeWithOptions(rt, opts, "locale.default", "greeter.en.alias")
	}
}

func BenchmarkAdvancedScopeResolve(b *testing.B) {
	rt := buildAdvancedBenchmarkRuntime(b)
	names := make([]string, b.N)
	for i := range b.N {
		names[i] = "request-bench-" + strconv.Itoa(i)
	}
	scopePackage := dix.ScopeFunc(func(c *dix.Container) {
		dix.ProvideValueT(c, benchRequestContext{RequestID: "req-42"})
		dix.Provide2T(c, func(service *benchService, req benchRequestContext) benchScopedHandler {
			return benchScopedHandler{service: service, req: req}
		})
	})

	b.ReportAllocs()
	b.ResetTimer()

	for i := range b.N {
		scope, err := rt.Scope(names[i], scopePackage)
		if err != nil {
			b.Fatal(err)
		}
		handler, err := dix.ResolveAs[benchScopedHandler](scope)
		if err != nil {
			b.Fatal(err)
		}
		benchScopedHandlerSink = handler
	}
}
