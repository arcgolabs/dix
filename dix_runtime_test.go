//revive:disable:file-length-limit Runtime tests intentionally keep related behavior scenarios in one file.

package dix_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/arcgolabs/collectionx"
	"github.com/arcgolabs/dix"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type frameworkLoggerCarrier struct {
	logger *slog.Logger
}

type frameworkEventLoggerCarrier struct {
	logger dix.EventLogger
}

type recordingEventLogger struct {
	mu          sync.Mutex
	messages    []dix.MessageEvent
	builds      []dix.BuildEvent
	starts      []dix.StartEvent
	stops       []dix.StopEvent
	health      []dix.HealthCheckEvent
	transitions []dix.StateTransitionEvent
}

func (r *recordingEventLogger) LogEvent(_ context.Context, event dix.Event) {
	r.mu.Lock()
	defer r.mu.Unlock()

	switch e := event.(type) {
	case dix.MessageEvent:
		r.messages = append(r.messages, e)
	case dix.BuildEvent:
		r.builds = append(r.builds, e)
	case dix.StartEvent:
		r.starts = append(r.starts, e)
	case dix.StopEvent:
		r.stops = append(r.stops, e)
	case dix.HealthCheckEvent:
		r.health = append(r.health, e)
	case dix.StateTransitionEvent:
		r.transitions = append(r.transitions, e)
	}
}

func TestBuildDebugLogging(t *testing.T) {
	logger, buf := newDebugLogger()
	app := dix.New("debug-build",
		dix.WithLogger(logger),
		dix.WithModule(
			dix.NewModule("debug",
				dix.WithModuleProviders(
					dix.Provider0(func() string { return "value" }),
				),
				dix.WithModuleHooks(
					dix.OnStart(func(context.Context, string) error { return nil }),
				),
				dix.WithModuleSetups(
					dix.SetupWithMetadata(func(*dix.Container, dix.Lifecycle) error { return nil }, dix.SetupMetadata{
						Label:        "DebugSetup",
						Dependencies: dix.ServiceRefs(dix.TypedService[string]()),
					}),
				),
				dix.WithModuleInvokes(
					dix.Invoke1(func(string) {}),
				),
			),
		),
	)

	_, err := app.Build()
	require.NoError(t, err)

	logs := buf.String()
	assert.True(t, strings.Contains(logs, "build plan ready"), logs)
	assert.True(t, strings.Contains(logs, "registering provider"), logs)
	assert.True(t, strings.Contains(logs, "binding lifecycle hook"), logs)
	assert.True(t, strings.Contains(logs, "module setup completed"), logs)
	assert.True(t, strings.Contains(logs, "invoke completed"), logs)
}

func TestWithLoggerRoutesInternalEventsThroughSlog(t *testing.T) {
	logger, buf := newDebugLogger()
	app := dix.New("slog-event-logger",
		dix.UseLogger(logger),
		dix.WithModule(
			dix.NewModule("slog-event-logger",
				dix.WithModuleProviders(
					dix.Provider0(func() string { return "value" }),
				),
				dix.WithModuleSetups(
					dix.SetupContainer(func(c *dix.Container) error {
						c.RegisterHealthCheck("ready", func(context.Context) error { return nil })
						return nil
					}),
				),
				dix.WithModuleHooks(
					dix.OnStartFunc(func() error { return nil }),
					dix.OnStopFunc(func() error { return nil }),
				),
			),
		),
	)

	rt := buildRuntime(t, app)
	require.NoError(t, rt.Start(context.Background()))
	assert.True(t, rt.CheckHealth(context.Background()).Healthy())
	require.NoError(t, rt.Stop(context.Background()))

	logs := buf.String()
	assert.Contains(t, logs, "app built")
	assert.Contains(t, logs, "app started")
	assert.Contains(t, logs, "health check passed")
	assert.Contains(t, logs, "app stopped")
}

