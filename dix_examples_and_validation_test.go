//revive:disable:file-length-limit Examples and validation tests are kept together to cover the public surface.

package dix_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/arcgolabs/dix"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ExampleApp() {
	app := dix.NewApp("myapp",
		DatabaseModule,
		ServerModule,
	)
	_ = app
}

func ExampleNewModule() {
	module := dix.NewModule("example",
		dix.WithModuleProviders(
			dix.Provider0(func() string { return "hello" }),
			dix.Provider1(func(s string) int { return len(s) }),
		),
		dix.WithModuleSetup(func(c *dix.Container, lc dix.Lifecycle) error {
			lc.OnStart(func(_ context.Context) error {
				s, err := dix.ResolveAs[string](c)
				if err != nil {
					return err
				}
				_, err = fmt.Println("Starting with:", s)
				if err != nil {
					return fmt.Errorf("print startup message: %w", err)
				}
				return nil
			})
			return nil
		}),
	)

	app := dix.NewApp("test", module)
	if _, err := app.Build(); err != nil {
		panic(err)
	}
}

func ExampleWithModuleHooks() {
	module := dix.NewModule("example",
		dix.WithModuleProviders(
			dix.Provider0(func() *http.Server {
				return &http.Server{Addr: ":8080", ReadHeaderTimeout: time.Second}
			}),
		),
		dix.WithModuleHooks(
			dix.OnStart(func(_ context.Context, s *http.Server) error {
				go func() {
					if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
						panic(err)
					}
				}()
				return nil
			}),
			dix.OnStop(func(ctx context.Context, s *http.Server) error {
				return s.Shutdown(ctx)
			}),
		),
	)

	app := dix.NewApp("test", module)
	_ = app
}

func TestResolveOptional(t *testing.T) {
	rt := buildRuntime(t, dix.NewApp("test"))
	_, ok := dix.ResolveOptional[string](rt.Container())
	assert.False(t, ok)
}

func TestResolveOptionAndResolveOr(t *testing.T) {
	rt := buildRuntime(t, dix.NewApp("test",
		dix.NewModule("deps",
			dix.WithModuleProviders(
				dix.Provider0(func() string { return "configured" }),
			),
		),
	))

	option := dix.ResolveOption[string](rt.Container())
	value, ok := option.Get()
	require.True(t, ok)
	assert.Equal(t, "configured", value)

	assert.Equal(t, "configured", dix.ResolveOr[string](rt.Container(), "fallback"))
	assert.Equal(t, 42, dix.ResolveOr[int](rt.Container(), 42))
}

func TestResolveOptionEAndResolveOrErrExposeProviderErrors(t *testing.T) {
	rt := buildRuntime(t, dix.NewApp("test",
		dix.NewModule("deps",
			dix.WithModuleProviders(
				dix.ProviderErr0(func() (string, error) {
					return "", errors.New("boom")
				}),
			),
		),
	))

	option, err := dix.ResolveOptionE[string](rt.Container())
	require.Error(t, err)
	assert.False(t, option.IsPresent())
	assert.Contains(t, err.Error(), "boom")

	value, ok, err := dix.ResolveOptionalE[string](rt.Container())
	require.Error(t, err)
	assert.False(t, ok)
	assert.Equal(t, "", value)

	fallback, err := dix.ResolveOrErr[string](rt.Container(), "fallback")
	require.Error(t, err)
	assert.Equal(t, "fallback", fallback)
}

func TestProfileFromEnv(t *testing.T) {
	t.Setenv("ARCGO_DIX_PROFILE", string(dix.ProfileDev))
	assert.Equal(t, dix.ProfileDev, dix.ProfileFromEnv("ARCGO_DIX_PROFILE", dix.ProfileProd))

	t.Setenv("ARCGO_DIX_PROFILE", "custom.profile")
	assert.Equal(t, dix.Profile("custom.profile"), dix.ProfileFromEnv("ARCGO_DIX_PROFILE", dix.ProfileProd))
}

