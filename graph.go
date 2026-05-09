//revive:disable:file-length-limit Dependency graph builder is kept together around one cohesive API surface.

package dix

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	collectiongraph "github.com/arcgolabs/collectionx/graph"
	collectionlist "github.com/arcgolabs/collectionx/list"
	collectionmapping "github.com/arcgolabs/collectionx/mapping"
	collectionset "github.com/arcgolabs/collectionx/set"
	"github.com/samber/oops"
)

// DependencyGraphNodeKind identifies the kind of one dependency graph node.
type DependencyGraphNodeKind string

const (
	// DependencyGraphNodeApp identifies an app or subapp scope.
	DependencyGraphNodeApp DependencyGraphNodeKind = "app"
	// DependencyGraphNodeModule identifies a dix module.
	DependencyGraphNodeModule DependencyGraphNodeKind = "module"
	// DependencyGraphNodeService identifies a service known to dix validation metadata.
	DependencyGraphNodeService DependencyGraphNodeKind = "service"
	// DependencyGraphNodeOperation identifies a provider, setup, invoke, or lifecycle hook.
	DependencyGraphNodeOperation DependencyGraphNodeKind = "operation"
)

// DependencyGraphEdgeKind identifies the relationship between two dependency graph nodes.
type DependencyGraphEdgeKind string

const (
	// DependencyGraphEdgeSubApp links a parent app scope to a child app scope.
	DependencyGraphEdgeSubApp DependencyGraphEdgeKind = "subapp"
	// DependencyGraphEdgeContains links an app to modules, or modules to operations.
	DependencyGraphEdgeContains DependencyGraphEdgeKind = "contains"
	// DependencyGraphEdgeImports links an imported module to the module that imports it.
	DependencyGraphEdgeImports DependencyGraphEdgeKind = "imports"
	// DependencyGraphEdgeConsumes links a service to an operation that resolves it.
	DependencyGraphEdgeConsumes DependencyGraphEdgeKind = "consumes"
	// DependencyGraphEdgeProvides links an operation to a service it registers.
	DependencyGraphEdgeProvides DependencyGraphEdgeKind = "provides"
	// DependencyGraphEdgeAliases links an operation to an alias service it registers.
	DependencyGraphEdgeAliases DependencyGraphEdgeKind = "aliases"
	// DependencyGraphEdgeContributes links an operation to a collection contribution service.
	DependencyGraphEdgeContributes DependencyGraphEdgeKind = "contributes"
	// DependencyGraphEdgeOverrides links a setup operation to a service it overrides.
	DependencyGraphEdgeOverrides DependencyGraphEdgeKind = "overrides"
)

// DependencyGraphNode describes one app, module, service, or operation node.
type DependencyGraphNode struct {
	ID        string
	Kind      DependencyGraphNodeKind
	Label     string
	App       string
	Module    string
	Service   string
	Operation string
	Eager     bool
	Raw       bool
}

// DependencyGraphEdge describes one directed dependency graph edge.
type DependencyGraphEdge struct {
	From   string
	To     string
	Kind   DependencyGraphEdgeKind
	Label  string
	App    string
	Module string
}

// DependencyGraph is a structured view of an app build plan.
type DependencyGraph struct {
	Nodes *collectionlist.List[DependencyGraphNode]
	Edges *collectionlist.List[DependencyGraphEdge]

	graph *collectiongraph.Graph[string, DependencyGraphNode]
}

// Explanation summarizes the static dix build plan and validation result.
type Explanation struct {
	App        AppMeta
	Profile    Profile
	Graph      DependencyGraph
	Validation ValidationReport
}

// DependencyGraph returns the static dependency graph for the app.
func (a *App) DependencyGraph() (DependencyGraph, error) {
	return a.DependencyGraphContext(context.Background())
}

// DependencyGraphContext returns the static dependency graph for the app using ctx during profile resolution.
func (a *App) DependencyGraphContext(ctx context.Context) (DependencyGraph, error) {
	plan, _, err := a.cachedBuildPlan(ctx)
	if plan == nil {
		return DependencyGraph{}, err
	}
	dependencyGraph, graphErr := newDependencyGraph(plan)
	return dependencyGraph, errors.Join(err, graphErr)
}