func TestWithLoggerTakesPriorityOverDIProvidedLogger(t *testing.T) {
	configuredLogger, configuredBuf := newDebugLogger()
	diLogger, diBuf := newDebugLogger()

	app := dix.New("logger-priority",
		dix.UseLogger(configuredLogger),
		dix.WithModule(
			dix.NewModule("logger",
				dix.WithModuleProviders(
					dix.Value(diLogger),
				),
			),
		),
	)

	rt := buildRuntime(t, app)
	resolved, err := dix.ResolveAs[*slog.Logger](rt.Container())
	require.NoError(t, err)

	require.Same(t, configuredLogger, rt.Logger())
	require.Same(t, configuredLogger, resolved)

	require.NoError(t, rt.Start(context.Background()))
	require.NoError(t, rt.Stop(context.Background()))

	assert.Contains(t, configuredBuf.String(), "app built")
	assert.Empty(t, diBuf.String())
}

func TestWithLoggerTakesPriorityOverLoggerFromContainer(t *testing.T) {
	configuredLogger, _ := newDebugLogger()
	diLogger, _ := newDebugLogger()

	app := dix.New("logger-priority-resolver",
		dix.UseLogger(configuredLogger),
		dix.UseLogger1(func(carrier *frameworkLoggerCarrier) *slog.Logger {
			return carrier.logger
		}),
		dix.WithModule(
			dix.NewModule("logger",
				dix.WithModuleProvider(
					dix.Provider0(func() *frameworkLoggerCarrier {
						return &frameworkLoggerCarrier{logger: diLogger}
					}),
				),
			),
		),
	)

	rt := buildRuntime(t, app)
	resolved, err := dix.ResolveAs[*slog.Logger](rt.Container())
	require.NoError(t, err)

	require.Same(t, configuredLogger, rt.Logger())
	require.Same(t, configuredLogger, resolved)
}

func TestRuntimeStartRollbackDebugLogging(t *testing.T) {
	logger, buf := newDebugLogger()
	app := dix.New("debug-start",
		dix.WithModule(loggerModule(logger)),
		dix.WithModule(
			dix.NewModule("debug-start",
				dix.WithModuleProviders(
					dix.Provider0(func() string { return "value" }),
				),
				dix.WithModuleHooks(
					dix.OnStart(func(context.Context, string) error { return nil }),
					dix.OnStop(func(context.Context, string) error { return nil }),
					dix.OnStart0(func(context.Context) error { return errors.New("boom") }),
				),
			),
		),
	)

	rt := buildRuntime(t, app)
	err := rt.Start(context.Background())
	require.Error(t, err)

	logs := buf.String()
	assert.True(t, strings.Contains(logs, "runtime state transition"), logs)
	assert.True(t, strings.Contains(logs, "executing start hook"), logs)
	assert.True(t, strings.Contains(logs, "rolling back app start"), logs)
	assert.True(t, strings.Contains(logs, "executing stop hook"), logs)
	assert.True(t, strings.Contains(logs, "shutting down container"), logs)
}

func TestHealthCheckReport(t *testing.T) {
	module := dix.NewModule("health",
		dix.WithModuleSetup(func(c *dix.Container, _ dix.Lifecycle) error {
			c.RegisterHealthCheck("db", func(_ context.Context) error { return nil })
			c.RegisterHealthCheck("cache", func(_ context.Context) error { return errors.New("down") })
			return nil
		}),
	)

	rt := buildRuntime(t, dix.NewApp("test", module))
	report := rt.CheckHealth(context.Background())
	assert.False(t, report.Healthy())
	require.Error(t, report.Error())
	assert.Contains(t, report.Error().Error(), "cache")
}

func TestRuntime_HealthHandlers(t *testing.T) {
	module := dix.NewModule("health",
		dix.WithModuleSetup(func(c *dix.Container, _ dix.Lifecycle) error {
			c.RegisterLivenessCheck("live", func(_ context.Context) error { return nil })
			c.RegisterReadinessCheck("ready", func(_ context.Context) error { return errors.New("booting") })
			return nil
		}),
	)

	rt := buildRuntime(t, dix.NewApp("health", module))
	reqCtx := context.Background()

	liveReq := httptest.NewRequestWithContext(reqCtx, http.MethodGet, "/livez", http.NoBody)
	liveRes := httptest.NewRecorder()
	rt.LivenessHandler()(liveRes, liveReq)
	assert.Equal(t, http.StatusOK, liveRes.Code)

	readyReq := httptest.NewRequestWithContext(reqCtx, http.MethodGet, "/readyz", http.NoBody)
	readyRes := httptest.NewRecorder()
	rt.ReadinessHandler()(readyRes, readyReq)
	assert.Equal(t, http.StatusServiceUnavailable, readyRes.Code)
}

