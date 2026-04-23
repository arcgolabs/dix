package dix

import (
	"context"
	"sync"

	"github.com/arcgolabs/collectionx"
	"github.com/samber/oops"
)

type appPlanCache struct {
	mu     sync.RWMutex
	plan   *buildPlan
	report ValidationReport
	err    error
	ready  bool
}

func (a *App) cachedBuildPlan(ctx context.Context) (*buildPlan, ValidationReport, error) {
	if a == nil || a.spec == nil {
		err := oops.In("dix").
			With("op", "cached_build_plan").
			New("app is nil")
		return nil, ValidationReport{Errors: collectionx.NewList(err)}, err
	}

	if !a.buildPlanCacheable() {
		plan, report, err := computeBuildPlan(ctx, a)
		return plan, cloneValidationReport(report), err
	}

	a.planCache.mu.RLock()
	if a.planCache.ready {
		plan := a.planCache.plan
		report := cloneValidationReport(a.planCache.report)
		err := a.planCache.err
		a.planCache.mu.RUnlock()
		return plan, report, err
	}
	a.planCache.mu.RUnlock()

	plan, report, err := computeBuildPlan(ctx, a)

	a.planCache.mu.Lock()
	if !a.planCache.ready {
		a.planCache.plan = plan
		a.planCache.report = report
		a.planCache.err = err
		a.planCache.ready = true
	}
	cachedPlan := a.planCache.plan
	cachedReport := cloneValidationReport(a.planCache.report)
	cachedErr := a.planCache.err
	a.planCache.mu.Unlock()

	return cachedPlan, cachedReport, cachedErr
}

func computeBuildPlan(ctx context.Context, app *App) (*buildPlan, ValidationReport, error) {
	plan, err := newUnvalidatedBuildPlan(ctx, app)
	if err != nil {
		report := ValidationReport{Errors: collectionx.NewList(err)}
		return nil, report, err
	}

	report := validateBuildPlanTreeReport(plan)
	if reportErr := report.Err(); reportErr != nil {
		return plan, report, reportErr
	}

	return plan, report, nil
}

func (a *App) buildPlanCacheable() bool {
	if a == nil || a.spec == nil {
		return false
	}

	return appProfileResolutionCacheable(a)
}

func appProfileResolutionCacheable(app *App) bool {
	if app == nil || app.spec == nil {
		return false
	}

	if !app.spec.profileConfigured {
		plan, err := newProfileBootstrapPlan(app)
		if err != nil {
			return false
		}
		if plan.declaresProviderOutput(TypedService[Profile]()) {
			return false
		}
	}

	cacheable := true
	app.spec.subapps.Range(func(_ int, subapp *App) bool {
		cacheable = appProfileResolutionCacheable(subapp)
		return cacheable
	})
	return cacheable
}
