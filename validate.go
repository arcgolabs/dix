package dix

import (
	"context"
	"errors"

	collectionlist "github.com/arcgolabs/collectionx/list"
	collectionset "github.com/arcgolabs/collectionx/set"
)

// Validate validates the immutable app spec and current module graph.
func (a *App) Validate() error {
	return a.ValidateContext(context.Background())
}

// ValidateReport validates the app and returns the full validation report.
func (a *App) ValidateReport() ValidationReport {
	return a.ValidateReportContext(context.Background())
}

// ValidateContext validates the immutable app spec and current module graph using the provided context.
func (a *App) ValidateContext(ctx context.Context) error {
	return a.ValidateReportContext(ctx).Err()
}

// ValidateReportContext validates the app and returns the full validation report using the provided context.
func (a *App) ValidateReportContext(ctx context.Context) ValidationReport {
	_, report, err := a.cachedBuildPlan(ctx)
	if err != nil && (report.Errors == nil || report.Errors.IsEmpty()) {
		report.Errors = collectionlist.NewList(err)
	}
	return report
}

func cloneValidationReport(report ValidationReport) ValidationReport {
	return ValidationReport{
		Errors:        cloneList(report.Errors),
		Warnings:      cloneList(report.Warnings),
		WarningCounts: cloneMultiSet(report.WarningCounts),
		ServiceCounts: cloneMultiSet(report.ServiceCounts),
	}
}

func cloneList[T any](items *collectionlist.List[T]) *collectionlist.List[T] {
	if items == nil {
		return collectionlist.NewList[T]()
	}
	return items.Clone()
}

func cloneMultiSet[T comparable](items *collectionset.MultiSet[T]) *collectionset.MultiSet[T] {
	out := collectionset.NewMultiSet[T]()
	if items == nil {
		return out
	}
	items.Range(func(item T, count int) bool {
		out.AddN(item, count)
		return true
	})
	return out
}

// HasWarnings reports whether the validation report contains warnings.
func (r ValidationReport) HasWarnings() bool {
	return r.Warnings != nil && r.Warnings.Len() > 0
}

// HasErrors reports whether the validation report contains errors.
func (r ValidationReport) HasErrors() bool {
	return r.Errors != nil && r.Errors.Len() > 0
}

// WarningKindCounts returns validation warning counts by kind.
func (r ValidationReport) WarningKindCounts() *collectionset.MultiSet[ValidationWarningKind] {
	if r.WarningCounts != nil {
		return cloneMultiSet(r.WarningCounts)
	}
	counts := collectionset.NewMultiSet[ValidationWarningKind]()
	if r.Warnings != nil {
		r.Warnings.Range(func(_ int, warning ValidationWarning) bool {
			counts.Add(warning.Kind)
			return true
		})
	}
	return counts
}

// DeclaredServiceCounts returns service declaration counts collected during validation.
func (r ValidationReport) DeclaredServiceCounts() *collectionset.MultiSet[string] {
	return cloneMultiSet(r.ServiceCounts)
}

// Err returns the combined validation error.
func (r ValidationReport) Err() error {
	if r.Errors == nil {
		return nil
	}
	return errors.Join(r.Errors.Values()...)
}

// WarningSummary renders the validation warnings as a newline-delimited summary.
func (r ValidationReport) WarningSummary() string {
	if r.Warnings == nil || r.Warnings.Len() == 0 {
		return ""
	}

	return r.Warnings.Join("\n", func(_ int, warning ValidationWarning) string {
		line := string(warning.Kind)
		if warning.Module != "" {
			line += " module=" + warning.Module
		}
		if warning.Label != "" {
			line += " label=" + warning.Label
		}
		if warning.Details != "" {
			line += " " + warning.Details
		}
		return line
	})
}