func TestNew_WithModulesOption(t *testing.T) {
	rt := buildRuntime(t, dix.New("test",
		dix.WithProfile(dix.ProfileDev),
		dix.WithModule(DatabaseModule),
	))

	logger, err := dix.ResolveAs[*slog.Logger](rt.Container())
	require.NoError(t, err)
	assert.NotNil(t, logger)
}

func TestHealthKinds(t *testing.T) {
	mod := dix.NewModule("health",
		dix.WithModuleSetup(func(c *dix.Container, _ dix.Lifecycle) error {
			c.RegisterLivenessCheck("live", func(_ context.Context) error { return nil })
			c.RegisterReadinessCheck("ready", func(_ context.Context) error { return nil })
			return nil
		}),
	)

	rt := buildRuntime(t, dix.New("health-app", dix.WithModule(mod)))
	live := rt.CheckLiveness(context.Background())
	ready := rt.CheckReadiness(context.Background())

	assert.True(t, live.Healthy())
	assert.True(t, ready.Healthy())
	require.NotNil(t, live.Checks)
	require.NotNil(t, ready.Checks)
	assert.Equal(t, 1, live.Checks.Len())
	assert.Equal(t, 1, ready.Checks.Len())
}

func TestNewDefault(t *testing.T) {
	app := dix.NewDefault()
	assert.Equal(t, dix.DefaultAppName, app.Name())
}

func TestApp_StartBuildsAndStartsRuntime(t *testing.T) {
	app := dix.New("start",
		dix.WithModule(
			dix.NewModule("start",
				dix.WithModuleProvider(dix.Provider0(func() string { return "value" })),
				dix.WithModuleHook(dix.OnStart(func(context.Context, string) error { return nil })),
			),
		),
	)

	rt, err := app.Start(context.Background())
	require.NoError(t, err)
	assert.Equal(t, dix.AppStateStarted, rt.State())

	value, err := dix.ResolveAs[string](rt.Container())
	require.NoError(t, err)
	assert.Equal(t, "value", value)

	require.NoError(t, rt.Stop(context.Background()))
}

func TestApp_RunContextStartsAndStopsRuntime(t *testing.T) {
	started := make(chan struct{}, 1)
	stopped := make(chan struct{}, 1)

	app := dix.New("run-context",
		dix.WithModule(
			dix.NewModule("run-context",
				dix.WithModuleProvider(dix.Provider0(func() string { return "value" })),
				dix.WithModuleHooks(
					dix.OnStart(func(context.Context, string) error {
						started <- struct{}{}
						return nil
					}),
					dix.OnStop(func(context.Context, string) error {
						stopped <- struct{}{}
						return nil
					}),
				),
			),
		),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- app.RunContext(ctx)
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("run context did not start runtime")
	}

	cancel()

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("run context did not return")
	}

	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("run context did not stop runtime")
	}
}

func TestApp_RunContextStopUsesConfiguredTimeout(t *testing.T) {
	started := make(chan struct{}, 1)
	stopErr := make(chan error, 1)

	app := dix.New("run-context-stop-timeout",
		dix.RunStopTimeout(20*time.Millisecond),
		dix.Modules(
			dix.NewModule("run-context-stop-timeout",
				dix.Hooks(
					dix.OnStart0(func(context.Context) error {
						started <- struct{}{}
						return nil
					}),
					dix.OnStop0(func(ctx context.Context) error {
						<-ctx.Done()
						err := ctx.Err()
						stopErr <- err
						return fmt.Errorf("stop timeout: %w", err)
					}),
				),
			),
		),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- app.RunContext(ctx)
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("run context did not start runtime")
	}

	cancel()

	select {
	case err := <-stopErr:
		require.ErrorIs(t, err, context.DeadlineExceeded)
	case <-time.After(2 * time.Second):
		t.Fatal("stop hook did not receive timeout")
	}

	select {
	case err := <-errCh:
		require.ErrorIs(t, err, context.DeadlineExceeded)
	case <-time.After(2 * time.Second):
		t.Fatal("run context did not return")
	}
}

