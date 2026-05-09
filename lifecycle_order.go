package dix

import (
	"errors"
	"sort"

	collectionlist "github.com/arcgolabs/collectionx/list"
	"github.com/samber/oops"
)

type lifecycleOrderDirection uint8

const (
	lifecycleOrderStart lifecycleOrderDirection = iota
	lifecycleOrderStop
)

func (l *lifecycleImpl) validateHookOrder() error {
	if l == nil {
		return nil
	}
	_, startErr := l.startOrder(l.startHooks)
	_, stopErr := l.stopOrder(l.stopHooks)
	return errors.Join(startErr, stopErr)
}

func (l *lifecycleImpl) startOrder(hooks *collectionlist.List[lifecycleHookEntry]) ([]lifecycleHookEntry, error) {
	return orderLifecycleHooks(hooks, lifecycleOrderStart)
}

func (l *lifecycleImpl) stopOrder(hooks *collectionlist.List[lifecycleHookEntry]) ([]lifecycleHookEntry, error) {
	return orderLifecycleHooks(hooks, lifecycleOrderStop)
}

func (l *lifecycleImpl) startOrderForSummary(hooks *collectionlist.List[lifecycleHookEntry]) []lifecycleHookEntry {
	entries, err := l.startOrder(hooks)
	if err == nil {
		return entries
	}
	return baseLifecycleOrder(hooks, lifecycleOrderStart)
}

func (l *lifecycleImpl) stopOrderForSummary(hooks *collectionlist.List[lifecycleHookEntry]) []lifecycleHookEntry {
	entries, err := l.stopOrder(hooks)
	if err == nil {
		return entries
	}
	return baseLifecycleOrder(hooks, lifecycleOrderStop)
}

func orderLifecycleHooks(
	hooks *collectionlist.List[lifecycleHookEntry],
	direction lifecycleOrderDirection,
) ([]lifecycleHookEntry, error) {
	entries := baseLifecycleOrder(hooks, direction)
	if len(entries) == 0 || !hasLifecycleOrdering(entries) {
		return entries, nil
	}
	ordered, err := topologicalLifecycleOrder(entries)
	if err != nil {
		return entries, err
	}
	return ordered, nil
}

func baseLifecycleOrder(
	hooks *collectionlist.List[lifecycleHookEntry],
	direction lifecycleOrderDirection,
) []lifecycleHookEntry {
	if hooks == nil || hooks.Len() == 0 {
		return nil
	}
	entries := append([]lifecycleHookEntry(nil), hooks.Values()...)
	sort.SliceStable(entries, func(i, j int) bool {
		return lifecycleEntryLess(entries[i], entries[j], direction)
	})
	return entries
}

func lifecycleEntryLess(left, right lifecycleHookEntry, direction lifecycleOrderDirection) bool {
	if left.meta.Priority != right.meta.Priority {
		if direction == lifecycleOrderStop {
			return left.meta.Priority > right.meta.Priority
		}
		return left.meta.Priority < right.meta.Priority
	}
	if direction == lifecycleOrderStop {
		return left.sequence > right.sequence
	}
	return left.sequence < right.sequence
}

func topologicalLifecycleOrder(entries []lifecycleHookEntry) ([]lifecycleHookEntry, error) {
	index, duplicates := lifecycleHookNameIndex(entries)
	edges := make([]map[int]struct{}, len(entries))
	indegree := make([]int, len(entries))

	for from, entry := range entries {
		if err := addLifecycleOrderingEdges(from, entry, index, duplicates, edges, indegree); err != nil {
			return nil, err
		}
	}

	return consumeLifecycleOrder(entries, edges, indegree)
}

func lifecycleHookNameIndex(entries []lifecycleHookEntry) (map[string]int, map[string]bool) {
	index := make(map[string]int, len(entries))
	duplicates := make(map[string]bool)
	for i, entry := range entries {
		name := hookName(entry.meta)
		if name == "" {
			continue
		}
		if _, found := index[name]; found {
			duplicates[name] = true
			continue
		}
		index[name] = i
	}
	return index, duplicates
}

