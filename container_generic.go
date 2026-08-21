package dix

import (
	"context"
	"errors"

	"github.com/samber/do/v2"
)

// Resolve resolves a service using its Go type as the service identity.
func (c *Container) Resolve[T any]() (T, error) {
	return ResolveAs[T](c)
}

// ResolveContext resolves a service and honors context cancellation before resolution.
func (c *Container) ResolveContext[T any](ctx context.Context) (T, error) {
	return ResolveAsContext[T](ctx, c)
}

// ResolveKey resolves a service using a typed service key.
func (c *Container) ResolveKey[T any](key ServiceKey[T]) (T, error) {
	return ResolveKey(c, key)
}

// ResolveKeyContext resolves a keyed service and honors context cancellation before resolution.
func (c *Container) ResolveKeyContext[T any](ctx context.Context, key ServiceKey[T]) (T, error) {
	return ResolveKeyContext(ctx, c, key)
}

// MustResolve resolves a service and panics when resolution fails.
func (c *Container) MustResolve[T any]() T {
	return MustResolveAs[T](c)
}

// MustResolveKey resolves a keyed service and panics when resolution fails.
func (c *Container) MustResolveKey[T any](key ServiceKey[T]) T {
	return MustResolveKey(c, key)
}

// ResolveOptional resolves a service and reports whether it is registered.
func (c *Container) ResolveOptional[T any]() (T, bool) {
	return ResolveOptional[T](c)
}

// ResolveOptionalE resolves a service and returns non-missing resolution errors.
func (c *Container) ResolveOptionalE[T any]() (T, bool, error) {
	return ResolveOptionalE[T](c)
}

// ResolveOptionalKey resolves a keyed service and returns non-missing resolution errors.
func (c *Container) ResolveOptionalKey[T any](key ServiceKey[T]) (T, bool, error) {
	value, err := c.ResolveKey(key)
	if err == nil {
		return value, true, nil
	}
	if errors.Is(err, do.ErrServiceNotFound) {
		var zero T
		return zero, false, nil
	}
	var zero T
	return zero, false, err
}

// ResolveOption resolves a service as an optional value.
func (c *Container) ResolveOption[T any]() Conditional[T] {
	return ResolveOption[T](c)
}

// ResolveOptionE resolves a service as an optional value and returns non-missing errors.
func (c *Container) ResolveOptionE[T any]() (Conditional[T], error) {
	return ResolveOptionE[T](c)
}

// ResolveOr resolves a service or returns fallback when the service is missing.
func (c *Container) ResolveOr[T any](fallback T) T {
	return ResolveOr[T](c, fallback)
}

// ResolveOrErr resolves a service or returns fallback and reports non-missing errors.
func (c *Container) ResolveOrErr[T any](fallback T) (T, error) {
	return ResolveOrErr[T](c, fallback)
}

// Provide registers a typed singleton provider without dependencies.
func (c *Container) Provide[T any](fn func() T) {
	ProvideT(c, fn)
}

// ProvideErr registers a fallible typed singleton provider without dependencies.
func (c *Container) ProvideErr[T any](fn func() (T, error)) {
	ProvideTErr(c, fn)
}

// Provide1 registers a typed singleton provider with one dependency.
func (c *Container) Provide1[T, D1 any](fn func(D1) T) {
	Provide1T(c, fn)
}

// Provide1Err registers a fallible typed singleton provider with one dependency.
func (c *Container) Provide1Err[T, D1 any](fn func(D1) (T, error)) {
	Provide1TErr(c, fn)
}

// Provide2 registers a typed singleton provider with two dependencies.
func (c *Container) Provide2[T, D1, D2 any](fn func(D1, D2) T) {
	Provide2T(c, fn)
}

// Provide2Err registers a fallible typed singleton provider with two dependencies.
func (c *Container) Provide2Err[T, D1, D2 any](fn func(D1, D2) (T, error)) {
	Provide2TErr(c, fn)
}

// Provide3 registers a typed singleton provider with three dependencies.
func (c *Container) Provide3[T, D1, D2, D3 any](fn func(D1, D2, D3) T) {
	Provide3T(c, fn)
}

// Provide3Err registers a fallible typed singleton provider with three dependencies.
func (c *Container) Provide3Err[T, D1, D2, D3 any](fn func(D1, D2, D3) (T, error)) {
	Provide3TErr(c, fn)
}

// Provide4 registers a typed singleton provider with four dependencies.
func (c *Container) Provide4[T, D1, D2, D3, D4 any](fn func(D1, D2, D3, D4) T) {
	Provide4T(c, fn)
}

// Provide4Err registers a fallible typed singleton provider with four dependencies.
func (c *Container) Provide4Err[T, D1, D2, D3, D4 any](fn func(D1, D2, D3, D4) (T, error)) {
	Provide4TErr(c, fn)
}

// Provide5 registers a typed singleton provider with five dependencies.
func (c *Container) Provide5[T, D1, D2, D3, D4, D5 any](fn func(D1, D2, D3, D4, D5) T) {
	Provide5T(c, fn)
}

// Provide5Err registers a fallible typed singleton provider with five dependencies.
func (c *Container) Provide5Err[T, D1, D2, D3, D4, D5 any](fn func(D1, D2, D3, D4, D5) (T, error)) {
	Provide5TErr(c, fn)
}

// Provide6 registers a typed singleton provider with six dependencies.
func (c *Container) Provide6[T, D1, D2, D3, D4, D5, D6 any](fn func(D1, D2, D3, D4, D5, D6) T) {
	Provide6T(c, fn)
}

// Provide6Err registers a fallible typed singleton provider with six dependencies.
func (c *Container) Provide6Err[T, D1, D2, D3, D4, D5, D6 any](fn func(D1, D2, D3, D4, D5, D6) (T, error)) {
	Provide6TErr(c, fn)
}

// ProvideKey registers a keyed singleton provider without dependencies.
func (c *Container) ProvideKey[T any](key ServiceKey[T], fn func() T) {
	ProvideKey(c, key, fn)
}

// ProvideKeyErr registers a fallible keyed singleton provider without dependencies.
func (c *Container) ProvideKeyErr[T any](key ServiceKey[T], fn func() (T, error)) {
	ProvideKeyErr(c, key, fn)
}

// ProvideKey1 registers a keyed singleton provider with one dependency.
func (c *Container) ProvideKey1[T, D1 any](key ServiceKey[T], fn func(D1) T) {
	ProvideKey1(c, key, fn)
}

// ProvideKey1Err registers a fallible keyed singleton provider with one dependency.
func (c *Container) ProvideKey1Err[T, D1 any](key ServiceKey[T], fn func(D1) (T, error)) {
	ProvideKeyErr1(c, key, fn)
}
