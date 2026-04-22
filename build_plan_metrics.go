package dix

import (
	"github.com/arcgolabs/collectionx"
)

func countModules(modules collectionx.List[*moduleSpec]) int {
	if modules == nil {
		return 0
	}
	return modules.Len()
}

func countModuleProviders(modules collectionx.List[*moduleSpec]) int {
	return sumModuleCounts(modules, func(mod *moduleSpec) int { return mod.providers.Len() })
}

func countModuleHooks(modules collectionx.List[*moduleSpec]) int {
	return sumModuleCounts(modules, func(mod *moduleSpec) int { return mod.hooks.Len() })
}

func countModuleSetups(modules collectionx.List[*moduleSpec]) int {
	return sumModuleCounts(modules, func(mod *moduleSpec) int { return mod.setups.Len() })
}

func countModuleInvokes(modules collectionx.List[*moduleSpec]) int {
	return sumModuleCounts(modules, func(mod *moduleSpec) int { return mod.invokes.Len() })
}

func sumModuleCounts(modules collectionx.List[*moduleSpec], selector func(*moduleSpec) int) int {
	if modules == nil || selector == nil {
		return 0
	}
	return collectionx.ReduceList(modules, 0, func(acc int, _ int, mod *moduleSpec) int {
		if mod == nil {
			return acc
		}
		return acc + selector(mod)
	})
}

func serviceRefNames(refs collectionx.List[ServiceRef]) collectionx.List[string] {
	if refs == nil || refs.Len() == 0 {
		return collectionx.NewList[string]()
	}
	return collectionx.FilterMapList(refs, func(_ int, ref ServiceRef) (string, bool) {
		return ref.Name, ref.Name != ""
	})
}
