package dix

import (
	"context"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
)

// Observer receives asynchronously dispatched framework lifecycle events emitted by dix.
//
// Observer callbacks must be safe for concurrent use. Implementations should
// avoid long blocking work or panics; dix recovers observer panics and continues.
type Observer interface {
	OnBuild(context.Context, BuildEvent)
	OnStart(context.Context, StartEvent)
	OnStop(context.Context, StopEvent)
	OnHealthCheck(context.Context, HealthCheckEvent)
	OnStateTransition(context.Context, StateTransitionEvent)
}

// BuildEvent describes a completed runtime build attempt.
type BuildEvent struct {
	Meta          AppMeta
	Profile       Profile
	ModuleCount   int
	ProviderCount int
	HookCount     int
	SetupCount    int
	InvokeCount   int
	Duration      time.Duration
	Err           error
}

// StartEvent describes a completed runtime start attempt.
type StartEvent struct {
	Meta             AppMeta
	Profile          Profile
	StartHookCount   int
	StartedHookCount int
	RolledBack       bool
	Duration         time.Duration
	Err              error
}

// StopEvent describes a completed runtime stop attempt.
type StopEvent struct {
	Meta               AppMeta
	Profile            Profile
	StopHookCount      int
	HookError          bool
	ShutdownErrorCount int
	Duration           time.Duration
	Err                error
}

// HealthCheckEvent describes a completed framework health check invocation.
type HealthCheckEvent struct {
	Meta     AppMeta
	Profile  Profile
	Kind     HealthKind
	Name     string
	Duration time.Duration
	Err      error
}

// StateTransitionEvent describes an internal runtime state transition.
type StateTransitionEvent struct {
	Meta    AppMeta
	Profile Profile
	From    AppState
	To      AppState
	Reason  string
}

func (spec *appSpec) emitBuild(ctx context.Context, event BuildEvent) {
	if spec == nil {
		return
	}
	emitEventLogger(ctx, spec.resolvedEventLogger(), event)
	emitObservers(ctx, spec.observerDispatchers, func(ctx context.Context, observer Observer) {
		observer.OnBuild(ctx, event)
	})
}

func (r *Runtime) emitStart(ctx context.Context, event StartEvent) {
	if r == nil || r.spec == nil {
		return
	}
	emitEventLogger(ctx, r.eventLogger, event)
	emitObservers(ctx, r.spec.observerDispatchers, func(ctx context.Context, observer Observer) {
		observer.OnStart(ctx, event)
	})
}

func (r *Runtime) emitStop(ctx context.Context, event StopEvent) {
	if r == nil || r.spec == nil {
		return
	}
	emitEventLogger(ctx, r.eventLogger, event)
	emitObservers(ctx, r.spec.observerDispatchers, func(ctx context.Context, observer Observer) {
		observer.OnStop(ctx, event)
	})
}

func (r *Runtime) emitHealthCheck(ctx context.Context, event HealthCheckEvent) {
	if r == nil || r.spec == nil {
		return
	}
	emitEventLogger(ctx, r.eventLogger, event)
	emitObservers(ctx, r.spec.observerDispatchers, func(ctx context.Context, observer Observer) {
		observer.OnHealthCheck(ctx, event)
	})
}

func (r *Runtime) emitStateTransition(ctx context.Context, event StateTransitionEvent) {
	if r == nil || r.spec == nil {
		return
	}
	emitEventLogger(ctx, r.eventLogger, event)
	emitObservers(ctx, r.spec.observerDispatchers, func(ctx context.Context, observer Observer) {
		observer.OnStateTransition(ctx, event)
	})
}

func emitObservers(ctx context.Context, dispatchers collectionx.List[*observerDispatcher], emit func(context.Context, Observer)) {
	if dispatchers == nil || dispatchers.Len() == 0 || emit == nil {
		return
	}
	dispatchers.Range(func(_ int, dispatcher *observerDispatcher) bool {
		if dispatcher == nil {
			return true
		}
		dispatcher.enqueue(ctx, emit)
		return true
	})
}

func (p *buildPlan) buildEvent(duration time.Duration, err error) BuildEvent {
	if p == nil || p.spec == nil {
		return BuildEvent{Duration: duration, Err: err}
	}
	return BuildEvent{
		Meta:          p.spec.meta,
		Profile:       p.profile,
		ModuleCount:   countModules(p.modules),
		ProviderCount: countModuleProviders(p.modules),
		HookCount:     countModuleHooks(p.modules),
		SetupCount:    countModuleSetups(p.modules),
		InvokeCount:   countModuleInvokes(p.modules),
		Duration:      duration,
		Err:           err,
	}
}

func (p *buildPlan) runtimeBuildEvent(rt *Runtime, duration time.Duration, err error) BuildEvent {
	if p == nil || rt == nil {
		return BuildEvent{Duration: duration, Err: err}
	}
	return BuildEvent{
		Meta:          rt.Meta(),
		Profile:       rt.Profile(),
		ModuleCount:   countModules(p.modules),
		ProviderCount: countModuleProviders(p.modules),
		HookCount:     countModuleHooks(p.modules),
		SetupCount:    countModuleSetups(p.modules),
		InvokeCount:   countModuleInvokes(p.modules),
		Duration:      duration,
		Err:           err,
	}
}

func (r *Runtime) startEvent(duration time.Duration, startedHooks int, err error) StartEvent {
	if r == nil {
		return StartEvent{Duration: duration, Err: err}
	}
	startHookCount := 0
	if r.lifecycle != nil {
		startHookCount = r.lifecycle.startHooks.Len()
	}
	return StartEvent{
		Meta:             r.Meta(),
		Profile:          r.Profile(),
		StartHookCount:   startHookCount,
		StartedHookCount: startedHooks,
		RolledBack:       err != nil && startedHooks > 0,
		Duration:         duration,
		Err:              err,
	}
}

func (r *Runtime) stopEvent(duration time.Duration, report *StopReport, err error) StopEvent {
	if r == nil {
		return StopEvent{Duration: duration, Err: err}
	}
	stopHookCount := 0
	if r.lifecycle != nil {
		stopHookCount = r.lifecycle.stopHooks.Len()
	}
	return StopEvent{
		Meta:               r.Meta(),
		Profile:            r.Profile(),
		StopHookCount:      stopHookCount,
		HookError:          report != nil && report.HookError != nil,
		ShutdownErrorCount: shutdownErrorCount(report),
		Duration:           duration,
		Err:                err,
	}
}

func (r *Runtime) healthCheckEvent(check healthCheckEntry, duration time.Duration, err error) HealthCheckEvent {
	if r == nil {
		return HealthCheckEvent{
			Kind:     check.kind,
			Name:     check.name,
			Duration: duration,
			Err:      err,
		}
	}
	return HealthCheckEvent{
		Meta:     r.Meta(),
		Profile:  r.Profile(),
		Kind:     check.kind,
		Name:     check.name,
		Duration: duration,
		Err:      err,
	}
}
