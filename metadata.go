package dix

import (
	"log/slog"

	"github.com/arcgolabs/collectionx"
	collectionset "github.com/arcgolabs/collectionx/set"
	"github.com/samber/oops"
)

func validateTypedGraphReport(plan *buildPlan) ValidationReport {
	return validateTypedGraphReportWithInherited(plan, nil)
}

func validateBuildPlanTreeReport(plan *buildPlan) ValidationReport {
	return validateBuildPlanTreeReportWithInherited(plan, nil)
}

func validateBuildPlanTreeReportWithInherited(plan *buildPlan, inherited *collectionset.Set[string]) ValidationReport {
	report := validateTypedGraphReportWithInherited(plan, inherited)
	if plan == nil {
		return report
	}

	nextInherited := mergeServiceNameSets(inherited, declaredServiceNames(plan))
	plan.subplans.Range(func(_ int, subplan *buildPlan) bool {
		report = mergeValidationReports(report, validateBuildPlanTreeReportWithInherited(subplan, nextInherited))
		return true
	})
	return report
}

func validateTypedGraphReportWithInherited(plan *buildPlan, inherited *collectionset.Set[string]) ValidationReport {
	if plan == nil || plan.modules == nil {
		return ValidationReport{}
	}

	state := newValidationState(
		!plan.declaresProviderOutput(TypedService[*slog.Logger]()),
		!plan.declaresProviderOutput(TypedService[AppMeta]()),
		!plan.declaresProviderOutput(TypedService[Profile]()),
		inherited,
	)
	collectDeclaredOutputs(plan.modules, state)
	validateDeclaredDependencies(plan.modules, state)

	return ValidationReport{
		Errors:   collectionx.NewListWithCapacity(state.err.Len(), state.err.Values()...),
		Warnings: collectionx.NewListWithCapacity(state.warnings.Len(), state.warnings.Values()...),
	}
}

type validationState struct {
	known     *collectionset.Set[string]
	inherited *collectionset.Set[string]
	err       collectionx.List[error]
	warnings  collectionx.List[ValidationWarning]
}

func newValidationState(
	includeDefaultLogger bool,
	includeDefaultAppMeta bool,
	includeDefaultProfile bool,
	inherited *collectionset.Set[string],
) *validationState {
	known := collectionset.NewSetWithCapacity[string](64)
	if includeDefaultLogger {
		known.Add(serviceNameOf[*slog.Logger]())
	}
	if includeDefaultAppMeta {
		known.Add(serviceNameOf[AppMeta]())
	}
	if includeDefaultProfile {
		known.Add(serviceNameOf[Profile]())
	}

	return &validationState{
		known:     known,
		inherited: cloneServiceNameSet(inherited),
		err:       collectionx.NewListWithCapacity[error](4),
		warnings:  collectionx.NewListWithCapacity[ValidationWarning](2),
	}
}

func declaredServiceNames(plan *buildPlan) *collectionset.Set[string] {
	if plan == nil {
		return collectionset.NewSet[string]()
	}
	state := newValidationState(
		!plan.declaresProviderOutput(TypedService[*slog.Logger]()),
		!plan.declaresProviderOutput(TypedService[AppMeta]()),
		!plan.declaresProviderOutput(TypedService[Profile]()),
		nil,
	)
	collectDeclaredOutputs(plan.modules, state)
	return state.known
}

func cloneServiceNameSet(items *collectionset.Set[string]) *collectionset.Set[string] {
	if items == nil {
		return collectionset.NewSet[string]()
	}
	return collectionset.NewSetWithCapacity[string](items.Len(), items.Values()...)
}

func mergeServiceNameSets(left *collectionset.Set[string], right *collectionset.Set[string]) *collectionset.Set[string] {
	merged := cloneServiceNameSet(left)
	if right != nil {
		merged.Add(right.Values()...)
	}
	return merged
}

func mergeValidationReports(left ValidationReport, right ValidationReport) ValidationReport {
	return ValidationReport{
		Errors:   mergeLists(left.Errors, right.Errors),
		Warnings: mergeLists(left.Warnings, right.Warnings),
	}
}

