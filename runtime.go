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
		subapps:     collectionx.NewList[*Runtime](),
	}

	rt.container.logger = rt.logger
	rt.container.eventLogger = rt.eventLogger
	rt.lifecycle.logger = rt.logger
	rt.lifecycle.eventLogger = rt.eventLogger
	rt.spec.rebuildObserverDispatchers(func() *slog.Logger { return rt.logger })

	return rt
}

func newChildRuntime(spec *appSpec, plan *buildPlan, parent *Runtime) (*Runtime, error) {
	if parent == nil || parent.container == nil {
		return nil, oops.In("dix").
			With("op", "new_child_runtime").
			New("parent runtime is nil")
	}

	logger := parent.logger
	runtimeSpec := cloneAppSpec(spec)
	if runtimeSpec != nil && runtimeSpec.loggerConfigured && runtimeSpec.logger != nil {
		logger = runtimeSpec.logger
	}
	eventLogger := parent.eventLogger
	if runtimeSpec != nil && runtimeSpec.eventLoggerConfigured && runtimeSpec.eventLogger != nil {
		eventLogger = runtimeSpec.eventLogger
	}
	if runtimeSpec != nil && plan != nil {
		runtimeSpec.profile = plan.profile
	}

	container, err := newChildContainer(parent.container, runtimeSpec.meta.Name, logger)
	if err != nil {
		return nil, oops.In("dix").
			With("op", "new_child_runtime", "app", runtimeSpec.meta.Name).
			Wrapf(err, "create child container failed")
	}

	rt := &Runtime{
		spec:        runtimeSpec,
		plan:        plan,
		container:   container,
		lifecycle:   newLifecycle(logger),
		logger:      logger,
		eventLogger: eventLogger,
		state:       AppStateCreated,
		subapps:     collectionx.NewList[*Runtime](),
	}

	rt.container.logger = rt.logger
	rt.container.eventLogger = rt.eventLogger
	rt.lifecycle.logger = rt.logger
	rt.lifecycle.eventLogger = rt.eventLogger
	rt.spec.rebuildObserverDispatchers(func() *slog.Logger { return rt.logger })

	return rt, nil
}

func cloneAppSpec(spec *appSpec) *appSpec {
	if spec == nil {
		return nil
	}
	cloned := *spec
	cloned.observers = spec.observers.Clone()
	cloned.observerDispatchers = collectionx.NewList[*observerDispatcher]()
	cloned.subapps = spec.subapps.Clone()
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

// SubApps returns built child runtimes in declaration order.
func (r *Runtime) SubApps() collectionx.List[*Runtime] {
	if r == nil || r.subapps == nil {
		return collectionx.NewList[*Runtime]()
	}
	return r.subapps.Clone()
}

// SubApp returns a built child runtime by app name.
func (r *Runtime) SubApp(name string) (*Runtime, bool) {
	if r == nil || r.subapps == nil {
		return nil, false
	}
	var found *Runtime
	r.subapps.Range(func(_ int, subapp *Runtime) bool {
		if subapp != nil && subapp.Name() == name {
			found = subapp
			return false
		}
		return true
	})
	return found, found != nil
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
		err = r.rollbackStart(ctx, startedHooks, 0, err)
		return err
	}

	startedSubApps, err := r.startSubApps(ctx)
	if err != nil {
		err = r.rollbackStart(ctx, startedHooks, startedSubApps, err)
		return err
	}

	r.transitionState(ctx, AppStateStarted, "start completed")
	return nil
}

func (r *Runtime) startSubApps(ctx context.Context) (int, error) {
	if r == nil || r.subapps == nil || r.subapps.Len() == 0 {
		return 0, nil
	}

	started := 0
	var startErr error
	r.subapps.Range(func(_ int, subapp *Runtime) bool {
		if subapp == nil {
			return true
		}
		if err := subapp.Start(ctx); err != nil {
			startErr = oops.In("dix").
				With("op", "start_subapp", "app", r.Name(), "subapp", subapp.Name()).
				Wrapf(err, "start subapp failed")
			return false
		}
		started++
		return true
	})
	return started, startErr
}

func (r *Runtime) rollbackStart(ctx context.Context, startedHooks int, startedSubApps int, cause error) error {
	r.logMessage(ctx, EventLevelDebug, "rolling back app start",
		"app", r.Name(),
		"started_hooks", startedHooks,
		"rollback_stop_hooks", startedHooks,
		"started_subapps", startedSubApps,
	)
	subappErr := r.stopSubAppsSubset(ctx, startedSubApps)
	rollbackErr := r.lifecycle.executeStopHooksSubset(ctx, startedHooks)
	shutdownReport := r.container.ShutdownReport(ctx)
	startErr := errors.Join(cause, subappErr, rollbackErr, shutdownReport)
	r.transitionState(ctx, AppStateStopped, "start failed")
	return startErr
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
	if err := r.stopSubApps(ctx); err != nil {
		r.logMessage(ctx, EventLevelError, "subapp stop failed", "app", r.Name(), "error", err)
		report.SubAppError = err
	}
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

func (r *Runtime) stopSubApps(ctx context.Context) error {
	if r == nil || r.subapps == nil {
		return nil
	}
	return r.stopSubAppsSubset(ctx, r.subapps.Len())
}

func (r *Runtime) stopSubAppsSubset(ctx context.Context, count int) error {
	if r == nil || r.subapps == nil || count <= 0 {
		return nil
	}
	if count > r.subapps.Len() {
		count = r.subapps.Len()
	}

	errs := collectionx.NewList[error]()
	for i := count - 1; i >= 0; i-- {
		subapp, _ := r.subapps.Get(i)
		if subapp == nil || subapp.State() != AppStateStarted {
			continue
		}
		if err := subapp.Stop(ctx); err != nil {
			errs.Add(oops.In("dix").
				With("op", "stop_subapp", "app", r.Name(), "subapp", subapp.Name()).
				Wrapf(err, "stop subapp failed"))
		}
	}
	return errors.Join(errs.Values()...)
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
