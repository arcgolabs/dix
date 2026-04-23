package dix

import (
	"errors"

	"github.com/samber/do/v2"
	"github.com/samber/mo"
	"github.com/samber/oops"
)

// ResolveAs resolves a typed value from the container.
func ResolveAs[T any](c *Container) (T, error) {
	if c == nil || c.injector == nil {
		var zero T
		return zero, oops.In("dix").
			With("op", "resolve", "service", serviceNameOf[T]()).
			New("container is nil")
	}
	return resolveContainerAs[T](c)
}

// ResolveOptionalAs resolves an optional typed value from the container.
func ResolveOptionalAs[T any](c *Container) (value T, ok bool) {
	return ResolveOptional[T](c)
}

// ResolveOptionalE resolves an optional typed value from the container and returns an error
// when resolution fails for reasons other than a missing service.
func ResolveOptionalE[T any](c *Container) (value T, ok bool, err error) {
	option, err := ResolveOptionE[T](c)
	if err != nil {
		var zero T
		return zero, false, err
	}
	value, ok = option.Get()
	return value, ok, nil
}

// ResolveOptional resolves an optional typed value from the container.
func ResolveOptional[T any](c *Container) (value T, ok bool) {
	value, ok, err := ResolveOptionalE[T](c)
	if err != nil {
		panic(err)
	}
	return value, ok
}

// ResolveOptionAs resolves an optional dependency as mo.Option.
func ResolveOptionAs[T any](c *Container) mo.Option[T] {
	return ResolveOption[T](c)
}

// ResolveOptionE resolves an optional dependency as mo.Option and returns an error
// when resolution fails for reasons other than a missing service.
func ResolveOptionE[T any](c *Container) (mo.Option[T], error) {
	value, err := ResolveAs[T](c)
	if err == nil {
		return mo.Some(value), nil
	}
	if errors.Is(err, do.ErrServiceNotFound) {
		return mo.None[T](), nil
	}
	return mo.None[T](), oops.In("dix").
		With("op", "resolve_option", "service", serviceNameOf[T]()).
		Wrapf(err, "resolve optional dependency")
}

// ResolveOption resolves an optional dependency as mo.Option.
func ResolveOption[T any](c *Container) mo.Option[T] {
	option, err := ResolveOptionE[T](c)
	if err != nil {
		panic(err)
	}
	return option
}

// ResolveOrElse resolves a typed value or returns the provided fallback.
func ResolveOrElse[T any](c *Container, fallback T) T {
	return ResolveOr[T](c, fallback)
}

// ResolveOr resolves a typed value or returns the provided fallback.
func ResolveOr[T any](c *Container, fallback T) T {
	value, err := ResolveOrErr[T](c, fallback)
	if err != nil {
		panic(err)
	}
	return value
}

// ResolveOrErr resolves a typed value or returns the provided fallback, and returns an error
// when resolution fails for reasons other than a missing service.
func ResolveOrErr[T any](c *Container, fallback T) (T, error) {
	option, err := ResolveOptionE[T](c)
	if err != nil {
		return fallback, err
	}
	return option.OrElse(fallback), nil
}

// MustResolveAs resolves a typed value and panics on failure.
func MustResolveAs[T any](c *Container) T {
	result, err := ResolveAs[T](c)
	if err != nil {
		panic(oops.In("dix").
			With("op", "must_resolve", "service", serviceNameOf[T]()).
			Wrapf(err, "resolve dependency"))
	}
	return result
}
