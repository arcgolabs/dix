package dix

import (
	"context"
	"errors"
	"log/slog"

	"github.com/arcgolabs/collectionx"
	"github.com/samber/oops"
)

// StartHook is executed when the application starts.
type StartHook func(ctx context.Context) error

// StopHook is executed when the application stops.
type StopHook func(ctx context.Context) error

// Lifecycle manages application lifecycle hooks.
type Lifecycle interface {
	OnStart(hook StartHook)
	OnStop(hook StopHook)
}

// HookFunc describes a typed lifecycle hook registration.
type HookFunc struct {
	register func(*Container, Lifecycle)
	meta     HookMetadata
}

func (h HookFunc) bind(c *Container, lc Lifecycle) {
	if h.register != nil {
		h.register(c, lc)
	}
}

// RawHook registers an untyped lifecycle hook.
func RawHook(fn func(*Container, Lifecycle)) HookFunc {
	return RawHookWithMetadata(fn, HookMetadata{
		Label: "RawHook",
	})
}

// RawHookWithMetadata registers an untyped lifecycle hook with metadata.
func RawHookWithMetadata(fn func(*Container, Lifecycle), meta HookMetadata) HookFunc {
	meta.Raw = true
	return NewHookFunc(fn, meta)
}

// lifecycleImpl is the internal implementation.
type lifecycleImpl struct {
	startHooks  collectionx.List[StartHook]
	stopHooks   collectionx.List[StopHook]
	logger      *slog.Logger
	eventLogger EventLogger
}

func newLifecycle(logger *slog.Logger) *lifecycleImpl {
	if logger == nil {
		logger = defaultLogger()
	}
	return &lifecycleImpl{
		startHooks: collectionx.NewList[StartHook](),
		stopHooks:  collectionx.NewList[StopHook](),
		logger:     logger,
	}
}

func (l *lifecycleImpl) OnStart(hook StartHook) {
	l.startHooks.Add(hook)
}

func (l *lifecycleImpl) OnStop(hook StopHook) {
	l.stopHooks.Add(hook)
}

func (l *lifecycleImpl) executeStartHooks(ctx context.Context, _ *Container) (int, error) {
	debugEnabled := l.debugEnabled(ctx)
	l.logDebug(ctx, debugEnabled, "executing start hooks", "count", l.startHooks.Len())

	completed := 0
	var startErr error
	l.startHooks.Range(func(i int, hook StartHook) bool {
		l.logDebug(ctx, debugEnabled, "executing start hook", "index", i)
		if err := hook(ctx); err != nil {
			logMessageEvent(ctx, l.eventLogger, EventLevelError, "start hook failed", "index", i, "error", err)
			startErr = oops.In("dix").
				With("op", "start_hook", "index", i).
				Wrapf(err, "start hook %d failed", i)
			return false
		}
		l.logDebug(ctx, debugEnabled, "start hook completed", "index", i)
		completed++
		return true
	})
	return completed, startErr
}

func (l *lifecycleImpl) executeStopHooks(ctx context.Context, _ *Container) error {
	return l.executeStopHooksSubset(ctx, l.stopHooks.Len())
}

func (l *lifecycleImpl) executeStopHooksSubset(ctx context.Context, count int) error {
	if count <= 0 {
		return nil
	}

	registered := l.stopHooks.Len()
	if count > registered {
		count = registered
	}
	debugEnabled := l.debugEnabled(ctx)
	l.logDebug(ctx, debugEnabled, "executing stop hooks", "count", count, "registered", registered)

	errs := collectionx.NewListWithCapacity[error](1)
	for i := count - 1; i >= 0; i-- {
		hook, _ := l.stopHooks.Get(i)
		l.logDebug(ctx, debugEnabled, "executing stop hook", "index", count-1-i)
		if err := hook(ctx); err != nil {
			logMessageEvent(ctx, l.eventLogger, EventLevelError, "stop hook failed", "index", count-1-i, "error", err)
			errs.Add(oops.In("dix").
				With("op", "stop_hook", "index", count-1-i).
				Wrapf(err, "stop hook %d failed", count-1-i))
			continue
		}
		l.logDebug(ctx, debugEnabled, "stop hook completed", "index", count-1-i)
	}
	return errors.Join(errs.Values()...)
}

func (l *lifecycleImpl) debugEnabled(ctx context.Context) bool {
	return eventLoggerEnabled(ctx, l.eventLogger, EventLevelDebug)
}

func (l *lifecycleImpl) logDebug(ctx context.Context, enabled bool, msg string, args ...any) {
	if enabled {
		logMessageEvent(ctx, l.eventLogger, EventLevelDebug, msg, args...)
	}
}

// OnStart0 registers a start hook with no resolved dependencies.
func OnStart0(fn func(context.Context) error) HookFunc {
	return NewHookFunc(func(_ *Container, lc Lifecycle) {
		lc.OnStart(fn)
	}, HookMetadata{
		Label: "OnStart0",
		Kind:  HookKindStart,
	})
}

