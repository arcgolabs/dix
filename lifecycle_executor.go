package dix

import (
	"context"
	"errors"
	goruntime "runtime"
	"sync"
	"time"

	collectionlist "github.com/arcgolabs/collectionx/list"
	"github.com/panjf2000/ants/v2"
	"github.com/samber/oops"
)

func (l *lifecycleImpl) executeStartHooks(ctx context.Context, _ *Container) (int, error) {
	debugEnabled := l.debugEnabled(ctx)
	l.logDebug(ctx, debugEnabled, "executing start hooks", "count", l.startHooks.Len())

	entries, err := l.startOrder(l.startHooks)
	if err != nil {
		return 0, err
	}
	return l.executeHookEntries(ctx, HookKindStart, entries, true)
}

func (l *lifecycleImpl) executeStopHooks(ctx context.Context, _ *Container) error {
	entries, err := l.stopOrder(l.stopHooks)
	if err != nil {
		return err
	}
	_, err = l.executeHookEntries(ctx, HookKindStop, entries, false)
	return err
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

	entries, err := l.startOrder(l.stopHooks)
	if err != nil {
		return err
	}
	entries = entries[:count]
	_, err = l.executeHookEntries(ctx, HookKindStop, reverseLifecycleEntries(entries), false)
	return err
}

func (l *lifecycleImpl) executeHookEntries(
	ctx context.Context,
	kind HookKind,
	entries []lifecycleHookEntry,
	failFast bool,
) (int, error) {
	completed := 0
	errs := collectionlist.NewList[error]()
	for index := 0; index < len(entries); {
		next, batchCompleted, err := l.executeNextHookBatch(ctx, kind, entries, index, failFast)
		completed += batchCompleted
		if err != nil {
			errs.Add(err)
			if failFast {
				return completed, errors.Join(errs.Values()...)
			}
		}
		index = next
	}
	return completed, errors.Join(errs.Values()...)
}

func (l *lifecycleImpl) executeNextHookBatch(
	ctx context.Context,
	kind HookKind,
	entries []lifecycleHookEntry,
	index int,
	failFast bool,
) (int, int, error) {
	entry := entries[index]
	if !entry.meta.Parallel {
		if err := l.executeHookEntry(ctx, kind, entry); err != nil {
			return index + 1, 0, err
		}
		return index + 1, 1, nil
	}

	end := parallelGroupEnd(entries, index)
	completed, err := l.executeParallelHookGroup(ctx, kind, entries[index:end], failFast)
	return end, completed, err
}

func parallelGroupEnd(entries []lifecycleHookEntry, index int) int {
	if lifecycleHookHasOrdering(entries[index].meta) {
		return index + 1
	}
	priority := entries[index].meta.Priority
	end := index + 1
	for end < len(entries) &&
		entries[end].meta.Parallel &&
		entries[end].meta.Priority == priority &&
		!lifecycleHookHasOrdering(entries[end].meta) {
		end++
	}
	return end
}

func (l *lifecycleImpl) executeParallelHookGroup(
	ctx context.Context,
	kind HookKind,
	entries []lifecycleHookEntry,
	failFast bool,
) (int, error) {
	switch len(entries) {
	case 0:
		return 0, nil
	case 1:
		return l.executeSingleHook(ctx, kind, entries[0])
	}

	pool, err := l.newLifecyclePool(kind)
	if err != nil {
		return 0, err
	}
	defer pool.Release()

	groupCtx, cancel := lifecycleGroupContext(ctx, failFast)
	defer cancel()

	result := newLifecycleParallelResult(failFast, cancel)
	var wg sync.WaitGroup
	for _, entry := range entries {
		wg.Add(1)
		if err := pool.Submit(func() {
			defer wg.Done()
			result.record(l.executeHookEntry(groupCtx, kind, entry))
		}); err != nil {
			wg.Done()
			result.record(submitLifecycleHookError(kind, entry, err))
			if failFast {
				break
			}
		}
	}
	wg.Wait()
	return result.completedCount(), result.err()
}

