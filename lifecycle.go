package dix

import (
	"context"

	collectionlist "github.com/arcgolabs/collectionx/list"
	"github.com/samber/oops"
	"log/slog"
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
		h.register(c, lifecycleWithMetadata{Lifecycle: lc, meta: h.meta})
	}
}

// RawHook registers an untyped lifecycle hook.
func RawHook(fn func(*Container, Lifecycle), opts ...LifecycleHookOption) HookFunc {
	return RawHookWithMetadata(fn, applyLifecycleHookOptions(HookMetadata{
		Label: "RawHook",
	}, opts...))
}

// RawHookWithMetadata registers an untyped lifecycle hook with metadata.
func RawHookWithMetadata(fn func(*Container, Lifecycle), meta HookMetadata) HookFunc {
	meta.Raw = true
	return NewHookFunc(fn, meta)
}

type lifecycleWithMetadata struct {
	Lifecycle
	meta HookMetadata
}

func (l lifecycleWithMetadata) OnStart(hook StartHook) {
	if target, ok := l.Lifecycle.(interface {
		onStartWithMetadata(StartHook, HookMetadata)
	}); ok {
		target.onStartWithMetadata(hook, l.meta)
		return
	}
	l.Lifecycle.OnStart(hook)
}

func (l lifecycleWithMetadata) OnStop(hook StopHook) {
	if target, ok := l.Lifecycle.(interface {
		onStopWithMetadata(StopHook, HookMetadata)
	}); ok {
		target.onStopWithMetadata(hook, l.meta)
		return
	}
	l.Lifecycle.OnStop(hook)
}

type lifecycleHookEntry struct {
	run      func(context.Context) error
	meta     HookMetadata
	sequence int
}

// lifecycleImpl is the internal implementation.
type lifecycleImpl struct {
	startHooks  *collectionlist.List[lifecycleHookEntry]
	stopHooks   *collectionlist.List[lifecycleHookEntry]
	nextSeq     int
	concurrency int
	logger      *slog.Logger
	eventLogger EventLogger
	emitHook    func(context.Context, LifecycleHookEvent)
}

func newLifecycle(logger *slog.Logger, concurrency ...int) *lifecycleImpl {
	if logger == nil {
		logger = defaultLogger()
	}
	resolvedConcurrency := 0
	if len(concurrency) > 0 {
		resolvedConcurrency = concurrency[0]
	}
	return &lifecycleImpl{
		startHooks:  collectionlist.NewList[lifecycleHookEntry](),
		stopHooks:   collectionlist.NewList[lifecycleHookEntry](),
		concurrency: resolvedConcurrency,
		logger:      logger,
	}
}

func (l *lifecycleImpl) OnStart(hook StartHook) {
	l.onStartWithMetadata(hook, HookMetadata{Label: "OnStart", Kind: HookKindStart})
}

func (l *lifecycleImpl) OnStop(hook StopHook) {
	l.onStopWithMetadata(hook, HookMetadata{Label: "OnStop", Kind: HookKindStop})
}

func (l *lifecycleImpl) onStartWithMetadata(hook StartHook, meta HookMetadata) {
	if hook == nil {
		return
	}
	meta.Kind = HookKindStart
	l.startHooks.Add(l.newHookEntry(func(ctx context.Context) error {
		return hook(ctx)
	}, normalizeHookMetadata(meta)))
}

func (l *lifecycleImpl) onStopWithMetadata(hook StopHook, meta HookMetadata) {
	if hook == nil {
		return
	}
	meta.Kind = HookKindStop
	l.stopHooks.Add(l.newHookEntry(func(ctx context.Context) error {
		return hook(ctx)
	}, normalizeHookMetadata(meta)))
}