func TestValidateReportReturnsIndependentCollectionsAndStillBuilds(t *testing.T) {
	app := dix.NewApp("warnings",
		dix.NewModule("advanced",
			dix.WithModuleProviders(
				dix.RawProvider(func(*dix.Container) {}),
			),
		),
	)

	first := app.ValidateReport()
	require.False(t, first.HasErrors())
	require.True(t, first.HasWarnings())
	baselineWarnings := first.Warnings.Len()

	first.Warnings.Add(dix.ValidationWarning{Kind: "custom"})
	assert.Equal(t, baselineWarnings+1, first.Warnings.Len())

	second := app.ValidateReport()
	assert.Equal(t, baselineWarnings, second.Warnings.Len())

	rt, err := app.Build()
	require.NoError(t, err)
	assert.NotNil(t, rt)
}

func TestValidateReportDoesNotFreezeDIResolvedProfile(t *testing.T) {
	t.Setenv("ARCGO_DIX_PROFILE", "preview")

	app := dix.New("profile-cache",
		dix.WithModules(
			dix.NewModule("profile",
				dix.WithModuleProviders(
					dix.Provider0(func() dix.Profile {
						return dix.ProfileFromEnv("ARCGO_DIX_PROFILE", dix.ProfileProd)
					}),
				),
			),
			dix.NewModule("preview-only",
				dix.WithModuleProfiles(dix.Profile("preview")),
				dix.WithModuleProviders(
					dix.Provider0(func() string { return "preview" }),
				),
			),
			dix.NewModule("prod-only",
				dix.WithModuleProfiles(dix.ProfileProd),
				dix.WithModuleProviders(
					dix.Provider0(func() string { return "prod" }),
				),
			),
		),
	)

	report := app.ValidateReport()
	require.False(t, report.HasErrors(), report.Err())

	t.Setenv("ARCGO_DIX_PROFILE", string(dix.ProfileProd))

	rt := buildRuntime(t, app)
	assert.Equal(t, dix.ProfileProd, rt.Profile())

	value, err := dix.ResolveAs[string](rt.Container())
	require.NoError(t, err)
	assert.Equal(t, "prod", value)
}

func TestBuildFailureShutsDownResolvedServices(t *testing.T) {
	svc := &cleanupService{}
	app := dix.NewApp("cleanup",
		dix.NewModule("cleanup",
			dix.WithModuleProviders(
				dix.Provider0(func() *cleanupService { return svc }),
			),
			dix.WithModuleSetup(func(c *dix.Container, _ dix.Lifecycle) error {
				_, err := dix.ResolveAs[*cleanupService](c)
				require.NoError(t, err)
				return errors.New("setup failed")
			}),
		),
	)

	_, err := app.Build()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "setup failed")
	assert.Equal(t, 1, svc.shutdowns)
}

func TestRuntimeStartFailureRollsBackStopHooks(t *testing.T) {
	type lifecycleService struct {
		starts int
		stops  int
	}

	svc := &lifecycleService{}
	app := dix.NewApp("rollback",
		dix.NewModule("rollback",
			dix.WithModuleProviders(
				dix.Provider0(func() *lifecycleService { return svc }),
			),
			dix.WithModuleHooks(
				dix.OnStart(func(context.Context, *lifecycleService) error {
					svc.starts++
					return nil
				}),
				dix.OnStop(func(context.Context, *lifecycleService) error {
					svc.stops++
					return nil
				}),
				dix.OnStart0(func(context.Context) error {
					return errors.New("boom")
				}),
			),
		),
	)

	rt := buildRuntime(t, app)
	err := rt.Start(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "boom")
	assert.Equal(t, 1, svc.starts)
	assert.Equal(t, 1, svc.stops)
	assert.Equal(t, dix.AppStateStopped, rt.State())
}
