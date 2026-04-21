//revive:disable:file-length-limit Core behavior tests intentionally keep related scenarios in one file.

package dix_test

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/arcgolabs/arcgo/dix"
	"github.com/stretchr/testify/assert"
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

func (d *Database) Connect() error {
	return nil
}

type testGreeter interface {
	Greet() string
}

type testGreeterImpl struct{}

func (g *testGreeterImpl) Greet() string {
	return "hello"
}

type cleanupService struct {
	shutdowns int
}

func (s *cleanupService) Shutdown() error {
	s.shutdowns++
	return nil
}

func (d *Database) Close() error {
	return nil
}

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

func (s *Server) Start() error {
	return nil
}

func (s *Server) Stop(_ context.Context) error {
	return nil
}

var DatabaseModule = dix.NewModule("database",
	dix.WithModuleProviders(
		dix.Provider0(ProvideConfig),
		dix.Provider1(ProvideDatabase),
	),
	dix.WithModuleHooks(
		dix.OnStart(func(_ context.Context, db *Database) error {
			return db.Connect()
		}),
		dix.OnStop(func(_ context.Context, db *Database) error {
			return db.Close()
		}),
	),
)

var ServerModule = dix.NewModule("server",
	dix.WithModuleProviders(
		dix.Provider1(ProvideServer),
	),
	dix.WithModuleImports(DatabaseModule),
	dix.WithModuleHooks(
		dix.OnStart(func(_ context.Context, s *Server) error {
			return s.Start()
		}),
		dix.OnStop(func(ctx context.Context, s *Server) error {
			return s.Stop(ctx)
		}),
	),
)

func buildRuntime(t *testing.T, app *dix.App) *dix.Runtime {
	t.Helper()
	rt, err := app.Build()
	require.NoError(t, err)
	require.NotNil(t, rt)
	return rt
}

func newDebugLogger() (*slog.Logger, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	return logger, buf
}

func loggerModule(logger *slog.Logger) dix.Module {
	return dix.NewModule("logger",
		dix.WithModuleProvider(dix.Value(logger)),
	)
}

func TestApp_Build(t *testing.T) {
	app := dix.NewApp("testapp", ServerModule)
	rt := buildRuntime(t, app)

	db, err := dix.ResolveAs[*Database](rt.Container())
	require.NoError(t, err)
	assert.NotNil(t, db)
	assert.Equal(t, "sqlite://test.db", db.dsn)

	cfg, err := dix.ResolveAs[Config](rt.Container())
	require.NoError(t, err)
	assert.Equal(t, 8080, cfg.Port)
	assert.Equal(t, dix.AppStateBuilt, rt.State())
}

func TestApp_BuildCreatesIndependentRuntimes(t *testing.T) {
	app := dix.NewApp("testapp", ServerModule)

	first := buildRuntime(t, app)
	second := buildRuntime(t, app)

	assert.NotSame(t, first, second)
	assert.NotSame(t, first.Container(), second.Container())
	assert.Equal(t, first.Name(), second.Name())
}

func TestRuntime_StartStop(t *testing.T) {
	rt := buildRuntime(t, dix.NewApp("testapp", ServerModule))
	ctx := context.Background()

	require.Equal(t, dix.AppStateBuilt, rt.State())
	require.NoError(t, rt.Start(ctx))
	assert.Equal(t, dix.AppStateStarted, rt.State())

	require.NoError(t, rt.Stop(ctx))
	assert.Equal(t, dix.AppStateStopped, rt.State())
}

func TestModule_WithProfiles(t *testing.T) {
	devOnlyCalled := false
	prodOnlyCalled := false

	devModule := dix.NewModule("dev-only",
		dix.WithModuleProfiles(dix.ProfileDev),
		dix.WithModuleProviders(
			dix.Provider0(func() string {
				devOnlyCalled = true
				return "dev"
			}),
		),
	)

	prodModule := dix.NewModule("prod-only",
		dix.WithModuleProfiles(dix.ProfileProd),
		dix.WithModuleProviders(
			dix.Provider0(func() string {
				prodOnlyCalled = true
				return "prod"
			}),
		),
	)

	appDev := dix.New("test",
		dix.WithProfile(dix.ProfileDev),
		dix.WithModules(devModule, prodModule),
	)

	rtDev := buildRuntime(t, appDev)
	devStr, err := dix.ResolveAs[string](rtDev.Container())
	require.NoError(t, err)
	assert.Equal(t, "dev", devStr)
	assert.True(t, devOnlyCalled)
	assert.False(t, prodOnlyCalled)

	appProd := dix.New("test",
		dix.WithProfile(dix.ProfileProd),
		dix.WithModules(devModule, prodModule),
	)

	rtProd := buildRuntime(t, appProd)
	prodStr, err := dix.ResolveAs[string](rtProd.Container())
	require.NoError(t, err)
	assert.Equal(t, "prod", prodStr)
}