func mergeLists[T any](left collectionx.List[T], right collectionx.List[T]) collectionx.List[T] {
	size := 0
	if left != nil {
		size += left.Len()
	}
	if right != nil {
		size += right.Len()
	}
	merged := collectionx.NewListWithCapacity[T](size)
	if left != nil {
		merged.Add(left.Values()...)
	}
	if right != nil {
		merged.Add(right.Values()...)
	}
	return merged
}

func collectDeclaredOutputs(modules collectionx.List[*moduleSpec], state *validationState) {
	collectExplicitOutputs(modules, state)
	collectContributionCollectionOutputs(modules, state)
}

func collectExplicitOutputs(modules collectionx.List[*moduleSpec], state *validationState) {
	modules.Range(func(_ int, mod *moduleSpec) bool {
		if mod == nil {
			return true
		}
		collectProviderOutputs(mod, state)
		collectSetupOutputs(mod, state)
		return true
	})
}

func collectProviderOutputs(mod *moduleSpec, state *validationState) {
	mod.providers.Range(func(_ int, provider ProviderFunc) bool {
		meta := provider.meta
		collectProviderOutput(mod.name, meta, state)
		collectProviderAliases(mod.name, meta, state)
		if meta.Output.Name == "" && meta.Raw {
			state.addWarning(
				ValidationWarningRawProviderUndeclaredOutput,
				mod.name,
				meta.Label,
				"raw provider has no declared output; validation cannot model services it registers",
			)
		}
		return true
	})
}

func collectProviderOutput(moduleName string, meta ProviderMetadata, state *validationState) {
	if meta.Output.Name == "" {
		return
	}
	if state.known.Contains(meta.Output.Name) {
		state.err.Add(oops.In("dix").
			With("op", "validate_provider_output", "module", moduleName, "label", meta.Label, "service", meta.Output.Name).
			Errorf("duplicate provider output `%s` in module `%s` via %s", meta.Output.Name, moduleName, meta.Label))
		return
	}
	state.known.Add(meta.Output.Name)
}

func collectProviderAliases(moduleName string, meta ProviderMetadata, state *validationState) {
	meta.Aliases.Range(func(_ int, alias ServiceRef) bool {
		if state.known.Contains(alias.Name) {
			state.err.Add(oops.In("dix").
				With("op", "validate_provider_alias", "module", moduleName, "label", meta.Label, "service", alias.Name).
				Errorf("duplicate provider alias `%s` in module `%s` via %s", alias.Name, moduleName, meta.Label))
			return true
		}
		state.known.Add(alias.Name)
		return true
	})
}

func collectContributionCollectionOutputs(modules collectionx.List[*moduleSpec], state *validationState) {
	newContributionPlan(modules).syntheticOutputs().Range(func(_ int, output ServiceRef) bool {
		if !state.known.Contains(output.Name) {
			state.known.Add(output.Name)
		}
		return true
	})
}

func collectSetupOutputs(mod *moduleSpec, state *validationState) {
	mod.setups.Range(func(_ int, setup SetupFunc) bool {
		meta := setup.meta
		meta.Provides.Range(func(_ int, provide ServiceRef) bool {
			if state.known.Contains(provide.Name) {
				state.err.Add(oops.In("dix").
					With("op", "validate_setup_output", "module", mod.name, "label", meta.Label, "service", provide.Name).
					Errorf("duplicate setup output `%s` in module `%s` via %s", provide.Name, mod.name, meta.Label))
				return true
			}
			state.known.Add(provide.Name)
			return true
		})
		if meta.Raw && meta.Provides.Len() == 0 && meta.Overrides.Len() == 0 && meta.GraphMutation {
			state.addWarning(
				ValidationWarningRawSetupUndeclaredGraph,
				mod.name,
				meta.Label,
				"raw setup mutates the graph without declared provides/overrides; validation cannot model its graph effects",
			)
		}
		return true
	})
}

