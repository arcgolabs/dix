package dix

import (
	collectionlist "github.com/arcgolabs/collectionx/list"
	"time"
)

// ServiceRef identifies a service in the container graph.
// Typed services should use TypedService[T](). Named services should use NamedService[T](name).Ref().
type ServiceRef struct {
	Name string
}

// Ref returns the service reference itself.
func (r ServiceRef) Ref() ServiceRef {
	return r
}

// ServiceKey is a typed handle to a service identity.
//
// A zero-value ServiceKey[T] identifies the default typed service T. Use
// NamedService[T](name) for named services with compile-time value type attached.
type ServiceKey[T any] struct {
	name string
}

// TypedServiceKey returns a typed key for the default service T.
func TypedServiceKey[T any]() ServiceKey[T] {
	return ServiceKey[T]{}
}

// TypedService returns a typed service reference for T.
func TypedService[T any]() ServiceRef {
	return ServiceRef{Name: serviceNameOf[T]()}
}

// NamedService returns a typed key for a named service.
func NamedService[T any](name string) ServiceKey[T] {
	return ServiceKey[T]{name: name}
}

// Name returns the concrete service name represented by this key.
func (k ServiceKey[T]) Name() string {
	return k.nameWith(nil)
}

// Ref returns a graph service reference for this key.
func (k ServiceKey[T]) Ref() ServiceRef {
	return ServiceRef{Name: k.Name()}
}

func (k ServiceKey[T]) nameWith(n *serviceNamer) string {
	if k.name != "" {
		return k.name
	}
	return serviceNameOfWith[T](n)
}

func namedServiceRef(name string) ServiceRef {
	return ServiceRef{Name: name}
}

// ProviderMetadata describes a provider registration for validation and inspection.
type ProviderMetadata struct {
	Label         string
	Output        ServiceRef
	Dependencies  *collectionlist.List[ServiceRef]
	Aliases       *collectionlist.List[ServiceRef]
	Contributions *collectionlist.List[ContributionRef]
	Conditions    *collectionlist.List[Condition]
	Eager         bool
	Raw           bool
}

// ContributionRef describes one provider contribution to a typed collection role.
type ContributionRef struct {
	Target  ServiceRef
	Service ServiceRef
	Key     string
	HasKey  bool
	Order   int

	sequence int
}

// InvokeMetadata describes an invoke registration for validation and inspection.
type InvokeMetadata struct {
	Label        string
	Dependencies *collectionlist.List[ServiceRef]
	Raw          bool
}

// HookKind identifies a lifecycle hook phase.
type HookKind string

const (
	// HookKindStart identifies start hooks.
	HookKindStart HookKind = "start"
	// HookKindStop identifies stop hooks.
	HookKindStop HookKind = "stop"
)

// HookMetadata describes a lifecycle hook registration.
type HookMetadata struct {
	Label        string
	Kind         HookKind
	Dependencies *collectionlist.List[ServiceRef]
	Name         string
	After        *collectionlist.List[string]
	Before       *collectionlist.List[string]
	Priority     int
	Parallel     bool
	Timeout      time.Duration
	Raw          bool
}

// SetupMetadata describes a setup registration.
type SetupMetadata struct {
	Label         string
	Dependencies  *collectionlist.List[ServiceRef]
	Provides      *collectionlist.List[ServiceRef]
	Overrides     *collectionlist.List[ServiceRef]
	GraphMutation bool
	Raw           bool
}

// ServiceRefs constructs a filtered collectionx list of service references.
func ServiceRefs(refs ...ServiceRef) *collectionlist.List[ServiceRef] {
	if len(refs) == 0 {
		return collectionlist.NewList[ServiceRef]()
	}
	filtered := collectionlist.NewListWithCapacity[ServiceRef](len(refs))
	for _, ref := range refs {
		if ref.Name != "" {
			filtered.Add(ref)
		}
	}
	return filtered
}

// NewProviderFunc constructs a provider registration from a callback and metadata.
func NewProviderFunc(register func(*Container), meta ProviderMetadata) ProviderFunc {
	return ProviderFunc{
		register: register,
		meta:     normalizeProviderMetadata(meta),
	}
}

