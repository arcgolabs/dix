package advanced

import (
	"github.com/arcgolabs/arcgo/dix"
	"github.com/samber/do/v2"
)

// NamedValue registers a named singleton value.
func NamedValue[T any](name string, value T) dix.ProviderFunc {
	return newProvider("NamedValue", dix.NamedService(name), func(c *dix.Container) {
		do.ProvideNamedValue(c.Raw(), name, value)
	})
}

// Named registers a named singleton value.
func Named[T any](name string, value T) dix.ProviderFunc {
	return NamedValue(name, value)
}

// NamedProvider0 registers a named singleton provider with no dependencies.
func NamedProvider0[T any](name string, fn func() T) dix.ProviderFunc {
	return newProvider("NamedProvider0", dix.NamedService(name), func(c *dix.Container) {
		do.ProvideNamed(c.Raw(), name, func(do.Injector) (T, error) { return fn(), nil })
	})
}

// NamedProviderErr0 registers a named singleton provider with no dependencies.
func NamedProviderErr0[T any](name string, fn func() (T, error)) dix.ProviderFunc {
	return newProvider("NamedProviderErr0", dix.NamedService(name), func(c *dix.Container) {
		do.ProvideNamed(c.Raw(), name, func(do.Injector) (T, error) { return fn() })
	})
}

// NamedProvider1 registers a named singleton provider with one dependency.
func NamedProvider1[T, D1 any](name string, fn func(D1) T) dix.ProviderFunc {
	return newProvider("NamedProvider1", dix.NamedService(name), func(c *dix.Container) {
		do.ProvideNamed(c.Raw(), name, func(i do.Injector) (T, error) {
			d1, err := invokeTyped[D1](i)
			if err != nil {
				var zero T
				return zero, err
			}
			return fn(d1), nil
		})
	}, dix.TypedService[D1]())
}

// NamedProviderErr1 registers a named singleton provider with one dependency.
func NamedProviderErr1[T, D1 any](name string, fn func(D1) (T, error)) dix.ProviderFunc {
	return newProvider("NamedProviderErr1", dix.NamedService(name), func(c *dix.Container) {
		do.ProvideNamed(c.Raw(), name, func(i do.Injector) (T, error) {
			d1, err := invokeTyped[D1](i)
			if err != nil {
				var zero T
				return zero, err
			}
			return fn(d1)
		})
	}, dix.TypedService[D1]())
}

// NamedProvider2 registers a named singleton provider with two dependencies.
func NamedProvider2[T, D1, D2 any](name string, fn func(D1, D2) T) dix.ProviderFunc {
	return newProvider("NamedProvider2", dix.NamedService(name), func(c *dix.Container) {
		do.ProvideNamed(c.Raw(), name, func(i do.Injector) (T, error) {
			d1, err := invokeTyped[D1](i)
			if err != nil {
				var zero T
				return zero, err
			}
			d2, err := invokeTyped[D2](i)
			if err != nil {
				var zero T
				return zero, err
			}
			return fn(d1, d2), nil
		})
	}, dix.TypedService[D1](), dix.TypedService[D2]())
}

// NamedProviderErr2 registers a named singleton provider with two dependencies.
func NamedProviderErr2[T, D1, D2 any](name string, fn func(D1, D2) (T, error)) dix.ProviderFunc {
	return newProvider("NamedProviderErr2", dix.NamedService(name), func(c *dix.Container) {
		do.ProvideNamed(c.Raw(), name, func(i do.Injector) (T, error) {
			d1, err := invokeTyped[D1](i)
			if err != nil {
				var zero T
				return zero, err
			}
			d2, err := invokeTyped[D2](i)
			if err != nil {
				var zero T
				return zero, err
			}
			return fn(d1, d2)
		})
	}, dix.TypedService[D1](), dix.TypedService[D2]())
}

// NamedProvider3 registers a named singleton provider with three dependencies.
func NamedProvider3[T, D1, D2, D3 any](name string, fn func(D1, D2, D3) T) dix.ProviderFunc {
	return newProvider("NamedProvider3", dix.NamedService(name), func(c *dix.Container) {
		do.ProvideNamed(c.Raw(), name, func(i do.Injector) (T, error) {
			d1, err := invokeTyped[D1](i)
			if err != nil {
				var zero T
				return zero, err
			}
			d2, err := invokeTyped[D2](i)
			if err != nil {
				var zero T
				return zero, err
			}
			d3, err := invokeTyped[D3](i)
			if err != nil {
				var zero T
				return zero, err
			}
			return fn(d1, d2, d3), nil
		})
	}, dix.TypedService[D1](), dix.TypedService[D2](), dix.TypedService[D3]())
}

