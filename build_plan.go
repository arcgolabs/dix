package dix

import (
	"context"
	"time"

	"github.com/arcgolabs/collectionx"
	collectionset "github.com/arcgolabs/collectionx/set"
	"github.com/samber/oops"
)

type buildPlan struct {
	spec              *appSpec
	modules           collectionx.List[*moduleSpec]
	profile           Profile
	parent            *buildPlan
	inheritedServices *collectionset.Set[string]
	subplans          collectionx.List[*buildPlan]
}

func newUnvalidatedBuildPlan(ctx context.Context, app *App) (*buildPlan, error) {
	return newUnvalidatedBuildPlanWithParent(ctx, app, nil)
}

func newUnvalidatedBuildPlanWithParent(ctx context.Context, app *App, parent *buildPlan) (*buildPlan, error) {
	if app == nil || app.spec == nil {
		return nil, oops.In("dix").
			With("op", "new_unvalidated_build_plan").
			New("app is nil")
	}

	profile, err := resolveBuildProfileWithParent(ctx, app, parent)
	if err != nil {
		logMessageEvent(ctx, app.spec.resolvedEventLogger(), EventLevelError, "profile resolution failed", "app", app.Name(), "error", err)
		return nil, oops.In("dix").
			With("op", "resolve_build_profile", "app", app.Name()).
			Wrapf(err, "resolve build profile failed")
	}

	modules, err := flattenModuleList(app.spec.modules, profile)
	if err != nil {
		logMessageEvent(ctx, app.spec.resolvedEventLogger(), EventLevelError, "module flatten failed", "app", app.Name(), "error", err)
		return nil, oops.In("dix").
			With("op", "flatten_modules", "app", app.Name()).
			Wrapf(err, "module flatten failed")
	}

	plan := &buildPlan{
		spec:              app.spec,
		modules:           modules,
		profile:           profile,
		parent:            parent,
		inheritedServices: inheritedServicesForParent(parent),
		subplans:          collectionx.NewList[*buildPlan](),
	}

	subplans, err := buildSubPlans(ctx, app.spec.subapps, plan)
	if err != nil {
		return nil, err
	}
	plan.subplans = subplans

	return plan, nil
}

func validateProfileResolutionApp(app *App) error {
	if app == nil || app.spec == nil {
		return oops.In("dix").
			With("op", "resolve_build_profile").
			New("app is nil")
	}
	return nil
}

func newProfileBootstrapPlan(app *App) (*buildPlan, error) {
	return newProfileBootstrapPlanWithProfile(app, app.spec.profile)
}

func newProfileBootstrapPlanWithProfile(app *App, profile Profile) (*buildPlan, error) {
	modules, err := flattenProfileBootstrapModuleList(app.spec.modules)
	if err != nil {
		return nil, err
	}
	return &buildPlan{
		spec:              app.spec,
		modules:           modules,
		profile:           profile,
		inheritedServices: collectionset.NewSet[string](),
		subplans:          collectionx.NewList[*buildPlan](),
	}, nil
}

func (p *buildPlan) Build(ctx context.Context) (_ *Runtime, err error) {
	return p.build(ctx, nil)
}

func (p *buildPlan) build(ctx context.Context, parent *Runtime) (_ *Runtime, err error) {
	startedAt := time.Now()
	var rt *Runtime
	defer func() {
		p.emitBuildResult(ctx, rt, time.Since(startedAt), err)
	}()

	if p == nil || p.spec == nil {
		err = oops.In("dix").
			With("op", "build_runtime").
			New("build plan is nil")
		return nil, err
	}

	rt, err = p.newRuntime(parent)
	if err != nil {
		return nil, err
	}

	debugEnabled, err := p.prepareRuntimeBuild(ctx, rt)
	if err != nil {
		err = cleanupBuildFailure(ctx, rt, err)
		return nil, err
	}

	if err := p.completeRuntimeBuild(ctx, rt, debugEnabled); err != nil {
		err = cleanupBuildFailure(ctx, rt, err)
		return nil, err
	}
	return rt, nil
}

func (p *buildPlan) declaresProviderOutput(ref ServiceRef) bool {
	if p == nil || p.modules == nil || ref.Name == "" {
		return false
	}
	_, found := collectionx.FindList(p.modules, func(_ int, mod *moduleSpec) bool {
		return mod != nil && mod.providers.AnyMatch(func(_ int, provider ProviderFunc) bool {
			return provider.meta.Output.Name == ref.Name
		})
	})
	return found
}

func (p *buildPlan) registerProviders(ctx context.Context, rt *Runtime, debugEnabled bool) {
	p.modules.Range(func(_ int, mod *moduleSpec) bool {
		if debugEnabled {
			rt.logMessage(ctx, EventLevelDebug, "registering module",
				"module", mod.name,
				"providers", mod.providers.Len(),
				"hooks", mod.hooks.Len(),
				"setups", mod.setups.Len(),
				"invokes", mod.invokes.Len(),
			)
		}
		mod.providers.Range(func(_ int, provider ProviderFunc) bool {
			if debugEnabled {
				rt.logMessage(ctx, EventLevelDebug, "registering provider",
					"module", mod.name,
					"label", provider.meta.Label,
					"output", provider.meta.Output.Name,
					"dependencies", serviceRefNames(provider.meta.Dependencies),
					"raw", provider.meta.Raw,
				)
			}
			provider.apply(rt.container)
			return true
		})
		return true
	})
}

