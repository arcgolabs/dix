package advanced_test

import (
	"context"
	"errors"
	"testing"

	"github.com/arcgolabs/dix"
	dixadvanced "github.com/arcgolabs/dix/advanced"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdvancedOverrideValue(t *testing.T) {
	app := dix.New("override",
		dix.WithModule(
			dix.NewModule("override",
				dix.WithModuleProviders(
					dix.Provider0(func() string { return "base" }),
				),
				dix.WithModuleSetups(
					dixadvanced.Override0(func() string { return "override" }),
				),
			),
		),
	)

	require.NoError(t, app.Validate())
	rt := buildRuntime(t, app)

	value, err := dix.ResolveAs[string](rt.Container())
	require.NoError(t, err)
	assert.Equal(t, "override", value)
}

func TestAdvancedShortAliases(t *testing.T) {
	app := dix.New("advanced-short",
		dix.WithModule(
			dix.NewModule("advanced-short",
				dix.Providers(
					dix.Provider0(func() *testGreeterImpl { return &testGreeterImpl{} }),
					dix.Value("base"),
					dixadvanced.Named("tenant.default", "public"),
					dixadvanced.NamedProvider0[*testGreeterImpl]("tenant.default.greeter", func() *testGreeterImpl {
						return &testGreeterImpl{}
					}),
				),
				dix.Setups(
					dixadvanced.Alias[*testGreeterImpl, testGreeter](),
					dixadvanced.NamedAlias[*testGreeterImpl, testGreeter]("tenant.default.greeter", "tenant.default.greeter.alias"),
					dixadvanced.Override(func() string { return "override" }),
				),
			),
		),
	)

	rt := buildRuntime(t, app)

	named, err := dixadvanced.ResolveNamedAs[string](rt.Container(), "tenant.default")
	require.NoError(t, err)
	assert.Equal(t, "public", named)

	greeterValue, err := dix.ResolveAs[testGreeter](rt.Container())
	require.NoError(t, err)
	assert.NotNil(t, greeterValue)

	namedGreeter, err := dixadvanced.ResolveNamedAs[testGreeter](rt.Container(), "tenant.default.greeter.alias")
	require.NoError(t, err)
	assert.NotNil(t, namedGreeter)

	value, err := dix.ResolveAs[string](rt.Container())
	require.NoError(t, err)
	assert.Equal(t, "override", value)
}

func TestAdvancedTransientProvider(t *testing.T) {
	counter := 0
	rt := buildRuntime(t, dix.New("transient",
		dix.WithModule(
			dix.NewModule("transient",
				dix.WithModuleProviders(
					dixadvanced.Transient(func() int {
						counter++
						return counter
					}),
				),
			),
		),
	))

	first, err := dix.ResolveAs[int](rt.Container())
	require.NoError(t, err)
	second, err := dix.ResolveAs[int](rt.Container())
	require.NoError(t, err)
	assert.Equal(t, 1, first)
	assert.Equal(t, 2, second)
}

func TestAdvancedOverrideTransient(t *testing.T) {
	counter := 100
	rt := buildRuntime(t, dix.New("override-transient",
		dix.WithModule(
			dix.NewModule("override-transient",
				dix.WithModuleProviders(
					dix.Provider0(func() int { return 1 }),
				),
				dix.WithModuleSetups(
					dixadvanced.TransientOverride(func() int {
						counter++
						return counter
					}),
				),
			),
		),
	))

	first, err := dix.ResolveAs[int](rt.Container())
	require.NoError(t, err)
	second, err := dix.ResolveAs[int](rt.Container())
	require.NoError(t, err)
	assert.Equal(t, 101, first)
	assert.Equal(t, 102, second)
}

func TestAdvancedInspectRuntime(t *testing.T) {
	rt := buildRuntime(t, dix.New("inspect",
		dix.WithModule(
			dix.NewModule("inspect",
				dix.WithModuleProviders(
					dixadvanced.NamedValue("tenant.default", "public"),
				),
			),
		),
	))

	_, err := dixadvanced.ResolveNamedAs[string](rt.Container(), "tenant.default")
	require.NoError(t, err)

	report := dixadvanced.InspectRuntime(rt, "tenant.default")
	assert.NotEmpty(t, report.ScopeTree)
	require.NotNil(t, report.ProvidedServices)
	require.NotNil(t, report.InvokedServices)
	require.NotNil(t, report.NamedDependencies)
	assert.Positive(t, report.ProvidedServices.Len())
	assert.Positive(t, report.InvokedServices.Len())
	_, ok := report.NamedDependencies.Get("tenant.default")
	assert.True(t, ok)
}

func TestAdvancedInspectRuntimeWithOptions(t *testing.T) {
	rt := buildRuntime(t, dix.New("inspect-light",
		dix.WithModule(
			dix.NewModule("inspect-light",
				dix.WithModuleProviders(
					dixadvanced.NamedValue("tenant.default", "public"),
				),
			),
		),
	))

	report := dixadvanced.InspectRuntimeWithOptions(rt, dixadvanced.InspectOptions{
		IncludeNamedDeps: true,
	}, "tenant.default")

	assert.Empty(t, report.ScopeTree)
	assert.Nil(t, report.ProvidedServices)
	assert.Nil(t, report.InvokedServices)
	require.NotNil(t, report.NamedDependencies)
	_, ok := report.NamedDependencies.Get("tenant.default")
	assert.True(t, ok)
}

func TestAdvancedScopeNamedHelpers(t *testing.T) {
	rt := buildRuntime(t, dix.New("scope-named",
		dix.WithModule(
			dix.NewModule("scope-named",
				dix.WithModuleProviders(
					dix.Provider0(func() string { return "root" }),
				),
			),
		),
	))

	scope := dixadvanced.Scope(rt, "named-scope", func(injector do.Injector) {
		dixadvanced.ProvideScopedNamed1[string, string](injector, "greeting", func(root string) string {
			return root + "-scoped"
		})
	})

	value, err := dixadvanced.ResolveScopedNamedAs[string](scope, "greeting")
	require.NoError(t, err)
	assert.Equal(t, "root-scoped", value)
}

func TestAdvancedNamedProviderErr0PropagatesError(t *testing.T) {
	expectedErr := errors.New("named provider failed")
	rt := buildRuntime(t, dix.NewApp("named-provider-err",
		dix.NewModule("named-provider-err",
			dix.WithModuleProviders(
				dixadvanced.NamedProviderErr0("tenant.default", func() (string, error) {
					return "", expectedErr
				}),
			),
		),
	))

	_, err := dixadvanced.ResolveNamedAs[string](rt.Container(), "tenant.default")
	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
}

func TestAdvancedTransientProviderErr0KeepsTransientBehavior(t *testing.T) {
	counter := 0
	rt := buildRuntime(t, dix.NewApp("transient-provider-err",
		dix.NewModule("transient-provider-err",
			dix.WithModuleProviders(
				dixadvanced.TransientProviderErr0(func() (int, error) {
					counter++
					return counter, nil
				}),
			),
		),
	))

	first, err := dix.ResolveAs[int](rt.Container())
	require.NoError(t, err)
	second, err := dix.ResolveAs[int](rt.Container())
	require.NoError(t, err)
	assert.Equal(t, 1, first)
	assert.Equal(t, 2, second)
}

func TestAdvancedNamedTransientProviderErr1ResolvesDependencies(t *testing.T) {
	calls := 0
	rt := buildRuntime(t, dix.NewApp("named-transient-provider-err",
		dix.NewModule("named-transient-provider-err",
			dix.WithModuleProviders(
				dix.Provider0(func() string { return "root" }),
				dixadvanced.NamedTransientProviderErr1("greeting", func(root string) (string, error) {
					calls++
					return root + "-scoped", nil
				}),
			),
		),
	))

	first, err := dixadvanced.ResolveNamedAs[string](rt.Container(), "greeting")
	require.NoError(t, err)
	second, err := dixadvanced.ResolveNamedAs[string](rt.Container(), "greeting")
	require.NoError(t, err)
	assert.Equal(t, "root-scoped", first)
	assert.Equal(t, "root-scoped", second)
	assert.Equal(t, 2, calls)
}

func TestAdvancedScopedProviderErr1PropagatesError(t *testing.T) {
	expectedErr := errors.New("scoped provider failed")
	rt := buildRuntime(t, dix.NewApp("scoped-provider-err",
		dix.NewModule("scoped-provider-err",
			dix.WithModuleProviders(
				dix.Provider0(func() string { return "root" }),
			),
		),
	))

	scope := dixadvanced.Scope(rt, "err-scope", func(injector do.Injector) {
		dixadvanced.ProvideScopedErr1(injector, func(root string) (int, error) {
			if root != "root" {
				return 0, errors.New("unexpected root")
			}
			return 0, expectedErr
		})
	})

	_, err := dixadvanced.ResolveScopedAs[int](scope)
	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
}

func TestAdvancedScopedNamedProviderErr0PropagatesError(t *testing.T) {
	expectedErr := errors.New("scoped named provider failed")
	rt := buildRuntime(t, dix.NewApp("scoped-named-provider-err",
		dix.NewModule("scoped-named-provider-err"),
	))

	scope := dixadvanced.Scope(rt, "named-err-scope", func(injector do.Injector) {
		dixadvanced.ProvideScopedNamedErr0(injector, "tenant.default", func() (string, error) {
			return "", expectedErr
		})
	})

	_, err := dixadvanced.ResolveScopedNamedAs[string](scope, "tenant.default")
	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
}

func TestAdvancedOverrideErr0PropagatesError(t *testing.T) {
	expectedErr := errors.New("override failed")
	rt := buildRuntime(t, dix.NewApp("override-err",
		dix.NewModule("override-err",
			dix.WithModuleProviders(
				dix.Provider0(func() string { return "base" }),
			),
			dix.WithModuleSetups(
				dixadvanced.OverrideErr0(func() (string, error) {
					return "", expectedErr
				}),
			),
		),
	))

	_, err := dix.ResolveAs[string](rt.Container())
	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
}

func TestAdvancedNamedOverrideErr0PropagatesError(t *testing.T) {
	expectedErr := errors.New("named override failed")
	rt := buildRuntime(t, dix.NewApp("named-override-err",
		dix.NewModule("named-override-err",
			dix.WithModuleProviders(
				dixadvanced.NamedProvider0("tenant.default", func() string { return "base" }),
			),
			dix.WithModuleSetups(
				dixadvanced.NamedOverrideErr0("tenant.default", func() (string, error) {
					return "", expectedErr
				}),
			),
		),
	))

	_, err := dixadvanced.ResolveNamedAs[string](rt.Container(), "tenant.default")
	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
}

func TestAdvancedOverrideTransientErr1KeepsTransientBehavior(t *testing.T) {
	calls := 0
	rt := buildRuntime(t, dix.NewApp("override-transient-err",
		dix.NewModule("override-transient-err",
			dix.WithModuleProviders(
				dix.Provider0(func() string { return "root" }),
				dix.Provider0(func() int { return 1 }),
			),
			dix.WithModuleSetups(
				dixadvanced.OverrideTransientErr1(func(root string) (int, error) {
					calls++
					return len(root) + calls, nil
				}),
			),
		),
	))

	first, err := dix.ResolveAs[int](rt.Container())
	require.NoError(t, err)
	second, err := dix.ResolveAs[int](rt.Container())
	require.NoError(t, err)
	assert.Equal(t, 5, first)
	assert.Equal(t, 6, second)
	assert.Equal(t, 2, calls)
}

func TestWithDoSetup(t *testing.T) {
	called := false
	module := dix.NewModule("advanced",
		dix.WithModuleSetups(
			dixadvanced.DoSetup(func(raw do.Injector) error {
				called = raw != nil
				return nil
			}),
		),
	)
	buildRuntime(t, dix.NewApp("test", module))
	assert.True(t, called)
}

func TestValidateReportWarnsForUndeclaredRawEscapes(t *testing.T) {
	module := dix.NewModule("advanced",
		dix.WithModuleProviders(
			dix.RawProvider(func(*dix.Container) {}),
		),
		dix.WithModuleInvokes(
			dix.RawInvoke(func(*dix.Container) error { return nil }),
		),
		dix.WithModuleHooks(
			dix.RawHook(func(*dix.Container, dix.Lifecycle) {}),
		),
		dix.WithModuleSetups(
			dixadvanced.DoSetup(func(raw do.Injector) error {
				_ = raw
				return nil
			}),
		),
	)

	report := dix.NewApp("warnings", module).ValidateReport()
	require.False(t, report.HasErrors())
	require.True(t, report.HasWarnings())
	assert.Contains(t, report.WarningSummary(), string(dix.ValidationWarningRawProviderUndeclaredOutput))
	assert.Contains(t, report.WarningSummary(), string(dix.ValidationWarningRawInvokeUndeclaredDeps))
	assert.Contains(t, report.WarningSummary(), string(dix.ValidationWarningRawHookUndeclaredDeps))
	assert.Contains(t, report.WarningSummary(), string(dix.ValidationWarningRawSetupUndeclaredGraph))
}

func TestValidateReportUsesDeclaredRawMetadata(t *testing.T) {
	module := dix.NewModule("advanced",
		dix.WithModuleProviders(
			dix.Provider0(ProvideConfig),
			dix.RawProviderWithMetadata(func(c *dix.Container) {
				dix.ProvideValueT(c, &Database{dsn: "sqlite://raw.db"})
			}, dix.ProviderMetadata{
				Label:        "RawDatabaseProvider",
				Output:       dix.TypedService[*Database](),
				Dependencies: dix.ServiceRefs(dix.TypedService[Config]()),
			}),
		),
		dix.WithModuleInvokes(
			dix.RawInvokeWithMetadata(func(c *dix.Container) error {
				_, err := dix.ResolveAs[*Database](c)
				return err
			}, dix.InvokeMetadata{
				Label:        "RawInvokeDatabase",
				Dependencies: dix.ServiceRefs(dix.TypedService[*Database]()),
			}),
		),
		dix.WithModuleHooks(
			dix.RawHookWithMetadata(func(c *dix.Container, lc dix.Lifecycle) {
				lc.OnStart(func(context.Context) error {
					_, err := dix.ResolveAs[*Database](c)
					return err
				})
			}, dix.HookMetadata{
				Label:        "RawStartDatabase",
				Kind:         dix.HookKindStart,
				Dependencies: dix.ServiceRefs(dix.TypedService[*Database]()),
			}),
		),
		dix.WithModuleSetups(
			dixadvanced.DoSetupWithMetadata(func(raw do.Injector) error {
				_ = raw
				return nil
			}, dix.SetupMetadata{
				Label:         "RawDoSetup",
				Dependencies:  dix.ServiceRefs(dix.TypedService[Config]()),
				Provides:      dix.ServiceRefs(dix.NamedService("tenant.default")),
				GraphMutation: true,
			}),
		),
	)

	report := dix.NewApp("warnings", module).ValidateReport()
	require.False(t, report.HasErrors())
	assert.False(t, report.HasWarnings(), report.WarningSummary())
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
