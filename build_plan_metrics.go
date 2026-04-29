package dix

import (
	collectionlist "github.com/arcgolabs/collectionx/list"
)

func countModules(modules *collectionlist.List[*moduleSpec]) int {
	if modules == nil {
		return 0
	}
	return modules.Len()
}

func countModuleProviders(modules *collectionlist.List[*moduleSpec]) int {
	return sumModuleCounts(modules, func(mod *moduleSpec) int { return mod.providers.Len() })
}

func countModuleHooks(modules *collectionlist.List[*moduleSpec]) int {
	return sumModuleCounts(modules, func(mod *moduleSpec) int { return mod.hooks.Len() })
}

func countModuleSetups(modules *collectionlist.List[*moduleSpec]) int {
	return sumModuleCounts(modules, func(mod *moduleSpec) int { return mod.setups.Len() })
}

func countModuleInvokes(modules *collectionlist.List[*moduleSpec]) int {
	return sumModuleCounts(modules, func(mod *moduleSpec) int { return mod.invokes.Len() })
}

func sumModuleCounts(modules *collectionlist.List[*moduleSpec], selector func(*moduleSpec) int) int {
	if modules == nil || selector == nil {
		return 0
	}
	return collectionlist.ReduceList(modules, 0, func(acc int, _ int, mod *moduleSpec) int {
		if mod == nil {
			return acc
		}
		return acc + selector(mod)
	})
}

func serviceRefNames(refs *collectionlist.List[ServiceRef]) *collectionlist.List[string] {
	if refs == nil || refs.Len() == 0 {
		return collectionlist.NewList[string]()
	}
	return collectionlist.FilterMapList(refs, func(_ int, ref ServiceRef) (string, bool) {
		return ref.Name, ref.Name != ""
	})
}
