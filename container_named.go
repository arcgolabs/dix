package dix

import "github.com/samber/do/v2"

// ProvideNamedValueT registers a named singleton value.
func ProvideNamedValueT[T any](c *Container, name string, value T) {
	do.ProvideNamedValue(c.injector, name, value)
}

// ProvideNamedT registers a named singleton provider with no dependencies.
func ProvideNamedT[T any](c *Container, name string, fn func() T) {
	ProvideNamedTErr(c, name, func() (T, error) { return fn(), nil })
}

// ProvideNamedTErr registers a named singleton provider with no dependencies.
func ProvideNamedTErr[T any](c *Container, name string, fn func() (T, error)) {
	provideNamedTimed(c, name, func(_ do.Injector) (T, error) { return fn() })
}

// ProvideNamed1T registers a named singleton provider with one dependency.
func ProvideNamed1T[T, D1 any](c *Container, name string, fn func(D1) T) {
	ProvideNamed1TErr(c, name, func(d1 D1) (T, error) { return fn(d1), nil })
}

// ProvideNamed1TErr registers a named singleton provider with one dependency.
func ProvideNamed1TErr[T, D1 any](c *Container, name string, fn func(D1) (T, error)) {
	provideNamedTimed(c, name, func(i do.Injector) (T, error) {
		d1, err := resolveDependency1[D1](i)
		if err != nil {
			var zero T
			return zero, err
		}
		return fn(d1)
	})
}
