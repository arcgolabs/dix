package dix

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/arcgolabs/collectionx"
	"github.com/samber/do/v2"
	"github.com/samber/oops"
)

func newRuntime(spec *appSpec, plan *buildPlan) *Runtime {
	logger := defaultLogger()
	runtimeSpec := cloneAppSpec(spec)
	if runtimeSpec != nil && runtimeSpec.logger != nil {
		logger = runtimeSpec.logger
	}
	eventLogger := NewSlogEventLogger(logger)
	if runtimeSpec != nil && runtimeSpec.eventLogger != nil {
		eventLogger = runtimeSpec.eventLogger
	}
	if runtimeSpec != nil && plan != nil {
		runtimeSpec.profile = plan.profile
	}

	rt := &Runtime{
		spec:        runtimeSpec,
		plan:        plan,
		container:   newContainer(logger),
		lifecycle:   newLifecycle(logger),
		logger:      logger,
		eventLogger: eventLogger,
		state:       AppStateCreated,
	}

	rt.container.logger = rt.logger
	rt.container.eventLogger = rt.eventLogger
	rt.lifecycle.logger = rt.logger
	rt.lifecycle.eventLogger = rt.eventLogger
	rt.spec.rebuildObserverDispatchers(func() *slog.Logger { return rt.logger })

	return rt
}

func cloneAppSpec(spec *appSpec) *appSpec {
	if spec == nil {
		return nil
	}
	cloned := *spec
	cloned.observers = spec.observers.Clone()
	cloned.observerDispatchers = collectionx.NewList[*observerDispatcher]()
	return &cloned
}

// Name returns the runtime application name.
func (r *Runtime) Name() string {
	if r == nil || r.spec == nil {
		return ""
	}
	return r.spec.meta.Name
}

// Profile returns the runtime application profile.
func (r *Runtime) Profile() Profile {
	if r == nil || r.spec == nil {
		return ""
	}
	return r.spec.profile
}

// Logger returns the runtime logger.
func (r *Runtime) Logger() *slog.Logger {
	if r == nil {
		return nil
	}
	return r.logger
}

// EventLogger returns the runtime event logger.
func (r *Runtime) EventLogger() EventLogger {
	if r == nil {
		return nil
	}
	return r.eventLogger
}

// Meta returns the runtime application metadata.
func (r *Runtime) Meta() AppMeta {
	if r == nil || r.spec == nil {
		return AppMeta{}
	}
	return r.spec.meta
}

// State returns the current runtime state.
func (r *Runtime) State() AppState {
	if r == nil {
		return AppStateCreated
	}
	return r.state
}

// Container returns the runtime container wrapper.
func (r *Runtime) Container() *Container {
	if r == nil {
		return nil
	}
	return r.container
}

// Raw returns the underlying do injector for the runtime.
func (r *Runtime) Raw() do.Injector {
	if r == nil || r.container == nil {
		return nil
	}
	return r.container.Raw()
}

// Start executes lifecycle start hooks for the runtime.
func (r *Runtime) Start(ctx context.Context) (err error) {
	startedAt := time.Now()
	startedHooks := 0
	defer func() {
		if r != nil {
			r.emitStart(ctx, r.startEvent(time.Since(startedAt), startedHooks, err))
		}
	}()

	if r == nil {
		err = oops.In("dix").
			With("op", "start").
			New("runtime is nil")
		return err
	}
	if r.state != AppStateBuilt {
		err = oops.In("dix").
			With("op", "start", "app", r.Name(), "state", r.state.String()).
			Errorf("runtime must be built before starting")
		return err
	}

	r.transitionState(ctx, AppStateStarting, "start requested")
	r.logMessage(ctx, EventLevelInfo, "starting app", "app", r.Name())
	startedHooks, err = r.lifecycle.executeStartHooks(ctx, r.container)
	if err != nil {
		r.logMessage(ctx, EventLevelDebug, "rolling back app start",
			"app", r.Name(),
			"started_hooks", startedHooks,
			"rollback_stop_hooks", startedHooks,
		)
		rollbackErr := r.lifecycle.executeStopHooksSubset(ctx, startedHooks)
		shutdownReport := r.container.ShutdownReport(ctx)
		startErr := errors.Join(err, rollbackErr, shutdownReport)
		r.transitionState(ctx, AppStateStopped, "start failed")
		err = startErr
		return err
	}

	r.transitionState(ctx, AppStateStarted, "start completed")
	return nil
}

