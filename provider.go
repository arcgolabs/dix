package dix

import "github.com/DaiYuANg/arcgo/collectionx"

// ProviderFunc describes a typed provider registration.
type ProviderFunc struct {
	register            func(*Container)
	meta                ProviderMetadata
	collectionFactories collectionx.List[collectionFactory]
}

func (p ProviderFunc) apply(c *Container) {
	if p.register != nil {
		p.register(c)
	}
}

// RawProvider registers an untyped provider callback.
func RawProvider(fn func(*Container)) ProviderFunc {
	return RawProviderWithMetadata(fn, ProviderMetadata{
		Label: "RawProvider",
	})
}

// RawProviderWithMetadata registers an untyped provider callback with metadata.
func RawProviderWithMetadata(fn func(*Container), meta ProviderMetadata) ProviderFunc {
	return NewProviderFunc(fn, ProviderMetadata{
		Label:         meta.Label,
		Output:        meta.Output,
		Dependencies:  meta.Dependencies,
		Aliases:       meta.Aliases,
		Contributions: meta.Contributions,
		Raw:           true,
	})
}

// Value registers a typed singleton value.
func Value[T any](value T, opts ...ProviderOption) ProviderFunc {
	return newTypedProviderFunc[T](
		"Value",
		func(c *Container) { ProvideValueT(c, value) },
		nil,
		opts...,
	)
}

// Provider0 registers a typed singleton provider with no dependencies.
func Provider0[T any](fn func() T, opts ...ProviderOption) ProviderFunc {
	return newTypedProviderFunc[T](
		"Provider0",
		func(c *Container) { ProvideT(c, fn) },
		nil,
		opts...,
	)
}

// ProviderErr0 registers a typed singleton provider with no dependencies.
func ProviderErr0[T any](fn func() (T, error), opts ...ProviderOption) ProviderFunc {
	return newTypedProviderFunc[T](
		"ProviderErr0",
		func(c *Container) { ProvideTErr(c, fn) },
		nil,
		opts...,
	)
}

// Provider1 registers a typed singleton provider with one dependency.
func Provider1[T, D1 any](fn func(D1) T, opts ...ProviderOption) ProviderFunc {
	return newTypedProviderFunc[T](
		"Provider1",
		func(c *Container) { Provide1T(c, fn) },
		ServiceRefs(TypedService[D1]()),
		opts...,
	)
}

// ProviderErr1 registers a typed singleton provider with one dependency.
func ProviderErr1[T, D1 any](fn func(D1) (T, error), opts ...ProviderOption) ProviderFunc {
	return newTypedProviderFunc[T](
		"ProviderErr1",
		func(c *Container) { Provide1TErr(c, fn) },
		ServiceRefs(TypedService[D1]()),
		opts...,
	)
}

// Provider2 registers a typed singleton provider with two dependencies.
func Provider2[T, D1, D2 any](fn func(D1, D2) T, opts ...ProviderOption) ProviderFunc {
	return newTypedProviderFunc[T](
		"Provider2",
		func(c *Container) { Provide2T(c, fn) },
		ServiceRefs(TypedService[D1](), TypedService[D2]()),
		opts...,
	)
}

// ProviderErr2 registers a typed singleton provider with two dependencies.
func ProviderErr2[T, D1, D2 any](fn func(D1, D2) (T, error), opts ...ProviderOption) ProviderFunc {
	return newTypedProviderFunc[T](
		"ProviderErr2",
		func(c *Container) { Provide2TErr(c, fn) },
		ServiceRefs(TypedService[D1](), TypedService[D2]()),
		opts...,
	)
}

// Provider3 registers a typed singleton provider with three dependencies.
func Provider3[T, D1, D2, D3 any](fn func(D1, D2, D3) T, opts ...ProviderOption) ProviderFunc {
	return newTypedProviderFunc[T](
		"Provider3",
		func(c *Container) { Provide3T(c, fn) },
		ServiceRefs(TypedService[D1](), TypedService[D2](), TypedService[D3]()),
		opts...,
	)
}

