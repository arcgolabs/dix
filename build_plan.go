package dix

import (
	"context"
	collectionlist "github.com/arcgolabs/collectionx/list"
	collectionset "github.com/arcgolabs/collectionx/set"
	"github.com/samber/oops"
	"time"
)

type buildPlan struct {
	spec              *appSpec
	modules           *collectionlist.List[*moduleSpec]
	profile           Profile
	parent            *buildPlan
	inheritedServices *collectionset.Set[string]
	subplans          *collectionlist.List[*buildPlan]
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
		subplans:          collectionlist.NewList[*buildPlan](),
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
		subplans:          collectionlist.NewList[*buildPlan](),
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
	_, found := collectionlist.FindList(p.modules, func(_ int, mod *moduleSpec) bool {
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
			startedAt := time.Now()
			if debugEnabled {
				rt.logMessage(ctx, EventLevelDebug, "registering provider",
					"module", mod.name,
					"label", provider.meta.Label,
					"output", provider.meta.Output.Name,
					"dependencies", serviceRefNames(provider.meta.Dependencies),
					"eager", provider.meta.Eager,
					"raw", provider.meta.Raw,
				)
			}
			provider.apply(rt.container)
			if debugEnabled {
				rt.logMessage(ctx, EventLevelDebug, "provider registered",
					"module", mod.name,
					"label", provider.meta.Label,
					"output", provider.meta.Output.Name,
					"eager", provider.meta.Eager,
					"duration", time.Since(startedAt),
				)
			}
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
				"eager", provider.meta.Eager,
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
				"name", hook.meta.Name,
				"kind", hook.meta.Kind,
				"priority", hook.meta.Priority,
				"parallel", hook.meta.Parallel,
				"dependencies", serviceRefNames(hook.meta.Dependencies),
				"raw", hook.meta.Raw,
			)
		}
		hook.bind(rt.container, rt.lifecycle)
		return true
	})
}
