---
title: 'dix Metrics And Observability'
linkTitle: 'metrics-and-observability'
description: 'Use dix/metrics to expose internal runtime events through Prometheus or OpenTelemetry'
weight: 4
---

## Metrics And Observability

`dix` itself does not embed an exporter, an HTTP server, or an OTel pipeline.  
The new standalone subpackage `github.com/arcgolabs/dix/metrics` only translates internal `dix` runtime events into `observabilityx` metrics:

- build
- start
- stop
- health check
- state transition

That means:

- if you already have a Prometheus backend, connect `dix/metrics` to `observabilityx/prometheus`
- if you already have an OTel meter, connect `dix/metrics` to `observabilityx/otel`
- `/metrics`, OTLP exporters, collectors, and HTTP routing stay owned by your outer application

## Install

```bash
go get github.com/arcgolabs/dix@latest
go get github.com/arcgolabs/dix/metrics@latest
go get github.com/arcgolabs/observabilityx/prometheus@latest
go get github.com/arcgolabs/observabilityx/otel@latest
```

## Minimal Prometheus wiring

In the example below, `dix/metrics` only emits metrics into `promobs.Adapter`; your own HTTP server still owns `/metrics`.

```go
package main

import (
	"context"
	"net/http"

	"github.com/arcgolabs/dix"
	dixmetrics "github.com/arcgolabs/dix/metrics"
	promobs "github.com/arcgolabs/observabilityx/prometheus"
)

func main() {
	prom := promobs.New(promobs.WithNamespace("arcgo"))

	app := dix.New(
		"orders",
		dixmetrics.WithObservability(prom),
	)

	http.Handle("/metrics", prom.Handler())

	rt, err := app.Start(context.Background())
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = rt.Stop(context.Background())
	}()
}
```

If you already use `httpx`, `chi`, `gin`, `echo`, or `fiber`, the model is the same: mount `prom.Handler()` into the existing server instead of asking `dix` to create a dedicated metrics server.

## Minimal OpenTelemetry wiring

If your process already owns OTel exporter / SDK setup, wrap the meter with `observabilityx/otel` and attach it through `dix/metrics`:

```go
package main

import (
	"context"

	"github.com/arcgolabs/dix"
	dixmetrics "github.com/arcgolabs/dix/metrics"
	otelobs "github.com/arcgolabs/observabilityx/otel"
)

func main() {
	obs := otelobs.New()

	app := dix.New(
		"orders",
		dixmetrics.WithObservability(obs),
	)

	rt, err := app.Start(context.Background())
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = rt.Stop(context.Background())
	}()
}
```

If you want the lower-level hook point, you can bypass the convenience helper and attach the observer directly:

```go
app := dix.New(
	"orders",
	dix.WithObserver(dixmetrics.NewObserver(obs)),
)
```

## Default metric names

The default prefix is `dix_`.

- `dix_build_total`
- `dix_build_duration_ms`
- `dix_build_modules`
- `dix_build_providers`
- `dix_build_hooks`
- `dix_build_setups`
- `dix_build_invokes`
- `dix_start_total`
- `dix_start_duration_ms`
- `dix_start_registered_hooks`
- `dix_start_completed_hooks`
- `dix_start_rollback_total`
- `dix_stop_total`
- `dix_stop_duration_ms`
- `dix_stop_registered_hooks`
- `dix_stop_shutdown_errors`
- `dix_stop_hook_error_total`
- `dix_health_check_total`
- `dix_health_check_duration_ms`
- `dix_state_transition_total`

## Default labels

Common labels:

- `app`
- `profile`
- `version`
- `result`

Extra labels:

- health checks: `kind`, optional `check`
- state transitions: `from`, `to`

## Customizing prefix and labels

`dix/metrics` exposes a few common knobs:

- `dixmetrics.WithMetricPrefix("arc_dix")`
- `dixmetrics.WithVersionAttribute(false)`
- `dixmetrics.WithHealthCheckNameAttribute(false)`

Example:

```go
app := dix.New(
	"orders",
	dixmetrics.WithObservability(
		prom,
		dixmetrics.WithMetricPrefix("arc_dix"),
		dixmetrics.WithHealthCheckNameAttribute(false),
	),
)
```

## Design boundary

- `dix` core only emits observer events and does not depend on a specific metrics backend
- `dix/metrics` only translates those events into metrics
- the Prometheus `/metrics` handler still comes from `observabilityx/prometheus`
- OTel exporter and SDK initialization still belong to your application

If you need a different shape or finer-grained runtime telemetry, implement your own `dix.Observer` and attach it with `dix.WithObserver(...)`.

## Next

- `observabilityx` overview: [observabilityx](../observabilityx)
- Prometheus `/metrics` handler: [observabilityx Prometheus metrics endpoint](../observabilityx/prometheus-metrics)
- OTel backend: [observabilityx OpenTelemetry backend](../observabilityx/otel-backend)
