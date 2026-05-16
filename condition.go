package dix

import (
	"context"
	"fmt"
	"os"

	collectionlist "github.com/arcgolabs/collectionx/list"
	collectionset "github.com/arcgolabs/collectionx/set"
	"github.com/samber/oops"
)

// ConditionContext is the build-time context available to provider conditions.
type ConditionContext struct {
	Context context.Context
	Meta    AppMeta
	Profile Profile
	Module  string
	Label   string
	Service string
}

// Condition decides whether a build-time conditional provider is enabled.
type Condition func(ConditionContext) (bool, error)

// When enables a provider only when condition evaluates to true.
//
// Supported input shapes are bool, func() bool, func() (bool, error),
// func(context.Context) bool, func(context.Context) (bool, error),
// func(Profile) bool, func(Profile) (bool, error),
// func(ConditionContext) bool, func(ConditionContext) (bool, error), and Condition.
func When(condition any) ProviderOption {
	normalized := normalizeConditionInput(condition)
	return func(opts *providerOptions) {
		if opts != nil {
			opts.conditions.Add(normalized)
		}
	}
}

// Unless enables a provider only when condition evaluates to false.
func Unless(condition any) ProviderOption {
	normalized := normalizeConditionInput(condition)
	return func(opts *providerOptions) {
		if opts != nil {
			opts.conditions.Add(func(ctx ConditionContext) (bool, error) {
				enabled, err := normalized(ctx)
				if err != nil {
					return false, err
				}
				return !enabled, nil
			})
		}
	}
}

// ProfileIs matches one of the provided build profiles.
func ProfileIs(profiles ...Profile) Condition {
	allowed := collectionset.NewSet(profiles...)
	return func(ctx ConditionContext) (bool, error) {
		return allowed.Contains(ctx.Profile), nil
	}
}

// EnvEquals matches an environment variable value.
func EnvEquals(key, value string) Condition {
	return func(ConditionContext) (bool, error) {
		return os.Getenv(key) == value, nil
	}
}

// EnvSet matches when an environment variable is present and non-empty.
func EnvSet(key string) Condition {
	return func(ConditionContext) (bool, error) {
		return os.Getenv(key) != "", nil
	}
}

func normalizeConditionInput(condition any) Condition {
	if normalized, ok := normalizeSimpleConditionInput(condition); ok {
		return normalized
	}
	if normalized, ok := normalizeContextConditionInput(condition); ok {
		return normalized
	}
	if normalized, ok := normalizeDixConditionInput(condition); ok {
		return normalized
	}
	return unsupportedConditionInput(condition)
}

func normalizeSimpleConditionInput(condition any) (Condition, bool) {
	switch current := condition.(type) {
	case bool:
		return func(ConditionContext) (bool, error) { return current, nil }, true
	case func() bool:
		return func(ConditionContext) (bool, error) { return current(), nil }, true
	case func() (bool, error):
		return func(ConditionContext) (bool, error) { return current() }, true
	default:
		return nil, false
	}
}

func normalizeContextConditionInput(condition any) (Condition, bool) {
	switch current := condition.(type) {
	case func(context.Context) bool:
		return func(ctx ConditionContext) (bool, error) { return current(contextOrBackground(ctx.Context)), nil }, true
	case func(context.Context) (bool, error):
		return func(ctx ConditionContext) (bool, error) { return current(contextOrBackground(ctx.Context)) }, true
	case func(Profile) bool:
		return func(ctx ConditionContext) (bool, error) { return current(ctx.Profile), nil }, true
	case func(Profile) (bool, error):
		return func(ctx ConditionContext) (bool, error) { return current(ctx.Profile) }, true
	default:
		return nil, false
	}
}

func normalizeDixConditionInput(condition any) (Condition, bool) {
	switch current := condition.(type) {
	case Condition:
		return current, true
	case func(ConditionContext) bool:
		return func(ctx ConditionContext) (bool, error) { return current(ctx), nil }, true
	case func(ConditionContext) (bool, error):
		return current, true
	default:
		return nil, false
	}
}

