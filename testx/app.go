package testx

import (
	"context"
	"testing"

	"github.com/arcgolabs/dix"
)

// Validate validates app and fails the test on validation errors.
func Validate(tb testing.TB, app *dix.App) dix.ValidationReport {
	tb.Helper()
	report := app.ValidateReport()
	if err := report.Err(); err != nil {
		tb.Fatalf("dix validation failed: %v", err)
	}
	return report
}

// Build builds app and fails the test on build errors.
func Build(tb testing.TB, app *dix.App) *dix.Runtime {
	tb.Helper()
	runtime, err := app.Build()
	if err != nil {
		tb.Fatalf("dix build failed: %v", err)
	}
	return runtime
}

// Start starts app and registers test cleanup that stops the runtime.
func Start(ctx context.Context, tb testing.TB, app *dix.App) *dix.Runtime {
	tb.Helper()
	runtime, err := app.Start(ctx)
	if err != nil {
		tb.Fatalf("dix start failed: %v", err)
	}
	tb.Cleanup(func() {
		stopCtx := context.WithoutCancel(contextOrBackground(ctx))
		if err := runtime.Stop(stopCtx); err != nil {
			tb.Fatalf("dix stop failed: %v", err)
		}
	})
	return runtime
}

func contextOrBackground(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