func (l *lifecycleImpl) executeSingleHook(ctx context.Context, kind HookKind, entry lifecycleHookEntry) (int, error) {
	if err := l.executeHookEntry(ctx, kind, entry); err != nil {
		return 0, err
	}
	return 1, nil
}

func (l *lifecycleImpl) newLifecyclePool(kind HookKind) (*ants.Pool, error) {
	pool, err := ants.NewPool(l.resolvedConcurrency())
	if err == nil {
		return pool, nil
	}
	return nil, oops.In("dix").
		With("op", "lifecycle_parallel_pool", "kind", string(kind)).
		Wrapf(err, "create lifecycle worker pool failed")
}

func lifecycleGroupContext(ctx context.Context, cancellable bool) (context.Context, context.CancelFunc) {
	if cancellable {
		return context.WithCancel(ctx)
	}
	return ctx, func() {}
}

type lifecycleParallelResult struct {
	mu       sync.Mutex
	errs     *collectionlist.List[error]
	done     int
	failFast bool
	cancel   context.CancelFunc
}

func newLifecycleParallelResult(failFast bool, cancel context.CancelFunc) *lifecycleParallelResult {
	return &lifecycleParallelResult{
		errs:     collectionlist.NewList[error](),
		failFast: failFast,
		cancel:   cancel,
	}
}

func (r *lifecycleParallelResult) record(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err != nil {
		r.errs.Add(err)
		if r.failFast {
			r.cancel()
		}
		return
	}
	r.done++
}

func (r *lifecycleParallelResult) completedCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.done
}

func (r *lifecycleParallelResult) err() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return errors.Join(r.errs.Values()...)
}

func submitLifecycleHookError(kind HookKind, entry lifecycleHookEntry, err error) error {
	return oops.In("dix").
		With("op", "lifecycle_parallel_submit", "kind", string(kind), "hook", hookName(entry.meta)).
		Wrapf(err, "submit lifecycle hook failed")
}

func (l *lifecycleImpl) executeHookEntry(ctx context.Context, kind HookKind, entry lifecycleHookEntry) error {
	debugEnabled := l.debugEnabled(ctx)
	name := hookName(entry.meta)
	hookCtx, cancel := lifecycleHookContext(ctx, entry.meta.Timeout)
	defer cancel()

	startedAt := time.Now()
	l.logDebug(ctx, debugEnabled, "executing "+string(kind)+" hook",
		"kind", string(kind),
		"name", name,
		"label", entry.meta.Label,
		"priority", entry.meta.Priority,
		"parallel", entry.meta.Parallel,
		"timeout", entry.meta.Timeout,
		"sequence", entry.sequence,
	)
	err := entry.run(hookCtx)
	duration := time.Since(startedAt)
	l.emitLifecycleHookResult(ctx, kind, entry, duration, err)
	if err != nil {
		return oops.In("dix").
			With("op", string(kind)+"_hook", "name", name, "priority", entry.meta.Priority, "sequence", entry.sequence).
			Wrapf(err, "%s hook %s failed", kind, name)
	}
	return nil
}

func (l *lifecycleImpl) emitLifecycleHookResult(
	ctx context.Context,
	kind HookKind,
	entry lifecycleHookEntry,
	duration time.Duration,
	err error,
) {
	event := LifecycleHookEvent{
		Kind:     kind,
		Name:     hookName(entry.meta),
		Label:    entry.meta.Label,
		Priority: entry.meta.Priority,
		Parallel: entry.meta.Parallel,
		Timeout:  entry.meta.Timeout,
		Sequence: entry.sequence,
		Duration: duration,
		Err:      err,
	}
	if l == nil {
		return
	}
	if l.emitHook != nil {
		l.emitHook(ctx, event)
		return
	}
	emitEventLogger(ctx, l.eventLogger, event)
}

func (l *lifecycleImpl) resolvedConcurrency() int {
	if l == nil || l.concurrency <= 0 {
		return max(1, goruntime.GOMAXPROCS(0))
	}
	return l.concurrency
}