func TestApp_RunContextStopTimeoutCanBeDisabled(t *testing.T) {
	started := make(chan struct{}, 1)
	stopped := make(chan struct{}, 1)

	app := dix.New("run-context-stop-timeout-disabled",
		dix.WithRunStopTimeout(0),
		dix.Modules(
			dix.NewModule("run-context-stop-timeout-disabled",
				dix.Hooks(
					dix.OnStart0(func(context.Context) error {
						started <- struct{}{}
						return nil
					}),
					dix.OnStop0(func(ctx context.Context) error {
						if _, ok := ctx.Deadline(); ok {
							return context.DeadlineExceeded
						}
						select {
						case <-ctx.Done():
							return ctx.Err()
						default:
						}
						stopped <- struct{}{}
						return nil
					}),
				),
			),
		),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- app.RunContext(ctx)
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("run context did not start runtime")
	}

	cancel()

	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("run context did not stop runtime")
	}

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("run context did not return")
	}
}

func TestApp_ShortOptionAliases(t *testing.T) {
	app := dix.New("aliases",
		dix.UseProfile(dix.ProfileDev),
		dix.Version("1.2.3"),
		dix.Modules(loggerModule(slog.Default()), DatabaseModule),
	)

	assert.Equal(t, dix.ProfileDev, app.Profile())
	assert.Equal(t, "1.2.3", app.Meta().Version)
	assert.NotNil(t, app.Logger())
	assert.Equal(t, dix.DefaultRunStopTimeout, app.RunStopTimeout())
	assert.Equal(t, 2, app.Modules().Len())

	rt := buildRuntime(t, app)
	cfg, err := dix.ResolveAs[Config](rt.Container())
	require.NoError(t, err)
	assert.Equal(t, 8080, cfg.Port)
}

func TestApp_MoreShortOptionAliases(t *testing.T) {
	observer := &recordingObserver{}
	buf := &bytes.Buffer{}
	baseLogger := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	app := dix.New("more-aliases",
		dix.AppDescription("alias app"),
		dix.LoggerFrom0(func() *slog.Logger { return baseLogger }),
		dix.Observers(observer),
		dix.DebugScopeTree(true),
		dix.DebugNamedServiceDependencies("tenant.default"),
		dix.Modules(
			dix.NewModule("more-aliases",
				dix.Providers(
					dix.Provider0(func() string { return "value" }),
				),
				dix.Hooks(
					dix.OnStartFunc(func() error { return nil }),
					dix.OnStopFunc(func() error { return nil }),
				),
			),
		),
	)

	assert.Equal(t, "alias app", app.Meta().Description)

	rt := buildRuntime(t, app)
	require.NotNil(t, rt.Logger())
	require.NoError(t, rt.Start(context.Background()))
	require.NoError(t, rt.Stop(context.Background()))
	waitForObserverEvents(t, observer, 1, 1, 0, 4)
	assert.NotEmpty(t, observer.starts)
	assert.NotEmpty(t, observer.stops)
	assert.Contains(t, buf.String(), "scope tree")
}

func TestWithLoggerFrom1_UsesDIProvidedLogger(t *testing.T) {
	buf := &bytes.Buffer{}
	diLogger := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	app := dix.New("di-logger",
		dix.WithLoggerFrom1(func(carrier *frameworkLoggerCarrier) *slog.Logger {
			return carrier.logger
		}),
		dix.WithModule(
			dix.NewModule("logger",
				dix.WithModuleProviders(
					dix.Provider0(func() *frameworkLoggerCarrier {
						return &frameworkLoggerCarrier{logger: diLogger}
					}),
				),
				dix.WithModuleHooks(
					dix.OnStart0(func(context.Context) error { return nil }),
					dix.OnStop0(func(context.Context) error { return nil }),
				),
			),
		),
	)

	rt := buildRuntime(t, app)

	resolved, err := dix.ResolveAs[*slog.Logger](rt.Container())
	require.NoError(t, err)
	assert.Same(t, diLogger, rt.Logger())
	assert.Same(t, diLogger, resolved)

	require.NoError(t, rt.Start(context.Background()))
	require.NoError(t, rt.Stop(context.Background()))

	logs := buf.String()
	assert.Contains(t, logs, "building app")
	assert.Contains(t, logs, "registering provider")
	assert.Contains(t, logs, "starting app")
	assert.Contains(t, logs, "app stopped")
}

