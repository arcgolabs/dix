package dix

import (
	collectionlist "github.com/arcgolabs/collectionx/list"
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
	return items.Clone()
}

func mergeServiceNameSets(left, right *collectionset.Set[string]) *collectionset.Set[string] {
	return cloneServiceNameSet(left).Merge(right)
}

func mergeValidationReports(left, right ValidationReport) ValidationReport {
	return ValidationReport{
		Errors:        mergeLists(left.Errors, right.Errors),
		Warnings:      mergeLists(left.Warnings, right.Warnings),
		WarningCounts: mergeMultiSets(left.WarningCounts, right.WarningCounts),
		ServiceCounts: mergeMultiSets(left.ServiceCounts, right.ServiceCounts),
	}
}

func mergeLists[T any](left, right *collectionlist.List[T]) *collectionlist.List[T] {
	size := 0
	if left != nil {
		size += left.Len()
	}
	if right != nil {
		size += right.Len()
	}
	return collectionlist.NewListWithCapacity[T](size).Merge(left).Merge(right)
}

func mergeMultiSets[T comparable](left, right *collectionset.MultiSet[T]) *collectionset.MultiSet[T] {
	out := cloneMultiSet(left)
	if right == nil {
		return out
	}
	right.Range(func(item T, count int) bool {
		out.AddN(item, count)
		return true
	})
	return out
}