func NewProviderFuncWithCollections(
	register func(*Container),
	meta ProviderMetadata,
	collections *collectionlist.List[collectionFactory],
) ProviderFunc {
	return ProviderFunc{
		register:            register,
		meta:                normalizeProviderMetadata(meta),
		collectionFactories: collections,
	}
}

// NewInvokeFunc constructs an invoke registration from a callback and metadata.
func NewInvokeFunc(run func(*Container) error, meta InvokeMetadata) InvokeFunc {
	return InvokeFunc{
		run:  run,
		meta: normalizeInvokeMetadata(meta),
	}
}

// NewHookFunc constructs a hook registration from a callback and metadata.
func NewHookFunc(register func(*Container, Lifecycle), meta HookMetadata) HookFunc {
	return HookFunc{
		register: register,
		meta:     normalizeHookMetadata(meta),
	}
}

// NewSetupFunc constructs a setup registration from a callback and metadata.
func NewSetupFunc(run func(*Container, Lifecycle) error, meta SetupMetadata) SetupFunc {
	return SetupFunc{
		run:  run,
		meta: normalizeSetupMetadata(meta),
	}
}

func normalizeProviderMetadata(meta ProviderMetadata) ProviderMetadata {
	if meta.Label == "" {
		meta.Label = "Provider"
	}
	meta.Dependencies = normalizeServiceRefs(meta.Dependencies)
	meta.Aliases = normalizeServiceRefs(meta.Aliases)
	meta.Contributions = normalizeContributionRefs(meta.Contributions)
	meta.Conditions = normalizeConditions(meta.Conditions)
	return meta
}

func normalizeInvokeMetadata(meta InvokeMetadata) InvokeMetadata {
	if meta.Label == "" {
		meta.Label = "Invoke"
	}
	meta.Dependencies = normalizeServiceRefs(meta.Dependencies)
	return meta
}

func normalizeHookMetadata(meta HookMetadata) HookMetadata {
	if meta.Label == "" {
		meta.Label = "Hook"
	}
	meta.Dependencies = normalizeServiceRefs(meta.Dependencies)
	meta.After = normalizeHookNames(meta.After)
	meta.Before = normalizeHookNames(meta.Before)
	return meta
}

func normalizeSetupMetadata(meta SetupMetadata) SetupMetadata {
	if meta.Label == "" {
		meta.Label = "Setup"
	}
	meta.Dependencies = normalizeServiceRefs(meta.Dependencies)
	meta.Provides = normalizeServiceRefs(meta.Provides)
	meta.Overrides = normalizeServiceRefs(meta.Overrides)
	return meta
}

func normalizeServiceRefs(refs *collectionlist.List[ServiceRef]) *collectionlist.List[ServiceRef] {
	if refs == nil || refs.Len() == 0 {
		return collectionlist.NewList[ServiceRef]()
	}
	return ServiceRefs(refs.Values()...)
}

func normalizeContributionRefs(refs *collectionlist.List[ContributionRef]) *collectionlist.List[ContributionRef] {
	if refs == nil || refs.Len() == 0 {
		return collectionlist.NewList[ContributionRef]()
	}
	return collectionlist.FilterList(refs, func(_ int, ref ContributionRef) bool {
		return ref.Target.Name != "" && ref.Service.Name != ""
	})
}

func normalizeConditions(conditions *collectionlist.List[Condition]) *collectionlist.List[Condition] {
	if conditions == nil || conditions.Len() == 0 {
		return collectionlist.NewList[Condition]()
	}
	return collectionlist.FilterList(conditions, func(_ int, condition Condition) bool {
		return condition != nil
	})
}

func normalizeHookNames(names *collectionlist.List[string]) *collectionlist.List[string] {
	if names == nil || names.Len() == 0 {
		return collectionlist.NewList[string]()
	}
	return collectionlist.FilterList(names, func(_ int, name string) bool {
		return name != ""
	})
}
