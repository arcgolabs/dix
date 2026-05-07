package dix

import collectionlist "github.com/arcgolabs/collectionx/list"

// LifecycleSummary reports lifecycle hooks registered on a built runtime.
type LifecycleSummary struct {
	StartHooks int
	StopHooks  int
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
		StartHooks: r.lifecycle.startHooks.Len(),
		StopHooks:  r.lifecycle.stopHooks.Len(),
	}
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
	values := names.Values()
	for i, j := 0, len(values)-1; i < j; i, j = i+1, j-1 {
		values[i], values[j] = values[j], values[i]
	}
	return collectionlist.NewListWithCapacity[string](len(values), values...)
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
