package dix

import (
	collectionlist "github.com/arcgolabs/collectionx/list"
	"time"
)

// LifecycleSummary reports lifecycle hooks registered on a built runtime.
type LifecycleSummary struct {
	StartHooks  int
	StopHooks   int
	Start       *collectionlist.List[LifecycleHookSummary]
	Stop        *collectionlist.List[LifecycleHookSummary]
	Concurrency int
}

// LifecycleHookSummary reports one registered lifecycle hook.
type LifecycleHookSummary struct {
	Name     string
	Label    string
	Kind     HookKind
	After    *collectionlist.List[string]
	Before   *collectionlist.List[string]
	Priority int
	Parallel bool
	Timeout  time.Duration
	Sequence int
}

// SubAppSummary reports one built child runtime.
type SubAppSummary struct {
	Name       string
	ParentName string
	Profile    Profile
	State      AppState
	SubApps    int
}

// LifecycleSummary returns the lifecycle hook counts currently bound to the runtime.
func (r *Runtime) LifecycleSummary() LifecycleSummary {
	if r == nil || r.lifecycle == nil {
		return LifecycleSummary{}
	}
	return LifecycleSummary{
		StartHooks:  r.lifecycle.startHooks.Len(),
		StopHooks:   r.lifecycle.stopHooks.Len(),
		Start:       lifecycleHookSummaries(r.lifecycle.startOrderForSummary(r.lifecycle.startHooks)),
		Stop:        lifecycleHookSummaries(r.lifecycle.stopOrderForSummary(r.lifecycle.stopHooks)),
		Concurrency: r.lifecycle.resolvedConcurrency(),
	}
}

func lifecycleHookSummaries(entries []lifecycleHookEntry) *collectionlist.List[LifecycleHookSummary] {
	summaries := collectionlist.NewListWithCapacity[LifecycleHookSummary](len(entries))
	for _, entry := range entries {
		summaries.Add(LifecycleHookSummary{
			Name:     hookName(entry.meta),
			Label:    entry.meta.Label,
			Kind:     entry.meta.Kind,
			After:    entry.meta.After.Clone(),
			Before:   entry.meta.Before.Clone(),
			Priority: entry.meta.Priority,
			Parallel: entry.meta.Parallel,
			Timeout:  entry.meta.Timeout,
			Sequence: entry.sequence,
		})
	}
	return summaries
}

// IsSubApp reports whether this runtime was built below a parent app.
func (r *Runtime) IsSubApp() bool {
	return r != nil && r.plan != nil && r.plan.parent != nil
}

// ParentName returns the parent app name when this runtime is a subapp.
func (r *Runtime) ParentName() (string, bool) {
	if !r.IsSubApp() || r.plan.parent.spec == nil {
		return "", false
	}
	return r.plan.parent.spec.meta.Name, true
}

// ScopePath returns the app scope names from root app to this runtime.
func (r *Runtime) ScopePath() *collectionlist.List[string] {
	if r == nil || r.plan == nil {
		return collectionlist.NewList[string]()
	}
	names := collectionlist.NewList[string]()
	for plan := r.plan; plan != nil && plan.spec != nil; plan = plan.parent {
		names.Add(plan.spec.meta.Name)
	}
	return names.Reverse()
}

// SubAppSummaries returns built child runtime summaries in declaration order.
func (r *Runtime) SubAppSummaries() *collectionlist.List[SubAppSummary] {
	if r == nil || r.subapps == nil {
		return collectionlist.NewList[SubAppSummary]()
	}
	summaries := collectionlist.NewListWithCapacity[SubAppSummary](r.subapps.Len())
	r.subapps.Range(func(_ int, subapp *Runtime) bool {
		if subapp == nil {
			return true
		}
		summary := SubAppSummary{
			Name:       subapp.Name(),
			ParentName: r.Name(),
			Profile:    subapp.Profile(),
			State:      subapp.State(),
			SubApps:    subapp.SubApps().Len(),
		}
		summaries.Add(summary)
		return true
	})
	return summaries
}
