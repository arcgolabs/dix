package dix

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/arcgolabs/collectionx"
	"github.com/samber/oops"
)

// HealthKind is the category of a health check.
type HealthKind string

const (
	// HealthKindGeneral identifies general health checks.
	HealthKindGeneral HealthKind = "general"
	// HealthKindLiveness identifies liveness health checks.
	HealthKindLiveness HealthKind = "liveness"
	// HealthKindReadiness identifies readiness health checks.
	HealthKindReadiness HealthKind = "readiness"
)

// HealthCheckFunc is a framework-level health check.
type HealthCheckFunc func(context.Context) error

type healthCheckEntry struct {
	name string
	kind HealthKind
	fn   HealthCheckFunc
}

// RegisterHealthCheck registers a framework-managed general health check.
func (c *Container) RegisterHealthCheck(name string, fn HealthCheckFunc) {
	c.RegisterHealthCheckOfKind(HealthKindGeneral, name, fn)
}

// RegisterLivenessCheck registers a liveness health check.
func (c *Container) RegisterLivenessCheck(name string, fn HealthCheckFunc) {
	c.RegisterHealthCheckOfKind(HealthKindLiveness, name, fn)
}

// RegisterReadinessCheck registers a readiness health check.
func (c *Container) RegisterReadinessCheck(name string, fn HealthCheckFunc) {
	c.RegisterHealthCheckOfKind(HealthKindReadiness, name, fn)
}

// RegisterHealthCheckOfKind registers a categorized health check.
func (c *Container) RegisterHealthCheckOfKind(kind HealthKind, name string, fn HealthCheckFunc) {
	if c == nil || fn == nil {
		return
	}
	if kind == "" {
		kind = HealthKindGeneral
	}
	c.healthChecks.Add(healthCheckEntry{name: name, kind: kind, fn: fn})
}

// HealthReport describes the current health status.
type HealthReport struct {
	Kind   HealthKind                     `json:"kind"`
	Checks collectionx.Map[string, error] `json:"-"`
}

// Healthy reports whether all checks passed.
func (r HealthReport) Healthy() bool {
	if r.Checks == nil || r.Checks.Len() == 0 {
		return true
	}
	return r.Checks.AllEntryMatch(func(_ string, err error) bool { return err == nil })
}

// Error returns a combined error when one or more checks fail.
func (r HealthReport) Error() error {
	if r.Healthy() {
		return nil
	}

	parts := collectionx.NewList[string]()
	if r.Checks != nil {
		r.Checks.Range(func(name string, err error) bool {
			if err != nil {
				parts.Add(fmt.Sprintf("%s: %v", name, err))
			}
			return true
		})
	}
	items := parts.Values()
	sort.Strings(items)
	return oops.In("dix").
		With("op", "health_report", "kind", r.Kind, "failed_checks", len(items)).
		Errorf("health check failed: %s", strings.Join(items, "; "))
}

// MarshalJSON renders a user-friendly JSON payload for HTTP endpoints.
func (r HealthReport) MarshalJSON() ([]byte, error) {
	type payload struct {
		Kind    HealthKind         `json:"kind"`
		Healthy bool               `json:"healthy"`
		Checks  map[string]*string `json:"checks"`
	}

	checksLen := 0
	if r.Checks != nil {
		checksLen = r.Checks.Len()
	}
	checks := collectionx.NewMapWithCapacity[string, *string](checksLen)
	if r.Checks != nil {
		r.Checks.Range(func(name string, err error) bool {
			if err == nil {
				checks.Set(name, nil)
				return true
			}
			message := err.Error()
			checks.Set(name, &message)
			return true
		})
	}

	data, err := json.Marshal(payload{Kind: r.Kind, Healthy: r.Healthy(), Checks: checks.All()})
	if err != nil {
		return nil, oops.In("dix").
			With("op", "marshal_health_report", "kind", r.Kind).
			Wrapf(err, "marshal health report")
	}
	return data, nil
}

// CheckHealth executes all general health checks.
func (r *Runtime) CheckHealth(ctx context.Context) HealthReport {
	return r.checkHealthByKind(ctx, HealthKindGeneral)
}

// CheckLiveness executes all liveness checks.
func (r *Runtime) CheckLiveness(ctx context.Context) HealthReport {
	return r.checkHealthByKind(ctx, HealthKindLiveness)
}

// CheckReadiness executes all readiness checks.
func (r *Runtime) CheckReadiness(ctx context.Context) HealthReport {
	return r.checkHealthByKind(ctx, HealthKindReadiness)
}

func (r *Runtime) checkHealthByKind(ctx context.Context, kind HealthKind) HealthReport {
	report := HealthReport{Kind: kind, Checks: collectionx.NewMap[string, error]()}
	if r == nil || r.container == nil {
		return report
	}

	reportChecks := collectionx.NewMapWithCapacity[string, error](r.container.healthChecks.Len())
	r.container.healthChecks.Range(func(_ int, check healthCheckEntry) bool {
		if check.kind != kind {
			return true
		}
		reportChecks.Set(check.name, r.runHealthCheck(ctx, check))
		return true
	})
	r.subapps.Range(func(_ int, subapp *Runtime) bool {
		if subapp == nil {
			return true
		}
		subReport := subapp.checkHealthByKind(ctx, kind)
		if subReport.Checks == nil {
			return true
		}
		subReport.Checks.Range(func(name string, err error) bool {
			reportChecks.Set(fmt.Sprintf("%s/%s", subapp.Name(), name), err)
			return true
		})
		return true
	})
	report.Checks = reportChecks
	return report
}

func (r *Runtime) runHealthCheck(ctx context.Context, check healthCheckEntry) error {
	startedAt := time.Now()
	err := check.fn(ctx)
	r.emitHealthCheck(ctx, r.healthCheckEvent(check, time.Since(startedAt), err))
	return err
}

// HealthHandler returns a HTTP handler for general health checks.
func (r *Runtime) HealthHandler() http.HandlerFunc {
	return r.healthHandler(HealthKindGeneral)
}

// LivenessHandler returns a HTTP handler for liveness checks.
func (r *Runtime) LivenessHandler() http.HandlerFunc {
	return r.healthHandler(HealthKindLiveness)
}

// ReadinessHandler returns a HTTP handler for readiness checks.
func (r *Runtime) ReadinessHandler() http.HandlerFunc {
	return r.healthHandler(HealthKindReadiness)
}

func (r *Runtime) healthHandler(kind HealthKind) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		var report HealthReport

		switch kind {
		case HealthKindGeneral:
			report = r.CheckHealth(ctx)
		case HealthKindLiveness:
			report = r.CheckLiveness(ctx)
		case HealthKindReadiness:
			report = r.CheckReadiness(ctx)
		default:
			report = r.CheckHealth(ctx)
		}

		status := http.StatusOK
		if !report.Healthy() {
			status = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if err := json.NewEncoder(w).Encode(report); err != nil {
			r.logMessage(ctx, EventLevelError, "write health response failed", "kind", kind, "error", err)
		}
	}
}