// OnStartFunc registers a start hook with no resolved dependencies and no context usage.
func OnStartFunc(fn func() error) HookFunc {
	return OnStart0(func(context.Context) error {
		return fn()
	})
}

// OnStop0 registers a stop hook with no resolved dependencies.
func OnStop0(fn func(context.Context) error) HookFunc {
	return NewHookFunc(func(_ *Container, lc Lifecycle) {
		lc.OnStop(fn)
	}, HookMetadata{
		Label: "OnStop0",
		Kind:  HookKindStop,
	})
}

// OnStopFunc registers a stop hook with no resolved dependencies and no context usage.
func OnStopFunc(fn func() error) HookFunc {
	return OnStop0(func(context.Context) error {
		return fn()
	})
}

// OnStart registers a start hook with one resolved dependency.
func OnStart[T any](fn func(context.Context, T) error) HookFunc {
	return NewHookFunc(func(c *Container, lc Lifecycle) {
		lc.OnStart(func(ctx context.Context) error {
			t, err := resolveDependency1[T](c.Raw())
			if err != nil {
				return oops.In("dix").
					With("op", "resolve_hook_dependency", "hook", "OnStart").
					Wrapf(err, "resolving dependency")
			}
			return fn(ctx, t)
		})
	}, HookMetadata{
		Label:        "OnStart",
		Kind:         HookKindStart,
		Dependencies: ServiceRefs(TypedService[T]()),
	})
}

// OnStop registers a stop hook with one resolved dependency.
func OnStop[T any](fn func(context.Context, T) error) HookFunc {
	return NewHookFunc(func(c *Container, lc Lifecycle) {
		lc.OnStop(func(ctx context.Context) error {
			t, err := resolveDependency1[T](c.Raw())
			if err != nil {
				return oops.In("dix").
					With("op", "resolve_hook_dependency", "hook", "OnStop").
					Wrapf(err, "resolving dependency")
			}
			return fn(ctx, t)
		})
	}, HookMetadata{
		Label:        "OnStop",
		Kind:         HookKindStop,
		Dependencies: ServiceRefs(TypedService[T]()),
	})
}

// OnStart2 registers a start hook with two resolved dependencies.
func OnStart2[T1, T2 any](fn func(context.Context, T1, T2) error) HookFunc {
	return NewHookFunc(func(c *Container, lc Lifecycle) {
		lc.OnStart(func(ctx context.Context) error {
			t1, t2, err := resolveDependencies2[T1, T2](c.Raw())
			if err != nil {
				return err
			}
			return fn(ctx, t1, t2)
		})
	}, HookMetadata{
		Label:        "OnStart2",
		Kind:         HookKindStart,
		Dependencies: ServiceRefs(TypedService[T1](), TypedService[T2]()),
	})
}

// OnStop2 registers a stop hook with two resolved dependencies.
func OnStop2[T1, T2 any](fn func(context.Context, T1, T2) error) HookFunc {
	return NewHookFunc(func(c *Container, lc Lifecycle) {
		lc.OnStop(func(ctx context.Context) error {
			t1, t2, err := resolveDependencies2[T1, T2](c.Raw())
			if err != nil {
				return err
			}
			return fn(ctx, t1, t2)
		})
	}, HookMetadata{
		Label:        "OnStop2",
		Kind:         HookKindStop,
		Dependencies: ServiceRefs(TypedService[T1](), TypedService[T2]()),
	})
}

// OnStart3 registers a start hook with three resolved dependencies.
func OnStart3[T1, T2, T3 any](fn func(context.Context, T1, T2, T3) error) HookFunc {
	return NewHookFunc(func(c *Container, lc Lifecycle) {
		lc.OnStart(func(ctx context.Context) error {
			t1, t2, t3, err := resolveDependencies3[T1, T2, T3](c.Raw())
			if err != nil {
				return err
			}
			return fn(ctx, t1, t2, t3)
		})
	}, HookMetadata{
		Label:        "OnStart3",
		Kind:         HookKindStart,
		Dependencies: ServiceRefs(TypedService[T1](), TypedService[T2](), TypedService[T3]()),
	})
}

// OnStop3 registers a stop hook with three resolved dependencies.
func OnStop3[T1, T2, T3 any](fn func(context.Context, T1, T2, T3) error) HookFunc {
	return NewHookFunc(func(c *Container, lc Lifecycle) {
		lc.OnStop(func(ctx context.Context) error {
			t1, t2, t3, err := resolveDependencies3[T1, T2, T3](c.Raw())
			if err != nil {
				return err
			}
			return fn(ctx, t1, t2, t3)
		})
	}, HookMetadata{
		Label:        "OnStop3",
		Kind:         HookKindStop,
		Dependencies: ServiceRefs(TypedService[T1](), TypedService[T2](), TypedService[T3]()),
	})
}
