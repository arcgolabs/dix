package dix

import "time"

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
