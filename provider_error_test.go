package dix_test

import (
	"errors"
	"testing"

	"github.com/arcgolabs/dix"
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
