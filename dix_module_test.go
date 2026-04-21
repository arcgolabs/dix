//revive:disable:file-length-limit Module tests intentionally keep related behavior scenarios in one file.

package dix_test

import (
	"context"
	"errors"
	"testing"

	"github.com/arcgolabs/arcgo/dix"
	dixadvanced "github.com/arcgolabs/arcgo/dix/advanced"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModule_Imports(t *testing.T) {
	baseModule := dix.NewModule("base",
		dix.WithModuleProviders(
			dix.Provider0(ProvideConfig),
		),
	)

	dependentModule := dix.NewModule("dependent",
		dix.WithModuleImports(baseModule),
		dix.WithModuleProviders(
			dix.Provider1(ProvideDatabase),
		),
	)

	rt := buildRuntime(t, dix.NewApp("test", dependentModule))

	cfg, err := dix.ResolveAs[Config](rt.Container())
	require.NoError(t, err)

	db, err := dix.ResolveAs[*Database](rt.Container())
	require.NoError(t, err)

	assert.NotNil(t, cfg)
	assert.NotNil(t, db)
}

func TestModule_ImportDeduplicatesSharedDependency(t *testing.T) {
	called := 0

	shared := dix.NewModule("shared",
		dix.WithModuleProviders(
			dix.Provider0(func() string {
				called++
				return "shared"
			}),
		),
	)

	left := dix.NewModule("left", dix.WithModuleImports(shared))
	right := dix.NewModule("right", dix.WithModuleImports(shared))

	rt := buildRuntime(t, dix.NewApp("test", left, right))
	value, err := dix.ResolveAs[string](rt.Container())
	require.NoError(t, err)
	assert.Equal(t, "shared", value)
	assert.Equal(t, 1, called)
}

func TestApp_ValidateDetectsDuplicateModuleNames(t *testing.T) {
	app := dix.NewApp("duplicate-modules",
		dix.NewModule("shared",
			dix.WithModuleProviders(
				dix.Provider0(func() string { return "left" }),
			),
		),
		dix.NewModule("shared",
			dix.WithModuleProviders(
				dix.Provider0(func() int { return 42 }),
			),
		),
	)

	err := app.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate module name detected: shared")
}

func TestApp_ValidateGraph(t *testing.T) {
	app := dix.NewApp("test", DatabaseModule, ServerModule)
	require.NoError(t, app.Validate())
}

func TestApp_ValidateAllowsProviderDependencyDeclaredLater(t *testing.T) {
	app := dix.NewApp("validate-order",
		dix.NewModule("ordered",
			dix.WithModuleProviders(
				dix.Provider1(ProvideDatabase),
				dix.Provider0(ProvideConfig),
			),
		),
	)

	require.NoError(t, app.Validate())

	rt := buildRuntime(t, app)
	db, err := dix.ResolveAs[*Database](rt.Container())
	require.NoError(t, err)
	assert.Equal(t, "sqlite://test.db", db.dsn)
}

func TestApp_ValidateDetectsMissingDependency(t *testing.T) {
	app := dix.NewApp("validate-missing",
		dix.NewModule("broken",
			dix.WithModuleProviders(
				dix.Provider1(func(cfg Config) *Database {
					return &Database{dsn: cfg.DSN}
				}),
			),
		),
	)

	err := app.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), do.NameOf[Config]())
}

func TestApp_ValidateDetectsDuplicateProvider(t *testing.T) {
	app := dix.NewApp("validate-duplicate",
		dix.NewModule("dup",
			dix.WithModuleProviders(
				dix.Provider0(ProvideConfig),
				dix.Provider0(ProvideConfig),
			),
		),
	)

	err := app.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate provider output")
	assert.Contains(t, err.Error(), do.NameOf[Config]())
}

