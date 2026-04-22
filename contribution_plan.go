package dix

import (
	"context"

	"github.com/arcgolabs/collectionx"
)

type serviceNameSet struct {
	values collectionx.Set[string]
}

func newServiceNameSet(capacity int) serviceNameSet {
	return serviceNameSet{values: collectionx.NewSetWithCapacity[string](capacity)}
}

func (s serviceNameSet) Add(name string) {
	if name != "" {
		s.values.Add(name)
	}
}

func (s serviceNameSet) Contains(name string) bool {
	return s.values.Contains(name)
}

type contributionPlan struct {
	contributions collectionx.MultiMap[string, ContributionRef]
	factories     collectionx.Map[string, collectionFactory]
	targets       collectionx.List[string]
	explicit      serviceNameSet
}

func newContributionPlan(modules collectionx.List[*moduleSpec]) contributionPlan {
	plan := contributionPlan{
		contributions: collectionx.NewMultiMap[string, ContributionRef](),
		factories:     collectionx.NewMap[string, collectionFactory](),
		targets:       collectionx.NewList[string](),
		explicit:      explicitServiceOutputs(modules),
	}
	if modules == nil {
		return plan
	}

	sequence := 0
	modules.Range(func(_ int, mod *moduleSpec) bool {
		plan.collectModule(mod, &sequence)
		return true
	})

	return plan
}

func (p contributionPlan) collectModule(mod *moduleSpec, sequence *int) {
	if mod == nil {
		return
	}
	mod.providers.Range(func(_ int, provider ProviderFunc) bool {
		p.collectProviderContributions(provider, sequence)
		p.collectProviderCollectionFactories(provider)
		return true
	})
}

func (p contributionPlan) collectProviderContributions(provider ProviderFunc, sequence *int) {
	provider.meta.Contributions.Range(func(_ int, contribution ContributionRef) bool {
		contribution.sequence = *sequence
		(*sequence)++
		p.addContribution(contribution)
		return true
	})
}

func (p contributionPlan) addContribution(contribution ContributionRef) {
	target := contribution.Target.Name
	if len(p.contributions.Get(target)) == 0 {
		p.targets.Add(target)
	}
	p.contributions.Put(target, contribution)
}

func (p contributionPlan) collectProviderCollectionFactories(provider ProviderFunc) {
	provider.collectionFactories.Range(func(_ int, factory collectionFactory) bool {
		target := factory.target.Name
		if target != "" {
			if _, exists := p.factories.Get(target); !exists {
				p.factories.Set(target, factory)
			}
		}
		return true
	})
}

func explicitServiceOutputs(modules collectionx.List[*moduleSpec]) serviceNameSet {
	known := newServiceNameSet(64)
	if modules == nil {
		return known
	}
	modules.Range(func(_ int, mod *moduleSpec) bool {
		if mod == nil {
			return true
		}
		mod.providers.Range(func(_ int, provider ProviderFunc) bool {
			known.Add(provider.meta.Output.Name)
			provider.meta.Aliases.Range(func(_ int, alias ServiceRef) bool {
				known.Add(alias.Name)
				return true
			})
			return true
		})
		mod.setups.Range(func(_ int, setup SetupFunc) bool {
			setup.meta.Provides.Range(func(_ int, provide ServiceRef) bool {
				known.Add(provide.Name)
				return true
			})
			return true
		})
		return true
	})
	return known
}

func (p contributionPlan) register(ctx context.Context, rt *Runtime, debugEnabled bool) {
	p.targets.Range(func(_ int, target string) bool {
		factory, found := p.factories.Get(target)
		if !found {
			return true
		}
		contributionValues := p.contributions.GetCopy(target)
		contributions := collectionx.NewListWithCapacity(len(contributionValues), contributionValues...)
		if debugEnabled {
			rt.logMessage(ctx, EventLevelDebug, "registering contribution collection",
				"target", target,
				"contributions", contributions.Len(),
			)
		}
		factory.register(rt.container, contributions, p.explicit)
		return true
	})
}

func (p contributionPlan) syntheticOutputs() collectionx.List[ServiceRef] {
	outputs := newServiceNameSet(16)
	refs := collectionx.NewList[ServiceRef]()
	p.targets.Range(func(_ int, target string) bool {
		factory, found := p.factories.Get(target)
		if !found {
			return true
		}
		factory.outputs.Range(func(_ int, output ServiceRef) bool {
			if !p.explicit.Contains(output.Name) && !outputs.Contains(output.Name) {
				outputs.Add(output.Name)
				refs.Add(output)
			}
			return true
		})
		return true
	})
	return refs
}
