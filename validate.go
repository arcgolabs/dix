package dix

import (
	"context"
	"errors"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
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
		report.Errors = collectionx.NewList(err)
	}
	return report
}

func cloneValidationReport(report ValidationReport) ValidationReport {
	return ValidationReport{
		Errors:   cloneList(report.Errors),
		Warnings: cloneList(report.Warnings),
	}
}

func cloneList[T any](items collectionx.List[T]) collectionx.List[T] {
	if items == nil || items.Len() == 0 {
		return collectionx.NewList[T]()
	}
	return collectionx.NewListWithCapacity(items.Len(), items.Values()...)
}

// HasWarnings reports whether the validation report contains warnings.
func (r ValidationReport) HasWarnings() bool {
	return r.Warnings != nil && r.Warnings.Len() > 0
}

// HasErrors reports whether the validation report contains errors.
func (r ValidationReport) HasErrors() bool {
	return r.Errors != nil && r.Errors.Len() > 0
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

	lines := collectionx.MapList(r.Warnings, func(_ int, warning ValidationWarning) string {
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
	return strings.Join(lines.Values(), "\n")
}
