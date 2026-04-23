package dix

import (
	"errors"
	"strings"

	"github.com/arcgolabs/collectionx"
	"github.com/samber/do/v2"
)

// StopReport aggregates errors produced while stopping a runtime.
type StopReport struct {
	SubAppError    error
	HookError      error
	ShutdownReport *do.ShutdownReport
}

// HasErrors reports whether the stop report contains any errors.
func (r *StopReport) HasErrors() bool {
	return r != nil && (r.SubAppError != nil || r.HookError != nil || (r.ShutdownReport != nil && len(r.ShutdownReport.Errors) > 0))
}

// Errors returns stop errors as a collectionx list.
func (r *StopReport) Errors() collectionx.List[error] {
	if r == nil {
		return collectionx.NewList[error]()
	}
	errs := collectionx.NewListWithCapacity[error](3)
	if r.SubAppError != nil {
		errs.Add(r.SubAppError)
	}
	if r.HookError != nil {
		errs.Add(r.HookError)
	}
	if r.ShutdownReport != nil && len(r.ShutdownReport.Errors) > 0 {
		errs.Add(r.ShutdownReport)
	}
	return errs
}

// Err returns the combined stop error.
func (r *StopReport) Err() error {
	return errors.Join(r.Errors().Values()...)
}

// Error returns the combined stop error string.
func (r *StopReport) Error() string {
	if !r.HasErrors() {
		return ""
	}
	errs := r.Errors()
	lines := collectionx.MapList(errs, func(_ int, err error) string {
		return err.Error()
	})
	return strings.Join(lines.Values(), "\n")
}
