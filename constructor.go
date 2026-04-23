package dix

import (
	"log/slog"
	"time"

	"github.com/arcgolabs/collectionx"
	"github.com/arcgolabs/pkg/option"
)

// NewDefault creates an application with the default framework name.
func NewDefault(opts ...AppOption) *App {
	return New(DefaultAppName, opts...)
}

// New creates an immutable application specification.
func New(name string, opts ...AppOption) *App {
	spec := &appSpec{
		meta:                AppMeta{Name: name},
		profile:             ProfileDefault,
		modules:             collectionx.NewList[Module](),
		logger:              defaultLogger(),
		observers:           collectionx.NewList[Observer](),
		observerDispatchers: collectionx.NewList[*observerDispatcher](),
		subapps:             collectionx.NewList[*App](),
		runStopTimeout:      DefaultRunStopTimeout,
		debug: debugSettings{
			namedServiceDependencies: collectionx.NewOrderedSet[string](),
		},
	}

	option.Apply(spec, opts...)

	return &App{spec: spec}
}

// NewApp keeps backward compatibility with the v0.3 style constructor surface.
func NewApp(name string, modules ...Module) *App {
	return New(name, WithModules(modules...))
}

// NewSubApp creates an application spec intended to be mounted below a parent app.
func NewSubApp(name string, opts ...AppOption) *App {
	return New(name, opts...)
}

// WithProfile selects the runtime profile for the application.
func WithProfile(profile Profile) AppOption {
	return func(spec *appSpec) {
		spec.profile = profile
		spec.profileConfigured = true
	}
}

// UseProfile selects the runtime profile for the application.
func UseProfile(profile Profile) AppOption {
	return WithProfile(profile)
}

// WithVersion sets application version metadata.
func WithVersion(version string) AppOption {
	return func(spec *appSpec) {
		spec.meta.Version = version
		spec.versionConfigured = true
	}
}

// Version sets application version metadata.
func Version(version string) AppOption {
	return WithVersion(version)
}

// WithAppDescription sets application description metadata.
func WithAppDescription(description string) AppOption {
	return func(spec *appSpec) {
		spec.meta.Description = description
		spec.descriptionConfigured = true
	}
}

// AppDescription sets application description metadata.
func AppDescription(description string) AppOption {
	return WithAppDescription(description)
}

// WithRunStopTimeout sets the graceful shutdown timeout used by RunContext and Run.
//
// A non-positive timeout disables the automatic shutdown timeout.
func WithRunStopTimeout(timeout time.Duration) AppOption {
	return func(spec *appSpec) {
		spec.runStopTimeout = timeout
	}
}

// RunStopTimeout sets the graceful shutdown timeout used by RunContext and Run.
func RunStopTimeout(timeout time.Duration) AppOption {
	return WithRunStopTimeout(timeout)
}

// WithLogger sets the framework slog logger.
//
// The logger also becomes the default dix EventLogger, so internal build,
// lifecycle, health, and debug events are emitted through the same instance.
func WithLogger(logger *slog.Logger) AppOption {
	return func(spec *appSpec) {
		if logger != nil {
			spec.logger = logger
			spec.loggerConfigured = true
			spec.eventLogger = NewSlogEventLogger(logger)
			spec.eventLoggerConfigured = true
		}
	}
}

// UseLogger sets the framework slog logger.
func UseLogger(logger *slog.Logger) AppOption {
	return WithLogger(logger)
}

// UseLogger0 resolves the framework logger from a zero-dependency callback.
func UseLogger0(fn func() *slog.Logger) AppOption {
	return WithLoggerFrom0(fn)
}

// UseLoggerErr0 resolves the framework logger from a zero-dependency callback that can fail.
func UseLoggerErr0(fn func() (*slog.Logger, error)) AppOption {
	if fn == nil {
		return func(*appSpec) {}
	}
	return WithLoggerFrom(func(*Container) (*slog.Logger, error) {
		return fn()
	})
}

// UseLogger1 resolves the framework logger from a one-dependency callback.
func UseLogger1[D1 any](fn func(D1) *slog.Logger) AppOption {
	return WithLoggerFrom1(fn)
}

// UseLoggerErr1 resolves the framework logger from a one-dependency callback that can fail.
func UseLoggerErr1[D1 any](fn func(D1) (*slog.Logger, error)) AppOption {
	if fn == nil {
		return func(*appSpec) {}
	}
	return WithLoggerFrom(func(c *Container) (*slog.Logger, error) {
		d1, err := ResolveAs[D1](c)
		if err != nil {
			return nil, err
		}
		return fn(d1)
	})
}

