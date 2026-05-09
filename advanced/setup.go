package advanced

import (
	"github.com/arcgolabs/dix"
	"github.com/samber/do/v2"
)

// DoSetup registers a raw setup against a do injector.
func DoSetup(fn func(do.Injector) error) dix.SetupFunc {
	return DoSetupWithMetadata(fn, dix.SetupMetadata{
		Label:         "DoSetup",
		GraphMutation: true,
	})
}

// DoSetupWithMetadata registers a raw setup against a do injector with metadata.
func DoSetupWithMetadata(fn func(do.Injector) error, meta dix.SetupMetadata) dix.SetupFunc {
	meta.Raw = true
	return dix.NewSetupFunc(func(c *dix.Container, _ dix.Lifecycle) error {
		return fn(c.Raw())
	}, meta)
}

// BindAlias binds one typed service to another interface or alias type.
func BindAlias[From, To any]() dix.SetupFunc {
	return newSetup("BindAlias", func(c *dix.Container) error {
		return do.As[From, To](c.Raw())
	}, dix.ServiceRefs(dix.TypedService[From]()), dix.ServiceRefs(dix.TypedService[To]()), nil)
}

// Alias binds one typed service to another interface or alias type.
func Alias[From, To any]() dix.SetupFunc {
	return BindAlias[From, To]()
}

// BindNamedAlias binds one named service key to another named alias key.
func BindNamedAlias[From, To any](source dix.ServiceKey[From], alias dix.ServiceKey[To]) dix.SetupFunc {
	sourceName := source.Name()
	aliasName := alias.Name()
	return newSetup("BindNamedAlias", func(c *dix.Container) error {
		return do.AsNamed[From, To](c.Raw(), sourceName, aliasName)
	}, dix.ServiceRefs(source.Ref()), dix.ServiceRefs(alias.Ref()), nil)
}

// NamedAlias binds one named service key to another named alias key.
func NamedAlias[From, To any](source dix.ServiceKey[From], alias dix.ServiceKey[To]) dix.SetupFunc {
	return BindNamedAlias[From, To](source, alias)
}

// OverrideValue overrides a typed value registration.
func OverrideValue[T any](value T) dix.SetupFunc {
	return NamedOverrideValue(dix.TypedServiceKey[T](), value)
}

// NamedOverrideValue overrides a named value registration.
func NamedOverrideValue[T any](key dix.ServiceKey[T], value T) dix.SetupFunc {
	name := key.Name()
	return newSetup("OverrideValue", func(c *dix.Container) error {
		do.OverrideNamedValue(c.Raw(), name, value)
		return nil
	}, nil, nil, dix.ServiceRefs(key.Ref()))
}

// Override0 overrides a typed provider with no dependencies.
func Override0[T any](fn func() T) dix.SetupFunc {
	return NamedOverride0(dix.TypedServiceKey[T](), fn)
}

// Override overrides a typed provider with no dependencies.
func Override[T any](fn func() T) dix.SetupFunc {
	return Override0(fn)
}

// OverrideErr0 overrides a typed provider with no dependencies.
func OverrideErr0[T any](fn func() (T, error)) dix.SetupFunc {
	return NamedOverrideErr0(dix.TypedServiceKey[T](), fn)
}

// OverrideErr overrides a typed provider with no dependencies.
func OverrideErr[T any](fn func() (T, error)) dix.SetupFunc {
	return OverrideErr0(fn)
}

// NamedOverride0 overrides a named provider with no dependencies.
func NamedOverride0[T any](key dix.ServiceKey[T], fn func() T) dix.SetupFunc {
	name := key.Name()
	return newSetup("Override0", func(c *dix.Container) error {
		do.OverrideNamed(c.Raw(), name, func(do.Injector) (T, error) { return fn(), nil })
		return nil
	}, nil, nil, dix.ServiceRefs(key.Ref()))
}

// NamedOverrideErr0 overrides a named provider with no dependencies.
func NamedOverrideErr0[T any](key dix.ServiceKey[T], fn func() (T, error)) dix.SetupFunc {
	name := key.Name()
	return newSetup("OverrideErr0", func(c *dix.Container) error {
		do.OverrideNamed(c.Raw(), name, func(do.Injector) (T, error) { return fn() })
		return nil
	}, nil, nil, dix.ServiceRefs(key.Ref()))
}

// Override1 overrides a typed provider with one dependency.
func Override1[T, D1 any](fn func(D1) T) dix.SetupFunc {
	return NamedOverride1(dix.TypedServiceKey[T](), fn)
}

// OverrideErr1 overrides a typed provider with one dependency.
func OverrideErr1[T, D1 any](fn func(D1) (T, error)) dix.SetupFunc {
	return NamedOverrideErr1(dix.TypedServiceKey[T](), fn)
}

