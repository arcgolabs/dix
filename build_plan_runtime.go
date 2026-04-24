package dix

import (
	"context"

	"github.com/samber/oops"
)

func (p *buildPlan) newRuntime(parent *Runtime) (*Runtime, error) {
	if parent == nil {
		return newRuntime(p.spec, p), nil
	}
	return newChildRuntime(p.spec, p, parent)
}

func (p *buildPlan) prepareRuntimeBuild(ctx context.Context, rt *Runtime) (bool, error) {
	p.registerRuntimeCoreServices(rt)

	providersRegistered, err := p.prepareFrameworkConfig(ctx, rt)
	if err != nil {
		return false, err
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
	return debugEnabled, nil
}

func (p *buildPlan) completeRuntimeBuild(ctx context.Context, rt *Runtime, debugEnabled bool) error {
	if err := p.bindHooksAndRunSetups(ctx, rt, debugEnabled); err != nil {
		return err
	}
	if err := p.runInvokes(ctx, rt, debugEnabled); err != nil {
		return err
	}
	if err := p.buildSubApps(ctx, rt); err != nil {
		return err
	}

	rt.transitionState(ctx, AppStateBuilt, "build completed")
	rt.logDebugInformation(ctx)
	return nil
}

func (p *buildPlan) buildSubApps(ctx context.Context, rt *Runtime) error {
	if p == nil || p.subplans == nil || p.subplans.Len() == 0 {
		return nil
	}

	var buildErr error
	p.subplans.Range(func(_ int, subplan *buildPlan) bool {
		subrt, err := subplan.build(ctx, rt)
		if err != nil {
			buildErr = oops.In("dix").
				With("op", "build_subapp", "app", rt.Name(), "subapp", subplan.spec.meta.Name).
				Wrapf(err, "build subapp failed")
			return false
		}
		rt.subapps.Add(subrt)
		return true
	})
	return buildErr
}
