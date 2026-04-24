package dix

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/arcgolabs/collectionx"
	"github.com/samber/do/v2"
)

// Container wraps samber/do.Injector.
// Most code should stay on the typed dix helpers.
// Raw() exists as an explicit escape hatch for advanced integrations.
type Container struct {
	injector     do.Injector
	serviceNames *serviceNamer
	healthChecks collectionx.List[healthCheckEntry]
	logger       *slog.Logger
	eventLogger  EventLogger
}

func newContainer(logger *slog.Logger) *Container {
	return newContainerWithServiceNamer(logger, newServiceNamer())
}

func newContainerWithServiceNamer(logger *slog.Logger, serviceNames *serviceNamer) *Container {
	if logger == nil {
		logger = defaultLogger()
	}
	return newContainerWithInjector(logger, do.NewWithOpts(&do.InjectorOpts{
		HookBeforeRegistration: nil,
		HookAfterRegistration:  nil,
		HookBeforeInvocation:   nil,
		HookAfterInvocation:    nil,
		HookBeforeShutdown:     nil,
		HookAfterShutdown:      nil,
		Logf: func(format string, args ...any) {
			logger.Debug(fmt.Sprintf(format, args...))
		},
	}), serviceNames)
}

func newChildContainer(parent *Container, name string, logger *slog.Logger) (*Container, error) {
	if parent == nil || parent.injector == nil {
		return nil, errors.New("parent container is nil")
	}
	if name == "" {
		return nil, errors.New("child container name is required")
	}
	if logger == nil {
		logger = parent.logger
	}
	return newContainerWithInjector(logger, parent.injector.Scope(name), parent.serviceNames), nil
}

func newContainerWithInjector(logger *slog.Logger, injector do.Injector, serviceNames *serviceNamer) *Container {
	if logger == nil {
		logger = defaultLogger()
	}
	if serviceNames == nil {
		serviceNames = newServiceNamer()
	}
	return &Container{
		injector:     injector,
		serviceNames: serviceNames,
		healthChecks: collectionx.NewList[healthCheckEntry](),
		logger:       logger,
	}
}

// Raw returns the underlying do injector for advanced integrations.
func (c *Container) Raw() do.Injector { return c.injector }

// Shutdown shuts down all registered container services.
func (c *Container) Shutdown(ctx context.Context) error {
	report := c.ShutdownReport(ctx)
	if report == nil || len(report.Errors) == 0 {
		return nil
	}
	return report
}

// ShutdownReport shuts down the container and returns the do shutdown report.
func (c *Container) ShutdownReport(ctx context.Context) *do.ShutdownReport {
	if c == nil || c.injector == nil {
		return nil
	}
	logMessageEvent(ctx, c.eventLogger, EventLevelDebug, "shutting down container")
	report := c.injector.ShutdownWithContext(ctx)
	errorsCount := 0
	if report != nil {
		errorsCount = len(report.Errors)
	}
	logMessageEvent(ctx, c.eventLogger, EventLevelDebug, "container shutdown completed", "errors", errorsCount)
	return report
}

func resolveInjectorAs[T any](injector do.Injector) (T, error) {
	return do.InvokeNamed[T](injector, serviceNameOf[T]())
}

func resolveContainerAs[T any](c *Container) (T, error) {
	return do.InvokeNamed[T](c.injector, serviceNameOfWith[T](c.serviceNames))
}

// ProvideT registers a typed singleton provider with no dependencies.
func ProvideT[T any](c *Container, fn func() T) {
	ProvideTErr(c, func() (T, error) { return fn(), nil })
}

// ProvideTErr registers a typed singleton provider with no dependencies.
func ProvideTErr[T any](c *Container, fn func() (T, error)) {
	do.ProvideNamed(c.injector, serviceNameOfWith[T](c.serviceNames), func(_ do.Injector) (T, error) { return fn() })
}

// Provide1T registers a typed singleton provider with one dependency.
func Provide1T[T, D1 any](c *Container, fn func(D1) T) {
	Provide1TErr(c, func(d1 D1) (T, error) { return fn(d1), nil })
}

// Provide1TErr registers a typed singleton provider with one dependency.
func Provide1TErr[T, D1 any](c *Container, fn func(D1) (T, error)) {
	do.ProvideNamed(c.injector, serviceNameOfWith[T](c.serviceNames), func(i do.Injector) (T, error) {
		d1, err := resolveDependency1[D1](i)
		if err != nil {
			var zero T
			return zero, err
		}
		return fn(d1)
	})
}

