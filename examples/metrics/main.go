// Package main demonstrates exposing dix runtime metrics through an external Prometheus handler.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	promobs "github.com/DaiYuANg/arcgo/observabilityx/prometheus"
	"github.com/arcgolabs/dix"
	dixmetrics "github.com/arcgolabs/dix/metrics"
)

func main() {
	prom := promobs.New(promobs.WithNamespace("arcgo"))

	app := dix.New(
		"dix-metrics",
		dix.Version("0.0.1"),
		dixmetrics.WithObservability(
			prom,
			dixmetrics.WithMetricPrefix("dix_runtime"),
		),
		dix.WithModule(
			dix.NewModule("checks",
				dix.Setups(dix.SetupContainer(func(c *dix.Container) error {
					c.RegisterHealthCheck("database", func(context.Context) error { return nil })
					c.RegisterReadinessCheck("cache", func(context.Context) error { return nil })
					c.RegisterLivenessCheck("process", func(context.Context) error { return nil })
					return nil
				})),
				dix.Hooks(
					dix.OnStartFunc(func() error { return nil }),
					dix.OnStopFunc(func() error { return nil }),
				),
			),
		),
	)

	rt, err := app.Start(context.Background())
	if err != nil {
		panic(err)
	}
	defer stopOrPanic(rt)

	_ = rt.CheckHealth(context.Background())
	_ = rt.CheckReadiness(context.Background())
	_ = rt.CheckLiveness(context.Background())

	mux := http.NewServeMux()
	mux.Handle("/metrics", prom.Handler())

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/metrics", http.NoBody)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)

	body := resp.Body.String()
	printLine("registered external route: GET /metrics")
	printLine("sample dix metrics:")
	for _, metricName := range []string{
		"arcgo_dix_runtime_build_total",
		"arcgo_dix_runtime_start_total",
		"arcgo_dix_runtime_health_check_total",
		"arcgo_dix_runtime_state_transition_total",
	} {
		printMetricLine(body, metricName)
	}
}

func stopOrPanic(rt *dix.Runtime) {
	if err := rt.Stop(context.Background()); err != nil {
		panic(err)
	}
}

func printMetricLine(body, metricName string) {
	for line := range strings.SplitSeq(body, "\n") {
		if strings.HasPrefix(line, metricName) {
			printLine(line)
			return
		}
	}
	panic(errors.New("metric not found: " + metricName))
}

func printLine(value any) {
	if _, err := fmt.Println(value); err != nil {
		panic(err)
	}
}
