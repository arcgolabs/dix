package dix

import (
	"github.com/arcgolabs/collectionx"
	"github.com/samber/do/v2"
)

// Contribute0 registers a provider only as a contribution to the collection role T.
func Contribute0[T any](fn func() T, opts ...ContributionOption) ProviderFunc {
	return newContributionProviderFunc[T]("Contribute0", func(c *Container, service string) {
		do.ProvideNamed(c.Raw(), service, func(do.Injector) (T, error) { return fn(), nil })
	}, nil, opts...)
}

// ContributeErr0 registers an error-returning provider only as a contribution to the collection role T.
func ContributeErr0[T any](fn func() (T, error), opts ...ContributionOption) ProviderFunc {
	return newContributionProviderFunc[T]("ContributeErr0", func(c *Container, service string) {
		do.ProvideNamed(c.Raw(), service, func(do.Injector) (T, error) { return fn() })
	}, nil, opts...)
}

// Contribute1 registers a one-dependency provider only as a contribution to the collection role T.
func Contribute1[T, D1 any](fn func(D1) T, opts ...ContributionOption) ProviderFunc {
	return newContributionProviderFunc[T]("Contribute1", func(c *Container, service string) {
		do.ProvideNamed(c.Raw(), service, func(i do.Injector) (T, error) {
			d1, err := resolveDependency1[D1](i)
			if err != nil {
				var zero T
				return zero, err
			}
			return fn(d1), nil
		})
	}, ServiceRefs(TypedService[D1]()), opts...)
}

// ContributeErr1 registers an error-returning one-dependency provider only as a contribution to the collection role T.
func ContributeErr1[T, D1 any](fn func(D1) (T, error), opts ...ContributionOption) ProviderFunc {
	return newContributionProviderFunc[T]("ContributeErr1", func(c *Container, service string) {
		do.ProvideNamed(c.Raw(), service, func(i do.Injector) (T, error) {
			d1, err := resolveDependency1[D1](i)
			if err != nil {
				var zero T
				return zero, err
			}
			return fn(d1)
		})
	}, ServiceRefs(TypedService[D1]()), opts...)
}

// Contribute2 registers a two-dependency provider only as a contribution to the collection role T.
func Contribute2[T, D1, D2 any](fn func(D1, D2) T, opts ...ContributionOption) ProviderFunc {
	return newContributionProviderFunc[T]("Contribute2", func(c *Container, service string) {
		do.ProvideNamed(c.Raw(), service, func(i do.Injector) (T, error) {
			d1, d2, err := resolveDependencies2[D1, D2](i)
			if err != nil {
				var zero T
				return zero, err
			}
			return fn(d1, d2), nil
		})
	}, ServiceRefs(TypedService[D1](), TypedService[D2]()), opts...)
}

// ContributeErr2 registers an error-returning two-dependency provider only as a contribution to the collection role T.
func ContributeErr2[T, D1, D2 any](fn func(D1, D2) (T, error), opts ...ContributionOption) ProviderFunc {
	return newContributionProviderFunc[T]("ContributeErr2", func(c *Container, service string) {
		do.ProvideNamed(c.Raw(), service, func(i do.Injector) (T, error) {
			d1, d2, err := resolveDependencies2[D1, D2](i)
			if err != nil {
				var zero T
				return zero, err
			}
			return fn(d1, d2)
		})
	}, ServiceRefs(TypedService[D1](), TypedService[D2]()), opts...)
}

// Contribute3 registers a three-dependency provider only as a contribution to the collection role T.
func Contribute3[T, D1, D2, D3 any](fn func(D1, D2, D3) T, opts ...ContributionOption) ProviderFunc {
	return newContributionProviderFunc[T]("Contribute3", func(c *Container, service string) {
		do.ProvideNamed(c.Raw(), service, func(i do.Injector) (T, error) {
			d1, d2, d3, err := resolveDependencies3[D1, D2, D3](i)
			if err != nil {
				var zero T
				return zero, err
			}
			return fn(d1, d2, d3), nil
		})
	}, ServiceRefs(TypedService[D1](), TypedService[D2](), TypedService[D3]()), opts...)
}

// ContributeErr3 registers an error-returning three-dependency provider only as a contribution to the collection role T.
func ContributeErr3[T, D1, D2, D3 any](fn func(D1, D2, D3) (T, error), opts ...ContributionOption) ProviderFunc {
	return newContributionProviderFunc[T]("ContributeErr3", func(c *Container, service string) {
		do.ProvideNamed(c.Raw(), service, func(i do.Injector) (T, error) {
			d1, d2, d3, err := resolveDependencies3[D1, D2, D3](i)
			if err != nil {
				var zero T
				return zero, err
			}
			return fn(d1, d2, d3)
		})
	}, ServiceRefs(TypedService[D1](), TypedService[D2](), TypedService[D3]()), opts...)
}

func newContributionProviderFunc[T any](
	label string,
	register func(*Container, string),
	deps collectionx.List[ServiceRef],
	opts ...ContributionOption,
) ProviderFunc {
	target := TypedService[T]()
	contributionOpts := contributionOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(&contributionOpts)
		}
	}
	service := anonymousContributionServiceName(target.Name)
	ref := ContributionRef{
		Target:  target,
		Service: NamedService(service),
		Key:     contributionOpts.key,
		HasKey:  contributionOpts.hasKey,
		Order:   contributionOpts.order,
	}
	factory := collectionFactory{
		target:  target,
		outputs: collectionServiceRefs[T](),
		register: func(c *Container, contributions collectionx.List[ContributionRef], explicit serviceNameSet) {
			registerCollectionProviders[T](c, contributions, explicit)
		},
	}

	return NewProviderFuncWithCollections(
		func(c *Container) {
			register(c, service)
		},
		ProviderMetadata{
			Label:         label,
			Output:        NamedService(service),
			Dependencies:  deps,
			Contributions: collectionx.NewList(ref),
		},
		collectionx.NewList(factory),
	)
}
