package dix_test

import (
	"context"
	"testing"

	"github.com/arcgolabs/dix"
	"github.com/samber/mo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type conditionalCacheConfig struct {
	Enabled bool
}

type conditionalCache struct {
	Name string
}

func TestConditionalProviderResolvesNoneWhenDependencyDisablesService(t *testing.T) {
	rt := buildRuntime(t, dix.New("conditional",
		dix.Modules(dix.NewModule("cache",
			dix.Providers(
				dix.Value(conditionalCacheConfig{Enabled: false}),
				dix.ConditionalProvider1(func(cfg conditionalCacheConfig) dix.Conditional[conditionalCache] {
					if !cfg.Enabled {
						return mo.None[conditionalCache]()
					}
					return mo.Some(conditionalCache{Name: "redis"})
				}),
			),
		)),
	))

	cache, err := dix.ResolveAs[dix.Conditional[conditionalCache]](rt.Container())
	require.NoError(t, err)
	assert.False(t, cache.IsPresent())
}

func TestConditionalProviderResolvesSomeWhenDependencyEnablesService(t *testing.T) {
	rt := buildRuntime(t, dix.New("conditional",
		dix.Modules(dix.NewModule("cache",
			dix.Providers(
				dix.Value(conditionalCacheConfig{Enabled: true}),
				dix.ConditionalProvider1(func(cfg conditionalCacheConfig) dix.Conditional[conditionalCache] {
					if !cfg.Enabled {
						return mo.None[conditionalCache]()
					}
					return mo.Some(conditionalCache{Name: "redis"})
				}),
			),
		)),
	))

	cache, err := dix.ResolveAs[dix.Conditional[conditionalCache]](rt.Container())
	require.NoError(t, err)
	value, ok := cache.Get()
	require.True(t, ok)
	assert.Equal(t, "redis", value.Name)
}

func TestWhenSupportsDynamicFuncBool(t *testing.T) {
	enabled := false
	app := dix.New("when",
		dix.Modules(dix.NewModule("cache",
			dix.Providers(
				dix.Provider0(func() conditionalCache {
					return conditionalCache{Name: "redis"}
				}, dix.When(func() bool { return enabled })),
			),
		)),
	)

	first := buildRuntime(t, app)
	_, ok := dix.ResolveOptional[conditionalCache](first.Container())
	assert.False(t, ok)

	enabled = true
	second := buildRuntime(t, app)
	cache, err := dix.ResolveAs[conditionalCache](second.Container())
	require.NoError(t, err)
	assert.Equal(t, "redis", cache.Name)
}

func TestWhenConditionErrorFailsValidation(t *testing.T) {
	app := dix.New("bad-condition",
		dix.Modules(dix.NewModule("cache",
			dix.Providers(
				dix.Provider0(func() conditionalCache {
					return conditionalCache{Name: "redis"}
				}, dix.When("unsupported")),
			),
		)),
	)

	err := app.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported provider condition")
}

func TestWhenSupportsConditionContextAndProfile(t *testing.T) {
	app := dix.New("context-condition",
		dix.UseProfile(dix.ProfileProd),
		dix.Modules(dix.NewModule("cache",
			dix.Providers(
				dix.Provider0(func() conditionalCache {
					return conditionalCache{Name: "redis"}
				}, dix.When(func(ctx dix.ConditionContext) bool {
					return ctx.Meta.Name == "context-condition" &&
						ctx.Module == "cache" &&
						ctx.Label == "Provider0" &&
						ctx.Profile == dix.ProfileProd
				})),
				dix.Provider0(func() conditionalCacheConfig {
					return conditionalCacheConfig{Enabled: true}
				}, dix.When(func(profile dix.Profile) bool {
					return profile == dix.ProfileProd
				})),
			),
		)),
	)

	rt := buildRuntime(t, app)
	cache, err := dix.ResolveAs[conditionalCache](rt.Container())
	require.NoError(t, err)
	assert.Equal(t, "redis", cache.Name)
	cfg, err := dix.ResolveAs[conditionalCacheConfig](rt.Container())
	require.NoError(t, err)
	assert.True(t, cfg.Enabled)
}

func TestUnlessSupportsContextCancellationAwareCondition(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	app := dix.New("unless",
		dix.Modules(dix.NewModule("cache",
			dix.Providers(
				dix.Provider0(func() conditionalCache {
					return conditionalCache{Name: "redis"}
				}, dix.Unless(func(context.Context) bool {
					return ctx.Err() != nil
				})),
			),
		)),
	)

	rt := buildRuntime(t, app)
	_, ok := dix.ResolveOptional[conditionalCache](rt.Container())
	assert.False(t, ok)
}
