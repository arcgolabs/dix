package advanced

import (
	"github.com/arcgolabs/arcgo/dix"
	"github.com/samber/do/v2"
	"github.com/samber/oops"
)

// ResolveInjectorAs resolves a typed value directly from a do injector.
func ResolveInjectorAs[T any](injector do.Injector) (T, error) {
	if injector == nil {
		var zero T
		return zero, oops.In("dix/advanced").
			With("op", "resolve_injector", "service", typedName[T]()).
			New("injector is nil")
	}
	return do.InvokeNamed[T](injector, typedName[T]())
}

// MustResolveInjectorAs resolves a typed value directly from a do injector and panics on failure.
func MustResolveInjectorAs[T any](injector do.Injector) T {
	value, err := ResolveInjectorAs[T](injector)
	if err != nil {
		panic(oops.In("dix/advanced").
			With("op", "must_resolve_injector", "service", typedName[T]()).
			Wrapf(err, "resolve injector dependency"))
	}
	return value
}

// ResolveRuntimeAs resolves a typed value from a dix runtime.
func ResolveRuntimeAs[T any](rt *dix.Runtime) (T, error) {
	if rt == nil {
		var zero T
		return zero, oops.In("dix/advanced").
			With("op", "resolve_runtime", "service", typedName[T]()).
			New("runtime is nil")
	}
	return ResolveInjectorAs[T](rt.Raw())
}

// MustResolveRuntimeAs resolves a typed value from a dix runtime and panics on failure.
func MustResolveRuntimeAs[T any](rt *dix.Runtime) T {
	value, err := ResolveRuntimeAs[T](rt)
	if err != nil {
		panic(oops.In("dix/advanced").
			With("op", "must_resolve_runtime", "service", typedName[T]()).
			Wrapf(err, "resolve runtime dependency"))
	}
	return value
}

// ResolveNamedAs resolves a named value from a dix container.
func ResolveNamedAs[T any](c *dix.Container, name string) (T, error) {
	if c == nil || c.Raw() == nil {
		var zero T
		return zero, oops.In("dix/advanced").
			With("op", "resolve_named", "name", name).
			New("container is nil")
	}
	return do.InvokeNamed[T](c.Raw(), name)
}

// MustResolveNamedAs resolves a named value from a dix container and panics on failure.
func MustResolveNamedAs[T any](c *dix.Container, name string) T {
	value, err := ResolveNamedAs[T](c, name)
	if err != nil {
		panic(oops.In("dix/advanced").
			With("op", "must_resolve_named", "name", name).
			Wrapf(err, "resolve named dependency"))
	}
	return value
}

// ResolveAssignableAs resolves an assignable value from a dix container.
func ResolveAssignableAs[T any](c *dix.Container) (T, error) {
	if c == nil || c.Raw() == nil {
		var zero T
		return zero, oops.In("dix/advanced").
			With("op", "resolve_assignable").
			New("container is nil")
	}
	return do.InvokeAs[T](c.Raw())
}

// MustResolveAssignableAs resolves an assignable value from a dix container and panics on failure.
func MustResolveAssignableAs[T any](c *dix.Container) T {
	value, err := ResolveAssignableAs[T](c)
	if err != nil {
		panic(oops.In("dix/advanced").
			With("op", "must_resolve_assignable").
			Wrapf(err, "resolve assignable dependency"))
	}
	return value
}
