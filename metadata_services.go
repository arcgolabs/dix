package dix

import (
	"log/slog"

	collectionset "github.com/arcgolabs/collectionx/set"
)

func declaredServiceNames(plan *buildPlan) *collectionset.Set[string] {
	if plan == nil {
		return collectionset.NewSet[string]()
	}
	names := explicitServiceOutputs(plan.modules).values.Clone()
	providerOutputs := plan.providerOutputs
	if providerOutputs == nil {
		providerOutputs = providerOutputNames(plan.modules)
	}
	loggerName := serviceNameOfSpec[*slog.Logger](plan.spec)
	appMetaName := serviceNameOfSpec[AppMeta](plan.spec)
	profileName := serviceNameOfSpec[Profile](plan.spec)
	if !providerOutputs.Contains(loggerName) {
		names.Add(loggerName)
	}
	if !providerOutputs.Contains(appMetaName) {
		names.Add(appMetaName)
	}
	if !providerOutputs.Contains(profileName) {
		names.Add(profileName)
	}
	plan.contributionPlan().syntheticOutputs().Range(func(_ int, output ServiceRef) bool {
		if output.Name != "" {
			names.Add(output.Name)
		}
		return true
	})
	return names
}