func TestApp_ValidateDetectsDuplicateProviderBeforeDIProfileBootstrap(t *testing.T) {
	app := dix.New("validate-profile-bootstrap-duplicate",
		dix.Modules(
			dix.NewModule("bootstrap",
				dix.Providers(
					dix.Provider0(ProvideConfig),
					dix.Provider0(ProvideConfig),
					dix.Provider0(func() dix.Profile {
						return dix.ProfileProd
					}),
				),
			),
		),
	)

	var err error
	require.NotPanics(t, func() {
		err = app.Validate()
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate provider output")
	assert.Contains(t, err.Error(), do.NameOf[Config]())
}

func TestApp_ValidateRequiresDIProfileProviderDependenciesInBootstrapGraph(t *testing.T) {
	prodConfig := dix.NewModule("prod-config",
		dix.UseProfiles(dix.ProfileProd),
		dix.Providers(
			dix.Provider0(ProvideConfig),
		),
	)
	bootstrap := dix.NewModule("bootstrap",
		dix.Providers(
			dix.Provider1(func(Config) dix.Profile {
				return dix.ProfileProd
			}),
		),
	)
	app := dix.New("validate-profile-bootstrap-deps",
		dix.Modules(bootstrap, prodConfig),
	)

	err := app.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), do.NameOf[Config]())
}

func TestApp_ValidateDoesNotEscapeForCoreSetup(t *testing.T) {
	app := dix.NewApp("validate-setup",
		dix.NewModule("health",
			dix.WithModuleSetup(func(c *dix.Container, _ dix.Lifecycle) error {
				c.RegisterHealthCheck("ok", func(_ context.Context) error { return nil })
				return nil
			}),
			dix.WithModuleProviders(
				dix.Provider1(func(cfg Config) *Database {
					return &Database{dsn: cfg.DSN}
				}),
			),
		),
	)

	err := app.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), do.NameOf[Config]())
}

func TestApp_ValidateAdvancedAliasDependency(t *testing.T) {
	app := dix.NewApp("validate-alias",
		dix.NewModule("alias",
			dix.WithModuleProviders(
				dix.Provider0(func() *testGreeterImpl { return &testGreeterImpl{} }),
			),
			dix.WithModuleSetups(
				dixadvanced.BindAlias[*testGreeterImpl, testGreeter](),
			),
			dix.WithModuleInvokes(
				dix.Invoke1(func(testGreeter) {}),
			),
		),
	)

	require.NoError(t, app.Validate())
}

func TestProvider4AndAggregateDependencyStruct(t *testing.T) {
	type Params struct {
		Config Config
		DB     *Database
		Server *Server
		Label  string
	}

	module := dix.NewModule("test",
		dix.WithModuleProviders(
			dix.Provider0(ProvideConfig),
			dix.Provider1(ProvideDatabase),
			dix.Provider1(ProvideServer),
			dix.Provider0(func() string { return "ok" }),
			dix.Provider4(func(cfg Config, db *Database, srv *Server, label string) Params {
				return Params{Config: cfg, DB: db, Server: srv, Label: label}
			}),
		),
	)

	rt := buildRuntime(t, dix.NewApp("test", module))
	params, err := dix.ResolveAs[Params](rt.Container())
	require.NoError(t, err)
	assert.Equal(t, 8080, params.Config.Port)
	assert.Equal(t, "sqlite://test.db", params.DB.dsn)
	assert.Equal(t, ":8080", params.Server.addr)
	assert.Equal(t, "ok", params.Label)
}

func TestLifecycleHookReceivesContext(t *testing.T) {
	type ctxKey string

	const key ctxKey = "trace"
	received := ""

	module := dix.NewModule("ctx",
		dix.WithModuleProviders(dix.Provider0(func() string { return "value" })),
		dix.WithModuleHooks(
			dix.OnStart(func(ctx context.Context, value string) error {
				trace, ok := ctx.Value(key).(string)
				if !ok {
					return errors.New("trace context value missing")
				}
				received = trace + ":" + value
				return nil
			}),
		),
	)

	rt := buildRuntime(t, dix.NewApp("test", module))
	require.NoError(t, rt.Start(context.WithValue(context.Background(), key, "abc")))
	require.NoError(t, rt.Stop(context.Background()))
	assert.Equal(t, "abc:value", received)
}

