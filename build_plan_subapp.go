package dix

import (
	"context"
	"errors"

	"github.com/arcgolabs/collectionx"
	collectionset "github.com/arcgolabs/collectionx/set"
	"github.com/samber/oops"
)

func buildSubPlans(ctx context.Context, apps collectionx.List[*App], parent *buildPlan) (collectionx.List[*buildPlan], error) {
	subplans := collectionx.NewList[*buildPlan]()
	if apps == nil || apps.Len() == 0 {
		return subplans, nil
	}

	names := collectionset.NewSetWithCapacity[string](apps.Len())
	var buildErr error
	apps.Range(func(_ int, app *App) bool {
		subplan, err := buildSubPlan(ctx, app, parent, names)
		if err != nil {
			buildErr = err
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

func buildSubPlan(ctx context.Context, app *App, parent *buildPlan, names *collectionset.Set[string]) (*buildPlan, error) {
	name, err := validateSubApp(app, names)
	if err != nil {
		return nil, err
	}

	subplan, err := newUnvalidatedBuildPlanWithParent(ctx, app, parent)
	if err != nil {
		return nil, oops.In("dix").
			With("op", "build_subplan", "subapp", name).
			Wrapf(err, "subapp build plan failed")
	}
	return subplan, nil
}

func validateSubApp(app *App, names *collectionset.Set[string]) (string, error) {
	if app == nil || app.spec == nil {
		return "", oops.In("dix").With("op", "build_subplans").New("subapp is nil")
	}
	name := app.Name()
	if name == "" {
		return "", oops.In("dix").With("op", "build_subplans").New("subapp name is required")
	}
	if names.Contains(name) {
		return "", oops.In("dix").
			With("op", "build_subplans", "subapp", name).
			Errorf("duplicate subapp name detected: %s", name)
	}
	names.Add(name)
	return name, nil
}

func resolveBuildProfileWithParent(ctx context.Context, app *App, parent *buildPlan) (Profile, error) {
	if err := validateProfileResolutionApp(app); err != nil {
		return "", err
	}
	if app.spec.profileConfigured {
		return app.spec.profile, nil
	}

	defaultProfile := app.spec.profile
	if parent != nil {
		defaultProfile = parent.profile
	}

	plan, err := newProfileBootstrapPlanWithProfile(app, defaultProfile)
	if err != nil {
		return "", err
	}
	plan.parent = parent
	plan.inheritedServices = inheritedServicesForParent(parent)
	if !plan.declaresProviderOutput(TypedService[Profile]()) {
		return defaultProfile, nil
	}
	if reportErr := validateTypedGraphReportWithInherited(plan, plan.inheritedServices).Err(); reportErr != nil {
		return "", reportErr
	}

	return resolveDeclaredBuildProfile(ctx, app, plan, parent)
}

func resolveDeclaredBuildProfile(ctx context.Context, app *App, plan, parent *buildPlan) (Profile, error) {
	parentRuntime, cleanupParent, err := buildProfileResolutionRuntimeChain(ctx, parent)
	if err != nil {
		return "", err
	}

	rt, err := newProfileResolutionRuntime(app, plan, parentRuntime)
	if err != nil {
		if cleanupParent != nil {
			err = errors.Join(err, cleanupParent())
		}
		return "", err
	}
	plan.registerRuntimeCoreServices(rt)
	plan.registerProviders(ctx, rt, false)
	newContributionPlan(plan.modules).register(ctx, rt, false)

	profile, resolveErr := resolveProfileFromRuntime(ctx, app, rt)
	if cleanupParent == nil {
		return profile, resolveErr
	}
	return profile, errors.Join(resolveErr, cleanupParent())
}

func newProfileResolutionRuntime(app *App, plan *buildPlan, parentRuntime *Runtime) (*Runtime, error) {
	if parentRuntime == nil {
		return newRuntime(app.spec, plan), nil
	}
	return newChildRuntime(app.spec, plan, parentRuntime)
}

func resolveProfileFromRuntime(ctx context.Context, app *App, rt *Runtime) (Profile, error) {
	profile, resolveErr := ResolveAs[Profile](rt.container)
	if resolveErr != nil {
		resolveErr = oops.In("dix").
			With("op", "resolve_declared_profile", "app", app.Name(), "service", serviceNameOfSpec[Profile](app.spec)).
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

func buildProfileResolutionRuntimeChain(ctx context.Context, plan *buildPlan) (_ *Runtime, cleanup func() error, err error) {
	if plan == nil {
		return nil, nil, nil
	}

	if plan.parent == nil {
		return buildRootProfileResolutionRuntime(ctx, plan)
	}
	return buildChildProfileResolutionRuntime(ctx, plan)
}

func buildRootProfileResolutionRuntime(ctx context.Context, plan *buildPlan) (*Runtime, func() error, error) {
	rt := newRuntime(plan.spec, plan)
	plan.registerRuntimeCoreServices(rt)
	plan.registerProviders(ctx, rt, false)
	newContributionPlan(plan.modules).register(ctx, rt, false)
	return rt, func() error {
		report := rt.container.ShutdownReport(ctx)
		if report == nil || len(report.Errors) == 0 {
			return nil
		}
		return report
	}, nil
}

func buildChildProfileResolutionRuntime(ctx context.Context, plan *buildPlan) (*Runtime, func() error, error) {
	parentRuntime, cleanupParent, err := buildProfileResolutionRuntimeChain(ctx, plan.parent)
	if err != nil {
		return nil, nil, err
	}

	rt, err := newChildRuntime(plan.spec, plan, parentRuntime)
	if err != nil {
		if cleanupParent != nil {
			err = errors.Join(err, cleanupParent())
		}
		return nil, nil, err
	}
	plan.registerRuntimeCoreServices(rt)
	plan.registerProviders(ctx, rt, false)
	newContributionPlan(plan.modules).register(ctx, rt, false)
	return rt, cleanupParent, nil
}

func inheritedServicesForParent(parent *buildPlan) *collectionset.Set[string] {
	if parent == nil {
		return collectionset.NewSet[string]()
	}
	return parent.visibleServiceNames()
}

func (p *buildPlan) visibleServiceNames() *collectionset.Set[string] {
	if p == nil {
		return collectionset.NewSet[string]()
	}
	return mergeServiceNameSets(p.inheritedServices, declaredServiceNames(p))
}