// Provide2T registers a typed singleton provider with two dependencies.
func Provide2T[T, D1, D2 any](c *Container, fn func(D1, D2) T) {
	Provide2TErr(c, func(d1 D1, d2 D2) (T, error) { return fn(d1, d2), nil })
}

// Provide2TErr registers a typed singleton provider with two dependencies.
func Provide2TErr[T, D1, D2 any](c *Container, fn func(D1, D2) (T, error)) {
	do.ProvideNamed(c.injector, serviceNameOfWith[T](c.serviceNames), func(i do.Injector) (T, error) {
		d1, d2, err := resolveDependencies2[D1, D2](i)
		if err != nil {
			var zero T
			return zero, err
		}
		return fn(d1, d2)
	})
}

// Provide3T registers a typed singleton provider with three dependencies.
func Provide3T[T, D1, D2, D3 any](c *Container, fn func(D1, D2, D3) T) {
	Provide3TErr(c, func(d1 D1, d2 D2, d3 D3) (T, error) { return fn(d1, d2, d3), nil })
}

// Provide3TErr registers a typed singleton provider with three dependencies.
func Provide3TErr[T, D1, D2, D3 any](c *Container, fn func(D1, D2, D3) (T, error)) {
	do.ProvideNamed(c.injector, serviceNameOfWith[T](c.serviceNames), func(i do.Injector) (T, error) {
		d1, d2, d3, err := resolveDependencies3[D1, D2, D3](i)
		if err != nil {
			var zero T
			return zero, err
		}
		return fn(d1, d2, d3)
	})
}

// Provide4T registers a typed singleton provider with four dependencies.
func Provide4T[T, D1, D2, D3, D4 any](c *Container, fn func(D1, D2, D3, D4) T) {
	Provide4TErr(c, func(d1 D1, d2 D2, d3 D3, d4 D4) (T, error) { return fn(d1, d2, d3, d4), nil })
}

// Provide4TErr registers a typed singleton provider with four dependencies.
func Provide4TErr[T, D1, D2, D3, D4 any](c *Container, fn func(D1, D2, D3, D4) (T, error)) {
	do.ProvideNamed(c.injector, serviceNameOfWith[T](c.serviceNames), func(i do.Injector) (T, error) {
		d1, d2, d3, d4, err := resolveDependencies4[D1, D2, D3, D4](i)
		if err != nil {
			var zero T
			return zero, err
		}
		return fn(d1, d2, d3, d4)
	})
}

// Provide5T registers a typed singleton provider with five dependencies.
func Provide5T[T, D1, D2, D3, D4, D5 any](c *Container, fn func(D1, D2, D3, D4, D5) T) {
	Provide5TErr(c, func(d1 D1, d2 D2, d3 D3, d4 D4, d5 D5) (T, error) {
		return fn(d1, d2, d3, d4, d5), nil
	})
}

// Provide5TErr registers a typed singleton provider with five dependencies.
func Provide5TErr[T, D1, D2, D3, D4, D5 any](c *Container, fn func(D1, D2, D3, D4, D5) (T, error)) {
	do.ProvideNamed(c.injector, serviceNameOfWith[T](c.serviceNames), func(i do.Injector) (T, error) {
		d1, d2, d3, d4, d5, err := resolveDependencies5[D1, D2, D3, D4, D5](i)
		if err != nil {
			var zero T
			return zero, err
		}
		return fn(d1, d2, d3, d4, d5)
	})
}

// Provide6T registers a typed singleton provider with six dependencies.
func Provide6T[T, D1, D2, D3, D4, D5, D6 any](c *Container, fn func(D1, D2, D3, D4, D5, D6) T) {
	Provide6TErr(c, func(d1 D1, d2 D2, d3 D3, d4 D4, d5 D5, d6 D6) (T, error) {
		return fn(d1, d2, d3, d4, d5, d6), nil
	})
}

// Provide6TErr registers a typed singleton provider with six dependencies.
func Provide6TErr[T, D1, D2, D3, D4, D5, D6 any](c *Container, fn func(D1, D2, D3, D4, D5, D6) (T, error)) {
	do.ProvideNamed(c.injector, serviceNameOfWith[T](c.serviceNames), func(i do.Injector) (T, error) {
		d1, d2, d3, d4, d5, d6, err := resolveDependencies6[D1, D2, D3, D4, D5, D6](i)
		if err != nil {
			var zero T
			return zero, err
		}
		return fn(d1, d2, d3, d4, d5, d6)
	})
}

// ProvideValueT registers a typed singleton value.
func ProvideValueT[T any](c *Container, value T) {
	do.ProvideNamedValue(c.injector, serviceNameOfWith[T](c.serviceNames), value)
}
