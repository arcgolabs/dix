package dix

import "context"

// ModuleBuilder incrementally constructs an immutable Module.
//
// A builder is not safe for concurrent use. Build returns a snapshot, so later
// builder changes do not mutate modules that were already built.
type ModuleBuilder struct {
	module Module
}

// NewModuleBuilder creates a typed module builder.
func NewModuleBuilder(name string) *ModuleBuilder {
	return &ModuleBuilder{module: NewModule(name)}
}

// Build returns an immutable snapshot of the module configured so far.
func (b *ModuleBuilder) Build() Module {
	if b == nil {
		return Module{}
	}
	return cloneModule(b.module)
}

// Apply applies existing module options to the builder.
func (b *ModuleBuilder) Apply(opts ...ModuleOption) *ModuleBuilder {
	if b == nil || b.module.spec == nil {
		return b
	}
	for _, opt := range opts {
		if opt != nil {
			opt(b.module.spec)
		}
	}
	return b
}

// Description sets the module description.
func (b *ModuleBuilder) Description(description string) *ModuleBuilder {
	return b.Apply(WithModuleDescription(description))
}

// Disabled sets whether the module is disabled.
func (b *ModuleBuilder) Disabled(disabled bool) *ModuleBuilder {
	return b.Apply(WithModuleDisabled(disabled))
}

// Tags appends tags to the module.
func (b *ModuleBuilder) Tags(tags ...string) *ModuleBuilder {
	return b.Apply(WithModuleTags(tags...))
}

// Profiles restricts the module to the listed profiles.
func (b *ModuleBuilder) Profiles(profiles ...Profile) *ModuleBuilder {
	return b.Apply(WithModuleProfiles(profiles...))
}

// ExcludeProfiles excludes the module from the listed profiles.
func (b *ModuleBuilder) ExcludeProfiles(profiles ...Profile) *ModuleBuilder {
	return b.Apply(WithModuleExcludeProfiles(profiles...))
}

// Imports appends imported modules to the module.
func (b *ModuleBuilder) Imports(modules ...Module) *ModuleBuilder {
	return b.Apply(WithModuleImports(modules...))
}

// Provide registers a typed singleton provider without dependencies.
func (b *ModuleBuilder) Provide[T any](fn func() T, opts ...ProviderOption) *ModuleBuilder {
	return b.addProvider(Provider(fn, opts...))
}

// ProvideErr registers a fallible typed singleton provider without dependencies.
func (b *ModuleBuilder) ProvideErr[T any](fn func() (T, error), opts ...ProviderOption) *ModuleBuilder {
	return b.addProvider(ProviderErr(fn, opts...))
}

// Provide1 registers a typed singleton provider with one dependency.
func (b *ModuleBuilder) Provide1[T, D1 any](fn func(D1) T, opts ...ProviderOption) *ModuleBuilder {
	return b.addProvider(Provider1(fn, opts...))
}

// Provide1Err registers a fallible typed singleton provider with one dependency.
func (b *ModuleBuilder) Provide1Err[T, D1 any](fn func(D1) (T, error), opts ...ProviderOption) *ModuleBuilder {
	return b.addProvider(ProviderErr1(fn, opts...))
}

// Provide2 registers a typed singleton provider with two dependencies.
func (b *ModuleBuilder) Provide2[T, D1, D2 any](fn func(D1, D2) T, opts ...ProviderOption) *ModuleBuilder {
	return b.addProvider(Provider2(fn, opts...))
}

// Provide2Err registers a fallible typed singleton provider with two dependencies.
func (b *ModuleBuilder) Provide2Err[T, D1, D2 any](fn func(D1, D2) (T, error), opts ...ProviderOption) *ModuleBuilder {
	return b.addProvider(ProviderErr2(fn, opts...))
}

// Provide3 registers a typed singleton provider with three dependencies.
func (b *ModuleBuilder) Provide3[T, D1, D2, D3 any](fn func(D1, D2, D3) T, opts ...ProviderOption) *ModuleBuilder {
	return b.addProvider(Provider3(fn, opts...))
}

