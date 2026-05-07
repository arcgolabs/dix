package testx_test

import (
	"context"
	"testing"

	"github.com/arcgolabs/dix"
	"github.com/arcgolabs/dix/testx"
	"github.com/stretchr/testify/require"
)

func TestBuildAndValidateHelpers(t *testing.T) {
	t.Parallel()

	app := dix.New("test", dix.Modules(dix.NewModule("mod", dix.Providers(dix.Value("value")))))

	report := testx.Validate(t, app)
	require.False(t, report.HasErrors())

	runtime := testx.Build(t, app)
	value := dix.MustResolveAs[string](runtime.Container())
	require.Equal(t, "value", value)
}

func TestStartHelperStopsRuntime(t *testing.T) {
	app := dix.New("test")

	runtime := testx.Start(context.Background(), t, app)
	require.Equal(t, dix.AppStateStarted, runtime.State())
}
