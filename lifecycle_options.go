package dix

import (
	"strings"
	"time"

	collectionlist "github.com/arcgolabs/collectionx/list"
)

// LifecycleHookOption configures lifecycle hook scheduling metadata.
type LifecycleHookOption func(*HookMetadata)

// LifecycleName assigns a stable, human-readable lifecycle hook name.
func LifecycleName(name string) LifecycleHookOption {
	return func(meta *HookMetadata) {
		if meta != nil {
			meta.Name = name
		}
	}
}

// LifecyclePriority assigns lifecycle hook ordering.
//
// Lower priority hooks start earlier. Stop hooks run with the reverse ordering.
func LifecyclePriority(priority int) LifecycleHookOption {
	return func(meta *HookMetadata) {
		if meta != nil {
			meta.Priority = priority
		}
	}
}

// LifecycleAfter makes this hook run after the named hooks in the same lifecycle phase.
func LifecycleAfter(names ...string) LifecycleHookOption {
	return func(meta *HookMetadata) {
		if meta != nil {
			meta.After = appendLifecycleHookNames(meta.After, names...)
		}
	}
}

// LifecycleBefore makes this hook run before the named hooks in the same lifecycle phase.
func LifecycleBefore(names ...string) LifecycleHookOption {
	return func(meta *HookMetadata) {
		if meta != nil {
			meta.Before = appendLifecycleHookNames(meta.Before, names...)
		}
	}
}

// LifecycleParallel allows this hook to run concurrently with adjacent parallel hooks of the same priority.
func LifecycleParallel() LifecycleHookOption {
	return func(meta *HookMetadata) {
		if meta != nil {
			meta.Parallel = true
		}
	}
}

// LifecycleTimeout sets a cooperative timeout for this lifecycle hook.
//
// The hook receives a child context with this deadline. Non-positive values disable the timeout.
func LifecycleTimeout(timeout time.Duration) LifecycleHookOption {
	return func(meta *HookMetadata) {
		if meta != nil {
			meta.Timeout = timeout
		}
	}
}

func applyLifecycleHookOptions(meta HookMetadata, opts ...LifecycleHookOption) HookMetadata {
	for _, opt := range opts {
		if opt != nil {
			opt(&meta)
		}
	}
	return meta
}

func appendLifecycleHookNames(current *collectionlist.List[string], names ...string) *collectionlist.List[string] {
	if current == nil {
		current = collectionlist.NewList[string]()
	}
	for _, name := range names {
		clean := strings.TrimSpace(name)
		if clean != "" {
			current.Add(clean)
		}
	}
	return current
}