func (l *lifecycleImpl) newHookEntry(run func(context.Context) error, meta HookMetadata) lifecycleHookEntry {
	entry := lifecycleHookEntry{
		run:      run,
		meta:     meta,
		sequence: l.nextSeq,
	}
	l.nextSeq++
	return entry
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
func OnStart0(fn func(context.Context) error, opts ...LifecycleHookOption) HookFunc {
	return NewHookFunc(func(_ *Container, lc Lifecycle) {
		lc.OnStart(fn)
	}, applyLifecycleHookOptions(HookMetadata{
		Label: "OnStart0",
		Kind:  HookKindStart,
	}, opts...))
}

// OnStartFunc registers a start hook with no resolved dependencies and no context usage.
func OnStartFunc(fn func() error, opts ...LifecycleHookOption) HookFunc {
	return OnStart0(func(context.Context) error {
		return fn()
	}, opts...)
}

// OnStop0 registers a stop hook with no resolved dependencies.
func OnStop0(fn func(context.Context) error, opts ...LifecycleHookOption) HookFunc {
	return NewHookFunc(func(_ *Container, lc Lifecycle) {
		lc.OnStop(fn)
	}, applyLifecycleHookOptions(HookMetadata{
		Label: "OnStop0",
		Kind:  HookKindStop,
	}, opts...))
}

// OnStopFunc registers a stop hook with no resolved dependencies and no context usage.
func OnStopFunc(fn func() error, opts ...LifecycleHookOption) HookFunc {
	return OnStop0(func(context.Context) error {
		return fn()
	}, opts...)
}

// OnStart registers a start hook with one resolved dependency.
func OnStart[T any](fn func(context.Context, T) error, opts ...LifecycleHookOption) HookFunc {
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
	}, applyLifecycleHookOptions(HookMetadata{
		Label:        "OnStart",
		Kind:         HookKindStart,
		Dependencies: ServiceRefs(TypedService[T]()),
	}, opts...))
}

// OnStop registers a stop hook with one resolved dependency.
func OnStop[T any](fn func(context.Context, T) error, opts ...LifecycleHookOption) HookFunc {
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
	}, applyLifecycleHookOptions(HookMetadata{
		Label:        "OnStop",
		Kind:         HookKindStop,
		Dependencies: ServiceRefs(TypedService[T]()),
	}, opts...))
}

// OnStart2 registers a start hook with two resolved dependencies.
func OnStart2[T1, T2 any](fn func(context.Context, T1, T2) error, opts ...LifecycleHookOption) HookFunc {
	return NewHookFunc(func(c *Container, lc Lifecycle) {
		lc.OnStart(func(ctx context.Context) error {
			t1, t2, err := resolveDependencies2[T1, T2](c.Raw())
			if err != nil {
				return err
			}
			return fn(ctx, t1, t2)
		})
	}, applyLifecycleHookOptions(HookMetadata{
		Label:        "OnStart2",
		Kind:         HookKindStart,
		Dependencies: ServiceRefs(TypedService[T1](), TypedService[T2]()),
	}, opts...))
}

// OnStop2 registers a stop hook with two resolved dependencies.
func OnStop2[T1, T2 any](fn func(context.Context, T1, T2) error, opts ...LifecycleHookOption) HookFunc {
	return NewHookFunc(func(c *Container, lc Lifecycle) {
		lc.OnStop(func(ctx context.Context) error {
			t1, t2, err := resolveDependencies2[T1, T2](c.Raw())
			if err != nil {
				return err
			}
			return fn(ctx, t1, t2)
		})
	}, applyLifecycleHookOptions(HookMetadata{
		Label:        "OnStop2",
		Kind:         HookKindStop,
		Dependencies: ServiceRefs(TypedService[T1](), TypedService[T2]()),
	}, opts...))
}

// OnStart3 registers a start hook with three resolved dependencies.
func OnStart3[T1, T2, T3 any](fn func(context.Context, T1, T2, T3) error, opts ...LifecycleHookOption) HookFunc {
	return NewHookFunc(func(c *Container, lc Lifecycle) {
		lc.OnStart(func(ctx context.Context) error {
			t1, t2, t3, err := resolveDependencies3[T1, T2, T3](c.Raw())
			if err != nil {
				return err
			}
			return fn(ctx, t1, t2, t3)
		})
	}, applyLifecycleHookOptions(HookMetadata{
		Label:        "OnStart3",
		Kind:         HookKindStart,
		Dependencies: ServiceRefs(TypedService[T1](), TypedService[T2](), TypedService[T3]()),
	}, opts...))
}

// OnStop3 registers a stop hook with three resolved dependencies.
func OnStop3[T1, T2, T3 any](fn func(context.Context, T1, T2, T3) error, opts ...LifecycleHookOption) HookFunc {
	return NewHookFunc(func(c *Container, lc Lifecycle) {
		lc.OnStop(func(ctx context.Context) error {
			t1, t2, t3, err := resolveDependencies3[T1, T2, T3](c.Raw())
			if err != nil {
				return err
			}
			return fn(ctx, t1, t2, t3)
		})
	}, applyLifecycleHookOptions(HookMetadata{
		Label:        "OnStop3",
		Kind:         HookKindStop,
		Dependencies: ServiceRefs(TypedService[T1](), TypedService[T2](), TypedService[T3]()),
	}, opts...))
}
