package dix

import (
	collectionlist "github.com/arcgolabs/collectionx/list"
)

type moduleStats struct {
	modules   int
	providers int
	hooks     int
	setups    int
	invokes   int
}

func countModuleStats(modules *collectionlist.List[*moduleSpec]) moduleStats {
	if modules == nil {
		return moduleStats{}
	}
	stats := moduleStats{modules: modules.Len()}
	modules.Range(func(_ int, mod *moduleSpec) bool {
		if mod == nil {
			return true
		}
		stats.providers += mod.providers.Len()
		stats.hooks += mod.hooks.Len()
		stats.setups += mod.setups.Len()
		stats.invokes += mod.invokes.Len()
		return true
	})
	return stats
}

func serviceRefNames(refs *collectionlist.List[ServiceRef]) *collectionlist.List[string] {
	if refs == nil || refs.Len() == 0 {
		return collectionlist.NewList[string]()
	}
	return collectionlist.FilterMapList(refs, func(_ int, ref ServiceRef) (string, bool) {
		return ref.Name, ref.Name != ""
	})
}