func (p *buildPlan) logProviderRegistrations(ctx context.Context, rt *Runtime, debugEnabled bool) {
	if !debugEnabled {
		return
	}
	p.modules.Range(func(_ int, mod *moduleSpec) bool {
		rt.logMessage(ctx, EventLevelDebug, "registering module",
			"module", mod.name,
			"providers", mod.providers.Len(),
			"hooks", mod.hooks.Len(),
			"setups", mod.setups.Len(),
			"invokes", mod.invokes.Len(),
		)
		mod.providers.Range(func(_ int, provider ProviderFunc) bool {
			rt.logMessage(ctx, EventLevelDebug, "registering provider",
				"module", mod.name,
				"label", provider.meta.Label,
				"output", provider.meta.Output.Name,
				"dependencies", serviceRefNames(provider.meta.Dependencies),
				"raw", provider.meta.Raw,
			)
			return true
		})
		return true
	})
}

func (p *buildPlan) bindHooksAndRunSetups(ctx context.Context, rt *Runtime, debugEnabled bool) error {
	var setupErr error
	p.modules.Range(func(_ int, mod *moduleSpec) bool {
		bindModuleHooks(ctx, mod, rt, debugEnabled)
		setupErr = runModuleSetups(ctx, mod, rt, debugEnabled)
		return setupErr == nil
	})
	return setupErr
}

func bindModuleHooks(ctx context.Context, mod *moduleSpec, rt *Runtime, debugEnabled bool) {
	mod.hooks.Range(func(_ int, hook HookFunc) bool {
		if debugEnabled {
			rt.logMessage(ctx, EventLevelDebug, "binding lifecycle hook",
				"module", mod.name,
				"label", hook.meta.Label,
				"kind", hook.meta.Kind,
				"dependencies", serviceRefNames(hook.meta.Dependencies),
				"raw", hook.meta.Raw,
			)
		}
		hook.bind(rt.container, rt.lifecycle)
		return true
	})
}

func runModuleSetups(ctx context.Context, mod *moduleSpec, rt *Runtime, debugEnabled bool) error {
	var setupErr error
	mod.setups.Range(func(_ int, setup SetupFunc) bool {
		if debugEnabled {
			rt.logMessage(ctx, EventLevelDebug, "running module setup",
				"module", mod.name,
				"label", setup.meta.Label,
				"dependencies", serviceRefNames(setup.meta.Dependencies),
				"provides", serviceRefNames(setup.meta.Provides),
				"overrides", serviceRefNames(setup.meta.Overrides),
				"graph_mutation", setup.meta.GraphMutation,
				"raw", setup.meta.Raw,
			)
		}
		if err := setup.apply(rt.container, rt.lifecycle); err != nil {
			rt.logMessage(ctx, EventLevelError, "module setup failed", "module", mod.name, "label", setup.meta.Label, "error", err)
			setupErr = oops.In("dix").
				With("op", "module_setup", "module", mod.name, "label", setup.meta.Label).
				Wrapf(err, "setup failed for module %s via %s", mod.name, setup.meta.Label)
			return false
		}
		if debugEnabled {
			rt.logMessage(ctx, EventLevelDebug, "module setup completed", "module", mod.name, "label", setup.meta.Label)
		}
		return true
	})
	return setupErr
}

func (p *buildPlan) runInvokes(ctx context.Context, rt *Runtime, debugEnabled bool) error {
	var buildErr error
	p.modules.Range(func(_ int, mod *moduleSpec) bool {
		buildErr = runModuleInvokes(ctx, mod, rt, debugEnabled)
		return buildErr == nil
	})
	return buildErr
}

func runModuleInvokes(ctx context.Context, mod *moduleSpec, rt *Runtime, debugEnabled bool) error {
	var invokeErr error
	mod.invokes.Range(func(_ int, invoke InvokeFunc) bool {
		if debugEnabled {
			rt.logMessage(ctx, EventLevelDebug, "running invoke",
				"module", mod.name,
				"label", invoke.meta.Label,
				"dependencies", serviceRefNames(invoke.meta.Dependencies),
				"raw", invoke.meta.Raw,
			)
		}
		invokeErr = invoke.apply(rt.container)
		if invokeErr == nil && debugEnabled {
			rt.logMessage(ctx, EventLevelDebug, "invoke completed", "module", mod.name, "label", invoke.meta.Label)
		}
		return invokeErr == nil
	})
	if invokeErr != nil {
		rt.logMessage(ctx, EventLevelError, "invoke failed", "module", mod.name, "error", invokeErr)
		return oops.In("dix").
			With("op", "module_invoke", "module", mod.name).
			Wrapf(invokeErr, "invoke failed in module %s", mod.name)
	}
	return nil
}