// DependencyGraph returns the static dependency graph for a built runtime.
func (r *Runtime) DependencyGraph() (DependencyGraph, error) {
	if r == nil || r.plan == nil {
		err := oops.In("dix").
			With("op", "runtime_dependency_graph").
			New("runtime is nil")
		return DependencyGraph{}, err
	}
	return newDependencyGraph(r.plan)
}

// Explain returns a structured explanation of the app build plan.
func (a *App) Explain() (Explanation, error) {
	return a.ExplainContext(context.Background())
}

// ExplainContext returns a structured explanation of the app build plan using ctx during profile resolution.
func (a *App) ExplainContext(ctx context.Context) (Explanation, error) {
	plan, report, err := a.cachedBuildPlan(ctx)
	if plan == nil {
		return Explanation{Validation: report}, err
	}
	dependencyGraph, graphErr := newDependencyGraph(plan)
	out := Explanation{
		App:        plan.spec.meta,
		Profile:    plan.profile,
		Graph:      dependencyGraph,
		Validation: report,
	}
	return out, errors.Join(err, graphErr)
}

// Directed returns a shallow clone of the underlying collectionx directed graph.
func (g DependencyGraph) Directed() *collectiongraph.Graph[string, DependencyGraphNode] {
	if g.graph == nil {
		return collectiongraph.NewDirectedGraph[string, DependencyGraphNode]()
	}
	return g.graph.Clone()
}

// TopologicalOrder returns graph nodes in dependency-first order.
func (g DependencyGraph) TopologicalOrder() (*collectionlist.List[DependencyGraphNode], error) {
	if g.graph == nil {
		return collectionlist.NewList[DependencyGraphNode](), nil
	}
	ids, err := g.graph.TopologicalSort()
	if err != nil {
		return collectionlist.NewList[DependencyGraphNode](), oops.In("dix").
			With("op", "dependency_graph_topological_order").
			Wrapf(err, "dependency graph is not acyclic")
	}
	nodes := collectionlist.NewListWithCapacity[DependencyGraphNode](len(ids))
	for _, id := range ids {
		node, found := g.graph.GetNode(id)
		if found {
			nodes.Add(node)
		}
	}
	return nodes, nil
}

// DOT renders the graph in Graphviz DOT format.
func (g DependencyGraph) DOT() string {
	var builder strings.Builder
	writeGraphString(&builder, "digraph dix {\n")
	if g.Nodes != nil {
		g.Nodes.Range(func(_ int, node DependencyGraphNode) bool {
			writeGraphString(&builder, "  ")
			writeGraphString(&builder, strconv.Quote(node.ID))
			writeGraphString(&builder, " [label=")
			writeGraphString(&builder, strconv.Quote(node.Label))
			writeGraphString(&builder, ", shape=")
			writeGraphString(&builder, strconv.Quote(dotShape(node.Kind)))
			writeGraphString(&builder, "];\n")
			return true
		})
	}
	if g.Edges != nil {
		g.Edges.Range(func(_ int, edge DependencyGraphEdge) bool {
			writeGraphString(&builder, "  ")
			writeGraphString(&builder, strconv.Quote(edge.From))
			writeGraphString(&builder, " -> ")
			writeGraphString(&builder, strconv.Quote(edge.To))
			writeGraphString(&builder, " [label=")
			writeGraphString(&builder, strconv.Quote(string(edge.Kind)))
			writeGraphString(&builder, "];\n")
			return true
		})
	}
	writeGraphString(&builder, "}\n")
	return builder.String()
}

// String returns a compact text explanation.
func (e Explanation) String() string {
	lines := collectionlist.NewList[string]()
	lines.Add("app: " + e.App.Name)
	lines.Add("profile: " + string(e.Profile))
	if e.Graph.Nodes != nil || e.Graph.Edges != nil {
		nodeCount := 0
		edgeCount := 0
		if e.Graph.Nodes != nil {
			nodeCount = e.Graph.Nodes.Len()
		}
		if e.Graph.Edges != nil {
			edgeCount = e.Graph.Edges.Len()
		}
		lines.Add(fmt.Sprintf("graph: %d nodes, %d edges", nodeCount, edgeCount))
	}
	if e.Validation.Warnings != nil && e.Validation.Warnings.Len() > 0 {
		lines.Add(fmt.Sprintf("warnings: %d", e.Validation.Warnings.Len()))
	}
	if e.Validation.Errors != nil && e.Validation.Errors.Len() > 0 {
		lines.Add(fmt.Sprintf("errors: %d", e.Validation.Errors.Len()))
	}
	return lines.Join("\n")
}

