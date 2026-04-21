// Package metrics exposes dix internal lifecycle metrics through the shared
// observabilityx facade.
//
// The package is exporter-agnostic:
//   - pass observabilityx/prometheus.Adapter to emit Prometheus metrics and
//     mount the adapter Handler on your existing HTTP server
//   - pass observabilityx/otel.New(...) to emit OpenTelemetry metrics into
//     your existing OTel pipeline
//
// Example:
//
//	promObs := prometheus.New(prometheus.WithNamespace("arcgo"))
//	app := dix.New("my-app",
//		dixmetrics.WithObservability(promObs),
//	)
//	http.Handle("/metrics", promObs.Handler())
package metrics