func unsupportedConditionInput(condition any) Condition {
	return func(ConditionContext) (bool, error) {
		return false, oops.In("dix").
			With("op", "normalize_provider_condition", "type", fmt.Sprintf("%T", condition)).
			New("unsupported provider condition")
	}
}

func filterConditionalProviders(
	ctx context.Context,
	spec *appSpec,
	modules *collectionlist.List[*moduleSpec],
	profile Profile,
) (*collectionlist.List[*moduleSpec], error) {
	if modules == nil || modules.Len() == 0 {
		return collectionlist.NewList[*moduleSpec](), nil
	}
	filtered := collectionlist.NewListWithCapacity[*moduleSpec](modules.Len())
	var filterErr error
	modules.Range(func(_ int, mod *moduleSpec) bool {
		var next *moduleSpec
		next, filterErr = filterModuleConditionalProviders(ctx, spec, mod, profile)
		if filterErr != nil {
			return false
		}
		filtered.Add(next)
		return true
	})
	return filtered, filterErr
}

func filterModuleConditionalProviders(
	ctx context.Context,
	spec *appSpec,
	mod *moduleSpec,
	profile Profile,
) (*moduleSpec, error) {
	if mod == nil || mod.providers == nil || mod.providers.Len() == 0 {
		return mod, nil
	}
	providers := collectionlist.NewListWithCapacity[ProviderFunc](mod.providers.Len())
	changed := false
	var conditionErr error
	mod.providers.Range(func(_ int, provider ProviderFunc) bool {
		var enabled bool
		enabled, conditionErr = providerConditionsEnabled(ctx, spec, mod.name, profile, provider)
		if conditionErr != nil {
			return false
		}
		if enabled {
			providers.Add(provider)
		} else {
			changed = true
		}
		return true
	})
	if conditionErr != nil {
		return nil, conditionErr
	}
	if !changed {
		return mod, nil
	}
	cloned := *mod
	cloned.providers = providers
	return &cloned, nil
}

func providerConditionsEnabled(
	ctx context.Context,
	spec *appSpec,
	moduleName string,
	profile Profile,
	provider ProviderFunc,
) (bool, error) {
	conditions := provider.meta.Conditions
	if conditions == nil || conditions.Len() == 0 {
		return true, nil
	}
	conditionCtx := ConditionContext{
		Context: contextOrBackground(ctx),
		Profile: profile,
		Module:  moduleName,
		Label:   provider.meta.Label,
		Service: provider.meta.Output.Name,
	}
	if spec != nil {
		conditionCtx.Meta = spec.meta
	}
	var conditionErr error
	enabled := true
	conditions.Range(func(_ int, condition Condition) bool {
		enabled, conditionErr = condition(conditionCtx)
		return conditionErr == nil && enabled
	})
	if conditionErr != nil {
		return false, oops.In("dix").
			With("op", "evaluate_provider_condition", "module", moduleName, "label", provider.meta.Label).
			Wrapf(conditionErr, "evaluate provider condition failed")
	}
	return enabled, nil
}

func appHasProviderConditions(app *App) bool {
	if app == nil || app.spec == nil {
		return false
	}
	if moduleListHasProviderConditions(app.spec.modules) {
		return true
	}
	hasConditions := false
	app.spec.subapps.Range(func(_ int, subapp *App) bool {
		hasConditions = appHasProviderConditions(subapp)
		return !hasConditions
	})
	return hasConditions
}

func moduleListHasProviderConditions(modules *collectionlist.List[Module]) bool {
	hasConditions := false
	walkModuleSpecs(modules, collectionset.NewSet[*moduleSpec](), func(spec *moduleSpec) {
		if hasConditions || spec == nil || spec.providers == nil {
			return
		}
		spec.providers.Range(func(_ int, provider ProviderFunc) bool {
			hasConditions = providerHasConditions(provider)
			return !hasConditions
		})
	})
	return hasConditions
}

func providerHasConditions(provider ProviderFunc) bool {
	return provider.meta.Conditions != nil && provider.meta.Conditions.Len() > 0
}
