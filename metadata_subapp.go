package dix

import (
	"github.com/arcgolabs/collectionx"
	collectionset "github.com/arcgolabs/collectionx/set"
)

func validateBuildPlanTreeReport(plan *buildPlan) ValidationReport {
	report := validateTypedGraphReportWithInherited(plan, inheritedServicesForValidation(plan))
	if plan == nil {
		return report
	}

	plan.subplans.Range(func(_ int, subplan *buildPlan) bool {
		report = mergeValidationReports(report, validateBuildPlanTreeReport(subplan))
		return true
	})
	return report
}

func inheritedServicesForValidation(plan *buildPlan) *collectionset.Set[string] {
	if plan == nil {
		return nil
	}
	return plan.inheritedServices
}

func cloneServiceNameSet(items *collectionset.Set[string]) *collectionset.Set[string] {
	if items == nil {
		return collectionset.NewSet[string]()
	}
	return collectionset.NewSetWithCapacity[string](items.Len(), items.Values()...)
}

func mergeServiceNameSets(left, right *collectionset.Set[string]) *collectionset.Set[string] {
	merged := cloneServiceNameSet(left)
	if right != nil {
		merged.Add(right.Values()...)
	}
	return merged
}

func mergeValidationReports(left, right ValidationReport) ValidationReport {
	return ValidationReport{
		Errors:   mergeLists(left.Errors, right.Errors),
		Warnings: mergeLists(left.Warnings, right.Warnings),
	}
}

func mergeLists[T any](left, right collectionx.List[T]) collectionx.List[T] {
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