func TestLoggerFrom1Alias_UsesDIProvidedLogger(t *testing.T) {
	buf := &bytes.Buffer{}
	diLogger := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	app := dix.New("di-logger-alias",
		dix.LoggerFrom1(func(carrier *frameworkLoggerCarrier) *slog.Logger {
			return carrier.logger
		}),
		dix.Modules(
			dix.NewModule("logger",
				dix.Providers(
					dix.Provider0(func() *frameworkLoggerCarrier {
						return &frameworkLoggerCarrier{logger: diLogger}
					}),
				),
				dix.Hooks(
					dix.OnStartFunc(func() error { return nil }),
					dix.OnStopFunc(func() error { return nil }),
				),
			),
		),
	)

	rt := buildRuntime(t, app)

	resolved, err := dix.ResolveAs[*slog.Logger](rt.Container())
	require.NoError(t, err)
	assert.Same(t, diLogger, rt.Logger())
	assert.Same(t, diLogger, resolved)
}

func TestWithLoggerFrom1_MissingDependencyFailsBuild(t *testing.T) {
	app := dix.New("di-logger-missing",
		dix.WithLoggerFrom1(func(*frameworkLoggerCarrier) *slog.Logger {
			return slog.Default()
		}),
	)

	_, err := app.Build()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolve framework logger failed")
}