func TestModule_SingularOptionAliases(t *testing.T) {
	base := dix.NewModule("base",
		dix.WithModuleProvider(dix.Provider0(ProvideConfig)),
	)

	module := dix.NewModule("aliases",
		dix.WithModuleImport(base),
		dix.WithModuleProfile(dix.ProfileDev),
		dix.WithModuleExcludeProfile(dix.ProfileProd),
		dix.WithModuleProvider(dix.Provider1(ProvideDatabase)),
		dix.WithModuleInvoke(dix.Invoke1(func(*Database) {})),
		dix.WithModuleHook(dix.OnStart0(func(context.Context) error { return nil })),
	)

	app := dix.New("aliases",
		dix.WithProfile(dix.ProfileDev),
		dix.WithModule(module),
	)

	require.NoError(t, app.Validate())
	rt := buildRuntime(t, app)

	db, err := dix.ResolveAs[*Database](rt.Container())
	require.NoError(t, err)
	assert.Equal(t, "sqlite://test.db", db.dsn)

	require.NoError(t, rt.Start(context.Background()))
	require.NoError(t, rt.Stop(context.Background()))
}

func TestModule_ShortOptionAliases(t *testing.T) {
	base := dix.NewModule("base",
		dix.Providers(dix.Provider0(ProvideConfig)),
	)

	module := dix.NewModule("short-aliases",
		dix.Imports(base),
		dix.UseProfiles(dix.ProfileDev),
		dix.ExcludeProfiles(dix.ProfileProd),
		dix.Description("short module options"),
		dix.Tags("api", "public"),
		dix.Providers(dix.Provider1(ProvideDatabase)),
		dix.Invokes(dix.Invoke1(func(*Database) {})),
		dix.Hooks(dix.OnStart0(func(context.Context) error { return nil })),
		dix.Setups(dix.Setup(func(*dix.Container, dix.Lifecycle) error { return nil })),
	)

	assert.Equal(t, "short module options", module.Description())
	assert.ElementsMatch(t, []string{"api", "public"}, module.Tags().Values())

	app := dix.New("short-aliases",
		dix.WithProfile(dix.ProfileDev),
		dix.WithModule(module),
	)

	require.NoError(t, app.Validate())
	rt := buildRuntime(t, app)

	db, err := dix.ResolveAs[*Database](rt.Container())
	require.NoError(t, err)
	assert.Equal(t, "sqlite://test.db", db.dsn)

	require.NoError(t, rt.Start(context.Background()))
	require.NoError(t, rt.Stop(context.Background()))
}

func TestModule_ValueAndInvokeAliases(t *testing.T) {
	invoked := false

	module := dix.NewModule("aliases",
		dix.Providers(
			dix.Value(Config{Port: 9090, DSN: "sqlite://alias.db"}),
		),
		dix.Invokes(
			dix.Invoke(func() {
				invoked = true
			}),
		),
	)

	rt := buildRuntime(t, dix.New("aliases", dix.WithModule(module)))

	cfg, err := dix.ResolveAs[Config](rt.Container())
	require.NoError(t, err)
	assert.Equal(t, 9090, cfg.Port)
	assert.Equal(t, "sqlite://alias.db", cfg.DSN)
	assert.True(t, invoked)
}

func TestSetup_Shortcuts(t *testing.T) {
	started := false
	stopped := false
	setup0Called := false
	setupValue := ""

	module := dix.NewModule("setup-shortcuts",
		dix.Providers(
			dix.Value("setup-value"),
		),
		dix.Setups(
			dix.Setup0(func() error {
				setup0Called = true
				return nil
			}),
			dix.Setup1(func(value string) error {
				setupValue = value
				return nil
			}),
			dix.SetupContainer(func(c *dix.Container) error {
				c.RegisterHealthCheck("shortcut", func(context.Context) error { return nil })
				return nil
			}),
			dix.SetupLifecycle(func(lc dix.Lifecycle) error {
				lc.OnStart(func(context.Context) error {
					started = true
					return nil
				})
				lc.OnStop(func(context.Context) error {
					stopped = true
					return nil
				})
				return nil
			}),
		),
	)

	app := dix.New("setup-shortcuts", dix.Modules(module))
	rt := buildRuntime(t, app)

	assert.True(t, setup0Called)
	assert.Equal(t, "setup-value", setupValue)
	assert.True(t, rt.CheckHealth(context.Background()).Healthy())

	require.NoError(t, rt.Start(context.Background()))
	assert.True(t, started)
	require.NoError(t, rt.Stop(context.Background()))
	assert.True(t, stopped)
}