// UseEventLogger sets the framework event logger. When configured, dix internal logging routes through it.
func UseEventLogger(logger EventLogger) AppOption {
	return func(spec *appSpec) {
		if logger != nil {
			spec.eventLogger = logger
			spec.eventLoggerConfigured = true
		}
	}
}

// UseEventLogger0 resolves the framework event logger from a zero-dependency callback.
func UseEventLogger0(fn func() EventLogger) AppOption {
	if fn == nil {
		return func(*appSpec) {}
	}
	return UseEventLoggerErr0(func() (EventLogger, error) {
		return fn(), nil
	})
}

// UseEventLoggerErr0 resolves the framework event logger from a zero-dependency callback that can fail.
func UseEventLoggerErr0(fn func() (EventLogger, error)) AppOption {
	if fn == nil {
		return func(*appSpec) {}
	}
	return func(spec *appSpec) {
		spec.eventLoggerFromContainer = func(*Container) (EventLogger, error) {
			return fn()
		}
	}
}

// UseEventLogger1 resolves the framework event logger from a one-dependency callback.
func UseEventLogger1[D1 any](fn func(D1) EventLogger) AppOption {
	if fn == nil {
		return func(*appSpec) {}
	}
	return UseEventLoggerErr1(func(d1 D1) (EventLogger, error) {
		return fn(d1), nil
	})
}

// UseEventLoggerErr1 resolves the framework event logger from a one-dependency callback that can fail.
func UseEventLoggerErr1[D1 any](fn func(D1) (EventLogger, error)) AppOption {
	if fn == nil {
		return func(*appSpec) {}
	}
	return func(spec *appSpec) {
		spec.eventLoggerFromContainer = func(c *Container) (EventLogger, error) {
			d1, err := ResolveAs[D1](c)
			if err != nil {
				return nil, err
			}
			return fn(d1)
		}
	}
}

// WithLoggerFrom resolves the framework logger from the built DI container.
// The resolved logger overrides the default logger and updates runtime internals.
func WithLoggerFrom(fn func(*Container) (*slog.Logger, error)) AppOption {
	return func(spec *appSpec) {
		if fn != nil {
			spec.loggerFromContainer = fn
		}
	}
}

// LoggerFrom resolves the framework logger from the built DI container.
func LoggerFrom(fn func(*Container) (*slog.Logger, error)) AppOption {
	return WithLoggerFrom(fn)
}

// WithLoggerFrom0 resolves the framework logger from a zero-dependency callback.
func WithLoggerFrom0(fn func() *slog.Logger) AppOption {
	if fn == nil {
		return func(*appSpec) {}
	}
	return WithLoggerFrom(func(*Container) (*slog.Logger, error) {
		return fn(), nil
	})
}

// LoggerFrom0 resolves the framework logger from a zero-dependency callback.
func LoggerFrom0(fn func() *slog.Logger) AppOption {
	return WithLoggerFrom0(fn)
}

// WithLoggerFrom1 resolves the framework logger from a one-dependency callback.
func WithLoggerFrom1[D1 any](fn func(D1) *slog.Logger) AppOption {
	if fn == nil {
		return func(*appSpec) {}
	}
	return WithLoggerFrom(func(c *Container) (*slog.Logger, error) {
		d1, err := ResolveAs[D1](c)
		if err != nil {
			return nil, err
		}
		return fn(d1), nil
	})
}

// LoggerFrom1 resolves the framework logger from a one-dependency callback.
func LoggerFrom1[D1 any](fn func(D1) *slog.Logger) AppOption {
	return WithLoggerFrom1(fn)
}

// WithModules appends application modules.
func WithModules(modules ...Module) AppOption {
	return func(spec *appSpec) {
		spec.modules.Add(modules...)
	}
}

// WithSubApps appends child applications. Child applications are built in child DI scopes.
func WithSubApps(apps ...*App) AppOption {
	return func(spec *appSpec) {
		spec.subapps.Add(apps...)
	}
}

// SubApps appends child applications. Child applications are built in child DI scopes.
func SubApps(apps ...*App) AppOption {
	return WithSubApps(apps...)
}

// WithSubApp appends a single child application.
func WithSubApp(app *App) AppOption {
	return WithSubApps(app)
}