// Provide3Err registers a fallible typed singleton provider with three dependencies.
func (b *ModuleBuilder) Provide3Err[T, D1, D2, D3 any](fn func(D1, D2, D3) (T, error), opts ...ProviderOption) *ModuleBuilder {
	return b.addProvider(ProviderErr3(fn, opts...))
}

// Provide4 registers a typed singleton provider with four dependencies.
func (b *ModuleBuilder) Provide4[T, D1, D2, D3, D4 any](fn func(D1, D2, D3, D4) T, opts ...ProviderOption) *ModuleBuilder {
	return b.addProvider(Provider4(fn, opts...))
}

// Provide4Err registers a fallible typed singleton provider with four dependencies.
func (b *ModuleBuilder) Provide4Err[T, D1, D2, D3, D4 any](fn func(D1, D2, D3, D4) (T, error), opts ...ProviderOption) *ModuleBuilder {
	return b.addProvider(ProviderErr4(fn, opts...))
}

// Provide5 registers a typed singleton provider with five dependencies.
func (b *ModuleBuilder) Provide5[T, D1, D2, D3, D4, D5 any](fn func(D1, D2, D3, D4, D5) T, opts ...ProviderOption) *ModuleBuilder {
	return b.addProvider(Provider5(fn, opts...))
}

// Provide5Err registers a fallible typed singleton provider with five dependencies.
func (b *ModuleBuilder) Provide5Err[T, D1, D2, D3, D4, D5 any](fn func(D1, D2, D3, D4, D5) (T, error), opts ...ProviderOption) *ModuleBuilder {
	return b.addProvider(ProviderErr5(fn, opts...))
}

// Provide6 registers a typed singleton provider with six dependencies.
func (b *ModuleBuilder) Provide6[T, D1, D2, D3, D4, D5, D6 any](fn func(D1, D2, D3, D4, D5, D6) T, opts ...ProviderOption) *ModuleBuilder {
	return b.addProvider(Provider6(fn, opts...))
}

// Provide6Err registers a fallible typed singleton provider with six dependencies.
func (b *ModuleBuilder) Provide6Err[T, D1, D2, D3, D4, D5, D6 any](fn func(D1, D2, D3, D4, D5, D6) (T, error), opts ...ProviderOption) *ModuleBuilder {
	return b.addProvider(ProviderErr6(fn, opts...))
}

// Contribute registers a typed collection contribution without dependencies.
func (b *ModuleBuilder) Contribute[T any](fn func() T, opts ...ContributionOption) *ModuleBuilder {
	return b.addProvider(Contribute(fn, opts...))
}

// ContributeErr registers a fallible typed collection contribution without dependencies.
func (b *ModuleBuilder) ContributeErr[T any](fn func() (T, error), opts ...ContributionOption) *ModuleBuilder {
	return b.addProvider(ContributeErr(fn, opts...))
}

// Contribute1 registers a typed collection contribution with one dependency.
func (b *ModuleBuilder) Contribute1[T, D1 any](fn func(D1) T, opts ...ContributionOption) *ModuleBuilder {
	return b.addProvider(Contribute1(fn, opts...))
}

// ContributeErr1 registers a fallible typed collection contribution with one dependency.
func (b *ModuleBuilder) ContributeErr1[T, D1 any](fn func(D1) (T, error), opts ...ContributionOption) *ModuleBuilder {
	return b.addProvider(ContributeErr1(fn, opts...))
}

// Contribute2 registers a typed collection contribution with two dependencies.
func (b *ModuleBuilder) Contribute2[T, D1, D2 any](fn func(D1, D2) T, opts ...ContributionOption) *ModuleBuilder {
	return b.addProvider(Contribute2(fn, opts...))
}

// ContributeErr2 registers a fallible typed collection contribution with two dependencies.
func (b *ModuleBuilder) ContributeErr2[T, D1, D2 any](fn func(D1, D2) (T, error), opts ...ContributionOption) *ModuleBuilder {
	return b.addProvider(ContributeErr2(fn, opts...))
}

// Contribute3 registers a typed collection contribution with three dependencies.
func (b *ModuleBuilder) Contribute3[T, D1, D2, D3 any](fn func(D1, D2, D3) T, opts ...ContributionOption) *ModuleBuilder {
	return b.addProvider(Contribute3(fn, opts...))
}