func TestUseEventLogger1_RoutesAllDixLogsThroughConfiguredLogger(t *testing.T) {
	baseBuf := &bytes.Buffer{}
	baseLogger := slog.New(slog.NewTextHandler(baseBuf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	eventLogger := &recordingEventLogger{}

	app := dix.New("event-logger",
		dix.UseEventLogger1(func(carrier *frameworkEventLoggerCarrier) dix.EventLogger {
			return carrier.logger
		}),
		dix.Modules(
			loggerModule(baseLogger),
			dix.NewModule("event-logger",
				dix.Providers(
					dix.Provider0(func() *frameworkEventLoggerCarrier {
						return &frameworkEventLoggerCarrier{logger: eventLogger}
					}),
					dix.Provider0(func() string { return "value" }),
				),
				dix.Setups(
					dix.SetupContainer(func(c *dix.Container) error {
						c.RegisterHealthCheck("db", func(context.Context) error { return nil })
						return nil
					}),
				),
				dix.Hooks(
					dix.OnStartFunc(func() error { return nil }),
					dix.OnStopFunc(func() error { return nil }),
				),
			),
		),
	)

	rt := buildRuntime(t, app)
	require.NoError(t, rt.Start(context.Background()))
	assert.True(t, rt.CheckHealth(context.Background()).Healthy())
	require.NoError(t, rt.Stop(context.Background()))

	assert.Empty(t, baseBuf.String())
	assert.NotEmpty(t, eventLogger.messages)
	assert.NotEmpty(t, eventLogger.builds)
	assert.NotEmpty(t, eventLogger.starts)
	assert.NotEmpty(t, eventLogger.stops)
	assert.NotEmpty(t, eventLogger.health)
	assert.NotEmpty(t, eventLogger.transitions)

	messageTexts := collectionx.MapList(
		collectionx.NewListWithCapacity(len(eventLogger.messages), eventLogger.messages...),
		func(_ int, event dix.MessageEvent) string { return event.Message },
	).Values()
	assert.Contains(t, messageTexts, "registering provider")
	assert.Contains(t, messageTexts, "starting app")
	assert.Contains(t, messageTexts, "stopping app")
}

func TestDIProvidedEventLoggerRoutesDixLogs(t *testing.T) {
	eventLogger := &recordingEventLogger{}
	app := dix.New("di-event-logger",
		dix.Modules(
			dix.NewModule("event-logger",
				dix.Providers(
					dix.Provider0(func() dix.EventLogger {
						return eventLogger
					}),
					dix.Provider0(func() string { return "value" }),
				),
				dix.Hooks(
					dix.OnStartFunc(func() error { return nil }),
					dix.OnStopFunc(func() error { return nil }),
				),
			),
		),
	)

	rt := buildRuntime(t, app)
	require.NoError(t, rt.Start(context.Background()))
	require.NoError(t, rt.Stop(context.Background()))

	assert.NotEmpty(t, eventLogger.messages)
	assert.NotEmpty(t, eventLogger.builds)
	assert.NotEmpty(t, eventLogger.starts)
	assert.NotEmpty(t, eventLogger.stops)
}

func TestExplicitEventLoggerTakesPriorityOverDIProvidedEventLogger(t *testing.T) {
	explicitLogger := &recordingEventLogger{}
	diLogger := &recordingEventLogger{}
	app := dix.New("explicit-event-logger",
		dix.UseEventLogger(explicitLogger),
		dix.Modules(
			dix.NewModule("event-logger",
				dix.Providers(
					dix.Provider0(func() dix.EventLogger {
						return diLogger
					}),
				),
			),
		),
	)

	rt := buildRuntime(t, app)
	require.NoError(t, rt.Start(context.Background()))
	require.NoError(t, rt.Stop(context.Background()))

	assert.NotEmpty(t, explicitLogger.builds)
	assert.NotEmpty(t, explicitLogger.starts)
	assert.NotEmpty(t, explicitLogger.stops)
	assert.Empty(t, diLogger.builds)
	assert.Empty(t, diLogger.starts)
	assert.Empty(t, diLogger.stops)
}

func TestExplicitSlogLoggerTakesPriorityOverDIProvidedEventLogger(t *testing.T) {
	logger, buf := newDebugLogger()
	diLogger := &recordingEventLogger{}
	app := dix.New("explicit-slog-event-logger",
		dix.WithLogger(logger),
		dix.Modules(
			dix.NewModule("event-logger",
				dix.Providers(
					dix.Provider0(func() dix.EventLogger {
						return diLogger
					}),
				),
			),
		),
	)

	rt := buildRuntime(t, app)
	require.NoError(t, rt.Start(context.Background()))
	require.NoError(t, rt.Stop(context.Background()))

	assert.Contains(t, buf.String(), "app built")
	assert.Empty(t, diLogger.builds)
	assert.Empty(t, diLogger.starts)
	assert.Empty(t, diLogger.stops)
}

func TestDIProvidedAppMetaUpdatesRuntimeMeta(t *testing.T) {
	app := dix.New("di-meta",
		dix.Modules(
			dix.NewModule("meta",
				dix.Providers(
					dix.Provider0(func() dix.AppMeta {
						return dix.AppMeta{
							Name:        "ignored-name",
							Version:     "1.2.3",
							Description: "from di",
						}
					}),
				),
			),
		),
	)

	rt := buildRuntime(t, app)
	meta := rt.Meta()
	assert.Equal(t, "di-meta", meta.Name)
	assert.Equal(t, "1.2.3", meta.Version)
	assert.Equal(t, "from di", meta.Description)

	resolved, err := dix.ResolveAs[dix.AppMeta](rt.Container())
	require.NoError(t, err)
	assert.Equal(t, meta, resolved)
}

func TestExplicitAppMetaOptionsTakePriorityOverDIProvidedAppMeta(t *testing.T) {
	app := dix.New("di-meta-priority",
		dix.WithVersion("explicit-version"),
		dix.Modules(
			dix.NewModule("meta",
				dix.Providers(
					dix.Provider0(func() dix.AppMeta {
						return dix.AppMeta{
							Version:     "di-version",
							Description: "from di",
						}
					}),
				),
			),
		),
	)

	rt := buildRuntime(t, app)
	meta := rt.Meta()
	assert.Equal(t, "explicit-version", meta.Version)
	assert.Equal(t, "from di", meta.Description)
}

func TestUseEventLogger1_MissingDependencyFailsBuild(t *testing.T) {
	app := dix.New("event-logger-missing",
		dix.UseEventLogger1(func(*frameworkEventLoggerCarrier) dix.EventLogger {
			return &recordingEventLogger{}
		}),
	)

	_, err := app.Build()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolve framework event logger failed")
}
