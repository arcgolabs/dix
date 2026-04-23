//revive:disable:file-length-limit Build planning is kept together because the steps share one orchestration flow.

package dix

import (
	"context"
	"errors"
	"time"

	"github.com/arcgolabs/collectionx"
	collectionset "github.com/arcgolabs/collectionx/set"
	"github.com/samber/oops"
)

type buildPlan struct {
	spec     *appSpec
	modules  collectionx.List[*moduleSpec]
	profile  Profile
	subplans collectionx.List[*buildPlan]
}

func newUnvalidatedBuildPlan(ctx context.Context, app *App) (*buildPlan, error) {
	return newUnvalidatedBuildPlanWithParentProfile(ctx, app, "", false)
}

func newUnvalidatedBuildPlanWithParentProfile(ctx context.Context, app *App, parentProfile Profile, hasParent bool) (*buildPlan, error) {
	if app == nil || app.spec == nil {
		return nil, oops.In("dix").
			With("op", "new_unvalidated_build_plan").
			New("app is nil")
	}

	profile, err := resolveBuildProfileWithFallback(ctx, app, parentProfile, hasParent)
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

	subplans, err := buildSubPlans(ctx, app.spec.subapps, profile)
	if err != nil {
		return nil, err
	}

	plan := &buildPlan{
		spec:     app.spec,
		modules:  modules,
		profile:  profile,
		subplans: subplans,
	}

	return plan, nil
}

func buildSubPlans(ctx context.Context, apps collectionx.List[*App], parentProfile Profile) (collectionx.List[*buildPlan], error) {
	subplans := collectionx.NewList[*buildPlan]()
	if apps == nil || apps.Len() == 0 {
		return subplans, nil
	}

	names := collectionset.NewSetWithCapacity[string](apps.Len())
	var buildErr error
	apps.Range(func(_ int, app *App) bool {
		if app == nil || app.spec == nil {
			buildErr = oops.In("dix").With("op", "build_subplans").New("subapp is nil")
			return false
		}
		name := app.Name()
		if name == "" {
			buildErr = oops.In("dix").With("op", "build_subplans").New("subapp name is required")
			return false
		}
		if names.Contains(name) {
			buildErr = oops.In("dix").
				With("op", "build_subplans", "subapp", name).
				Errorf("duplicate subapp name detected: %s", name)
			return false
		}
		names.Add(name)

		subplan, err := newUnvalidatedBuildPlanWithParentProfile(ctx, app, parentProfile, true)
		if err != nil {
			buildErr = oops.In("dix").
				With("op", "build_subplan", "subapp", name).
				Wrapf(err, "subapp build plan failed")
			return false
		}
		subplans.Add(subplan)
		return true
	})
	if buildErr != nil {
		return nil, buildErr
	}
	return subplans, nil
}

func resolveBuildProfile(ctx context.Context, app *App) (Profile, error) {
	return resolveBuildProfileWithFallback(ctx, app, "", false)
}

func resolveBuildProfileWithFallback(ctx context.Context, app *App, fallback Profile, hasFallback bool) (Profile, error) {
	if err := validateProfileResolutionApp(app); err != nil {
		return "", err
	}
	if app.spec.profileConfigured {
		return app.spec.profile, nil
	}

	defaultProfile := app.spec.profile
	if hasFallback {
		defaultProfile = fallback
	}

	plan, err := newProfileBootstrapPlanWithProfile(app, defaultProfile)
	if err != nil {
		return "", err
	}
	if !plan.declaresProviderOutput(TypedService[Profile]()) {
		return defaultProfile, nil
	}
	if reportErr := validateTypedGraphReport(plan).Err(); reportErr != nil {
		return "", reportErr
	}

	return resolveDeclaredBuildProfile(ctx, app, plan)
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
		spec:     app.spec,
		modules:  modules,
		profile:  profile,
		subplans: collectionx.NewList[*buildPlan](),
	}, nil
}

