package dix

import (
	collectionlist "github.com/arcgolabs/collectionx/list"
	collectionset "github.com/arcgolabs/collectionx/set"
	"github.com/arcgolabs/pkg/option"
)

// ModuleOption configures a Module during construction.
type ModuleOption func(*moduleSpec)

// NewModule creates an immutable module specification.
func NewModule(name string, opts ...ModuleOption) Module {
	spec := &moduleSpec{
		name:            name,
		providers:       collectionlist.NewList[ProviderFunc](),
		setups:          collectionlist.NewList[SetupFunc](),
		invokes:         collectionlist.NewList[InvokeFunc](),
		hooks:           collectionlist.NewList[HookFunc](),
		imports:         collectionlist.NewList[Module](),
		profiles:        collectionset.NewSet[Profile](),
		excludeProfiles: collectionset.NewSet[Profile](),
		tags:            collectionset.NewOrderedSet[string](),
	}
	option.Apply(spec, opts...)
	return Module{spec: spec}
}

// Name returns the module name.
func (m Module) Name() string {
	if m.spec == nil {
		return ""
	}
	return m.spec.name
}

// Description returns the module description.
func (m Module) Description() string {
	if m.spec == nil {
		return ""
	}
	return m.spec.description
}

// Tags returns the module tags.
func (m Module) Tags() *collectionset.OrderedSet[string] {
	if m.spec == nil {
		return collectionset.NewOrderedSet[string]()
	}
	return m.spec.tags.Clone()
}

// Profiles returns the profiles this module is restricted to.
func (m Module) Profiles() *collectionset.Set[Profile] {
	if m.spec == nil {
		return collectionset.NewSet[Profile]()
	}
	return m.spec.profiles.Clone()
}

// ExcludeProfiles returns the profiles this module is excluded from.
func (m Module) ExcludeProfiles() *collectionset.Set[Profile] {
	if m.spec == nil {
		return collectionset.NewSet[Profile]()
	}
	return m.spec.excludeProfiles.Clone()
}

// Disabled reports whether the module is disabled.
func (m Module) Disabled() bool {
	return m.spec != nil && m.spec.disabled
}

// Imports returns the imported modules.
func (m Module) Imports() *collectionlist.List[Module] {
	if m.spec == nil {
		return collectionlist.NewList[Module]()
	}
	return m.spec.imports.Clone()
}
