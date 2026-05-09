package dix

import (
	"errors"
	"sort"

	collectionlist "github.com/arcgolabs/collectionx/list"
	collectionmapping "github.com/arcgolabs/collectionx/mapping"
	collectionset "github.com/arcgolabs/collectionx/set"
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
	entries := make([]lifecycleHookEntry, 0, hooks.Len())
	hooks.Range(func(_ int, entry lifecycleHookEntry) bool {
		entries = append(entries, entry)
		return true
	})
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
	edges := collectionmapping.NewMultiMapWithCapacity[int, int](len(entries))
	edgeSet := collectionset.NewSetWithCapacity[int](len(entries))
	indegree := make([]int, len(entries))

	for from, entry := range entries {
		if err := addLifecycleOrderingEdges(from, entry, index, duplicates, edges, edgeSet, indegree); err != nil {
			return nil, err
		}
	}

	return consumeLifecycleOrder(entries, edges, indegree)
}

func lifecycleHookNameIndex(entries []lifecycleHookEntry) (
	*collectionmapping.Map[string, int],
	*collectionset.Set[string],
) {
	index := collectionmapping.NewMapWithCapacity[string, int](len(entries))
	duplicates := collectionset.NewSet[string]()
	for i, entry := range entries {
		name := hookName(entry.meta)
		if name == "" {
			continue
		}
		if _, found := index.Get(name); found {
			duplicates.Add(name)
			continue
		}
		index.Set(name, i)
	}
	return index, duplicates
}

func addLifecycleOrderingEdges(
	from int,
	entry lifecycleHookEntry,
	index *collectionmapping.Map[string, int],
	duplicates *collectionset.Set[string],
	edges *collectionmapping.MultiMap[int, int],
	edgeSet *collectionset.Set[int],
	indegree []int,
) error {
	if err := addLifecycleAfterEdges(from, entry, index, duplicates, edges, edgeSet, indegree); err != nil {
		return err
	}
	return addLifecycleBeforeEdges(from, entry, index, duplicates, edges, edgeSet, indegree)
}

func addLifecycleAfterEdges(
	current int,
	entry lifecycleHookEntry,
	index *collectionmapping.Map[string, int],
	duplicates *collectionset.Set[string],
	edges *collectionmapping.MultiMap[int, int],
	edgeSet *collectionset.Set[int],
	indegree []int,
) error {
	var edgeErr error
	entry.meta.After.Range(func(_ int, name string) bool {
		target, err := resolveLifecycleHookTarget(entry, name, index, duplicates)
		if err != nil {
			edgeErr = err
			return false
		}
		addLifecycleEdge(target, current, edges, edgeSet, indegree)
		return true
	})
	return edgeErr
}

func addLifecycleBeforeEdges(
	current int,
	entry lifecycleHookEntry,
	index *collectionmapping.Map[string, int],
	duplicates *collectionset.Set[string],
	edges *collectionmapping.MultiMap[int, int],
	edgeSet *collectionset.Set[int],
	indegree []int,
) error {
	var edgeErr error
	entry.meta.Before.Range(func(_ int, name string) bool {
		target, err := resolveLifecycleHookTarget(entry, name, index, duplicates)
		if err != nil {
			edgeErr = err
			return false
		}
		addLifecycleEdge(current, target, edges, edgeSet, indegree)
		return true
	})
	return edgeErr
}

func resolveLifecycleHookTarget(
	entry lifecycleHookEntry,
	name string,
	index *collectionmapping.Map[string, int],
	duplicates *collectionset.Set[string],
) (int, error) {
	if duplicates.Contains(name) {
		return 0, oops.In("dix").
			With("op", "lifecycle_order", "hook", hookName(entry.meta), "target", name).
			Errorf("lifecycle hook target `%s` is ambiguous", name)
	}
	target, found := index.Get(name)
	if !found {
		return 0, oops.In("dix").
			With("op", "lifecycle_order", "hook", hookName(entry.meta), "target", name).
			Errorf("lifecycle hook target `%s` was not found", name)
	}
	return target, nil
}

func addLifecycleEdge(
	from int,
	to int,
	edges *collectionmapping.MultiMap[int, int],
	edgeSet *collectionset.Set[int],
	indegree []int,
) {
	if from == to || edges == nil || edgeSet == nil {
		return
	}
	key := lifecycleEdgeKey(from, to, len(indegree))
	if edgeSet.Contains(key) {
		return
	}
	edgeSet.Add(key)
	edges.Put(from, to)
	indegree[to]++
}

func lifecycleEdgeKey(from, to, size int) int {
	return from*size + to
}

func consumeLifecycleOrder(
	entries []lifecycleHookEntry,
	edges *collectionmapping.MultiMap[int, int],
	indegree []int,
) ([]lifecycleHookEntry, error) {
	ready, err := newLifecycleReadyQueue(indegree)
	if err != nil {
		return nil, err
	}

	ordered := make([]lifecycleHookEntry, 0, len(entries))
	for !ready.IsEmpty() {
		next, _ := ready.Pop()
		ordered = append(ordered, entries[next])
		for _, target := range edges.Get(next) {
			indegree[target]--
			if indegree[target] == 0 {
				ready.Push(target)
			}
		}
	}
	if len(ordered) < len(entries) {
		return nil, oops.In("dix").
			With("op", "lifecycle_order").
			New("lifecycle hook dependency cycle detected")
	}
	return ordered, nil
}

func newLifecycleReadyQueue(indegree []int) (*collectionlist.PriorityQueue[int], error) {
	ready, err := collectionlist.NewPriorityQueue[int](func(left, right int) bool {
		return left < right
	})
	if err != nil {
		return nil, err
	}
	for i := range indegree {
		if indegree[i] == 0 {
			ready.Push(i)
		}
	}
	return ready, nil
}

func reverseLifecycleEntries(entries []lifecycleHookEntry) []lifecycleHookEntry {
	reversed := make([]lifecycleHookEntry, len(entries))
	for i, entry := range entries {
		reversed[len(entries)-1-i] = entry
	}
	return reversed
}
