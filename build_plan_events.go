package dix

import (
	"context"
	"errors"
	"time"
)

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

	cleanupErr := rt.cleanupBuildFailure(ctx)
	if cleanupErr == nil {
		return buildErr
	}
	rt.logMessage(ctx, EventLevelError, "build cleanup failed", "app", rt.Name(), "error", cleanupErr)
	return errors.Join(buildErr, cleanupErr)
}

func (p *buildPlan) logBuildStart(ctx context.Context, rt *Runtime, infoEnabled, debugEnabled bool) {
	if infoEnabled {
		rt.logMessage(ctx, EventLevelInfo, "building app", "app", p.spec.meta.Name, "profile", p.profile)
	}
	if debugEnabled {
		stats := countModuleStats(p.modules)
		rt.logMessage(ctx, EventLevelDebug, "build plan ready",
			"app", p.spec.meta.Name,
			"modules", stats.modules,
			"providers", stats.providers,
			"hooks", stats.hooks,
			"setups", stats.setups,
			"invokes", stats.invokes,
		)
	}
}