// Stop executes lifecycle stop hooks and shuts down the runtime.
func (r *Runtime) Stop(ctx context.Context) error {
	report, err := r.StopWithReport(ctx)
	if err != nil {
		return err
	}
	if report != nil {
		return report.Err()
	}
	return nil
}

// StopWithReport executes runtime shutdown and returns a detailed stop report.
func (r *Runtime) StopWithReport(ctx context.Context) (report *StopReport, err error) {
	startedAt := time.Now()
	defer func() {
		if r != nil {
			r.emitStop(ctx, r.stopEvent(time.Since(startedAt), report, err))
		}
	}()

	if validateErr := r.validateStoppable(); validateErr != nil {
		return nil, validateErr
	}

	debugEnabled := eventLoggerEnabled(ctx, r.eventLogger, EventLevelDebug)
	r.logMessage(ctx, EventLevelInfo, "stopping app", "app", r.Name())
	r.logMessage(ctx, EventLevelDebug, "executing runtime stop",
		"app", r.Name(),
		"stop_hooks", r.lifecycle.stopHooks.Len(),
	)

	report = r.executeStopSequence(ctx)
	r.logStopReport(ctx, debugEnabled, report)

	r.transitionState(ctx, AppStateStopped, "stop completed")

	err = report.Err()
	return report, err
}

func (r *Runtime) logDebugInformation(ctx context.Context) {
	if r == nil || r.spec == nil {
		return
	}

	if r.spec.debug.scopeTree {
		injector := do.ExplainInjector(r.container.Raw())
		r.logMessage(ctx, EventLevelInfo, "do scope tree", "app", r.Name(), "tree", injector.String())
	}

	r.spec.debug.namedServiceDependencies.Range(func(name string) bool {
		if desc, found := do.ExplainNamedService(r.container.Raw(), name); found {
			r.logMessage(ctx, EventLevelInfo, "do named service dependencies", "app", r.Name(), "name", name, "dependencies", desc.String())
		} else {
			r.logMessage(ctx, EventLevelWarn, "do named service not found", "app", r.Name(), "name", name)
		}
		return true
	})
}

func (r *Runtime) transitionState(ctx context.Context, next AppState, reason string) {
	if r == nil {
		return
	}
	prev := r.state
	r.state = next
	if prev != next {
		r.emitStateTransition(ctx, StateTransitionEvent{
			Meta:    r.Meta(),
			Profile: r.Profile(),
			From:    prev,
			To:      next,
			Reason:  reason,
		})
	}
}

func (r *Runtime) validateStoppable() error {
	if r == nil {
		return oops.In("dix").
			With("op", "stop").
			New("runtime is nil")
	}
	if r.state == AppStateStarting {
		return oops.In("dix").
			With("op", "stop", "app", r.Name(), "state", r.state.String()).
			Errorf("runtime is still starting")
	}
	if r.state != AppStateStarted {
		return oops.In("dix").
			With("op", "stop", "app", r.Name(), "state", r.state.String()).
			Errorf("runtime must be started before stopping")
	}
	return nil
}

func (r *Runtime) executeStopSequence(ctx context.Context) *StopReport {
	report := &StopReport{}
	if err := r.lifecycle.executeStopHooks(ctx, r.container); err != nil {
		r.logMessage(ctx, EventLevelError, "stop hooks failed", "app", r.Name(), "error", err)
		report.HookError = err
	}

	report.ShutdownReport = r.container.ShutdownReport(ctx)
	if report.ShutdownReport != nil && len(report.ShutdownReport.Errors) > 0 {
		r.logMessage(ctx, EventLevelError, "container shutdown failed", "app", r.Name(), "error", report.ShutdownReport)
	}
	return report
}

func (r *Runtime) logStopReport(ctx context.Context, debugEnabled bool, report *StopReport) {
	if !debugEnabled {
		return
	}
	r.logMessage(ctx, EventLevelDebug, "runtime stop report",
		"app", r.Name(),
		"hook_error", report.HookError != nil,
		"shutdown_errors", shutdownErrorCount(report),
	)
}

func (r *Runtime) logMessage(ctx context.Context, level EventLevel, message string, args ...any) {
	if r == nil {
		return
	}
	logMessageEvent(ctx, r.eventLogger, level, message, args...)
}

func shutdownErrorCount(report *StopReport) int {
	if report == nil || report.ShutdownReport == nil {
		return 0
	}
	return len(report.ShutdownReport.Errors)
}