// ProviderErr3 registers a typed singleton provider with three dependencies.
func ProviderErr3[T, D1, D2, D3 any](fn func(D1, D2, D3) (T, error), opts ...ProviderOption) ProviderFunc {
	return newTypedProviderFunc[T](
		"ProviderErr3",
		func(c *Container) { Provide3TErr(c, fn) },
		ServiceRefs(TypedService[D1](), TypedService[D2](), TypedService[D3]()),
		opts...,
	)
}

// Provider4 registers a typed singleton provider with four dependencies.
func Provider4[T, D1, D2, D3, D4 any](fn func(D1, D2, D3, D4) T, opts ...ProviderOption) ProviderFunc {
	return newTypedProviderFunc[T](
		"Provider4",
		func(c *Container) { Provide4T(c, fn) },
		ServiceRefs(TypedService[D1](), TypedService[D2](), TypedService[D3](), TypedService[D4]()),
		opts...,
	)
}

// ProviderErr4 registers a typed singleton provider with four dependencies.
func ProviderErr4[T, D1, D2, D3, D4 any](fn func(D1, D2, D3, D4) (T, error), opts ...ProviderOption) ProviderFunc {
	return newTypedProviderFunc[T](
		"ProviderErr4",
		func(c *Container) { Provide4TErr(c, fn) },
		ServiceRefs(TypedService[D1](), TypedService[D2](), TypedService[D3](), TypedService[D4]()),
		opts...,
	)
}

// Provider5 registers a typed singleton provider with five dependencies.
func Provider5[T, D1, D2, D3, D4, D5 any](fn func(D1, D2, D3, D4, D5) T, opts ...ProviderOption) ProviderFunc {
	return newTypedProviderFunc[T](
		"Provider5",
		func(c *Container) { Provide5T(c, fn) },
		ServiceRefs(TypedService[D1](), TypedService[D2](), TypedService[D3](), TypedService[D4](), TypedService[D5]()),
		opts...,
	)
}

// ProviderErr5 registers a typed singleton provider with five dependencies.
func ProviderErr5[T, D1, D2, D3, D4, D5 any](fn func(D1, D2, D3, D4, D5) (T, error), opts ...ProviderOption) ProviderFunc {
	return newTypedProviderFunc[T](
		"ProviderErr5",
		func(c *Container) { Provide5TErr(c, fn) },
		ServiceRefs(TypedService[D1](), TypedService[D2](), TypedService[D3](), TypedService[D4](), TypedService[D5]()),
		opts...,
	)
}

// Provider6 registers a typed singleton provider with six dependencies.
func Provider6[T, D1, D2, D3, D4, D5, D6 any](fn func(D1, D2, D3, D4, D5, D6) T, opts ...ProviderOption) ProviderFunc {
	return newTypedProviderFunc[T](
		"Provider6",
		func(c *Container) { Provide6T(c, fn) },
		ServiceRefs(
			TypedService[D1](),
			TypedService[D2](),
			TypedService[D3](),
			TypedService[D4](),
			TypedService[D5](),
			TypedService[D6](),
		),
		opts...,
	)
}

// ProviderErr6 registers a typed singleton provider with six dependencies.
func ProviderErr6[T, D1, D2, D3, D4, D5, D6 any](fn func(D1, D2, D3, D4, D5, D6) (T, error), opts ...ProviderOption) ProviderFunc {
	return newTypedProviderFunc[T](
		"ProviderErr6",
		func(c *Container) { Provide6TErr(c, fn) },
		ServiceRefs(
			TypedService[D1](),
			TypedService[D2](),
			TypedService[D3](),
			TypedService[D4](),
			TypedService[D5](),
			TypedService[D6](),
		),
		opts...,
	)
}