func resolveDeclaredBuildProfile(ctx context.Context, app *App, plan *buildPlan) (Profile, error) {
	rt := newRuntime(app.spec, plan)
	plan.registerRuntimeCoreServices(rt)
	plan.registerProviders(ctx, rt, false)
	newContributionPlan(plan.modules).register(ctx, rt, false)

	profile, resolveErr := ResolveAs[Profile](rt.container)
	if resolveErr != nil {
		resolveErr = oops.In("dix").
			With("op", "resolve_declared_profile", "app", app.Name(), "service", serviceNameOf[Profile]()).
			Wrapf(resolveErr, "resolve declared profile failed")
	}

	if report := rt.container.ShutdownReport(ctx); report != nil && len(report.Errors) > 0 {
		return "", errors.Join(resolveErr, report)
	}
	if resolveErr != nil {
		return "", resolveErr
	}

	return profile, nil
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

	if parent == nil {
		rt = newRuntime(p.spec, p)
	} else {
		rt, err = newChildRuntime(p.spec, p, parent)
		if err != nil {
			return nil, err
		}
	}
	p.registerRuntimeCoreServices(rt)

	providersRegistered, err := p.prepareFrameworkConfig(ctx, rt)
	if err != nil {
		err = cleanupBuildFailure(ctx, rt, err)
		return nil, err
	}

	debugEnabled := eventLoggerEnabled(ctx, rt.eventLogger, EventLevelDebug)
	infoEnabled := eventLoggerEnabled(ctx, rt.eventLogger, EventLevelInfo)
	p.logBuildStart(ctx, rt, infoEnabled, debugEnabled)

	if providersRegistered {
		p.logProviderRegistrations(ctx, rt, debugEnabled)
	} else {
		p.registerProviders(ctx, rt, debugEnabled)
	}
	newContributionPlan(p.modules).register(ctx, rt, debugEnabled)

	if err := p.bindHooksAndRunSetups(ctx, rt, debugEnabled); err != nil {
		err = cleanupBuildFailure(ctx, rt, err)
		return nil, err
	}

	if err := p.runInvokes(ctx, rt, debugEnabled); err != nil {
		err = cleanupBuildFailure(ctx, rt, err)
		return nil, err
	}

	if err := p.buildSubApps(ctx, rt); err != nil {
		err = cleanupBuildFailure(ctx, rt, err)
		return nil, err
	}

	rt.transitionState(ctx, AppStateBuilt, "build completed")
	rt.logDebugInformation(ctx)
	return rt, nil
}

func (p *buildPlan) buildSubApps(ctx context.Context, rt *Runtime) error {
	if p == nil || p.subplans == nil || p.subplans.Len() == 0 {
		return nil
	}

	var buildErr error
	p.subplans.Range(func(_ int, subplan *buildPlan) bool {
		subrt, err := subplan.build(ctx, rt)
		if err != nil {
			buildErr = err
			return false
		}
		rt.subapps.Add(subrt)
		return true
	})
	return buildErr
}

func (p *buildPlan) emitBuildResult(ctx context.Context, rt *Runtime, duration time.Duration, err error) {
	if p == nil || p.spec == nil {
		return
	}
	if rt == nil {
		event := p.buildEvent(duration, err)
		p.spec.emitBuild(ctx, event)
		return
	}
	event := p.runtimeBuildEvent(rt, duration, err)
	emitEventLogger(ctx, rt.eventLogger, event)
	emitObservers(ctx, rt.spec.observerDispatchers, func(ctx context.Context, observer Observer) {
		observer.OnBuild(ctx, event)
	})
}

func cleanupBuildFailure(ctx context.Context, rt *Runtime, buildErr error) error {
	if rt == nil || rt.container == nil {
		return buildErr
	}

	report := rt.container.ShutdownReport(ctx)
	if report == nil || len(report.Errors) == 0 {
		return buildErr
	}
	rt.logMessage(ctx, EventLevelError, "build cleanup failed", "app", rt.Name(), "error", report)
	return errors.Join(buildErr, report)
}

func (p *buildPlan) logBuildStart(ctx context.Context, rt *Runtime, infoEnabled, debugEnabled bool) {
	if infoEnabled {
		rt.logMessage(ctx, EventLevelInfo, "building app", "app", p.spec.meta.Name, "profile", p.profile)
	}
	if debugEnabled {
		rt.logMessage(ctx, EventLevelDebug, "build plan ready",
			"app", p.spec.meta.Name,
			"modules", p.modules.Len(),
			"providers", countModuleProviders(p.modules),
			"hooks", countModuleHooks(p.modules),
			"setups", countModuleSetups(p.modules),
			"invokes", countModuleInvokes(p.modules),
		)
	}
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
