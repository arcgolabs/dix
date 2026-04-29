package dix

import (
	"fmt"
	collectionlist "github.com/arcgolabs/collectionx/list"
	collectionmapping "github.com/arcgolabs/collectionx/mapping"
	"github.com/samber/do/v2"
	"strings"
)

func registerCollectionProviders[T any](c *Container, refs *collectionlist.List[ContributionRef], explicit serviceNameSet) {
	ordered := orderedContributionRefs(refs)
	registerContributionListProvider[T](c, ordered, explicit)
	registerContributionMapProvider[T](c, ordered, explicit)
	registerContributionCollectionMapProvider[T](c, ordered, explicit)
	registerContributionOrderedMapProvider[T](c, ordered, explicit)
}

func registerContributionListProvider[T any](
	c *Container,
	refs *collectionlist.List[ContributionRef],
	explicit serviceNameSet,
) {
	if explicit.Contains(TypedService[*collectionlist.List[T]]().Name) {
		return
	}
	ProvideTErr[*collectionlist.List[T]](c, func() (*collectionlist.List[T], error) {
		return resolveContributionList[T](c.Raw(), refs)
	})
}

func registerContributionMapProvider[T any](
	c *Container,
	refs *collectionlist.List[ContributionRef],
	explicit serviceNameSet,
) {
	if explicit.Contains(TypedService[map[string]T]().Name) {
		return
	}
	ProvideTErr[map[string]T](c, func() (map[string]T, error) {
		values, err := resolveContributionMap[T](c.Raw(), refs)
		if err != nil {
			return nil, err
		}
		return values.All(), nil
	})
}

func registerContributionCollectionMapProvider[T any](
	c *Container,
	refs *collectionlist.List[ContributionRef],
	explicit serviceNameSet,
) {
	if explicit.Contains(TypedService[*collectionmapping.Map[string, T]]().Name) {
		return
	}
	ProvideTErr[*collectionmapping.Map[string, T]](c, func() (*collectionmapping.Map[string, T], error) {
		return resolveContributionMap[T](c.Raw(), refs)
	})
}

func registerContributionOrderedMapProvider[T any](
	c *Container,
	refs *collectionlist.List[ContributionRef],
	explicit serviceNameSet,
) {
	if explicit.Contains(TypedService[*collectionmapping.OrderedMap[string, T]]().Name) {
		return
	}
	ProvideTErr[*collectionmapping.OrderedMap[string, T]](c, func() (*collectionmapping.OrderedMap[string, T], error) {
		return resolveContributionOrderedMap[T](c.Raw(), refs)
	})
}

func orderedContributionRefs(refs *collectionlist.List[ContributionRef]) *collectionlist.List[ContributionRef] {
	return refs.Clone().Sort(func(left, right ContributionRef) int {
		if left.Order != right.Order {
			if left.Order < right.Order {
				return -1
			}
			return 1
		}
		if left.sequence < right.sequence {
			return -1
		}
		if left.sequence > right.sequence {
			return 1
		}
		return 0
	})
}

func resolveContributionList[T any](injector do.Injector, refs *collectionlist.List[ContributionRef]) (*collectionlist.List[T], error) {
	values := collectionlist.NewListWithCapacity[T](refs.Len())
	var resolveErr error
	refs.Range(func(_ int, ref ContributionRef) bool {
		value, err := do.InvokeNamed[T](injector, ref.Service.Name)
		if err != nil {
			resolveErr = fmt.Errorf("dix: resolve contribution `%s`: %w", ref.Service.Name, err)
			return false
		}
		values.Add(value)
		return true
	})
	if resolveErr != nil {
		return nil, resolveErr
	}
	return values, nil
}

func resolveContributionMap[T any](
	injector do.Injector,
	refs *collectionlist.List[ContributionRef],
) (*collectionmapping.Map[string, T], error) {
	values := collectionmapping.NewMapWithCapacity[string, T](refs.Len())
	var resolveErr error
	refs.Range(func(_ int, ref ContributionRef) bool {
		key, err := contributionKey(ref)
		if err != nil {
			resolveErr = err
			return false
		}
		if _, exists := values.Get(key); exists {
			resolveErr = fmt.Errorf("dix: duplicate contribution key `%s` for `%s`", key, ref.Target.Name)
			return false
		}
		value, err := do.InvokeNamed[T](injector, ref.Service.Name)
		if err != nil {
			resolveErr = fmt.Errorf("dix: resolve contribution `%s`: %w", ref.Service.Name, err)
			return false
		}
		values.Set(key, value)
		return true
	})
	if resolveErr != nil {
		return nil, resolveErr
	}
	return values, nil
}

func resolveContributionOrderedMap[T any](
	injector do.Injector,
	refs *collectionlist.List[ContributionRef],
) (*collectionmapping.OrderedMap[string, T], error) {
	values := collectionmapping.NewOrderedMapWithCapacity[string, T](refs.Len())
	var resolveErr error
	refs.Range(func(_ int, ref ContributionRef) bool {
		if err := contributionOrderedMapValue(injector, values, ref); err != nil {
			resolveErr = err
			return false
		}
		return true
	})
	if resolveErr != nil {
		return nil, resolveErr
	}
	return values, nil
}

func contributionOrderedMapValue[T any](injector do.Injector, values *collectionmapping.OrderedMap[string, T], ref ContributionRef) error {
	key, err := contributionKey(ref)
	if err != nil {
		return err
	}
	if _, exists := values.Get(key); exists {
		return fmt.Errorf("dix: duplicate contribution key `%s` for `%s`", key, ref.Target.Name)
	}
	value, err := do.InvokeNamed[T](injector, ref.Service.Name)
	if err != nil {
		return fmt.Errorf("dix: resolve contribution `%s`: %w", ref.Service.Name, err)
	}
	values.Set(key, value)
	return nil
}

func contributionKey(ref ContributionRef) (string, error) {
	if !ref.HasKey || strings.TrimSpace(ref.Key) == "" {
		return "", fmt.Errorf("dix: contribution to `%s` is missing a key", ref.Target.Name)
	}
	return ref.Key, nil
}
