//revive:disable:file-length-limit App configuration and runtime entrypoints are kept together as one API surface.

package dix

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/arcgolabs/collectionx"
	"github.com/samber/oops"
)

// AppOption configures an App specification during construction.
type AppOption func(*appSpec)

const (
	// DefaultAppName is the fallback name used by NewDefault.
	DefaultAppName = "dix application"
	// DefaultRunStopTimeout is the default graceful shutdown timeout for RunContext and Run.
	DefaultRunStopTimeout = 30 * time.Second
)

// Modules appends application modules.
func Modules(modules ...Module) AppOption {
	return WithModules(modules...)
}

// WithModule appends a single application module.
func WithModule(module Module) AppOption {
	return WithModules(module)
}

// WithObservers appends runtime observers that receive internal dix events.
func WithObservers(observers ...Observer) AppOption {
	return func(spec *appSpec) {
		spec.appendObserversWithLogger(func() *slog.Logger { return spec.logger }, observers...)
		if spec != nil && spec.observers.Len() > 0 {
			spec.observersConfigured = true
		}
	}
}

// Observers appends runtime observers that receive internal dix events.
func Observers(observers ...Observer) AppOption {
	return WithObservers(observers...)
}

// WithObserver appends a single runtime observer.
func WithObserver(observer Observer) AppOption {
	return WithObservers(observer)
}

// WithDebugScopeTree logs do's scope tree after build.
func WithDebugScopeTree(enabled bool) AppOption {
	return func(spec *appSpec) { spec.debug.scopeTree = enabled }
}

// DebugScopeTree logs do's scope tree after build.
func DebugScopeTree(enabled bool) AppOption {
	return WithDebugScopeTree(enabled)
}

// WithDebugNamedServiceDependencies logs dependency trees for named services after build.
func WithDebugNamedServiceDependencies(names ...string) AppOption {
	return func(spec *appSpec) {
		spec.debug.namedServiceDependencies.Add(names...)
	}
}

// DebugNamedServiceDependencies logs dependency trees for named services after build.
func DebugNamedServiceDependencies(names ...string) AppOption {
	return WithDebugNamedServiceDependencies(names...)
}

func defaultLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{}))
}

// Name returns the configured application name.
func (a *App) Name() string {
	if a == nil || a.spec == nil {
		return ""
	}
	return a.spec.meta.Name
}

// Profile returns the configured application profile.
func (a *App) Profile() Profile {
	if a == nil || a.spec == nil {
		return ""
	}
	return a.spec.profile
}

// Logger returns the application logger.
func (a *App) Logger() *slog.Logger {
	if a == nil || a.spec == nil {
		return nil
	}
	return a.spec.logger
}

// EventLogger returns the configured application event logger when one is explicitly configured.
func (a *App) EventLogger() EventLogger {
	if a == nil || a.spec == nil {
		return nil
	}
	return a.spec.resolvedEventLogger()
}

// Meta returns the application metadata.
func (a *App) Meta() AppMeta {
	if a == nil || a.spec == nil {
		return AppMeta{}
	}
	return a.spec.meta
}

// RunStopTimeout returns the graceful shutdown timeout used by RunContext and Run.
func (a *App) RunStopTimeout() time.Duration {
	if a == nil || a.spec == nil {
		return 0
	}
	return a.spec.runStopTimeout
}

// Modules returns the configured application modules.
func (a *App) Modules() collectionx.List[Module] {
	if a == nil || a.spec == nil {
		return collectionx.NewList[Module]()
	}
	return a.spec.modules.Clone()
}

// SubApps returns the configured child applications.
func (a *App) SubApps() collectionx.List[*App] {
	if a == nil || a.spec == nil {
		return collectionx.NewList[*App]()
	}
	return a.spec.subapps.Clone()
}

// Build compiles the immutable App spec into a Runtime.
func (a *App) Build() (*Runtime, error) {
	return a.BuildContext(context.Background())
}

// BuildContext compiles the immutable App spec into a Runtime using the provided context.
func (a *App) BuildContext(ctx context.Context) (*Runtime, error) {
	return a.buildWithContext(ctx)
}

func (a *App) buildWithContext(ctx context.Context) (*Runtime, error) {
	plan, _, err := a.cachedBuildPlan(ctx)
	if err != nil {
		return nil, err
	}
	return plan.Build(ctx)
}

// Start builds a Runtime and starts it with the provided context.
func (a *App) Start(ctx context.Context) (*Runtime, error) {
	rt, err := a.buildWithContext(ctx)
	if err != nil {
		return nil, oops.In("dix").
			With("op", "start", "app", a.Name()).
			Wrapf(err, "build failed")
	}
	if err := rt.Start(ctx); err != nil {
		return nil, oops.In("dix").
			With("op", "start", "app", a.Name()).
			Wrapf(err, "start failed")
	}
	return rt, nil
}

// RunContext builds a Runtime, starts it, waits for the context to finish, and stops it.
func (a *App) RunContext(ctx context.Context) error {
	ctx = contextOrBackground(ctx)
	rt, err := a.Start(ctx)
	if err != nil {
		return err
	}

	<-ctx.Done()

	stopCtx, cancel := a.runStopContext(ctx)
	defer cancel()
	if err := rt.Stop(stopCtx); err != nil {
		return oops.In("dix").
			With("op", "run_context_stop", "app", a.Name()).
			Wrapf(err, "stop failed")
	}

	return nil
}

func (a *App) runStopContext(ctx context.Context) (context.Context, context.CancelFunc) {
	stopCtx := context.WithoutCancel(contextOrBackground(ctx))
	timeout := a.RunStopTimeout()
	if timeout <= 0 {
		return stopCtx, func() {}
	}
	return context.WithTimeout(stopCtx, timeout)
}

// Run builds a Runtime, starts it, waits for shutdown signals, and stops it.
func (a *App) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	return a.RunContext(ctx)
}