type dependencyGraphBuilder struct {
	graph       *collectiongraph.Graph[string, DependencyGraphNode]
	edges       *collectionlist.List[DependencyGraphEdge]
	edgeSet     *collectionset.Set[string]
	parentPaths *collectionmapping.Map[string, string]
	moduleIDs   *collectionmapping.Map[string, string]
	services    *collectionmapping.Map[string, string]
}

func newDependencyGraph(plan *buildPlan) (DependencyGraph, error) {
	builder := newDependencyGraphBuilder()
	builder.collectPlan(plan)
	builder.connectPlan(plan)
	result := builder.snapshot()
	_, err := result.TopologicalOrder()
	return result, err
}

func newDependencyGraphBuilder() *dependencyGraphBuilder {
	return &dependencyGraphBuilder{
		graph:       collectiongraph.NewDirectedGraph[string, DependencyGraphNode](),
		edges:       collectionlist.NewList[DependencyGraphEdge](),
		edgeSet:     collectionset.NewSetWithCapacity[string](64),
		parentPaths: collectionmapping.NewMap[string, string](),
		moduleIDs:   collectionmapping.NewMap[string, string](),
		services:    collectionmapping.NewMap[string, string](),
	}
}

func (b *dependencyGraphBuilder) collectPlan(plan *buildPlan) {
	if plan == nil || plan.spec == nil {
		return
	}
	path := buildPlanPath(plan)
	parentPath := ""
	if plan.parent != nil {
		parentPath = buildPlanPath(plan.parent)
	}
	b.parentPaths.Set(path, parentPath)
	b.addAppNode(plan, path)
	b.collectModules(plan, path)
	b.collectCoreServices(plan, path)
	b.collectSyntheticContributionServices(plan, path)
	plan.subplans.Range(func(_ int, subplan *buildPlan) bool {
		b.collectPlan(subplan)
		return true
	})
}

func (b *dependencyGraphBuilder) connectPlan(plan *buildPlan) {
	if plan == nil || plan.spec == nil {
		return
	}
	path := buildPlanPath(plan)
	appID := appNodeID(path)
	plan.modules.Range(func(_ int, mod *moduleSpec) bool {
		moduleID := b.moduleID(path, mod)
		b.addEdge(appID, moduleID, DependencyGraphEdgeContains, "module", path, moduleKey(mod))
		b.connectModuleImports(path, mod)
		b.connectProviders(path, mod)
		b.connectSetups(path, mod)
		b.connectInvokes(path, mod)
		b.connectHooks(path, mod)
		return true
	})
	b.connectContributionCollections(path, plan)
	plan.subplans.Range(func(_ int, subplan *buildPlan) bool {
		subPath := buildPlanPath(subplan)
		b.addEdge(appID, appNodeID(subPath), DependencyGraphEdgeSubApp, "subapp", path, "")
		b.connectPlan(subplan)
		return true
	})
}

func (b *dependencyGraphBuilder) snapshot() DependencyGraph {
	nodes := collectionlist.NewListWithCapacity[DependencyGraphNode](b.graph.Len())
	b.graph.RangeNodes(func(_ string, node DependencyGraphNode) bool {
		nodes.Add(node)
		return true
	})
	return DependencyGraph{
		Nodes: nodes,
		Edges: b.edges.Clone(),
		graph: b.graph.Clone(),
	}
}

func (b *dependencyGraphBuilder) addAppNode(plan *buildPlan, path string) {
	b.addNode(DependencyGraphNode{
		ID:    appNodeID(path),
		Kind:  DependencyGraphNodeApp,
		Label: plan.spec.meta.Name,
		App:   path,
	})
}