func validateDeclaredDependencies(modules collectionx.List[*moduleSpec], state *validationState) {
	modules.Range(func(_ int, mod *moduleSpec) bool {
		if mod == nil {
			return true
		}
		validateProviderDependencies(mod, state)
		validateSetupDependencies(mod, state)
		validateInvokeDependencies(mod, state)
		validateHookDependencies(mod, state)
		return true
	})
}

func validateProviderDependencies(mod *moduleSpec, state *validationState) {
	mod.providers.Range(func(_ int, provider ProviderFunc) bool {
		meta := provider.meta
		if meta.Raw && meta.Dependencies.Len() == 0 {
			state.addWarning(
				ValidationWarningRawProviderUndeclaredDeps,
				mod.name,
				meta.Label,
				"raw provider has no declared dependencies; validation cannot verify what it resolves at registration time",
			)
		}
		state.validateDeps(mod.name, "provider", meta.Label, meta.Dependencies)
		return true
	})
}

func validateSetupDependencies(mod *moduleSpec, state *validationState) {
	mod.setups.Range(func(_ int, setup SetupFunc) bool {
		meta := setup.meta
		meta.Overrides.Range(func(_ int, override ServiceRef) bool {
			if !state.known.Contains(override.Name) {
				state.err.Add(oops.In("dix").
					With("op", "validate_setup_override", "module", mod.name, "label", meta.Label, "service", override.Name).
					Errorf("override target `%s` not found in module `%s` via %s", override.Name, mod.name, meta.Label))
			}
			return true
		})
		state.validateDeps(mod.name, "setup", meta.Label, meta.Dependencies)
		return true
	})
}

func validateInvokeDependencies(mod *moduleSpec, state *validationState) {
	mod.invokes.Range(func(_ int, invoke InvokeFunc) bool {
		if invoke.meta.Raw && invoke.meta.Dependencies.Len() == 0 {
			state.addWarning(
				ValidationWarningRawInvokeUndeclaredDeps,
				mod.name,
				invoke.meta.Label,
				"raw invoke has no declared dependencies; validation cannot verify what it resolves",
			)
			return true
		}
		state.validateDeps(mod.name, "invoke", invoke.meta.Label, invoke.meta.Dependencies)
		return true
	})
}

func validateHookDependencies(mod *moduleSpec, state *validationState) {
	mod.hooks.Range(func(_ int, hook HookFunc) bool {
		if hook.meta.Raw && hook.meta.Dependencies.Len() == 0 {
			state.addWarning(
				ValidationWarningRawHookUndeclaredDeps,
				mod.name,
				hook.meta.Label,
				"raw hook has no declared dependencies; validation cannot verify what it resolves during lifecycle execution",
			)
			return true
		}
		state.validateDeps(mod.name, string(hook.meta.Kind)+" hook", hook.meta.Label, hook.meta.Dependencies)
		return true
	})
}

func (s *validationState) addWarning(kind ValidationWarningKind, moduleName, label, details string) {
	s.warnings.Add(ValidationWarning{
		Kind:    kind,
		Module:  moduleName,
		Label:   label,
		Details: details,
	})
}

func (s *validationState) validateDeps(moduleName, kind, label string, deps collectionx.List[ServiceRef]) {
	validateDependencies(s.err, s, moduleName, kind, label, deps)
}

func validateDependencies(
	err collectionx.List[error],
	state *validationState,
	moduleName string,
	kind string,
	label string,
	deps collectionx.List[ServiceRef],
) {
	deps.Range(func(_ int, dep ServiceRef) bool {
		if !state.canResolve(dep.Name) {
			err.Add(oops.In("dix").
				With("op", "validate_dependency", "module", moduleName, "label", label, "dependency", dep.Name, "kind", kind).
				Errorf("missing dependency `%s` for %s %s in module `%s`", dep.Name, kind, label, moduleName))
		}
		return true
	})
}

func (s *validationState) canResolve(name string) bool {
	if s == nil || name == "" {
		return false
	}
	return s.known.Contains(name) || (s.inherited != nil && s.inherited.Contains(name))
}