// ContributeErr3 registers a fallible typed collection contribution with three dependencies.
func (b *ModuleBuilder) ContributeErr3[T, D1, D2, D3 any](fn func(D1, D2, D3) (T, error), opts ...ContributionOption) *ModuleBuilder {
	return b.addProvider(ContributeErr3(fn, opts...))
}

// Invoke registers an invoke callback without dependencies.
func (b *ModuleBuilder) Invoke(fn func()) *ModuleBuilder {
	return b.addInvoke(Invoke(fn))
}

// Invoke1 registers an invoke callback with one dependency.
func (b *ModuleBuilder) Invoke1[T any](fn func(T)) *ModuleBuilder {
	return b.addInvoke(Invoke1(fn))
}

// Invoke2 registers an invoke callback with two dependencies.
func (b *ModuleBuilder) Invoke2[T1, T2 any](fn func(T1, T2)) *ModuleBuilder {
	return b.addInvoke(Invoke2(fn))
}

// Invoke3 registers an invoke callback with three dependencies.
func (b *ModuleBuilder) Invoke3[T1, T2, T3 any](fn func(T1, T2, T3)) *ModuleBuilder {
	return b.addInvoke(Invoke3(fn))
}

// Invoke4 registers an invoke callback with four dependencies.
func (b *ModuleBuilder) Invoke4[T1, T2, T3, T4 any](fn func(T1, T2, T3, T4)) *ModuleBuilder {
	return b.addInvoke(Invoke4(fn))
}

// Invoke5 registers an invoke callback with five dependencies.
func (b *ModuleBuilder) Invoke5[T1, T2, T3, T4, T5 any](fn func(T1, T2, T3, T4, T5)) *ModuleBuilder {
	return b.addInvoke(Invoke5(fn))
}

// Invoke6 registers an invoke callback with six dependencies.
func (b *ModuleBuilder) Invoke6[T1, T2, T3, T4, T5, T6 any](fn func(T1, T2, T3, T4, T5, T6)) *ModuleBuilder {
	return b.addInvoke(Invoke6(fn))
}

// Setup registers a raw setup callback.
func (b *ModuleBuilder) Setup(fn func(*Container, Lifecycle) error) *ModuleBuilder {
	if b == nil || b.module.spec == nil {
		return b
	}
	b.module.spec.setups.Add(Setup(fn))
	return b
}

// Start appends a lifecycle start hook without dependencies.
func (b *ModuleBuilder) Start(fn func(context.Context) error, opts ...LifecycleHookOption) *ModuleBuilder {
	return b.addHook(OnStart0(fn, opts...))
}

// Stop appends a lifecycle stop hook without dependencies.
func (b *ModuleBuilder) Stop(fn func(context.Context) error, opts ...LifecycleHookOption) *ModuleBuilder {
	return b.addHook(OnStop0(fn, opts...))
}

func (b *ModuleBuilder) addProvider(provider ProviderFunc) *ModuleBuilder {
	if b != nil && b.module.spec != nil {
		b.module.spec.providers.Add(provider)
	}
	return b
}

func (b *ModuleBuilder) addInvoke(invoke InvokeFunc) *ModuleBuilder {
	if b != nil && b.module.spec != nil {
		b.module.spec.invokes.Add(invoke)
	}
	return b
}

func (b *ModuleBuilder) addHook(hook HookFunc) *ModuleBuilder {
	if b != nil && b.module.spec != nil {
		b.module.spec.hooks.Add(hook)
	}
	return b
}

func cloneModule(module Module) Module {
	if module.spec == nil {
		return Module{}
	}
	spec := *module.spec
	spec.providers = module.spec.providers.Clone()
	spec.setups = module.spec.setups.Clone()
	spec.invokes = module.spec.invokes.Clone()
	spec.hooks = module.spec.hooks.Clone()
	spec.imports = module.spec.imports.Clone()
	spec.profiles = module.spec.profiles.Clone()
	spec.excludeProfiles = module.spec.excludeProfiles.Clone()
	spec.tags = module.spec.tags.Clone()
	return Module{spec: &spec}
}
