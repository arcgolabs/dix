package dix_test

import (
	"context"
	"errors"
	"testing"

	"github.com/arcgolabs/dix"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type subappParentValue struct {
	Name string
}

type subappChildValue struct {
	ParentName string
}

type subappPrivateValue struct {
	Name string
}

func TestSubAppBuildsInChildScope(t *testing.T) {
	parentModule := dix.NewModule("parent",
		dix.Providers(
			dix.Value(subappParentValue{Name: "parent"}),
		),
	)
	childModule := dix.NewModule("child",
		dix.Providers(
			dix.Provider1(func(parent subappParentValue) subappChildValue {
				return subappChildValue{ParentName: parent.Name}
			}),
			dix.Value(subappPrivateValue{Name: "child-only"}),
		),
	)

	app := dix.New("parent",
		dix.Modules(parentModule),
		dix.SubApps(
			dix.NewSubApp("api", dix.Modules(childModule)),
		),
	)

	require.NoError(t, app.Validate())
	rt := buildRuntime(t, app)

	subrt, ok := rt.SubApp("api")
	require.True(t, ok)

	child, err := dix.ResolveAs[subappChildValue](subrt.Container())
	require.NoError(t, err)
	assert.Equal(t, "parent", child.ParentName)

	private, err := dix.ResolveAs[subappPrivateValue](subrt.Container())
	require.NoError(t, err)
	assert.Equal(t, "child-only", private.Name)

	_, err = dix.ResolveAs[subappPrivateValue](rt.Container())
	require.Error(t, err)
}

func TestSubAppLifecycleOrder(t *testing.T) {
	events := make([]string, 0, 4)
	parentModule := dix.NewModule("parent",
		dix.Hooks(
			dix.OnStart0(func(context.Context) error {
				events = append(events, "parent:start")
				return nil
			}),
			dix.OnStop0(func(context.Context) error {
				events = append(events, "parent:stop")
				return nil
			}),
		),
	)
	childModule := dix.NewModule("child",
		dix.Hooks(
			dix.OnStart0(func(context.Context) error {
				events = append(events, "child:start")
				return nil
			}),
			dix.OnStop0(func(context.Context) error {
				events = append(events, "child:stop")
				return nil
			}),
		),
	)

	rt := buildRuntime(t, dix.New("parent",
		dix.Modules(parentModule),
		dix.SubApps(dix.NewSubApp("api", dix.Modules(childModule))),
	))

	require.NoError(t, rt.Start(context.Background()))
	require.NoError(t, rt.Stop(context.Background()))

	assert.Equal(t, []string{
		"parent:start",
		"child:start",
		"child:stop",
		"parent:stop",
	}, events)
}

func TestSubAppHealthIsAggregatedByParent(t *testing.T) {
	parentModule := dix.NewModule("parent",
		dix.Setups(dix.SetupContainer(func(c *dix.Container) error {
			c.RegisterHealthCheck("parent", func(context.Context) error { return nil })
			return nil
		})),
	)
	childModule := dix.NewModule("child",
		dix.Setups(dix.SetupContainer(func(c *dix.Container) error {
			c.RegisterHealthCheck("child", func(context.Context) error { return errors.New("down") })
			return nil
		})),
	)

	rt := buildRuntime(t, dix.New("parent",
		dix.Modules(parentModule),
		dix.SubApps(dix.NewSubApp("api", dix.Modules(childModule))),
	))

	report := rt.CheckHealth(context.Background())
	assert.False(t, report.Healthy())
	require.NotNil(t, report.Checks)
	_, hasParent := report.Checks.Get("parent")
	_, hasChild := report.Checks.Get("api/child")
	assert.True(t, hasParent)
	assert.True(t, hasChild)
}

func TestSubAppInheritsParentProfile(t *testing.T) {
	childModule := dix.NewModule("child",
		dix.UseProfiles(dix.ProfileProd),
		dix.Providers(dix.Value(subappPrivateValue{Name: "prod-child"})),
	)

	rt := buildRuntime(t, dix.New("parent",
		dix.UseProfile(dix.ProfileProd),
		dix.SubApps(dix.NewSubApp("api", dix.Modules(childModule))),
	))
	subrt, ok := rt.SubApp("api")
	require.True(t, ok)
	assert.Equal(t, dix.ProfileProd, subrt.Profile())

	value, err := dix.ResolveAs[subappPrivateValue](subrt.Container())
	require.NoError(t, err)
	assert.Equal(t, "prod-child", value.Name)
}
