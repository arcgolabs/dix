package dix

import (
	"strconv"

	collectionlist "github.com/arcgolabs/collectionx/list"
	collectionset "github.com/arcgolabs/collectionx/set"
)

// TestOption configures a test-only App derived from an existing App.
type TestOption func(*testOptions)

type testOptions struct {
	disabledModules *collectionset.Set[string]
	providers       *collectionlist.List[ProviderFunc]
	modules         *collectionlist.List[Module]
}

func newTestOptions() testOptions {
	return testOptions{
		disabledModules: collectionset.NewSet[string](),
		providers:       collectionlist.NewList[ProviderFunc](),
		modules:         collectionlist.NewList[Module](),
	}
}

// Test returns a derived App with test-only module disables, provider replacements, and extra modules.
//
// The original App and its module specs are not mutated.
func (a *App) Test(opts ...TestOption) *App {
	if a == nil || a.spec == nil {
		return NewDefault()
	}

	spec := cloneAppSpecForMutation(a.spec)
	testOpts := newTestOptions()
	for _, opt := range opts {
		if opt != nil {
			opt(&testOpts)
		}
	}

	disableModuleTree(spec.modules, testOpts.disabledModules)
	replaceTestProviders(spec, testOpts.providers)
	spec.modules.Add(testOpts.modules.Values()...)
	return &App{spec: spec}
}

// TestDisableModules disables modules by name in a derived test App.
func TestDisableModules(names ...string) TestOption {
	return func(opts *testOptions) {
		if opts != nil {
			opts.disabledModules.Add(names...)
		}
	}
}

// TestProviders replaces providers with matching output or alias service names in a derived test App.
func TestProviders(providers ...ProviderFunc) TestOption {
	return func(opts *testOptions) {
		if opts != nil {
			opts.providers.Add(providers...)
		}
	}
}

// TestProvider replaces one provider in a derived test App.
func TestProvider(provider ProviderFunc) TestOption {
	return TestProviders(provider)
}

// TestValue replaces a typed service with a test value in a derived test App.
func TestValue[T any](value T, opts ...ProviderOption) TestOption {
	return TestProvider(Value(value, opts...))
}

// TestModules appends extra modules to a derived test App.
func TestModules(modules ...Module) TestOption {
	return func(opts *testOptions) {
		if opts != nil {
			opts.modules.Add(modules...)
		}
	}
}

func disableModuleTree(modules *collectionlist.List[Module], names *collectionset.Set[string]) {
	if modules == nil || names == nil || names.Len() == 0 {
		return
	}
	visited := collectionset.NewSet[*moduleSpec]()
	walkModuleSpecs(modules, visited, func(spec *moduleSpec) {
		if names.Contains(spec.name) {
			spec.disabled = true
		}
	})
}

func replaceTestProviders(spec *appSpec, providers *collectionlist.List[ProviderFunc]) {
	if spec == nil || providers == nil || providers.Len() == 0 {
		return
	}
	names := replacementProviderServiceNames(providers)
	removeProvidersByServiceName(spec.modules, names)
	spec.modules.Add(NewModule(nextTestOverrideModuleName(spec.modules), Providers(providers.Values()...)))
}

func replacementProviderServiceNames(providers *collectionlist.List[ProviderFunc]) serviceNameSet {
	names := newServiceNameSet(providers.Len())
	providers.Range(func(_ int, provider ProviderFunc) bool {
		meta := provider.meta
		names.Add(meta.Output.Name)
		meta.Aliases.Range(func(_ int, alias ServiceRef) bool {
			names.Add(alias.Name)
			return true
		})
		return true
	})
	return names
}

func removeProvidersByServiceName(modules *collectionlist.List[Module], names serviceNameSet) {
	if modules == nil || names.values == nil || names.values.Len() == 0 {
		return
	}
	visited := collectionset.NewSet[*moduleSpec]()
	walkModuleSpecs(modules, visited, func(spec *moduleSpec) {
		spec.providers = filterProvidersByServiceName(spec.providers, names)
	})
}

func filterProvidersByServiceName(providers *collectionlist.List[ProviderFunc], names serviceNameSet) *collectionlist.List[ProviderFunc] {
	if providers == nil || providers.Len() == 0 {
		return collectionlist.NewList[ProviderFunc]()
	}
	return collectionlist.RejectList(providers, func(_ int, provider ProviderFunc) bool {
		return providerMatchesAnyService(provider, names)
	})
}

func providerMatchesAnyService(provider ProviderFunc, names serviceNameSet) bool {
	meta := provider.meta
	if names.Contains(meta.Output.Name) {
		return true
	}
	return meta.Aliases.Stream().Any(func(alias ServiceRef) bool {
		return names.Contains(alias.Name)
	})
}

func nextTestOverrideModuleName(modules *collectionlist.List[Module]) string {
	const base = "dix:test-overrides"
	names := collectionset.NewSet[string]()
	walkModuleSpecs(modules, collectionset.NewSet[*moduleSpec](), func(spec *moduleSpec) {
		names.Add(spec.name)
	})
	if !names.Contains(base) {
		return base
	}
	for index := 2; ; index++ {
		name := base + ":" + strconv.Itoa(index)
		if !names.Contains(name) {
			return name
		}
	}
}

func walkModuleSpecs(
	modules *collectionlist.List[Module],
	visited *collectionset.Set[*moduleSpec],
	visit func(*moduleSpec),
) {
	if modules == nil || visited == nil || visit == nil {
		return
	}
	modules.Range(func(_ int, module Module) bool {
		walkModuleSpec(module.spec, visited, visit)
		return true
	})
}

func walkModuleSpec(spec *moduleSpec, visited *collectionset.Set[*moduleSpec], visit func(*moduleSpec)) {
	if spec == nil || visited.Contains(spec) {
		return
	}
	visited.Add(spec)
	visit(spec)
	spec.imports.Range(func(_ int, imported Module) bool {
		walkModuleSpec(imported.spec, visited, visit)
		return true
	})
}
