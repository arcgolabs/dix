package dix

import (
	"fmt"
	"sort"
	"strings"

	"github.com/arcgolabs/collectionx"
	"github.com/samber/do/v2"
)

func registerCollectionProviders[T any](c *Container, refs collectionx.List[ContributionRef], explicit serviceNameSet) {
	ordered := orderedContributionRefs(refs)
	registerContributionListProvider[T](c, ordered, explicit)
	registerContributionMapProvider[T](c, ordered, explicit)
	registerContributionCollectionMapProvider[T](c, ordered, explicit)
	registerContributionOrderedMapProvider[T](c, ordered, explicit)
}

func registerContributionListProvider[T any](
	c *Container,
	refs collectionx.List[ContributionRef],
	explicit serviceNameSet,
) {
	if explicit.Contains(TypedService[collectionx.List[T]]().Name) {
		return
	}
	ProvideTErr[collectionx.List[T]](c, func() (collectionx.List[T], error) {
		return resolveContributionList[T](c.Raw(), refs)
	})
}

func registerContributionMapProvider[T any](
	c *Container,
	refs collectionx.List[ContributionRef],
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
	refs collectionx.List[ContributionRef],
	explicit serviceNameSet,
) {
	if explicit.Contains(TypedService[collectionx.Map[string, T]]().Name) {
		return
	}
	ProvideTErr[collectionx.Map[string, T]](c, func() (collectionx.Map[string, T], error) {
		return resolveContributionMap[T](c.Raw(), refs)
	})
}

func registerContributionOrderedMapProvider[T any](
	c *Container,
	refs collectionx.List[ContributionRef],
	explicit serviceNameSet,
) {
	if explicit.Contains(TypedService[collectionx.OrderedMap[string, T]]().Name) {
		return
	}
	ProvideTErr[collectionx.OrderedMap[string, T]](c, func() (collectionx.OrderedMap[string, T], error) {
		return resolveContributionOrderedMap[T](c.Raw(), refs)
	})
}

func orderedContributionRefs(refs collectionx.List[ContributionRef]) collectionx.List[ContributionRef] {
	ordered := refs.Values()
	sort.SliceStable(ordered, func(left, right int) bool {
		if ordered[left].Order != ordered[right].Order {
			return ordered[left].Order < ordered[right].Order
		}
		return ordered[left].sequence < ordered[right].sequence
	})
	return collectionx.NewListWithCapacity(len(ordered), ordered...)
}

func resolveContributionList[T any](injector do.Injector, refs collectionx.List[ContributionRef]) (collectionx.List[T], error) {
	values := collectionx.NewListWithCapacity[T](refs.Len())
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
	refs collectionx.List[ContributionRef],
) (collectionx.Map[string, T], error) {
	values := collectionx.NewMapWithCapacity[string, T](refs.Len())
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
	refs collectionx.List[ContributionRef],
) (collectionx.OrderedMap[string, T], error) {
	values := collectionx.NewOrderedMapWithCapacity[string, T](refs.Len())
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

func contributionOrderedMapValue[T any](injector do.Injector, values collectionx.OrderedMap[string, T], ref ContributionRef) error {
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
