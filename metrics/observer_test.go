package metrics_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/DaiYuANg/arcgo/observabilityx"
	"github.com/arcgolabs/arcgo/dix"
	dixmetrics "github.com/arcgolabs/arcgo/dix/metrics"
)

type testObservability struct {
	counterCalls []metricCall
	histCalls    []metricCall
}

type metricCall struct {
	name  string
	value float64
	attrs map[string]any
}

func (t *testObservability) Logger() *slog.Logger {
	return slog.Default()
}

func (t *testObservability) StartSpan(ctx context.Context, _ string, _ ...observabilityx.Attribute) (context.Context, observabilityx.Span) {
	return ctx, testSpan{}
}

func (t *testObservability) Counter(spec observabilityx.CounterSpec) observabilityx.Counter {
	spec = observabilityx.NormalizeCounterSpec(spec)
	return testCounter{parent: t, name: spec.Name}
}

func (t *testObservability) UpDownCounter(spec observabilityx.UpDownCounterSpec) observabilityx.UpDownCounter {
	spec = observabilityx.NormalizeUpDownCounterSpec(spec)
	return testCounter{parent: t, name: spec.Name}
}

func (t *testObservability) Histogram(spec observabilityx.HistogramSpec) observabilityx.Histogram {
	spec = observabilityx.NormalizeHistogramSpec(spec)
	return testHistogram{parent: t, name: spec.Name}
}

func (t *testObservability) Gauge(spec observabilityx.GaugeSpec) observabilityx.Gauge {
	spec = observabilityx.NormalizeGaugeSpec(spec)
	return testGauge{parent: t, name: spec.Name}
}

type testCounter struct {
	parent *testObservability
	name   string
}

func (t testCounter) Add(_ context.Context, value int64, attrs ...observabilityx.Attribute) {
	t.parent.counterCalls = append(t.parent.counterCalls, metricCall{
		name:  t.name,
		value: float64(value),
		attrs: toAttrMap(attrs),
	})
}

type testHistogram struct {
	parent *testObservability
	name   string
}

func (t testHistogram) Record(_ context.Context, value float64, attrs ...observabilityx.Attribute) {
	t.parent.histCalls = append(t.parent.histCalls, metricCall{
		name:  t.name,
		value: value,
		attrs: toAttrMap(attrs),
	})
}

type testGauge struct {
	parent *testObservability
	name   string
}

func (t testGauge) Set(_ context.Context, value float64, attrs ...observabilityx.Attribute) {
	t.parent.histCalls = append(t.parent.histCalls, metricCall{
		name:  t.name,
		value: value,
		attrs: toAttrMap(attrs),
	})
}

type testSpan struct{}

func (testSpan) End()                                      {}
func (testSpan) RecordError(error)                         {}
func (testSpan) SetAttributes(...observabilityx.Attribute) {}

func TestObserverEmitsBuildAndStateMetrics(t *testing.T) {
	obs := &testObservability{}
	observer := dixmetrics.NewObserver(obs, dixmetrics.WithMetricPrefix("arc_dix"))

	observer.OnBuild(context.Background(), dix.BuildEvent{
		Meta:          dix.AppMeta{Name: "orders", Version: "1.2.3"},
		Profile:       dix.ProfileProd,
		ModuleCount:   2,
		ProviderCount: 5,
		HookCount:     3,
		SetupCount:    1,
		InvokeCount:   1,
		Err:           errors.New("boom"),
	})
	observer.OnStateTransition(context.Background(), dix.StateTransitionEvent{
		Meta:    dix.AppMeta{Name: "orders", Version: "1.2.3"},
		Profile: dix.ProfileProd,
		From:    dix.AppStateBuilt,
		To:      dix.AppStateStarting,
	})

	assertCounterCall(t, obs.counterCalls, "arc_dix_build_total", map[string]any{
		"app":     "orders",
		"profile": string(dix.ProfileProd),
		"version": "1.2.3",
		"result":  "error",
	})
	assertHistogramCall(t, obs.histCalls, "arc_dix_build_modules", map[string]any{
		"app":     "orders",
		"profile": string(dix.ProfileProd),
		"version": "1.2.3",
		"result":  "error",
	}, 2)
	assertCounterCall(t, obs.counterCalls, "arc_dix_state_transition_total", map[string]any{
		"app":     "orders",
		"profile": string(dix.ProfileProd),
		"version": "1.2.3",
		"from":    dix.AppStateBuilt.String(),
		"to":      dix.AppStateStarting.String(),
	})
}