func (b *dependencyGraphBuilder) collectModules(plan *buildPlan, path string) {
	plan.modules.Range(func(_ int, mod *moduleSpec) bool {
		if mod == nil {
			return true
		}
		moduleID := b.moduleID(path, mod)
		b.addNode(DependencyGraphNode{
			ID:     moduleID,
			Kind:   DependencyGraphNodeModule,
			Label:  moduleKey(mod),
			App:    path,
			Module: moduleKey(mod),
		})
		b.collectProviderServices(path, mod)
		b.collectSetupServices(path, mod)
		return true
	})
}

func (b *dependencyGraphBuilder) collectCoreServices(plan *buildPlan, path string) {
	if !plan.declaresProviderOutput(TypedService[*slog.Logger]()) {
		b.ensureService(path, TypedService[*slog.Logger]().Name, "dix core")
	}
	if !plan.declaresProviderOutput(TypedService[AppMeta]()) {
		b.ensureService(path, TypedService[AppMeta]().Name, "dix core")
	}
	if !plan.declaresProviderOutput(TypedService[Profile]()) {
		b.ensureService(path, TypedService[Profile]().Name, "dix core")
	}
}

func (b *dependencyGraphBuilder) collectSyntheticContributionServices(plan *buildPlan, path string) {
	newContributionPlan(plan.modules).syntheticOutputs().Range(func(_ int, output ServiceRef) bool {
		b.ensureService(path, output.Name, "dix contributions")
		return true
	})
}

func (b *dependencyGraphBuilder) collectProviderServices(path string, mod *moduleSpec) {
	mod.providers.Range(func(_ int, provider ProviderFunc) bool {
		meta := provider.meta
		if meta.Output.Name != "" {
			b.ensureService(path, meta.Output.Name, moduleKey(mod))
		}
		meta.Aliases.Range(func(_ int, alias ServiceRef) bool {
			b.ensureService(path, alias.Name, moduleKey(mod))
			return true
		})
		meta.Contributions.Range(func(_ int, contribution ContributionRef) bool {
			b.ensureService(path, contribution.Service.Name, moduleKey(mod))
			return true
		})
		return true
	})
}

func (b *dependencyGraphBuilder) collectSetupServices(path string, mod *moduleSpec) {
	mod.setups.Range(func(_ int, setup SetupFunc) bool {
		setup.meta.Provides.Range(func(_ int, provide ServiceRef) bool {
			b.ensureService(path, provide.Name, moduleKey(mod))
			return true
		})
		return true
	})
}

func (b *dependencyGraphBuilder) connectModuleImports(path string, mod *moduleSpec) {
	moduleID := b.moduleID(path, mod)
	mod.imports.Range(func(_ int, imported Module) bool {
		importID, found := b.moduleIDs.Get(moduleIndexKey(path, imported.spec))
		if found {
			b.addEdge(importID, moduleID, DependencyGraphEdgeImports, "import", path, moduleKey(mod))
		}
		return true
	})
}

func (b *dependencyGraphBuilder) connectProviders(path string, mod *moduleSpec) {
	mod.providers.Range(func(index int, provider ProviderFunc) bool {
		meta := provider.meta
		opID := b.addOperation(path, mod, "provider", meta.Label, index, meta.Raw, meta.Eager)
		b.connectDependencies(path, mod, opID, meta.Dependencies)
		if meta.Output.Name != "" {
			b.addEdge(opID, b.resolveService(path, meta.Output.Name), DependencyGraphEdgeProvides, meta.Label, path, moduleKey(mod))
		}
		meta.Aliases.Range(func(_ int, alias ServiceRef) bool {
			b.addEdge(opID, b.resolveService(path, alias.Name), DependencyGraphEdgeAliases, meta.Label, path, moduleKey(mod))
			return true
		})
		meta.Contributions.Range(func(_ int, contribution ContributionRef) bool {
			b.addEdge(opID, b.resolveService(path, contribution.Service.Name), DependencyGraphEdgeContributes, meta.Label, path, moduleKey(mod))
			if contribution.Target.Name != "" {
				b.addEdge(
					b.resolveService(path, contribution.Service.Name),
					b.resolveService(path, contribution.Target.Name),
					DependencyGraphEdgeContributes,
					meta.Label,
					path,
					moduleKey(mod),
				)
			}
			return true
		})
		return true
	})
}