func addLifecycleOrderingEdges(
	from int,
	entry lifecycleHookEntry,
	index map[string]int,
	duplicates map[string]bool,
	edges []map[int]struct{},
	indegree []int,
) error {
	if err := addLifecycleAfterEdges(from, entry, index, duplicates, edges, indegree); err != nil {
		return err
	}
	return addLifecycleBeforeEdges(from, entry, index, duplicates, edges, indegree)
}

func addLifecycleAfterEdges(
	current int,
	entry lifecycleHookEntry,
	index map[string]int,
	duplicates map[string]bool,
	edges []map[int]struct{},
	indegree []int,
) error {
	var edgeErr error
	entry.meta.After.Range(func(_ int, name string) bool {
		target, err := resolveLifecycleHookTarget(entry, name, index, duplicates)
		if err != nil {
			edgeErr = err
			return false
		}
		addLifecycleEdge(target, current, edges, indegree)
		return true
	})
	return edgeErr
}

func addLifecycleBeforeEdges(
	current int,
	entry lifecycleHookEntry,
	index map[string]int,
	duplicates map[string]bool,
	edges []map[int]struct{},
	indegree []int,
) error {
	var edgeErr error
	entry.meta.Before.Range(func(_ int, name string) bool {
		target, err := resolveLifecycleHookTarget(entry, name, index, duplicates)
		if err != nil {
			edgeErr = err
			return false
		}
		addLifecycleEdge(current, target, edges, indegree)
		return true
	})
	return edgeErr
}

func resolveLifecycleHookTarget(
	entry lifecycleHookEntry,
	name string,
	index map[string]int,
	duplicates map[string]bool,
) (int, error) {
	if duplicates[name] {
		return 0, oops.In("dix").
			With("op", "lifecycle_order", "hook", hookName(entry.meta), "target", name).
			Errorf("lifecycle hook target `%s` is ambiguous", name)
	}
	target, found := index[name]
	if !found {
		return 0, oops.In("dix").
			With("op", "lifecycle_order", "hook", hookName(entry.meta), "target", name).
			Errorf("lifecycle hook target `%s` was not found", name)
	}
	return target, nil
}

func addLifecycleEdge(from, to int, edges []map[int]struct{}, indegree []int) {
	if from == to {
		return
	}
	if edges[from] == nil {
		edges[from] = map[int]struct{}{}
	}
	if _, exists := edges[from][to]; exists {
		return
	}
	edges[from][to] = struct{}{}
	indegree[to]++
}

func consumeLifecycleOrder(
	entries []lifecycleHookEntry,
	edges []map[int]struct{},
	indegree []int,
) ([]lifecycleHookEntry, error) {
	used := make([]bool, len(entries))
	ordered := make([]lifecycleHookEntry, 0, len(entries))
	for len(ordered) < len(entries) {
		next := nextLifecycleReadyNode(used, indegree)
		if next < 0 {
			return nil, oops.In("dix").
				With("op", "lifecycle_order").
				New("lifecycle hook dependency cycle detected")
		}
		used[next] = true
		ordered = append(ordered, entries[next])
		for target := range edges[next] {
			indegree[target]--
		}
	}
	return ordered, nil
}

func nextLifecycleReadyNode(used []bool, indegree []int) int {
	for i := range indegree {
		if !used[i] && indegree[i] == 0 {
			return i
		}
	}
	return -1
}

func reverseLifecycleEntries(entries []lifecycleHookEntry) []lifecycleHookEntry {
	ordered := append([]lifecycleHookEntry(nil), entries...)
	for i, j := 0, len(ordered)-1; i < j; i, j = i+1, j-1 {
		ordered[i], ordered[j] = ordered[j], ordered[i]
	}
	return ordered
}

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
