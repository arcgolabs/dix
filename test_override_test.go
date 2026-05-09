package dix_test

import (
	"testing"

	"github.com/arcgolabs/dix"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testOverrideConfig struct {
	Name string
}

type testOverrideRepo struct {
	Config testOverrideConfig
}

type testOverrideGreeter interface {
	Greet() string
}

type testOverrideRealGreeter struct{}

func (testOverrideRealGreeter) Greet() string { return "real" }

type testOverrideMockGreeter struct{}

func (testOverrideMockGreeter) Greet() string { return "mock" }

func TestAppTestValueReplacesProviderWithoutMutatingOriginal(t *testing.T) {
	app := dix.New("test-overrides",
		dix.Modules(
			dix.NewModule("config",
				dix.Providers(dix.Value(testOverrideConfig{Name: "prod"})),
			),
		),
	)

	testRuntime := buildRuntime(t, app.Test(dix.TestValue(testOverrideConfig{Name: "test"})))
	testConfig, err := dix.ResolveAs[testOverrideConfig](testRuntime.Container())
	require.NoError(t, err)
	assert.Equal(t, "test", testConfig.Name)

	prodRuntime := buildRuntime(t, app)
	prodConfig, err := dix.ResolveAs[testOverrideConfig](prodRuntime.Container())
	require.NoError(t, err)
	assert.Equal(t, "prod", prodConfig.Name)
}

func TestAppTestDisableModuleAndProvideReplacement(t *testing.T) {
	configModule := dix.NewModule("config",
		dix.Providers(dix.Value(testOverrideConfig{Name: "prod"})),
	)
	repoModule := dix.NewModule("repo",
		dix.Imports(configModule),
		dix.Providers(dix.Provider1(func(config testOverrideConfig) testOverrideRepo {
			return testOverrideRepo{Config: config}
		})),
	)
	app := dix.New("test-disable-module", dix.Modules(repoModule))

	testRuntime := buildRuntime(t, app.Test(
		dix.TestDisableModules("config"),
		dix.TestValue(testOverrideConfig{Name: "mock"}),
	))
	repo, err := dix.ResolveAs[testOverrideRepo](testRuntime.Container())
	require.NoError(t, err)
	assert.Equal(t, "mock", repo.Config.Name)
}

func TestAppTestValueReplacesProviderAlias(t *testing.T) {
	app := dix.New("test-alias-override",
		dix.Modules(
			dix.NewModule("greeter",
				dix.Providers(
					dix.Provider0(func() testOverrideRealGreeter {
						return testOverrideRealGreeter{}
					}, dix.As[testOverrideGreeter]()),
				),
			),
		),
	)

	testRuntime := buildRuntime(t, app.Test(
		dix.TestValue[testOverrideGreeter](testOverrideMockGreeter{}),
	))
	greeter, err := dix.ResolveAs[testOverrideGreeter](testRuntime.Container())
	require.NoError(t, err)
	assert.Equal(t, "mock", greeter.Greet())
}

func TestAppTestProvidersAcceptsArbitraryProviderFunc(t *testing.T) {
	app := dix.New("test-provider-func",
		dix.Modules(
			dix.NewModule("config",
				dix.Providers(dix.Value(testOverrideConfig{Name: "prod"})),
			),
		),
	)

	testRuntime := buildRuntime(t, app.Test(
		dix.TestProviders(dix.Provider0(func() testOverrideConfig {
			return testOverrideConfig{Name: "custom"}
		})),
	))
	config, err := dix.ResolveAs[testOverrideConfig](testRuntime.Container())
	require.NoError(t, err)
	assert.Equal(t, "custom", config.Name)
}