// NamedProviderErr3 registers a named singleton provider with three dependencies.
func NamedProviderErr3[T, D1, D2, D3 any](name string, fn func(D1, D2, D3) (T, error)) dix.ProviderFunc {
	return newProvider("NamedProviderErr3", dix.NamedService(name), func(c *dix.Container) {
		do.ProvideNamed(c.Raw(), name, func(i do.Injector) (T, error) {
			d1, err := invokeTyped[D1](i)
			if err != nil {
				var zero T
				return zero, err
			}
			d2, err := invokeTyped[D2](i)
			if err != nil {
				var zero T
				return zero, err
			}
			d3, err := invokeTyped[D3](i)
			if err != nil {
				var zero T
				return zero, err
			}
			return fn(d1, d2, d3)
		})
	}, dix.TypedService[D1](), dix.TypedService[D2](), dix.TypedService[D3]())
}

// TransientProvider0 registers a typed transient provider with no dependencies.
func TransientProvider0[T any](fn func() T) dix.ProviderFunc {
	name := typedName[T]()
	return newProvider("TransientProvider0", dix.TypedService[T](), func(c *dix.Container) {
		do.ProvideNamedTransient(c.Raw(), name, func(do.Injector) (T, error) { return fn(), nil })
	})
}

// Transient registers a typed transient provider with no dependencies.
func Transient[T any](fn func() T) dix.ProviderFunc {
	return TransientProvider0(fn)
}

// TransientProviderErr0 registers a typed transient provider with no dependencies.
func TransientProviderErr0[T any](fn func() (T, error)) dix.ProviderFunc {
	name := typedName[T]()
	return newProvider("TransientProviderErr0", dix.TypedService[T](), func(c *dix.Container) {
		do.ProvideNamedTransient(c.Raw(), name, func(do.Injector) (T, error) { return fn() })
	})
}

// TransientErr registers a typed transient provider with no dependencies.
func TransientErr[T any](fn func() (T, error)) dix.ProviderFunc {
	return TransientProviderErr0(fn)
}

// TransientProvider1 registers a typed transient provider with one dependency.
func TransientProvider1[T, D1 any](fn func(D1) T) dix.ProviderFunc {
	name := typedName[T]()
	return newProvider("TransientProvider1", dix.TypedService[T](), func(c *dix.Container) {
		do.ProvideNamedTransient(c.Raw(), name, func(i do.Injector) (T, error) {
			d1, err := invokeTyped[D1](i)
			if err != nil {
				var zero T
				return zero, err
			}
			return fn(d1), nil
		})
	}, dix.TypedService[D1]())
}

// TransientProviderErr1 registers a typed transient provider with one dependency.
func TransientProviderErr1[T, D1 any](fn func(D1) (T, error)) dix.ProviderFunc {
	name := typedName[T]()
	return newProvider("TransientProviderErr1", dix.TypedService[T](), func(c *dix.Container) {
		do.ProvideNamedTransient(c.Raw(), name, func(i do.Injector) (T, error) {
			d1, err := invokeTyped[D1](i)
			if err != nil {
				var zero T
				return zero, err
			}
			return fn(d1)
		})
	}, dix.TypedService[D1]())
}

// NamedTransientProvider0 registers a named transient provider with no dependencies.
func NamedTransientProvider0[T any](name string, fn func() T) dix.ProviderFunc {
	return newProvider("NamedTransientProvider0", dix.NamedService(name), func(c *dix.Container) {
		do.ProvideNamedTransient(c.Raw(), name, func(do.Injector) (T, error) { return fn(), nil })
	})
}

// NamedTransient registers a named transient provider with no dependencies.
func NamedTransient[T any](name string, fn func() T) dix.ProviderFunc {
	return NamedTransientProvider0(name, fn)
}

// NamedTransientProviderErr0 registers a named transient provider with no dependencies.
func NamedTransientProviderErr0[T any](name string, fn func() (T, error)) dix.ProviderFunc {
	return newProvider("NamedTransientProviderErr0", dix.NamedService(name), func(c *dix.Container) {
		do.ProvideNamedTransient(c.Raw(), name, func(do.Injector) (T, error) { return fn() })
	})
}

// NamedTransientErr registers a named transient provider with no dependencies.
func NamedTransientErr[T any](name string, fn func() (T, error)) dix.ProviderFunc {
	return NamedTransientProviderErr0(name, fn)
}

// NamedTransientProvider1 registers a named transient provider with one dependency.
func NamedTransientProvider1[T, D1 any](name string, fn func(D1) T) dix.ProviderFunc {
	return newProvider("NamedTransientProvider1", dix.NamedService(name), func(c *dix.Container) {
		do.ProvideNamedTransient(c.Raw(), name, func(i do.Injector) (T, error) {
			d1, err := invokeTyped[D1](i)
			if err != nil {
				var zero T
				return zero, err
			}
			return fn(d1), nil
		})
	}, dix.TypedService[D1]())
}

// NamedTransientProviderErr1 registers a named transient provider with one dependency.
func NamedTransientProviderErr1[T, D1 any](name string, fn func(D1) (T, error)) dix.ProviderFunc {
	return newProvider("NamedTransientProviderErr1", dix.NamedService(name), func(c *dix.Container) {
		do.ProvideNamedTransient(c.Raw(), name, func(i do.Injector) (T, error) {
			d1, err := invokeTyped[D1](i)
			if err != nil {
				var zero T
				return zero, err
			}
			return fn(d1)
		})
	}, dix.TypedService[D1]())
}