// NamedOverride1 overrides a named provider with one dependency.
func NamedOverride1[T, D1 any](key dix.ServiceKey[T], fn func(D1) T) dix.SetupFunc {
	name := key.Name()
	return newSetup("Override1", func(c *dix.Container) error {
		do.OverrideNamed(c.Raw(), name, func(i do.Injector) (T, error) {
			d1, err := invokeTyped[D1](i)
			if err != nil {
				var zero T
				return zero, err
			}
			return fn(d1), nil
		})
		return nil
	}, dix.ServiceRefs(dix.TypedService[D1]()), nil, dix.ServiceRefs(key.Ref()))
}

// NamedOverrideErr1 overrides a named provider with one dependency.
func NamedOverrideErr1[T, D1 any](key dix.ServiceKey[T], fn func(D1) (T, error)) dix.SetupFunc {
	name := key.Name()
	return newSetup("OverrideErr1", func(c *dix.Container) error {
		do.OverrideNamed(c.Raw(), name, func(i do.Injector) (T, error) {
			d1, err := invokeTyped[D1](i)
			if err != nil {
				var zero T
				return zero, err
			}
			return fn(d1)
		})
		return nil
	}, dix.ServiceRefs(dix.TypedService[D1]()), nil, dix.ServiceRefs(key.Ref()))
}

// OverrideTransient0 overrides a typed transient provider with no dependencies.
func OverrideTransient0[T any](fn func() T) dix.SetupFunc {
	return NamedOverrideTransient0(dix.TypedServiceKey[T](), fn)
}

// TransientOverride overrides a typed transient provider with no dependencies.
func TransientOverride[T any](fn func() T) dix.SetupFunc {
	return OverrideTransient0(fn)
}

// OverrideTransientErr0 overrides a typed transient provider with no dependencies.
func OverrideTransientErr0[T any](fn func() (T, error)) dix.SetupFunc {
	return NamedOverrideTransientErr0(dix.TypedServiceKey[T](), fn)
}

// TransientOverrideErr overrides a typed transient provider with no dependencies.
func TransientOverrideErr[T any](fn func() (T, error)) dix.SetupFunc {
	return OverrideTransientErr0(fn)
}

// NamedOverrideTransient0 overrides a named transient provider with no dependencies.
func NamedOverrideTransient0[T any](key dix.ServiceKey[T], fn func() T) dix.SetupFunc {
	name := key.Name()
	return newSetup("OverrideTransient0", func(c *dix.Container) error {
		do.OverrideNamedTransient(c.Raw(), name, func(do.Injector) (T, error) { return fn(), nil })
		return nil
	}, nil, nil, dix.ServiceRefs(key.Ref()))
}

// NamedOverrideTransientErr0 overrides a named transient provider with no dependencies.
func NamedOverrideTransientErr0[T any](key dix.ServiceKey[T], fn func() (T, error)) dix.SetupFunc {
	name := key.Name()
	return newSetup("OverrideTransientErr0", func(c *dix.Container) error {
		do.OverrideNamedTransient(c.Raw(), name, func(do.Injector) (T, error) { return fn() })
		return nil
	}, nil, nil, dix.ServiceRefs(key.Ref()))
}

// OverrideTransient1 overrides a typed transient provider with one dependency.
func OverrideTransient1[T, D1 any](fn func(D1) T) dix.SetupFunc {
	return NamedOverrideTransient1(dix.TypedServiceKey[T](), fn)
}

// OverrideTransientErr1 overrides a typed transient provider with one dependency.
func OverrideTransientErr1[T, D1 any](fn func(D1) (T, error)) dix.SetupFunc {
	return NamedOverrideTransientErr1(dix.TypedServiceKey[T](), fn)
}

// NamedOverrideTransient1 overrides a named transient provider with one dependency.
func NamedOverrideTransient1[T, D1 any](key dix.ServiceKey[T], fn func(D1) T) dix.SetupFunc {
	name := key.Name()
	return newSetup("OverrideTransient1", func(c *dix.Container) error {
		do.OverrideNamedTransient(c.Raw(), name, func(i do.Injector) (T, error) {
			d1, err := invokeTyped[D1](i)
			if err != nil {
				var zero T
				return zero, err
			}
			return fn(d1), nil
		})
		return nil
	}, dix.ServiceRefs(dix.TypedService[D1]()), nil, dix.ServiceRefs(key.Ref()))
}

// NamedOverrideTransientErr1 overrides a named transient provider with one dependency.
func NamedOverrideTransientErr1[T, D1 any](key dix.ServiceKey[T], fn func(D1) (T, error)) dix.SetupFunc {
	name := key.Name()
	return newSetup("OverrideTransientErr1", func(c *dix.Container) error {
		do.OverrideNamedTransient(c.Raw(), name, func(i do.Injector) (T, error) {
			d1, err := invokeTyped[D1](i)
			if err != nil {
				var zero T
				return zero, err
			}
			return fn(d1)
		})
		return nil
	}, dix.ServiceRefs(dix.TypedService[D1]()), nil, dix.ServiceRefs(key.Ref()))
}
