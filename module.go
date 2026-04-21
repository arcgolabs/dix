package dix

import (
	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/pkg/option"
)

// ModuleOption configures a Module during construction.
type ModuleOption func(*moduleSpec)

// NewModule creates an immutable module specification.
func NewModule(name string, opts ...ModuleOption) Module {
	spec := &moduleSpec{
		name:            name,
		providers:       collectionx.NewList[ProviderFunc](),
		setups:          collectionx.NewList[SetupFunc](),
		invokes:         collectionx.NewList[InvokeFunc](),
		hooks:           collectionx.NewList[HookFunc](),
		imports:         collectionx.NewList[Module](),
		profiles:        collectionx.NewSet[Profile](),
		excludeProfiles: collectionx.NewSet[Profile](),
		tags:            collectionx.NewOrderedSet[string](),
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
func (m Module) Tags() collectionx.OrderedSet[string] {
	if m.spec == nil {
		return collectionx.NewOrderedSet[string]()
	}
	return m.spec.tags.Clone()
}

// Profiles returns the profiles this module is restricted to.
func (m Module) Profiles() collectionx.Set[Profile] {
	if m.spec == nil {
		return collectionx.NewSet[Profile]()
	}
	return m.spec.profiles.Clone()
}

// ExcludeProfiles returns the profiles this module is excluded from.
func (m Module) ExcludeProfiles() collectionx.Set[Profile] {
	if m.spec == nil {
		return collectionx.NewSet[Profile]()
	}
	return m.spec.excludeProfiles.Clone()
}

// Disabled reports whether the module is disabled.
func (m Module) Disabled() bool {
	return m.spec != nil && m.spec.disabled
}

// Imports returns the imported modules.
func (m Module) Imports() collectionx.List[Module] {
	if m.spec == nil {
		return collectionx.NewList[Module]()
	}
	return m.spec.imports.Clone()
}
