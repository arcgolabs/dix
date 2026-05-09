package dix_test

import (
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/arcgolabs/dix"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type eagerConfig struct {
	Name string
}

type eagerService struct {
	Config eagerConfig
}

func TestProviderEagerWarmsDuringBuild(t *testing.T) {
	var calls atomic.Int32
	app := dix.New("eager",
		dix.Modules(
			dix.NewModule("eager",
				dix.Providers(
					dix.Provider0(func() *eagerService {
						calls.Add(1)
						return &eagerService{}
					}, dix.Eager()),
				),
			),
		),
	)

	rt := buildRuntime(t, app)
	assert.Equal(t, int32(1), calls.Load())

	resolved, err := dix.ResolveAs[*eagerService](rt.Container())
	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Equal(t, int32(1), calls.Load())
}

func TestProviderWithoutEagerStaysLazy(t *testing.T) {
	var calls atomic.Int32
	app := dix.New("lazy",
		dix.Modules(
			dix.NewModule("lazy",
				dix.Providers(
					dix.Provider0(func() *eagerService {
						calls.Add(1)
						return &eagerService{}
					}),
				),
			),
		),
	)

	rt := buildRuntime(t, app)
	assert.Equal(t, int32(0), calls.Load())

	_, err := dix.ResolveAs[*eagerService](rt.Container())
	require.NoError(t, err)
	assert.Equal(t, int32(1), calls.Load())
}

func TestProviderEagerCanDependOnProviderDeclaredLater(t *testing.T) {
	var calls atomic.Int32
	app := dix.New("eager-deps",
		dix.Modules(
			dix.NewModule("eager-deps",
				dix.Providers(
					dix.Provider1(func(cfg eagerConfig) *eagerService {
						calls.Add(1)
						return &eagerService{Config: cfg}
					}, dix.Eager()),
					dix.Provider0(func() eagerConfig {
						return eagerConfig{Name: "cfg"}
					}),
				),
			),
		),
	)

	rt := buildRuntime(t, app)
	assert.Equal(t, int32(1), calls.Load())

	resolved, err := dix.ResolveAs[*eagerService](rt.Container())
	require.NoError(t, err)
	assert.Equal(t, "cfg", resolved.Config.Name)
}

func TestProviderEagerPropagatesConstructionError(t *testing.T) {
	app := dix.New("eager-error",
		dix.Modules(
			dix.NewModule("eager-error",
				dix.Providers(
					dix.ProviderErr0(func() (*eagerService, error) {
						return nil, errors.New("boom")
					}, dix.Eager()),
				),
			),
		),
	)

	_, err := app.Build()
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "eager provider failed"), err)
	assert.True(t, strings.Contains(err.Error(), "boom"), err)
}

func TestRawEagerProviderRequiresDeclaredOutput(t *testing.T) {
	app := dix.New("raw-eager",
		dix.Modules(
			dix.NewModule("raw-eager",
				dix.Providers(
					dix.RawProviderWithMetadata(func(*dix.Container) {}, dix.ProviderMetadata{
						Label: "raw eager",
						Eager: true,
					}),
				),
			),
		),
	)

	_, err := app.Build()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "eager provider requires a declared output")
}
