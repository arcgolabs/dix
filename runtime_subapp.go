package dix

import (
	"context"
	"errors"
	"log/slog"

	"github.com/arcgolabs/collectionx"
	"github.com/samber/oops"
)

func newChildRuntime(spec *appSpec, plan *buildPlan, parent *Runtime) (*Runtime, error) {
	if parent == nil || parent.container == nil {
		return nil, oops.In("dix").
			With("op", "new_child_runtime").
			New("parent runtime is nil")
	}

	runtimeSpec := cloneAppSpec(spec)
	logger := resolveChildRuntimeLogger(parent, runtimeSpec)
	eventLogger := resolveChildRuntimeEventLogger(parent, runtimeSpec)
	if runtimeSpec != nil && plan != nil {
		runtimeSpec.profile = plan.profile
	}

	container, err := newChildContainer(parent.container, runtimeSpec.meta.Name, logger)
	if err != nil {
		return nil, oops.In("dix").
			With("op", "new_child_runtime", "app", runtimeSpec.meta.Name).
			Wrapf(err, "create child container failed")
	}
	return newRuntimeFromParts(runtimeSpec, plan, container, logger, eventLogger), nil
}

func resolveChildRuntimeLogger(parent *Runtime, spec *appSpec) *slog.Logger {
	logger := parent.logger
	if spec != nil && spec.loggerConfigured && spec.logger != nil {
		logger = spec.logger
	}
	return logger
}

func resolveChildRuntimeEventLogger(parent *Runtime, spec *appSpec) EventLogger {
	eventLogger := parent.eventLogger
	if spec != nil && spec.eventLoggerConfigured && spec.eventLogger != nil {
		eventLogger = spec.eventLogger
	}
	return eventLogger
}

func newRuntimeFromParts(
	spec *appSpec,
	plan *buildPlan,
	container *Container,
	logger *slog.Logger,
	eventLogger EventLogger,
) *Runtime {
	rt := &Runtime{
		spec:        spec,
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
	return rt
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

func (r *Runtime) rollbackStart(ctx context.Context, startedHooks, startedSubApps int, cause error) error {
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

func (r *Runtime) cleanupBuildFailure(ctx context.Context) error {
	if r == nil || r.container == nil {
		return nil
	}

	errs := collectionx.NewList[error]()
	if err := r.cleanupBuiltSubApps(ctx); err != nil {
		errs.Add(err)
	}
	report := r.container.ShutdownReport(ctx)
	if report != nil && len(report.Errors) > 0 {
		errs.Add(report)
	}
	return errors.Join(errs.Values()...)
}

func (r *Runtime) cleanupBuiltSubApps(ctx context.Context) error {
	if r == nil || r.subapps == nil || r.subapps.Len() == 0 {
		return nil
	}

	errs := collectionx.NewList[error]()
	for i := r.subapps.Len() - 1; i >= 0; i-- {
		subapp, _ := r.subapps.Get(i)
		if subapp == nil {
			continue
		}
		if err := subapp.cleanupBuildFailure(ctx); err != nil {
			errs.Add(oops.In("dix").
				With("op", "cleanup_subapp_build_failure", "app", r.Name(), "subapp", subapp.Name()).
				Wrapf(err, "cleanup subapp build failure"))
		}
	}
	return errors.Join(errs.Values()...)
}
