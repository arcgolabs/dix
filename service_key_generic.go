package dix

import "context"

// Resolve resolves the keyed service from a container.
func (k ServiceKey[T]) Resolve(c *Container) (T, error) {
	return c.ResolveKey(k)
}

// ResolveContext resolves the keyed service and honors context cancellation before resolution.
func (k ServiceKey[T]) ResolveContext(ctx context.Context, c *Container) (T, error) {
	return c.ResolveKeyContext(ctx, k)
}

// MustResolve resolves the keyed service and panics when resolution fails.
func (k ServiceKey[T]) MustResolve(c *Container) T {
	return c.MustResolveKey(k)
}

// ResolveOptional resolves the keyed service and reports whether it is registered.
func (k ServiceKey[T]) ResolveOptional(c *Container) (T, bool) {
	value, ok, err := c.ResolveOptionalKey(k)
	if err != nil {
		panic(err)
	}
	return value, ok
}

// Provide registers a keyed singleton provider without dependencies.
func (k ServiceKey[T]) Provide(c *Container, fn func() T) {
	c.ProvideKey(k, fn)
}

// ProvideErr registers a fallible keyed singleton provider without dependencies.
func (k ServiceKey[T]) ProvideErr(c *Container, fn func() (T, error)) {
	c.ProvideKeyErr(k, fn)
}

// Provide1 registers a keyed singleton provider with one dependency.
func (k ServiceKey[T]) Provide1[D1 any](c *Container, fn func(D1) T) {
	c.ProvideKey1(k, fn)
}

// Provide1Err registers a fallible keyed singleton provider with one dependency.
func (k ServiceKey[T]) Provide1Err[D1 any](c *Container, fn func(D1) (T, error)) {
	c.ProvideKey1Err(k, fn)
}
