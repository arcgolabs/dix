package dix

import (
	"context"
	"time"

	"github.com/samber/oops"
)

func (p *buildPlan) runEagerProviders(ctx context.Context, rt *Runtime, debugEnabled bool) error {
	var buildErr error
	p.modules.Range(func(_ int, mod *moduleSpec) bool {
		buildErr = runModuleEagerProviders(ctx, mod, rt, debugEnabled)
		return buildErr == nil
	})
	return buildErr
}

func runModuleEagerProviders(ctx context.Context, mod *moduleSpec, rt *Runtime, debugEnabled bool) error {
	var eagerErr error
	mod.providers.Range(func(_ int, provider ProviderFunc) bool {
		if !provider.meta.Eager {
			return true
		}
		eagerErr = runEagerProvider(ctx, mod, rt, provider, debugEnabled)
		return eagerErr == nil
	})
	return eagerErr
}

func runEagerProvider(ctx context.Context, mod *moduleSpec, rt *Runtime, provider ProviderFunc, debugEnabled bool) error {
	service := provider.meta.Output.Name
	if service == "" {
		return oops.In("dix").
			With("op", "eager_provider", "module", mod.name, "label", provider.meta.Label).
			New("eager provider requires a declared output")
	}
	if err := contextOrBackground(ctx).Err(); err != nil {
		return oops.In("dix").
			With("op", "eager_provider", "module", mod.name, "label", provider.meta.Label, "service", service).
			Wrapf(err, "eager provider context canceled")
	}

	startedAt := time.Now()
	if debugEnabled {
		rt.logMessage(ctx, EventLevelDebug, "warming eager provider",
			"module", mod.name,
			"label", provider.meta.Label,
			"service", service,
			"dependencies", serviceRefNames(provider.meta.Dependencies),
			"raw", provider.meta.Raw,
		)
	}
	_, err := resolveNamedAny(rt.container.Raw(), service)
	duration := time.Since(startedAt)
	rt.container.logServiceResolution(ctx, service, "eager", duration, err)
	rt.emitProvider(ctx, ProviderEvent{
		Meta:      rt.Meta(),
		Profile:   rt.Profile(),
		Module:    mod.name,
		Label:     provider.meta.Label,
		Service:   service,
		Operation: "eager",
		Duration:  duration,
		Err:       err,
	})
	if err != nil {
		rt.logMessage(ctx, EventLevelError, "eager provider failed",
			"module", mod.name,
			"label", provider.meta.Label,
			"service", service,
			"duration", duration,
			"error", err,
		)
		return oops.In("dix").
			With("op", "eager_provider", "module", mod.name, "label", provider.meta.Label, "service", service).
			Wrapf(err, "eager provider failed in module %s via %s", mod.name, provider.meta.Label)
	}
	if debugEnabled {
		rt.logMessage(ctx, EventLevelDebug, "eager provider warmed",
			"module", mod.name,
			"label", provider.meta.Label,
			"service", service,
			"duration", duration,
		)
	}
	return nil
}

func runModuleSetups(ctx context.Context, mod *moduleSpec, rt *Runtime, debugEnabled bool) error {
	var setupErr error
	mod.setups.Range(func(_ int, setup SetupFunc) bool {
		startedAt := time.Now()
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
			rt.logMessage(ctx, EventLevelDebug, "module setup completed",
				"module", mod.name,
				"label", setup.meta.Label,
				"duration", time.Since(startedAt),
			)
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
		startedAt := time.Now()
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
			rt.logMessage(ctx, EventLevelDebug, "invoke completed",
				"module", mod.name,
				"label", invoke.meta.Label,
				"duration", time.Since(startedAt),
			)
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
