package dix

func hasLifecycleOrdering(entries []lifecycleHookEntry) bool {
	for _, entry := range entries {
		if lifecycleHookHasOrdering(entry.meta) {
			return true
		}
	}
	return false
}

func lifecycleHookHasOrdering(meta HookMetadata) bool {
	return (meta.After != nil && meta.After.Len() > 0) || (meta.Before != nil && meta.Before.Len() > 0)
}

func hookName(meta HookMetadata) string {
	if meta.Name != "" {
		return meta.Name
	}
	return meta.Label
}