func TestDIProvidedProfileSelectsProfileModules(t *testing.T) {
	devOnlyCalled := false
	prodOnlyCalled := false

	configModule := dix.NewModule("profile-config",
		dix.WithModuleProviders(
			dix.Provider0(func() dix.Profile {
				return dix.ProfileProd
			}),
		),
	)
	devModule := dix.NewModule("di-dev-only",
		dix.WithModuleProfiles(dix.ProfileDev),
		dix.WithModuleProviders(
			dix.Provider0(func() string {
				devOnlyCalled = true
				return "dev"
			}),
		),
	)
	prodModule := dix.NewModule("di-prod-only",
		dix.WithModuleProfiles(dix.ProfileProd),
		dix.WithModuleProviders(
			dix.Provider0(func() string {
				prodOnlyCalled = true
				return "prod"
			}),
		),
	)

	app := dix.New("di-profile",
		dix.WithModules(configModule, devModule, prodModule),
	)

	rt := buildRuntime(t, app)
	assert.Equal(t, dix.ProfileProd, rt.Profile())

	resolvedProfile, err := dix.ResolveAs[dix.Profile](rt.Container())
	require.NoError(t, err)
	assert.Equal(t, dix.ProfileProd, resolvedProfile)

	prodStr, err := dix.ResolveAs[string](rt.Container())
	require.NoError(t, err)
	assert.Equal(t, "prod", prodStr)
	assert.False(t, devOnlyCalled)
	assert.True(t, prodOnlyCalled)
}

func TestExplicitProfileTakesPriorityOverDIProvidedProfile(t *testing.T) {
	devOnlyCalled := false
	prodOnlyCalled := false

	configModule := dix.NewModule("profile-config",
		dix.WithModuleProviders(
			dix.Provider0(func() dix.Profile {
				return dix.ProfileProd
			}),
		),
	)
	devModule := dix.NewModule("explicit-dev-only",
		dix.WithModuleProfiles(dix.ProfileDev),
		dix.WithModuleProviders(
			dix.Provider0(func() string {
				devOnlyCalled = true
				return "dev"
			}),
		),
	)
	prodModule := dix.NewModule("explicit-prod-only",
		dix.WithModuleProfiles(dix.ProfileProd),
		dix.WithModuleProviders(
			dix.Provider0(func() string {
				prodOnlyCalled = true
				return "prod"
			}),
		),
	)

	app := dix.New("explicit-profile",
		dix.WithProfile(dix.ProfileDev),
		dix.WithModules(configModule, devModule, prodModule),
	)

	rt := buildRuntime(t, app)
	assert.Equal(t, dix.ProfileDev, rt.Profile())

	resolvedProfile, err := dix.ResolveAs[dix.Profile](rt.Container())
	require.NoError(t, err)
	assert.Equal(t, dix.ProfileDev, resolvedProfile)

	devStr, err := dix.ResolveAs[string](rt.Container())
	require.NoError(t, err)
	assert.Equal(t, "dev", devStr)
	assert.True(t, devOnlyCalled)
	assert.False(t, prodOnlyCalled)
}

func TestModule_WithExcludeProfiles(t *testing.T) {
	called := false

	module := dix.NewModule("not-test",
		dix.WithModuleExcludeProfiles(dix.ProfileTest),
		dix.WithModuleProviders(
			dix.Provider0(func() string {
				called = true
				return "value"
			}),
		),
	)

	appTest := dix.New("test",
		dix.WithProfile(dix.ProfileTest),
		dix.WithModule(module),
	)

	rt := buildRuntime(t, appTest)
	_, err := dix.ResolveAs[string](rt.Container())
	assert.Error(t, err)
	assert.False(t, called)
}

func TestResolveAs(t *testing.T) {
	rt := buildRuntime(t, dix.NewApp("test",
		dix.NewModule("test",
			dix.WithModuleProviders(
				dix.Provider0(ProvideConfig),
			),
		),
	))

	cfg, err := dix.ResolveAs[Config](rt.Container())
	require.NoError(t, err)
	assert.Equal(t, 8080, cfg.Port)
}

func TestMustResolveAs(t *testing.T) {
	rt := buildRuntime(t, dix.NewApp("test",
		dix.NewModule("test",
			dix.WithModuleProviders(
				dix.Provider0(ProvideConfig),
			),
		),
	))

	cfg := dix.MustResolveAs[Config](rt.Container())
	assert.Equal(t, 8080, cfg.Port)
}

func TestInvoke(t *testing.T) {
	invoked := false

	rt := buildRuntime(t, dix.NewApp("test",
		dix.NewModule("test",
			dix.WithModuleProviders(
				dix.Provider0(ProvideConfig),
			),
			dix.WithModuleInvokes(
				dix.Invoke1(func(cfg Config) {
					invoked = true
					assert.Equal(t, 8080, cfg.Port)
				}),
			),
		),
	))

	assert.True(t, invoked)
	assert.NotNil(t, rt)
}
