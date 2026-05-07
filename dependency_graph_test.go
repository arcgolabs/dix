package dix_test

import (
	"context"
	"strings"
	"testing"

	"github.com/arcgolabs/dix"
	"github.com/stretchr/testify/require"
)

type graphConfig struct {
	Name string
}

type graphRepo struct {
	Config graphConfig
}

func TestAppDependencyGraphIncludesSubAppsAndOperations(t *testing.T) {
	t.Parallel()

	root := dix.New(
		"root",
		dix.Modules(
			dix.NewModule("infra", dix.Providers(dix.Value(graphConfig{Name: "test"}))),
			dix.NewModule(
				"service",
				dix.Providers(dix.Provider1[graphRepo, graphConfig](func(config graphConfig) graphRepo {
					return graphRepo{Config: config}
				})),
				dix.Invokes(dix.Invoke1[graphRepo](func(graphRepo) {})),
			),
		),
		dix.SubApps(dix.NewSubApp(
			"worker",
			dix.Modules(dix.NewModule("worker", dix.Invokes(dix.Invoke1[graphConfig](func(graphConfig) {})))),
		)),
	)

	explanation, err := root.Explain()
	require.NoError(t, err)
	require.Equal(t, "root", explanation.App.Name)
	require.NotZero(t, explanation.Graph.Nodes.Len())
	require.NotZero(t, explanation.Graph.Edges.Len())

	order, err := explanation.Graph.TopologicalOrder()
	require.NoError(t, err)
	require.NotZero(t, order.Len())
	require.Contains(t, explanation.Graph.DOT(), "digraph dix")
	require.True(t, graphHasEdge(explanation.Graph, dix.DependencyGraphEdgeSubApp))
	require.True(t, graphHasEdge(explanation.Graph, dix.DependencyGraphEdgeConsumes))
	require.True(t, graphHasNode(explanation.Graph, dix.DependencyGraphNodeService, dix.TypedService[graphConfig]().Name))
}

func TestValidationMissingDependencyIncludesAvailableServices(t *testing.T) {
	t.Parallel()

	app := dix.New(
		"root",
		dix.Modules(dix.NewModule(
			"mod",
			dix.Providers(dix.Value(graphConfig{})),
			dix.Invokes(dix.RawInvokeWithMetadata(func(*dix.Container) error {
				return nil
			}, dix.InvokeMetadata{
				Label:        "missing",
				Dependencies: dix.ServiceRefs(dix.NamedService("graphConfig")),
			})),
		)),
	)

	err := app.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "available services include")
	require.Contains(t, err.Error(), "graphConfig")
}

func TestRuntimeScopeAndLifecycleSummary(t *testing.T) {
	t.Parallel()

	app := dix.New(
		"root",
		dix.Modules(dix.NewModule(
			"lifecycle",
			dix.Hooks(
				dix.OnStart0(func(context.Context) error { return nil }),
				dix.OnStop0(func(context.Context) error { return nil }),
			),
		)),
		dix.SubApps(dix.NewSubApp("worker")),
	)

	runtime, err := app.Build()
	require.NoError(t, err)
	require.False(t, runtime.IsSubApp())

	lifecycle := runtime.LifecycleSummary()
	require.Equal(t, 1, lifecycle.StartHooks)
	require.Equal(t, 1, lifecycle.StopHooks)

	worker, found := runtime.SubApp("worker")
	require.True(t, found)
	require.True(t, worker.IsSubApp())
	parentName, found := worker.ParentName()
	require.True(t, found)
	require.Equal(t, "root", parentName)
	require.Equal(t, []string{"root", "worker"}, worker.ScopePath().Values())

	summaries := runtime.SubAppSummaries()
	require.Equal(t, 1, summaries.Len())
	summary, _ := summaries.Get(0)
	require.Equal(t, "worker", summary.Name)
	require.Equal(t, "root", summary.ParentName)
}

func graphHasEdge(graph dix.DependencyGraph, kind dix.DependencyGraphEdgeKind) bool {
	found := false
	graph.Edges.Range(func(_ int, edge dix.DependencyGraphEdge) bool {
		found = edge.Kind == kind
		return !found
	})
	return found
}

func graphHasNode(graph dix.DependencyGraph, kind dix.DependencyGraphNodeKind, labelPart string) bool {
	found := false
	graph.Nodes.Range(func(_ int, node dix.DependencyGraphNode) bool {
		found = node.Kind == kind && strings.Contains(node.Label, labelPart)
		return !found
	})
	return found
}