func TestWithObservabilityAttachesToDixApp(t *testing.T) {
	obs := &testObservability{}
	app := dix.New("metrics-app",
		dixmetrics.WithObservability(obs, dixmetrics.WithHealthCheckNameAttribute(false)),
		dix.WithModule(
			dix.NewModule("health",
				dix.Setups(dix.Setup(func(c *dix.Container, _ dix.Lifecycle) error {
					c.RegisterHealthCheck("db", func(context.Context) error { return nil })
					return nil
				})),
				dix.Hooks(
					dix.OnStart0(func(context.Context) error { return nil }),
					dix.OnStop0(func(context.Context) error { return nil }),
				),
			),
		),
	)

	rt, err := app.Build()
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if err := rt.Start(context.Background()); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	rt.CheckHealth(context.Background())
	if err := rt.Stop(context.Background()); err != nil {
		t.Fatalf("stop failed: %v", err)
	}

	assertCounterCall(t, obs.counterCalls, "dix_build_total", map[string]any{
		"app":     "metrics-app",
		"profile": string(dix.ProfileDefault),
		"result":  "ok",
	})
	assertCounterCall(t, obs.counterCalls, "dix_start_total", map[string]any{
		"app":     "metrics-app",
		"profile": string(dix.ProfileDefault),
		"result":  "ok",
	})
	assertCounterCall(t, obs.counterCalls, "dix_stop_total", map[string]any{
		"app":     "metrics-app",
		"profile": string(dix.ProfileDefault),
		"result":  "ok",
	})
	assertCounterCall(t, obs.counterCalls, "dix_health_check_total", map[string]any{
		"app":     "metrics-app",
		"profile": string(dix.ProfileDefault),
		"kind":    string(dix.HealthKindGeneral),
		"result":  "ok",
	})
	healthCounter := findCounterCall(t, obs.counterCalls, "dix_health_check_total")
	if _, exists := healthCounter.attrs["check"]; exists {
		t.Fatal("expected health check name attribute to be disabled")
	}
}

func toAttrMap(attrs []observabilityx.Attribute) map[string]any {
	m := make(map[string]any, len(attrs))
	for _, attr := range attrs {
		m[attr.Key] = attr.Value
	}
	return m
}

func findCounterCall(t *testing.T, calls []metricCall, name string) metricCall {
	t.Helper()
	for _, call := range calls {
		if call.name == name {
			return call
		}
	}
	t.Fatalf("expected counter call %q", name)
	return metricCall{}
}

func assertCounterCall(t *testing.T, calls []metricCall, name string, wantAttrs map[string]any) {
	t.Helper()
	call := findCounterCall(t, calls, name)
	for key, want := range wantAttrs {
		if got := call.attrs[key]; got != want {
			t.Fatalf("counter %q expected attr %q=%v, got %v", name, key, want, got)
		}
	}
}

func assertHistogramCall(t *testing.T, calls []metricCall, name string, wantAttrs map[string]any, wantValue float64) {
	t.Helper()
	for _, call := range calls {
		if call.name != name {
			continue
		}
		if call.value != wantValue {
			continue
		}
		for key, want := range wantAttrs {
			if got := call.attrs[key]; got != want {
				t.Fatalf("histogram %q expected attr %q=%v, got %v", name, key, want, got)
			}
		}
		return
	}
	t.Fatalf("expected histogram call %q with value %v", name, wantValue)
}
