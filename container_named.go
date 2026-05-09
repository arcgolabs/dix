package dix

import "github.com/samber/do/v2"

// ProvideKeyValue registers a keyed singleton value.
func ProvideKeyValue[T any](c *Container, key ServiceKey[T], value T) {
	do.ProvideNamedValue(c.injector, serviceKeyName(c, key), value)
}

// ProvideKey registers a keyed singleton provider with no dependencies.
func ProvideKey[T any](c *Container, key ServiceKey[T], fn func() T) {
	ProvideKeyErr(c, key, func() (T, error) { return fn(), nil })
}

// ProvideKeyErr registers a keyed singleton provider with no dependencies.
func ProvideKeyErr[T any](c *Container, key ServiceKey[T], fn func() (T, error)) {
	provideServiceTimed(c, serviceKeyName(c, key), func(_ do.Injector) (T, error) { return fn() })
}

// ProvideKey1 registers a keyed singleton provider with one dependency.
func ProvideKey1[T, D1 any](c *Container, key ServiceKey[T], fn func(D1) T) {
	ProvideKeyErr1(c, key, func(d1 D1) (T, error) { return fn(d1), nil })
}

// ProvideKeyErr1 registers a keyed singleton provider with one dependency.
func ProvideKeyErr1[T, D1 any](c *Container, key ServiceKey[T], fn func(D1) (T, error)) {
	provideServiceTimed(c, serviceKeyName(c, key), func(i do.Injector) (T, error) {
		d1, err := resolveDependency1[D1](i)
		if err != nil {
			var zero T
			return zero, err
		}
		return fn(d1)
	})
}
