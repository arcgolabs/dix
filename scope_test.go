package dix_test

import (
	"errors"
	"testing"

	"github.com/arcgolabs/dix"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type scopedRequestContext struct {
	ID string
}

type scopedHandler struct {
	Root    string
	Request scopedRequestContext
}

func TestRuntimeScopeResolvesScopedAndInheritedServices(t *testing.T) {
	rt := buildRuntime(t, dix.New("scope",
		dix.Modules(
			dix.NewModule("root",
				dix.Providers(dix.Value("root")),
			),
		),
	))

	scope, err := rt.Scope("request-42", dix.ScopeFunc(func(c *dix.Container) {
		dix.ProvideValueT(c, scopedRequestContext{ID: "req-42"})
		dix.Provide2T(c, func(root string, req scopedRequestContext) scopedHandler {
			return scopedHandler{Root: root, Request: req}
		})
	}))
	require.NoError(t, err)

	handler, err := dix.ResolveAs[scopedHandler](scope)
	require.NoError(t, err)
	assert.Equal(t, "root", handler.Root)
	assert.Equal(t, "req-42", handler.Request.ID)

	_, ok := dix.ResolveOptionalAs[scopedRequestContext](rt.Container())
	assert.False(t, ok)
}

func TestRuntimeScopePropagatesPackageErrors(t *testing.T) {
	expectedErr := errors.New("scope package failed")
	rt := buildRuntime(t, dix.NewDefault())

	_, err := rt.Scope("broken", func(*dix.Container) error {
		return expectedErr
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
}

func TestRuntimeScopeSupportsNamedProviders(t *testing.T) {
	rt := buildRuntime(t, dix.NewDefault())

	scope, err := rt.Scope("named", dix.ScopeFunc(func(c *dix.Container) {
		dix.ProvideNamed1T(c, "greeting", func(root string) string {
			return root + "-scoped"
		})
		dix.ProvideValueT(c, "root")
	}))
	require.NoError(t, err)

	value, err := dix.ResolveNamedAs[string](scope, "greeting")
	require.NoError(t, err)
	assert.Equal(t, "root-scoped", value)
}
