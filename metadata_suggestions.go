package dix

import (
	"strings"

	collectionlist "github.com/arcgolabs/collectionx/list"
	collectionset "github.com/arcgolabs/collectionx/set"
)

func (s *validationState) missingDependencyHint(name string) string {
	suggestions := s.suggestServiceNames(name, 5)
	if suggestions.Len() == 0 {
		return ""
	}
	return "; available services include: " + suggestions.Join(", ")
}

func (s *validationState) suggestServiceNames(name string, limit int) *collectionlist.List[string] {
	if s == nil || limit <= 0 {
		return collectionlist.NewList[string]()
	}
	candidates := s.availableServiceNames()
	if candidates.Len() == 0 {
		return collectionlist.NewList[string]()
	}

	matches := collectionlist.NewListWithCapacity[string](limit)
	normalizedName := normalizeServiceNameForSuggestion(name)
	candidates.Range(func(_ int, candidate string) bool {
		if serviceNameLooksRelated(normalizedName, normalizeServiceNameForSuggestion(candidate)) {
			matches.Add(candidate)
		}
		return matches.Len() < limit
	})
	if matches.Len() > 0 {
		return matches
	}

	candidates.Range(func(_ int, candidate string) bool {
		matches.Add(candidate)
		return matches.Len() < limit
	})
	return matches
}

func (s *validationState) availableServiceNames() *collectionlist.List[string] {
	names := collectionset.NewSetWithCapacity[string](s.known.Len())
	s.known.Range(func(name string) bool {
		names.Add(name)
		return true
	})
	if s.inherited != nil {
		s.inherited.Range(func(name string) bool {
			names.Add(name)
			return true
		})
	}
	return collectionlist.NewListWithCapacity[string](names.Len(), names.Values()...).Sort(strings.Compare)
}

func normalizeServiceNameForSuggestion(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func serviceNameLooksRelated(missing, candidate string) bool {
	if missing == "" || candidate == "" {
		return false
	}
	if strings.Contains(candidate, missing) || strings.Contains(missing, candidate) {
		return true
	}
	missingTail := serviceNameTail(missing)
	candidateTail := serviceNameTail(candidate)
	return missingTail != "" &&
		candidateTail != "" &&
		(strings.Contains(candidateTail, missingTail) || strings.Contains(missingTail, candidateTail))
}

func serviceNameTail(name string) string {
	index := strings.LastIndexAny(name, "./:")
	if index < 0 || index+1 >= len(name) {
		return name
	}
	return name[index+1:]
}
