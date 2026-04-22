package dix_test

import (
	"context"
	"errors"
	"testing"

	"github.com/arcgolabs/dix"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type failingShutdownService struct{}

func (s *failingShutdownService) Shutdown() error {
	return errors.New("shutdown failed")
}

func TestRuntimeStopWithReportAggregatesErrors(t *testing.T) {
	app := dix.New("stop-report",
		dix.WithModule(
			dix.NewModule("runtime",
				dix.WithModuleProviders(
					dix.Provider0(func() *failingShutdownService { return &failingShutdownService{} }),
				),
				dix.WithModuleInvokes(
					dix.Invoke1(func(*failingShutdownService) {}),
				),
				dix.WithModuleHooks(
					dix.OnStop(func(_ context.Context, _ *failingShutdownService) error {
						return errors.New("stop hook failed")
					}),
				),
			),
		),
	)

	rt := buildRuntime(t, app)
	require.NoError(t, rt.Start(context.Background()))

	report, err := rt.StopWithReport(context.Background())
	require.Error(t, err)
	require.NotNil(t, report)
	require.ErrorContains(t, report.HookError, "stop hook failed")
	require.NotNil(t, report.ShutdownReport)
	assert.NotEmpty(t, report.ShutdownReport.Errors)
	assert.Equal(t, dix.AppStateStopped, rt.State())
}
