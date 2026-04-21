package dix

import (
	"context"
	"testing"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewModule_StoresCollectionBackedSpec(t *testing.T) {
	shared := NewModule("shared")
	mod := NewModule("feature",
		WithModuleProviders(
			Provider0(func() string { return "a" }),
			Provider0(func() string { return "b" }),
		),
		WithModuleInvokes(
			Invoke0(func() {}),
			Invoke0(func() {}),
		),
		WithModuleImports(shared, shared),
		WithModuleProfiles(ProfileDev, ProfileDev, ProfileProd),
		WithModuleExcludeProfiles(ProfileTest, ProfileTest),
		WithModuleTags("http", "http", "core"),
	)

	require.NotNil(t, mod.spec)
	assert.Equal(t, "feature", mod.Name())
	assert.Equal(t, 2, mod.spec.providers.Len())
	assert.Equal(t, 2, mod.spec.invokes.Len())
	assert.Equal(t, 2, mod.spec.imports.Len())
	assert.True(t, mod.spec.profiles.Contains(ProfileDev))
	assert.True(t, mod.spec.profiles.Contains(ProfileProd))
	assert.Equal(t, 2, mod.spec.profiles.Len())
	assert.True(t, mod.spec.excludeProfiles.Contains(ProfileTest))
	assert.Equal(t, 1, mod.spec.excludeProfiles.Len())
	assert.Equal(t, []string{"http", "core"}, mod.Tags().Values())
	assert.True(t, mod.Profiles().Contains(ProfileDev))
	assert.True(t, mod.Profiles().Contains(ProfileProd))
	assert.True(t, mod.ExcludeProfiles().Contains(ProfileTest))
	assert.False(t, mod.Disabled())
}

func TestModule_AccessorsReturnDefensiveCollections(t *testing.T) {
	dependency := NewModule("dependency")
	mod := NewModule("feature",
		WithModuleImports(dependency),
		WithModuleProfiles(ProfileDev),
		WithModuleExcludeProfiles(ProfileProd),
		WithModuleTags("http"),
		WithModuleDisabled(true),
	)

	mod.Tags().Add("mutated")
	mod.Profiles().Add(ProfileTest)
	mod.ExcludeProfiles().Add(ProfileDefault)
	mod.Imports().Add(NewModule("mutated"))

	assert.Equal(t, []string{"http"}, mod.Tags().Values())
	assert.True(t, mod.Profiles().Contains(ProfileDev))
	assert.False(t, mod.Profiles().Contains(ProfileTest))
	assert.True(t, mod.ExcludeProfiles().Contains(ProfileProd))
	assert.False(t, mod.ExcludeProfiles().Contains(ProfileDefault))
	assert.Equal(t, 1, mod.Imports().Len())
	assert.True(t, mod.Disabled())
}

func TestWithDebugNamedServiceDependencies_Deduplicates(t *testing.T) {
	app := New("debug",
		WithDebugNamedServiceDependencies("tenant.default", "tenant.default", "tenant.backup"),
	)

	require.NotNil(t, app.spec)
	require.Equal(t, 2, app.spec.debug.namedServiceDependencies.Len())
	assert.Equal(t, []string{"tenant.default", "tenant.backup"}, app.spec.debug.namedServiceDependencies.Values())
}

func TestWithModules_StoresSpecModules(t *testing.T) {
	left := NewModule("left")
	right := NewModule("right")

	app := New("modules", WithModules(left, right))

	require.NotNil(t, app.spec)
	require.Equal(t, 2, app.spec.modules.Len())
	values := app.spec.modules.Values()
	require.Len(t, values, 2)
	assert.Equal(t, "left", values[0].Name())
	assert.Equal(t, "right", values[1].Name())
}

func TestRuntimeDefaultLogger_IsSharedWithInternalComponents(t *testing.T) {
	app := New("logger-sharing")

	rt, err := app.Build()
	require.NoError(t, err)
	require.NotNil(t, rt)

	require.Same(t, app.Logger(), rt.Logger())
	require.Same(t, rt.Logger(), rt.container.logger)
	require.Same(t, rt.Logger(), rt.lifecycle.logger)

	eventLogger, ok := rt.eventLogger.(*slogEventLogger)
	require.True(t, ok)
	require.Same(t, rt.Logger(), eventLogger.logger)
}

func TestContainerHealthChecks_UsesCollectionList(t *testing.T) {
	c := newContainer(defaultLogger())

	c.RegisterHealthCheck("general", func(context.Context) error { return nil })
	c.RegisterLivenessCheck("live", func(context.Context) error { return nil })
	c.RegisterReadinessCheck("ready", func(context.Context) error { return nil })

	require.Equal(t, 3, c.healthChecks.Len())
	values := c.healthChecks.Values()
	require.Len(t, values, 3)
	assert.Equal(t, HealthKindGeneral, values[0].kind)
	assert.Equal(t, HealthKindLiveness, values[1].kind)
	assert.Equal(t, HealthKindReadiness, values[2].kind)
}

func TestLifecycle_UsesCollectionBackedHooks(t *testing.T) {
	logger := defaultLogger()
	lc := newLifecycle(logger)
	order := collectionx.NewListWithCapacity[string](4)

	lc.OnStart(func(context.Context) error {
		order.Add("start-1")
		return nil
	})
	lc.OnStart(func(context.Context) error {
		order.Add("start-2")
		return nil
	})
	lc.OnStop(func(context.Context) error {
		order.Add("stop-2")
		return nil
	})
	lc.OnStop(func(context.Context) error {
		order.Add("stop-1")
		return nil
	})

	started, err := lc.executeStartHooks(context.Background(), newContainer(logger))
	require.NoError(t, err)
	assert.Equal(t, 2, started)
	require.NoError(t, lc.executeStopHooks(context.Background(), newContainer(logger)))
	assert.Equal(t, []string{"start-1", "start-2", "stop-1", "stop-2"}, order.Values())
}

func TestFlattenModules_VisitorOrderIsDependencyFirst(t *testing.T) {
	shared := NewModule("shared")
	left := NewModule("left", WithModuleImports(shared))
	root := NewModule("root", WithModuleImports(left))

	modules, err := flattenModules(collectionx.NewList(root), ProfileDefault)
	require.NoError(t, err)
	require.Len(t, modules.Values(), 3)
	assert.Equal(t, []string{"shared", "left", "root"}, []string{
		modules.Values()[0].name,
		modules.Values()[1].name,
		modules.Values()[2].name,
	})
}

func TestWalkModules_DetectsImportCycle(t *testing.T) {
	left := NewModule("left").spec
	right := NewModule("right").spec

	left.imports.Add(Module{spec: right})
	right.imports.Add(Module{spec: left})

	err := walkModules(collectionx.NewList(Module{spec: left}), ProfileDefault, moduleVisitorFuncs{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "module import cycle detected")
	assert.Contains(t, err.Error(), "left")
	assert.Contains(t, err.Error(), "right")
}

func TestProfileFilter_FilterModulesEReturnsFlattenErrors(t *testing.T) {
	left := NewModule("left").spec
	right := NewModule("right").spec

	left.imports.Add(Module{spec: right})
	right.imports.Add(Module{spec: left})

	filter := NewProfileFilter(ProfileDefault)
	filtered, err := filter.FilterModulesE(collectionx.NewList(Module{spec: left}))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "module import cycle detected")
	assert.True(t, filtered.IsEmpty())
	assert.True(t, filter.FilterModules(collectionx.NewList(Module{spec: left})).IsEmpty())
}