func (b *dependencyGraphBuilder) connectSetups(path string, mod *moduleSpec) {
	mod.setups.Range(func(index int, setup SetupFunc) bool {
		meta := setup.meta
		opID := b.addOperation(path, mod, "setup", meta.Label, index, meta.Raw, false)
		b.connectDependencies(path, mod, opID, meta.Dependencies)
		meta.Provides.Range(func(_ int, provide ServiceRef) bool {
			b.addEdge(opID, b.resolveService(path, provide.Name), DependencyGraphEdgeProvides, meta.Label, path, moduleKey(mod))
			return true
		})
		meta.Overrides.Range(func(_ int, override ServiceRef) bool {
			b.addEdge(opID, b.resolveService(path, override.Name), DependencyGraphEdgeOverrides, meta.Label, path, moduleKey(mod))
			return true
		})
		return true
	})
}

func (b *dependencyGraphBuilder) connectInvokes(path string, mod *moduleSpec) {
	mod.invokes.Range(func(index int, invoke InvokeFunc) bool {
		meta := invoke.meta
		opID := b.addOperation(path, mod, "invoke", meta.Label, index, meta.Raw, false)
		b.connectDependencies(path, mod, opID, meta.Dependencies)
		return true
	})
}

func (b *dependencyGraphBuilder) connectHooks(path string, mod *moduleSpec) {
	mod.hooks.Range(func(index int, hook HookFunc) bool {
		meta := hook.meta
		opID := b.addOperation(path, mod, string(meta.Kind)+" hook", meta.Label, index, meta.Raw, false)
		b.connectDependencies(path, mod, opID, meta.Dependencies)
		return true
	})
}

func (b *dependencyGraphBuilder) connectContributionCollections(path string, plan *buildPlan) {
	contributionPlan := newContributionPlan(plan.modules)
	appID := appNodeID(path)
	contributionPlan.targets.Range(func(index int, target string) bool {
		factory, found := contributionPlan.factories.Get(target)
		if !found {
			return true
		}
		opID := operationNodeID(path, "dix contributions", "collection provider", target, index)
		b.addNode(DependencyGraphNode{
			ID:        opID,
			Kind:      DependencyGraphNodeOperation,
			Label:     "collection provider " + target,
			App:       path,
			Operation: "collection provider",
		})
		b.addEdge(appID, opID, DependencyGraphEdgeContains, "collection provider", path, "")
		for _, contribution := range contributionPlan.contributions.Get(target) {
			b.addEdge(
				b.resolveService(path, contribution.Service.Name),
				opID,
				DependencyGraphEdgeContributes,
				"collection contribution",
				path,
				"",
			)
		}
		factory.outputs.Range(func(_ int, output ServiceRef) bool {
			if contributionPlan.explicit.Contains(output.Name) {
				return true
			}
			b.addEdge(opID, b.resolveService(path, output.Name), DependencyGraphEdgeProvides, "collection provider", path, "")
			return true
		})
		return true
	})
}

func (b *dependencyGraphBuilder) connectDependencies(
	path string,
	mod *moduleSpec,
	operationID string,
	deps *collectionlist.List[ServiceRef],
) {
	deps.Range(func(_ int, dep ServiceRef) bool {
		b.addEdge(b.resolveService(path, dep.Name), operationID, DependencyGraphEdgeConsumes, "dependency", path, moduleKey(mod))
		return true
	})
}

func (b *dependencyGraphBuilder) addOperation(path string, mod *moduleSpec, kind, label string, index int, raw, eager bool) string {
	module := moduleKey(mod)
	id := operationNodeID(path, module, kind, label, index)
	b.addNode(DependencyGraphNode{
		ID:        id,
		Kind:      DependencyGraphNodeOperation,
		Label:     kind + " " + label,
		App:       path,
		Module:    module,
		Operation: kind,
		Eager:     eager,
		Raw:       raw,
	})
	b.addEdge(b.moduleID(path, mod), id, DependencyGraphEdgeContains, kind, path, module)
	return id
}

