package advanced

import (
	collectionlist "github.com/arcgolabs/collectionx/list"
	collectionmapping "github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/dix"
	"github.com/samber/do/v2"
)

// Inspection summarizes advanced runtime inspection output.
type Inspection struct {
	ScopeTree         string
	ProvidedServices  *collectionlist.List[do.ServiceDescription]
	InvokedServices   *collectionlist.List[do.ServiceDescription]
	NamedDependencies *collectionmapping.Map[string, string]
	DependencyGraph   dix.DependencyGraph
	Lifecycle         dix.LifecycleSummary
	SubApps           *collectionlist.List[dix.SubAppSummary]
}

// InspectOptions controls which inspection sections are populated.
type InspectOptions struct {
	IncludeScopeTree        bool
	IncludeProvidedServices bool
	IncludeInvokedServices  bool
	IncludeNamedDeps        bool
	IncludeDependencyGraph  bool
	IncludeLifecycle        bool
	IncludeSubApps          bool
}

// DefaultInspectOptions returns the default inspection option set.
func DefaultInspectOptions() InspectOptions {
	return InspectOptions{
		IncludeScopeTree:        true,
		IncludeProvidedServices: true,
		IncludeInvokedServices:  true,
		IncludeNamedDeps:        true,
		IncludeDependencyGraph:  true,
		IncludeLifecycle:        true,
		IncludeSubApps:          true,
	}
}

// ExplainScopeTree returns the textual do scope tree for a runtime.
func ExplainScopeTree(rt *dix.Runtime) string {
	if rt == nil {
		return ""
	}

	explainedScope := do.ExplainInjector(rt.Raw())
	return explainedScope.String()
}

// ListProvidedServices returns the services provided by the runtime injector.
func ListProvidedServices(rt *dix.Runtime) *collectionlist.List[do.ServiceDescription] {
	if rt == nil {
		return nil
	}

	items := rt.Raw().ListProvidedServices()
	return collectionlist.NewListWithCapacity(len(items), items...)
}

// ListInvokedServices returns the services invoked by the runtime injector.
func ListInvokedServices(rt *dix.Runtime) *collectionlist.List[do.ServiceDescription] {
	if rt == nil {
		return nil
	}

	items := rt.Raw().ListInvokedServices()
	return collectionlist.NewListWithCapacity(len(items), items...)
}

// ExplainNamedDependencies returns dependency trees for the requested named services.
func ExplainNamedDependencies(rt *dix.Runtime, namedServices ...string) *collectionmapping.Map[string, string] {
	if rt == nil || len(namedServices) == 0 {
		return nil
	}

	dependencies := collectionmapping.NewMapWithCapacity[string, string](len(namedServices))
	for _, name := range namedServices {
		if desc, found := do.ExplainNamedService(rt.Raw(), name); found {
			dependencies.Set(name, desc.String())
		}
	}

	return dependencies
}

// InspectRuntime inspects a runtime with the default options.
func InspectRuntime(rt *dix.Runtime, namedServices ...string) Inspection {
	return InspectRuntimeWithOptions(rt, DefaultInspectOptions(), namedServices...)
}

// InspectRuntimeWithOptions inspects a runtime with the provided options.
func InspectRuntimeWithOptions(rt *dix.Runtime, opts InspectOptions, namedServices ...string) Inspection {
	if rt == nil {
		return Inspection{}
	}

	return Inspection{
		ScopeTree:         inspectScopeTree(rt, opts.IncludeScopeTree),
		ProvidedServices:  inspectProvidedServices(rt, opts.IncludeProvidedServices),
		InvokedServices:   inspectInvokedServices(rt, opts.IncludeInvokedServices),
		NamedDependencies: inspectNamedDependencies(rt, opts.IncludeNamedDeps, namedServices...),
		DependencyGraph:   inspectDependencyGraph(rt, opts.IncludeDependencyGraph),
		Lifecycle:         inspectLifecycle(rt, opts.IncludeLifecycle),
		SubApps:           inspectSubApps(rt, opts.IncludeSubApps),
	}
}

func inspectScopeTree(rt *dix.Runtime, include bool) string {
	if !include {
		return ""
	}
	return ExplainScopeTree(rt)
}

func inspectProvidedServices(rt *dix.Runtime, include bool) *collectionlist.List[do.ServiceDescription] {
	if !include {
		return nil
	}
	return ListProvidedServices(rt)
}

func inspectInvokedServices(rt *dix.Runtime, include bool) *collectionlist.List[do.ServiceDescription] {
	if !include {
		return nil
	}
	return ListInvokedServices(rt)
}

func inspectNamedDependencies(
	rt *dix.Runtime,
	include bool,
	namedServices ...string,
) *collectionmapping.Map[string, string] {
	if !include || len(namedServices) == 0 {
		return nil
	}
	return ExplainNamedDependencies(rt, namedServices...)
}

func inspectDependencyGraph(rt *dix.Runtime, include bool) dix.DependencyGraph {
	if !include {
		return dix.DependencyGraph{}
	}
	graph, err := rt.DependencyGraph()
	if err != nil {
		return dix.DependencyGraph{}
	}
	return graph
}

func inspectLifecycle(rt *dix.Runtime, include bool) dix.LifecycleSummary {
	if !include {
		return dix.LifecycleSummary{}
	}
	return rt.LifecycleSummary()
}

func inspectSubApps(rt *dix.Runtime, include bool) *collectionlist.List[dix.SubAppSummary] {
	if !include {
		return nil
	}
	return rt.SubAppSummaries()
}
