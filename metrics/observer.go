package metrics

import (
	"context"
	"strings"
	"time"

	"github.com/DaiYuANg/arcgo/observabilityx"
	"github.com/arcgolabs/dix"
)

// Option configures the dix metrics observer.
type Option func(*config)

type config struct {
	metricPrefix            string
	includeVersionAttribute bool
	includeHealthCheckName  bool
}

// WithMetricPrefix overrides the metric prefix used by emitted metrics.
func WithMetricPrefix(prefix string) Option {
	return func(cfg *config) {
		clean := strings.TrimSpace(prefix)
		if clean != "" {
			cfg.metricPrefix = clean
		}
	}
}

// WithVersionAttribute controls whether the app version is attached when available.
func WithVersionAttribute(enabled bool) Option {
	return func(cfg *config) {
		cfg.includeVersionAttribute = enabled
	}
}

// WithHealthCheckNameAttribute controls whether health check names are attached.
func WithHealthCheckNameAttribute(enabled bool) Option {
	return func(cfg *config) {
		cfg.includeHealthCheckName = enabled
	}
}

// NewObserver creates a dix.Observer that emits metrics through observabilityx.
func NewObserver(obs observabilityx.Observability, opts ...Option) dix.Observer {
	cfg := config{
		metricPrefix:            "dix",
		includeVersionAttribute: true,
		includeHealthCheckName:  true,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	normalized := observabilityx.Normalize(obs, nil)

	return &observer{
		cfg:              cfg,
		buildTotal:       normalized.Counter(counterSpec(cfg.metricPrefix, "build_total", "Total number of build attempts.", "app", "profile", "version", "result")),
		buildDurationMS:  normalized.Histogram(histogramSpec(cfg.metricPrefix, "build_duration_ms", "Build duration in milliseconds.", "ms", "app", "profile", "version", "result")),
		buildModules:     normalized.Histogram(histogramSpec(cfg.metricPrefix, "build_modules", "Number of modules processed during build.", "", "app", "profile", "version", "result")),
		buildProviders:   normalized.Histogram(histogramSpec(cfg.metricPrefix, "build_providers", "Number of providers processed during build.", "", "app", "profile", "version", "result")),
		buildHooks:       normalized.Histogram(histogramSpec(cfg.metricPrefix, "build_hooks", "Number of hooks processed during build.", "", "app", "profile", "version", "result")),
		buildSetups:      normalized.Histogram(histogramSpec(cfg.metricPrefix, "build_setups", "Number of setups processed during build.", "", "app", "profile", "version", "result")),
		buildInvokes:     normalized.Histogram(histogramSpec(cfg.metricPrefix, "build_invokes", "Number of invokes processed during build.", "", "app", "profile", "version", "result")),
		startTotal:       normalized.Counter(counterSpec(cfg.metricPrefix, "start_total", "Total number of start attempts.", "app", "profile", "version", "result")),
		startDurationMS:  normalized.Histogram(histogramSpec(cfg.metricPrefix, "start_duration_ms", "Start duration in milliseconds.", "ms", "app", "profile", "version", "result")),
		startRegistered:  normalized.Histogram(histogramSpec(cfg.metricPrefix, "start_registered_hooks", "Number of registered start hooks.", "", "app", "profile", "version", "result")),
		startCompleted:   normalized.Histogram(histogramSpec(cfg.metricPrefix, "start_completed_hooks", "Number of completed start hooks.", "", "app", "profile", "version", "result")),
		startRollback:    normalized.Counter(counterSpec(cfg.metricPrefix, "start_rollback_total", "Total number of start rollbacks.", "app", "profile", "version", "result")),
		stopTotal:        normalized.Counter(counterSpec(cfg.metricPrefix, "stop_total", "Total number of stop attempts.", "app", "profile", "version", "result")),
		stopDurationMS:   normalized.Histogram(histogramSpec(cfg.metricPrefix, "stop_duration_ms", "Stop duration in milliseconds.", "ms", "app", "profile", "version", "result")),
		stopRegistered:   normalized.Histogram(histogramSpec(cfg.metricPrefix, "stop_registered_hooks", "Number of registered stop hooks.", "", "app", "profile", "version", "result")),
		stopErrors:       normalized.Histogram(histogramSpec(cfg.metricPrefix, "stop_shutdown_errors", "Number of shutdown errors observed during stop.", "", "app", "profile", "version", "result")),
		stopHookErrors:   normalized.Counter(counterSpec(cfg.metricPrefix, "stop_hook_error_total", "Total number of stop hook errors.", "app", "profile", "version", "result")),
		healthCheckTotal: normalized.Counter(counterSpec(cfg.metricPrefix, "health_check_total", "Total number of health checks.", "app", "profile", "version", "kind", "result", "check")),
		healthDurationMS: normalized.Histogram(histogramSpec(cfg.metricPrefix, "health_check_duration_ms", "Health check duration in milliseconds.", "ms", "app", "profile", "version", "kind", "result", "check")),
		stateTransition:  normalized.Counter(counterSpec(cfg.metricPrefix, "state_transition_total", "Total number of state transitions.", "app", "profile", "version", "from", "to")),
	}
}

// WithObservability adapts an observability backend into a dix AppOption.
func WithObservability(obs observabilityx.Observability, opts ...Option) dix.AppOption {
	return dix.WithObserver(NewObserver(obs, opts...))
}

type observer struct {
	cfg              config
	buildTotal       observabilityx.Counter
	buildDurationMS  observabilityx.Histogram
	buildModules     observabilityx.Histogram
	buildProviders   observabilityx.Histogram
	buildHooks       observabilityx.Histogram
	buildSetups      observabilityx.Histogram
	buildInvokes     observabilityx.Histogram
	startTotal       observabilityx.Counter
	startDurationMS  observabilityx.Histogram
	startRegistered  observabilityx.Histogram
	startCompleted   observabilityx.Histogram
	startRollback    observabilityx.Counter
	stopTotal        observabilityx.Counter
	stopDurationMS   observabilityx.Histogram
	stopRegistered   observabilityx.Histogram
	stopErrors       observabilityx.Histogram
	stopHookErrors   observabilityx.Counter
	healthCheckTotal observabilityx.Counter
	healthDurationMS observabilityx.Histogram
	stateTransition  observabilityx.Counter
}

func (o *observer) OnBuild(ctx context.Context, event dix.BuildEvent) {
	attrs := o.withResultAttrs(event.Meta, event.Profile, event.Err)
	o.buildTotal.Add(ctx, 1, attrs...)
	o.buildDurationMS.Record(ctx, durationMS(event.Duration), attrs...)
	o.buildModules.Record(ctx, float64(event.ModuleCount), attrs...)
	o.buildProviders.Record(ctx, float64(event.ProviderCount), attrs...)
	o.buildHooks.Record(ctx, float64(event.HookCount), attrs...)
	o.buildSetups.Record(ctx, float64(event.SetupCount), attrs...)
	o.buildInvokes.Record(ctx, float64(event.InvokeCount), attrs...)
}

func (o *observer) OnStart(ctx context.Context, event dix.StartEvent) {
	attrs := o.withResultAttrs(event.Meta, event.Profile, event.Err)
	o.startTotal.Add(ctx, 1, attrs...)
	o.startDurationMS.Record(ctx, durationMS(event.Duration), attrs...)
	o.startRegistered.Record(ctx, float64(event.StartHookCount), attrs...)
	o.startCompleted.Record(ctx, float64(event.StartedHookCount), attrs...)
	if event.RolledBack {
		o.startRollback.Add(ctx, 1, attrs...)
	}
}

func (o *observer) OnStop(ctx context.Context, event dix.StopEvent) {
	attrs := o.withResultAttrs(event.Meta, event.Profile, event.Err)
	o.stopTotal.Add(ctx, 1, attrs...)
	o.stopDurationMS.Record(ctx, durationMS(event.Duration), attrs...)
	o.stopRegistered.Record(ctx, float64(event.StopHookCount), attrs...)
	o.stopErrors.Record(ctx, float64(event.ShutdownErrorCount), attrs...)
	if event.HookError {
		o.stopHookErrors.Add(ctx, 1, attrs...)
	}
}

func (o *observer) OnHealthCheck(ctx context.Context, event dix.HealthCheckEvent) {
	attrs := o.commonAttrs(event.Meta, event.Profile)
	attrs = append(attrs,
		observabilityx.String("kind", string(event.Kind)),
		observabilityx.String("result", resultOf(event.Err)),
	)
	if o.cfg.includeHealthCheckName && event.Name != "" {
		attrs = append(attrs, observabilityx.String("check", event.Name))
	}
	o.healthCheckTotal.Add(ctx, 1, attrs...)
	o.healthDurationMS.Record(ctx, durationMS(event.Duration), attrs...)
}

func (o *observer) OnStateTransition(ctx context.Context, event dix.StateTransitionEvent) {
	attrs := o.commonAttrs(event.Meta, event.Profile)
	attrs = append(attrs,
		observabilityx.String("from", event.From.String()),
		observabilityx.String("to", event.To.String()),
	)
	o.stateTransition.Add(ctx, 1, attrs...)
}

func (o *observer) commonAttrs(meta dix.AppMeta, profile dix.Profile) []observabilityx.Attribute {
	attrs := []observabilityx.Attribute{
		observabilityx.String("app", meta.Name),
		observabilityx.String("profile", string(profile)),
	}
	if o.cfg.includeVersionAttribute && strings.TrimSpace(meta.Version) != "" {
		attrs = append(attrs, observabilityx.String("version", meta.Version))
	}
	return attrs
}

func (o *observer) withResultAttrs(meta dix.AppMeta, profile dix.Profile, err error) []observabilityx.Attribute {
	attrs := o.commonAttrs(meta, profile)
	return append(attrs, observabilityx.String("result", resultOf(err)))
}

func counterSpec(prefix, suffix, description string, labelKeys ...string) observabilityx.CounterSpec {
	return observabilityx.NewCounterSpec(
		prefix+"_"+suffix,
		observabilityx.WithDescription(description),
		observabilityx.WithLabelKeys(labelKeys...),
	)
}

func histogramSpec(prefix, suffix, description, unit string, labelKeys ...string) observabilityx.HistogramSpec {
	return observabilityx.NewHistogramSpec(
		prefix+"_"+suffix,
		observabilityx.WithDescription(description),
		observabilityx.WithUnit(unit),
		observabilityx.WithLabelKeys(labelKeys...),
	)
}

func resultOf(err error) string {
	if err != nil {
		return "error"
	}
	return "ok"
}

func durationMS(durationValue time.Duration) float64 {
	return float64(durationValue.Milliseconds())
}
