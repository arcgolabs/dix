package dix_test

import (
	"errors"
	"testing"

	"github.com/arcgolabs/arcgo/dix"
	dixadvanced "github.com/arcgolabs/arcgo/dix/advanced"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderErr0PropagatesError(t *testing.T) {
	expectedErr := errors.New("provider failed")
	rt := buildRuntime(t, dix.NewApp("provider-err",
		dix.NewModule("provider-err",
			dix.WithModuleProviders(
				dix.ProviderErr0(func() (string, error) {
					return "", expectedErr
				}),
			),
		),
	))

	_, err := dix.ResolveAs[string](rt.Container())
	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
}

func TestProviderErr1DeclaresDependencyAndPropagatesError(t *testing.T) {
	expectedErr := errors.New("database init failed")
	app := dix.NewApp("provider-err-deps",
		dix.NewModule("provider-err-deps",
			dix.WithModuleProviders(
				dix.Provider0(ProvideConfig),
				dix.ProviderErr1(func(cfg Config) (*Database, error) {
					if cfg.DSN == "" {
						return nil, errors.New("missing dsn")
					}
					return nil, expectedErr
				}),
			),
		),
	)

	require.NoError(t, app.Validate())
	rt := buildRuntime(t, app)

	_, err := dix.ResolveAs[*Database](rt.Container())
	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
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
