package dix

import (
	collectionlist "github.com/arcgolabs/collectionx/list"
	collectionmapping "github.com/arcgolabs/collectionx/mapping"
)

func cloneAppSpecForMutation(spec *appSpec) *appSpec {
	if spec == nil {
		return nil
	}
	cloned := *spec
	if cloned.serviceNames == nil {
		cloned.serviceNames = newServiceNamer()
	}
	cloned.modules = cloneModuleList(spec.modules)
	cloned.observers = spec.observers.Clone()
	cloned.observerDispatchers = collectionlist.NewList[*observerDispatcher]()
	cloned.subapps = spec.subapps.Clone()
	return &cloned
}

func cloneModuleList(modules *collectionlist.List[Module]) *collectionlist.List[Module] {
	if modules == nil || modules.Len() == 0 {
		return collectionlist.NewList[Module]()
	}
	memo := collectionmapping.NewMapWithCapacity[*moduleSpec, *moduleSpec](modules.Len())
	out := collectionlist.NewListWithCapacity[Module](modules.Len())
	modules.Range(func(_ int, module Module) bool {
		out.Add(Module{spec: cloneModuleSpec(module.spec, memo)})
		return true
	})
	return out
}

func cloneModuleSpec(spec *moduleSpec, memo *collectionmapping.Map[*moduleSpec, *moduleSpec]) *moduleSpec {
	if spec == nil {
		return nil
	}
	if cloned, ok := memo.Get(spec); ok {
		return cloned
	}

	cloned := *spec
	memo.Set(spec, &cloned)
	cloned.providers = spec.providers.Clone()
	cloned.setups = spec.setups.Clone()
	cloned.invokes = spec.invokes.Clone()
	cloned.hooks = spec.hooks.Clone()
	cloned.imports = collectionlist.NewListWithCapacity[Module](spec.imports.Len())
	spec.imports.Range(func(_ int, imported Module) bool {
		cloned.imports.Add(Module{spec: cloneModuleSpec(imported.spec, memo)})
		return true
	})
	cloned.profiles = spec.profiles.Clone()
	cloned.excludeProfiles = spec.excludeProfiles.Clone()
	cloned.tags = spec.tags.Clone()
	return &cloned
}