func (b *dependencyGraphBuilder) addNode(node DependencyGraphNode) {
	if node.ID == "" {
		return
	}
	if node.Label == "" {
		node.Label = node.ID
	}
	b.graph.AddNode(node.ID, node)
}

func (b *dependencyGraphBuilder) addEdge(from, to string, kind DependencyGraphEdgeKind, label, app, module string) {
	if from == "" || to == "" {
		return
	}
	key := from + "\x00" + to + "\x00" + string(kind) + "\x00" + label
	if b.edgeSet.Contains(key) {
		return
	}
	b.edgeSet.Add(key)
	if !b.graph.HasNode(from) {
		b.addNode(DependencyGraphNode{ID: from, Kind: DependencyGraphNodeService, Label: from, App: app, Module: module})
	}
	if !b.graph.HasNode(to) {
		b.addNode(DependencyGraphNode{ID: to, Kind: DependencyGraphNodeService, Label: to, App: app, Module: module})
	}
	if err := b.graph.AddEdge(from, to); err != nil {
		return
	}
	b.edges.Add(DependencyGraphEdge{
		From:   from,
		To:     to,
		Kind:   kind,
		Label:  label,
		App:    app,
		Module: module,
	})
}

func (b *dependencyGraphBuilder) moduleID(path string, mod *moduleSpec) string {
	if mod == nil {
		return moduleNodeID(path, "<nil>")
	}
	key := moduleIndexKey(path, mod)
	if id, found := b.moduleIDs.Get(key); found {
		return id
	}
	id := moduleNodeID(path, moduleKey(mod))
	b.moduleIDs.Set(key, id)
	return id
}

func (b *dependencyGraphBuilder) ensureService(path, name, module string) string {
	if name == "" {
		return ""
	}
	id := serviceNodeID(path, name)
	key := serviceIndexKey(path, name)
	b.services.Set(key, id)
	b.addNode(DependencyGraphNode{
		ID:      id,
		Kind:    DependencyGraphNodeService,
		Label:   name,
		App:     path,
		Module:  module,
		Service: name,
	})
	return id
}

func (b *dependencyGraphBuilder) resolveService(path, name string) string {
	if name == "" {
		return ""
	}
	current := path
	for {
		if id, found := b.services.Get(serviceIndexKey(current, name)); found {
			return id
		}
		parent, found := b.parentPaths.Get(current)
		if !found || parent == "" {
			break
		}
		current = parent
	}
	return b.ensureService(path, name, "unresolved")
}

func buildPlanPath(plan *buildPlan) string {
	if plan == nil || plan.spec == nil {
		return "<nil>"
	}
	names := []string{plan.spec.meta.Name}
	for parent := plan.parent; parent != nil && parent.spec != nil; parent = parent.parent {
		names = append(names, parent.spec.meta.Name)
	}
	for i, j := 0, len(names)-1; i < j; i, j = i+1, j-1 {
		names[i], names[j] = names[j], names[i]
	}
	return strings.Join(names, "/")
}

func appNodeID(path string) string {
	return "app:" + path
}

func moduleNodeID(path, name string) string {
	return "module:" + path + ":" + name
}

func serviceNodeID(path, name string) string {
	return "service:" + path + ":" + name
}

func operationNodeID(path, module, kind, label string, index int) string {
	return "operation:" + path + ":" + module + ":" + kind + ":" + label + ":" + strconv.Itoa(index)
}

func serviceIndexKey(path, name string) string {
	return path + "\x00" + name
}

func moduleIndexKey(path string, mod *moduleSpec) string {
	return path + "\x00" + fmt.Sprintf("%p", mod)
}

func dotShape(kind DependencyGraphNodeKind) string {
	switch kind {
	case DependencyGraphNodeApp:
		return "folder"
	case DependencyGraphNodeModule:
		return "box"
	case DependencyGraphNodeOperation:
		return "ellipse"
	case DependencyGraphNodeService:
		return "component"
	default:
		return "plain"
	}
}

func writeGraphString(builder *strings.Builder, value string) {
	if _, err := builder.WriteString(value); err != nil {
		panic(err)
	}
}
