package dix

import (
	collectionlist "github.com/arcgolabs/collectionx/list"
	collectionset "github.com/arcgolabs/collectionx/set"
	"github.com/samber/oops"
	"log/slog"
)

func validateTypedGraphReportWithInherited(plan *buildPlan, inherited *collectionset.Set[string]) ValidationReport {
	if plan == nil || plan.modules == nil {
		return ValidationReport{}
	}

	state := newValidationState(
		!declaresProviderOutputType[*slog.Logger](plan),
		!declaresProviderOutputType[AppMeta](plan),
		!declaresProviderOutputType[Profile](plan),
		inherited,
	)
	collectDeclaredOutputs(plan, state)
	validateDeclaredDependencies(plan.modules, state)

	return ValidationReport{
		Errors:        collectionlist.NewListWithCapacity(state.err.Len(), state.err.Values()...),
		Warnings:      collectionlist.NewListWithCapacity(state.warnings.Len(), state.warnings.Values()...),
		WarningCounts: cloneMultiSet(state.warningCounts),
		ServiceCounts: cloneMultiSet(state.serviceCounts),
	}
}

type validationState struct {
	known         *collectionset.Set[string]
	inherited     *collectionset.Set[string]
	err           *collectionlist.List[error]
	warnings      *collectionlist.List[ValidationWarning]
	warningCounts *collectionset.MultiSet[ValidationWarningKind]
	serviceCounts *collectionset.MultiSet[string]
}

func newValidationState(
	includeDefaultLogger bool,
	includeDefaultAppMeta bool,
	includeDefaultProfile bool,
	inherited *collectionset.Set[string],
) *validationState {
	state := &validationState{
		known:         collectionset.NewSetWithCapacity[string](64),
		inherited:     cloneServiceNameSet(inherited),
		err:           collectionlist.NewListWithCapacity[error](4),
		warnings:      collectionlist.NewListWithCapacity[ValidationWarning](2),
		warningCounts: collectionset.NewMultiSet[ValidationWarningKind](),
		serviceCounts: collectionset.NewMultiSet[string](),
	}
	if includeDefaultLogger {
		state.declareService(serviceNameOfSpec[*slog.Logger](nil))
	}
	if includeDefaultAppMeta {
		state.declareService(serviceNameOfSpec[AppMeta](nil))
	}
	if includeDefaultProfile {
		state.declareService(serviceNameOfSpec[Profile](nil))
	}

	return state
}

func collectDeclaredOutputs(plan *buildPlan, state *validationState) {
	if plan == nil {
		return
	}
	collectExplicitOutputs(plan.modules, state)
	collectContributionCollectionOutputs(plan.contributionPlan(), state)
}

func collectExplicitOutputs(modules *collectionlist.List[*moduleSpec], state *validationState) {
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
	if !state.declareService(meta.Output.Name) {
		state.err.Add(oops.In("dix").
			With("op", "validate_provider_output", "module", moduleName, "label", meta.Label, "service", meta.Output.Name).
			Errorf("duplicate provider output `%s` in module `%s` via %s", meta.Output.Name, moduleName, meta.Label))
		return
	}
}

func collectProviderAliases(moduleName string, meta ProviderMetadata, state *validationState) {
	meta.Aliases.Range(func(_ int, alias ServiceRef) bool {
		if !state.declareService(alias.Name) {
			state.err.Add(oops.In("dix").
				With("op", "validate_provider_alias", "module", moduleName, "label", meta.Label, "service", alias.Name).
				Errorf("duplicate provider alias `%s` in module `%s` via %s", alias.Name, moduleName, meta.Label))
			return true
		}
		return true
	})
}

func collectContributionCollectionOutputs(plan contributionPlan, state *validationState) {
	plan.syntheticOutputs().Range(func(_ int, output ServiceRef) bool {
		state.declareService(output.Name)
		return true
	})
}

func collectSetupOutputs(mod *moduleSpec, state *validationState) {
	mod.setups.Range(func(_ int, setup SetupFunc) bool {
		meta := setup.meta
		meta.Provides.Range(func(_ int, provide ServiceRef) bool {
			if !state.declareService(provide.Name) {
				state.err.Add(oops.In("dix").
					With("op", "validate_setup_output", "module", mod.name, "label", meta.Label, "service", provide.Name).
					Errorf("duplicate setup output `%s` in module `%s` via %s", provide.Name, mod.name, meta.Label))
				return true
			}
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

func validateDeclaredDependencies(modules *collectionlist.List[*moduleSpec], state *validationState) {
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
	s.warningCounts.Add(kind)
}

func (s *validationState) declareService(name string) bool {
	if s == nil || name == "" {
		return true
	}
	s.serviceCounts.Add(name)
	if s.known.Contains(name) {
		return false
	}
	s.known.Add(name)
	return true
}

func (s *validationState) validateDeps(moduleName, kind, label string, deps *collectionlist.List[ServiceRef]) {
	validateDependencies(s.err, s, moduleName, kind, label, deps)
}

func validateDependencies(
	err *collectionlist.List[error],
	state *validationState,
	moduleName string,
	kind string,
	label string,
	deps *collectionlist.List[ServiceRef],
) {
	deps.Range(func(_ int, dep ServiceRef) bool {
		if !state.canResolve(dep.Name) {
			hint := state.missingDependencyHint(dep.Name)
			err.Add(oops.In("dix").
				With("op", "validate_dependency", "module", moduleName, "label", label, "dependency", dep.Name, "kind", kind).
				Errorf("missing dependency `%s` for %s %s in module `%s`%s", dep.Name, kind, label, moduleName, hint))
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
