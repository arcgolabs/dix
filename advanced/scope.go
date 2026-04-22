package advanced

import (
	"github.com/arcgolabs/collectionx"
	"github.com/arcgolabs/dix"
	"github.com/samber/do/v2"
)

// ScopePackage configures a newly created do scope.
type ScopePackage func(do.Injector)

// Scope creates a named child scope from a runtime injector.
func Scope(rt *dix.Runtime, name string, packages ...ScopePackage) *do.Scope {
	if rt == nil {
		return nil
	}

	wrapped := collectionx.NewListWithCapacity[func(do.Injector)](len(packages))
	for _, pkg := range packages {
		if pkg == nil {
			continue
		}
		current := pkg
		wrapped.Add(func(injector do.Injector) {
			current(injector)
		})
	}

	switch wrapped.Len() {
	case 0:
		return rt.Raw().Scope(name)
	case 1:
		first, _ := wrapped.Get(0)
		return rt.Raw().Scope(name, first)
	default:
		return rt.Raw().Scope(name, wrapped.Values()...)
	}
}

// ProvideScopedValue registers a typed singleton value in a scope.
func ProvideScopedValue[T any](injector do.Injector, value T) {
	do.ProvideNamedValue(injector, typedName[T](), value)
}

// ProvideScopedNamedValue registers a named singleton value in a scope.
func ProvideScopedNamedValue[T any](injector do.Injector, name string, value T) {
	do.ProvideNamedValue(injector, name, value)
}

// ProvideScoped0 registers a typed scoped provider with no dependencies.
func ProvideScoped0[T any](injector do.Injector, fn func() T) {
	ProvideScopedErr0(injector, func() (T, error) { return fn(), nil })
}

// ProvideScopedErr0 registers a typed scoped provider with no dependencies.
func ProvideScopedErr0[T any](injector do.Injector, fn func() (T, error)) {
	do.ProvideNamed(injector, typedName[T](), func(do.Injector) (T, error) {
		return fn()
	})
}

// ProvideScopedNamed0 registers a named scoped provider with no dependencies.
func ProvideScopedNamed0[T any](injector do.Injector, name string, fn func() T) {
	ProvideScopedNamedErr0(injector, name, func() (T, error) { return fn(), nil })
}

// ProvideScopedNamedErr0 registers a named scoped provider with no dependencies.
func ProvideScopedNamedErr0[T any](injector do.Injector, name string, fn func() (T, error)) {
	do.ProvideNamed(injector, name, func(do.Injector) (T, error) {
		return fn()
	})
}

// ProvideScoped1 registers a typed scoped provider with one dependency.
func ProvideScoped1[T, D1 any](injector do.Injector, fn func(D1) T) {
	ProvideScopedErr1(injector, func(d1 D1) (T, error) { return fn(d1), nil })
}

// ProvideScopedErr1 registers a typed scoped provider with one dependency.
func ProvideScopedErr1[T, D1 any](injector do.Injector, fn func(D1) (T, error)) {
	do.ProvideNamed(injector, typedName[T](), func(i do.Injector) (T, error) {
		d1, err := invokeTyped[D1](i)
		if err != nil {
			var zero T
			return zero, err
		}
		return fn(d1)
	})
}

// ProvideScopedNamed1 registers a named scoped provider with one dependency.
func ProvideScopedNamed1[T, D1 any](injector do.Injector, name string, fn func(D1) T) {
	ProvideScopedNamedErr1(injector, name, func(d1 D1) (T, error) { return fn(d1), nil })
}

// ProvideScopedNamedErr1 registers a named scoped provider with one dependency.
func ProvideScopedNamedErr1[T, D1 any](injector do.Injector, name string, fn func(D1) (T, error)) {
	do.ProvideNamed(injector, name, func(i do.Injector) (T, error) {
		d1, err := invokeTyped[D1](i)
		if err != nil {
			var zero T
			return zero, err
		}
		return fn(d1)
	})
}

// ProvideScoped2 registers a typed scoped provider with two dependencies.
func ProvideScoped2[T, D1, D2 any](injector do.Injector, fn func(D1, D2) T) {
	ProvideScopedErr2(injector, func(d1 D1, d2 D2) (T, error) { return fn(d1, d2), nil })
}

// ProvideScopedErr2 registers a typed scoped provider with two dependencies.
func ProvideScopedErr2[T, D1, D2 any](injector do.Injector, fn func(D1, D2) (T, error)) {
	do.ProvideNamed(injector, typedName[T](), func(i do.Injector) (T, error) {
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
}

// ProvideScopedNamed2 registers a named scoped provider with two dependencies.
func ProvideScopedNamed2[T, D1, D2 any](injector do.Injector, name string, fn func(D1, D2) T) {
	ProvideScopedNamedErr2(injector, name, func(d1 D1, d2 D2) (T, error) { return fn(d1, d2), nil })
}

// ProvideScopedNamedErr2 registers a named scoped provider with two dependencies.
func ProvideScopedNamedErr2[T, D1, D2 any](injector do.Injector, name string, fn func(D1, D2) (T, error)) {
	do.ProvideNamed(injector, name, func(i do.Injector) (T, error) {
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
}

// ProvideScoped3 registers a typed scoped provider with three dependencies.
func ProvideScoped3[T, D1, D2, D3 any](injector do.Injector, fn func(D1, D2, D3) T) {
	ProvideScopedErr3(injector, func(d1 D1, d2 D2, d3 D3) (T, error) { return fn(d1, d2, d3), nil })
}

// ProvideScopedErr3 registers a typed scoped provider with three dependencies.
func ProvideScopedErr3[T, D1, D2, D3 any](injector do.Injector, fn func(D1, D2, D3) (T, error)) {
	do.ProvideNamed(injector, typedName[T](), func(i do.Injector) (T, error) {
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
}

// ProvideScopedNamed3 registers a named scoped provider with three dependencies.
func ProvideScopedNamed3[T, D1, D2, D3 any](injector do.Injector, name string, fn func(D1, D2, D3) T) {
	ProvideScopedNamedErr3(injector, name, func(d1 D1, d2 D2, d3 D3) (T, error) { return fn(d1, d2, d3), nil })
}

// ProvideScopedNamedErr3 registers a named scoped provider with three dependencies.
func ProvideScopedNamedErr3[T, D1, D2, D3 any](injector do.Injector, name string, fn func(D1, D2, D3) (T, error)) {
	do.ProvideNamed(injector, name, func(i do.Injector) (T, error) {
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
}

// ResolveScopedAs resolves a typed value from a scope injector.
func ResolveScopedAs[T any](injector do.Injector) (T, error) {
	return ResolveInjectorAs[T](injector)
}

// ResolveScopedNamedAs resolves a named value from a scope injector.
func ResolveScopedNamedAs[T any](injector do.Injector, name string) (T, error) {
	return do.InvokeNamed[T](injector, name)
}
