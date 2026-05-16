package dix

import "github.com/samber/mo"

// Conditional is an injectable optional service value.
type Conditional[T any] = mo.Option[T]

// ConditionalProvider0 registers a provider that returns an optional service value.
func ConditionalProvider0[T any](fn func() Conditional[T], opts ...ProviderOption) ProviderFunc {
	return newTypedProviderFunc[Conditional[T]](
		"ConditionalProvider0",
		func(c *Container) { ProvideT(c, fn) },
		nil,
		opts...,
	)
}

// ConditionalProviderErr0 registers a fallible provider that returns an optional service value.
func ConditionalProviderErr0[T any](fn func() (Conditional[T], error), opts ...ProviderOption) ProviderFunc {
	return newTypedProviderFunc[Conditional[T]](
		"ConditionalProviderErr0",
		func(c *Container) { ProvideTErr(c, fn) },
		nil,
		opts...,
	)
}

// ConditionalProvider1 registers a one-dependency provider that returns an optional service value.
func ConditionalProvider1[T, D1 any](fn func(D1) Conditional[T], opts ...ProviderOption) ProviderFunc {
	return newTypedProviderFunc[Conditional[T]](
		"ConditionalProvider1",
		func(c *Container) { Provide1T(c, fn) },
		ServiceRefs(TypedService[D1]()),
		opts...,
	)
}

// ConditionalProviderErr1 registers a fallible one-dependency provider that returns an optional service value.
func ConditionalProviderErr1[T, D1 any](fn func(D1) (Conditional[T], error), opts ...ProviderOption) ProviderFunc {
	return newTypedProviderFunc[Conditional[T]](
		"ConditionalProviderErr1",
		func(c *Container) { Provide1TErr(c, fn) },
		ServiceRefs(TypedService[D1]()),
		opts...,
	)
}

// ConditionalProvider2 registers a two-dependency provider that returns an optional service value.
func ConditionalProvider2[T, D1, D2 any](fn func(D1, D2) Conditional[T], opts ...ProviderOption) ProviderFunc {
	return newTypedProviderFunc[Conditional[T]](
		"ConditionalProvider2",
		func(c *Container) { Provide2T(c, fn) },
		ServiceRefs(TypedService[D1](), TypedService[D2]()),
		opts...,
	)
}

// ConditionalProviderErr2 registers a fallible two-dependency provider that returns an optional service value.
func ConditionalProviderErr2[T, D1, D2 any](
	fn func(D1, D2) (Conditional[T], error),
	opts ...ProviderOption,
) ProviderFunc {
	return newTypedProviderFunc[Conditional[T]](
		"ConditionalProviderErr2",
		func(c *Container) { Provide2TErr(c, fn) },
		ServiceRefs(TypedService[D1](), TypedService[D2]()),
		opts...,
	)
}

// ConditionalProvider3 registers a three-dependency provider that returns an optional service value.
func ConditionalProvider3[T, D1, D2, D3 any](fn func(D1, D2, D3) Conditional[T], opts ...ProviderOption) ProviderFunc {
	return newTypedProviderFunc[Conditional[T]](
		"ConditionalProvider3",
		func(c *Container) { Provide3T(c, fn) },
		ServiceRefs(TypedService[D1](), TypedService[D2](), TypedService[D3]()),
		opts...,
	)
}

// ConditionalProviderErr3 registers a fallible three-dependency provider that returns an optional service value.
func ConditionalProviderErr3[T, D1, D2, D3 any](
	fn func(D1, D2, D3) (Conditional[T], error),
	opts ...ProviderOption,
) ProviderFunc {
	return newTypedProviderFunc[Conditional[T]](
		"ConditionalProviderErr3",
		func(c *Container) { Provide3TErr(c, fn) },
		ServiceRefs(TypedService[D1](), TypedService[D2](), TypedService[D3]()),
		opts...,
	)
}
