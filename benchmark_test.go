package dix_test

import (
	"context"
	"log/slog"
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

func newBenchmarkApp() *dix.App {
	coreModule, domainModule, transportModule := newBenchmarkModules()

	return dix.New(
		"benchmark",
		dix.WithModules(loggerModule(benchmarkLogger()), coreModule, domainModule, transportModule),
	)
}

func newAdvancedBenchmarkApp() *dix.App {
	coreModule, domainModule, transportModule := newBenchmarkModules()
	advancedModule := dix.NewModule("advanced",
		dix.WithModuleImports(transportModule),
		dix.WithModuleProviders(
			dixadvanced.NamedValue("locale.default", "en-US"),
			dix.Provider1(func(logger *slog.Logger) *benchEnglishGreeter {
				return &benchEnglishGreeter{logger: logger}
			}),
			dixadvanced.NamedProvider1[*benchEnglishGreeter, *slog.Logger]("greeter.en", func(logger *slog.Logger) *benchEnglishGreeter {
				return &benchEnglishGreeter{logger: logger}
			}),
			dixadvanced.TransientProvider1[*benchTransientToken, *benchService](func(service *benchService) *benchTransientToken {
				return &benchTransientToken{service: service}
			}),
		),
		dix.WithModuleSetups(
			dixadvanced.BindAlias[*benchEnglishGreeter, benchGreeter](),
			dixadvanced.BindNamedAlias[*benchEnglishGreeter, benchGreeter]("greeter.en", "greeter.en.alias"),
		),
	)

	return dix.New(
		"advanced-benchmark",
		dix.WithModules(loggerModule(benchmarkLogger()), coreModule, domainModule, transportModule, advancedModule),
	)
}

func buildBenchmarkRuntime(b *testing.B) *dix.Runtime {
	b.Helper()

	rt, err := newBenchmarkApp().Build()
	if err != nil {
		b.Fatal(err)
	}

	return rt
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
	benchServiceSink        *benchService
	benchHandlerSink        *benchHandler
	benchGreeterSink        benchGreeter
	benchNamedValueSink     string
	benchTransientTokenSink *benchTransientToken
	benchScopedHandlerSink  benchScopedHandler
	benchInspectionSink     dixadvanced.Inspection
)
