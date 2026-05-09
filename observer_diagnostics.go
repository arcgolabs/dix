package dix

import (
	"context"
	"time"
)

// ProviderObserver optionally receives provider-level diagnostic events.
type ProviderObserver interface {
	OnProvider(context.Context, ProviderEvent)
}

// ResolveObserver optionally receives service resolution diagnostic events.
type ResolveObserver interface {
	OnResolve(context.Context, ResolveEvent)
}

// LifecycleHookObserver optionally receives individual lifecycle hook diagnostic events.
type LifecycleHookObserver interface {
	OnLifecycleHook(context.Context, LifecycleHookEvent)
}

// ProviderEvent describes provider registration or eager warmup diagnostics.
type ProviderEvent struct {
	Meta      AppMeta
	Profile   Profile
	Module    string
	Label     string
	Service   string
	Operation string
	Duration  time.Duration
	Err       error
}

// ResolveEvent describes service construction or explicit resolution diagnostics.
type ResolveEvent struct {
	Meta      AppMeta
	Profile   Profile
	Service   string
	Operation string
	Duration  time.Duration
	Err       error
}

// LifecycleHookEvent describes one lifecycle hook execution.
type LifecycleHookEvent struct {
	Meta     AppMeta
	Profile  Profile
	Kind     HookKind
	Name     string
	Label    string
	Priority int
	Parallel bool
	Timeout  time.Duration
	Sequence int
	Duration time.Duration
	Err      error
}

func (r *Runtime) emitProvider(ctx context.Context, event ProviderEvent) {
	if r == nil || r.spec == nil {
		return
	}
	event.Meta = r.Meta()
	event.Profile = r.Profile()
	emitEventLogger(ctx, r.eventLogger, event)
	emitObservers(ctx, r.spec.observerDispatchers, func(ctx context.Context, observer Observer) {
		if target, ok := observer.(ProviderObserver); ok {
			target.OnProvider(ctx, event)
		}
	})
}

func (r *Runtime) emitResolve(ctx context.Context, event ResolveEvent) {
	if r == nil || r.spec == nil {
		return
	}
	event.Meta = r.Meta()
	event.Profile = r.Profile()
	emitEventLogger(ctx, r.eventLogger, event)
	emitObservers(ctx, r.spec.observerDispatchers, func(ctx context.Context, observer Observer) {
		if target, ok := observer.(ResolveObserver); ok {
			target.OnResolve(ctx, event)
		}
	})
}

func (r *Runtime) emitLifecycleHook(ctx context.Context, event LifecycleHookEvent) {
	if r == nil || r.spec == nil {
		return
	}
	event.Meta = r.Meta()
	event.Profile = r.Profile()
	emitEventLogger(ctx, r.eventLogger, event)
	emitObservers(ctx, r.spec.observerDispatchers, func(ctx context.Context, observer Observer) {
		if target, ok := observer.(LifecycleHookObserver); ok {
			target.OnLifecycleHook(ctx, event)
		}
	})
}
