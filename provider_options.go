package dix

import (
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/arcgolabs/collectionx"
	"github.com/samber/do/v2"
)

var anonymousContributionID atomic.Uint64

// ProviderOption configures extra provider outputs such as aliases and collection contributions.
type ProviderOption func(*providerOptions)

// ContributionOption configures a provider contribution to a collection role.
type ContributionOption func(*contributionOptions)

type providerOptions struct {
	aliases       collectionx.List[aliasSpec]
	contributions collectionx.List[contributionOptionSpec]
}

type aliasSpec struct {
	target   ServiceRef
	register func(*Container, string)
}

type contributionOptionSpec struct {
	target   ServiceRef
	options  contributionOptions
	register func(*Container, string, string)
	factory  collectionFactory
}

type contributionOptions struct {
	key    string
	hasKey bool
	order  int
}

type collectionFactory struct {
	target   ServiceRef
	outputs  collectionx.List[ServiceRef]
	register func(*Container, collectionx.List[ContributionRef], serviceNameSet)
}

type contributionRegistration struct {
	ref      ContributionRef
	register func(*Container, string, string)
}

// As exposes a provider output as another typed service, usually an interface.
func As[T any]() ProviderOption {
	target := TypedService[T]()
	return func(opts *providerOptions) {
		if opts == nil {
			return
		}
		opts.aliases.Add(aliasSpec{
			target: target,
			register: func(c *Container, source string) {
				provideAlias[T](c, source, target.Name)
			},
		})
	}
}

// Into contributes a provider output into the collection role identified by T.
func Into[T any](options ...ContributionOption) ProviderOption {
	target := TypedService[T]()
	contributionOpts := contributionOptions{}
	for _, option := range options {
		if option != nil {
			option(&contributionOpts)
		}
	}
	return func(opts *providerOptions) {
		if opts == nil {
			return
		}
		opts.contributions.Add(contributionOptionSpec{
			target:  target,
			options: contributionOpts,
			register: func(c *Container, source, contribution string) {
				provideAlias[T](c, source, contribution)
			},
			factory: collectionFactory{
				target:  target,
				outputs: collectionServiceRefs[T](),
				register: func(c *Container, contributions collectionx.List[ContributionRef], explicit serviceNameSet) {
					registerCollectionProviders[T](c, contributions, explicit)
				},
			},
		})
	}
}

func collectionServiceRefs[T any]() collectionx.List[ServiceRef] {
	return ServiceRefs(
		TypedService[collectionx.List[T]](),
		TypedService[map[string]T](),
		TypedService[collectionx.Map[string, T]](),
		TypedService[collectionx.OrderedMap[string, T]](),
	)
}

// Key assigns the map key for a collection contribution.
func Key(key string) ContributionOption {
	return func(opts *contributionOptions) {
		if opts == nil {
			return
		}
		opts.key = strings.TrimSpace(key)
		opts.hasKey = opts.key != ""
	}
}

// Order assigns the relative order for collectionx list injection.
func Order(order int) ContributionOption {
	return func(opts *contributionOptions) {
		if opts != nil {
			opts.order = order
		}
	}
}

func newTypedProviderFunc[T any](
	label string,
	register func(*Container),
	deps collectionx.List[ServiceRef],
	opts ...ProviderOption,
) ProviderFunc {
	output := TypedService[T]()
	providerOpts := applyProviderOptions(opts...)
	contributions, factories, contributionRegistrations := buildContributionRefs(output.Name, providerOpts.contributions)
	aliases := aliasRefs(providerOpts.aliases)

	return NewProviderFuncWithCollections(
		func(c *Container) {
			register(c)
			registerAliases(c, output.Name, providerOpts.aliases)
			registerContributionAliases(c, output.Name, contributionRegistrations)
		},
		ProviderMetadata{
			Label:         label,
			Output:        output,
			Dependencies:  deps,
			Aliases:       aliases,
			Contributions: contributions,
		},
		factories,
	)
}

func applyProviderOptions(opts ...ProviderOption) providerOptions {
	out := providerOptions{
		aliases:       collectionx.NewList[aliasSpec](),
		contributions: collectionx.NewList[contributionOptionSpec](),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&out)
		}
	}
	return out
}

func aliasRefs(aliases collectionx.List[aliasSpec]) collectionx.List[ServiceRef] {
	out := collectionx.NewListWithCapacity[ServiceRef](aliases.Len())
	aliases.Range(func(_ int, alias aliasSpec) bool {
		out.Add(alias.target)
		return true
	})
	return out
}

func buildContributionRefs(
	sourceName string,
	specs collectionx.List[contributionOptionSpec],
) (
	collectionx.List[ContributionRef],
	collectionx.List[collectionFactory],
	collectionx.List[contributionRegistration],
) {
	refs := collectionx.NewListWithCapacity[ContributionRef](specs.Len())
	factories := collectionx.NewListWithCapacity[collectionFactory](specs.Len())
	registrations := collectionx.NewListWithCapacity[contributionRegistration](specs.Len())
	specs.Range(func(index int, spec contributionOptionSpec) bool {
		service := contributionServiceName(spec.target.Name, sourceName, index)
		ref := ContributionRef{
			Target:  spec.target,
			Service: NamedService(service),
			Key:     spec.options.key,
			HasKey:  spec.options.hasKey,
			Order:   spec.options.order,
		}
		refs.Add(ref)
		factories.Add(spec.factory)
		registrations.Add(contributionRegistration{
			ref:      ref,
			register: spec.register,
		})
		return true
	})
	return refs, factories, registrations
}

func contributionServiceName(targetName, sourceName string, index int) string {
	return "dix:contribution:" + targetName + ":" + sourceName + ":" + strconv.Itoa(index)
}

func anonymousContributionServiceName(targetName string) string {
	id := anonymousContributionID.Add(1)
	return "dix:contribution:" + targetName + ":anonymous:" + strconv.FormatUint(id, 10)
}

func registerAliases(c *Container, source string, aliases collectionx.List[aliasSpec]) {
	aliases.Range(func(_ int, alias aliasSpec) bool {
		alias.register(c, source)
		return true
	})
}

func registerContributionAliases(c *Container, source string, registrations collectionx.List[contributionRegistration]) {
	registrations.Range(func(_ int, registration contributionRegistration) bool {
		registration.register(c, source, registration.ref.Service.Name)
		return true
	})
}

func provideAlias[T any](c *Container, source, alias string) {
	do.ProvideNamed(c.Raw(), alias, func(i do.Injector) (T, error) {
		return resolveNamedAnyAs[T](i, source, alias)
	})
}

func resolveNamedAnyAs[T any](i do.Injector, source, target string) (T, error) {
	raw, err := resolveNamedAny(i, source)
	if err != nil {
		var zero T
		return zero, err
	}
	value, ok := raw.(T)
	if !ok {
		var zero T
		return zero, fmt.Errorf("dix: service `%s` is not assignable to `%s`", source, target)
	}
	return value, nil
}

func resolveNamedAny(i do.Injector, name string) (any, error) {
	return do.InvokeNamed[any](i, name)
}
