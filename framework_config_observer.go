package dix

import (
	collectionlist "github.com/arcgolabs/collectionx/list"
	"github.com/samber/oops"
	"log/slog"
)

func (p *buildPlan) applyFrameworkObservers(rt *Runtime, declared frameworkConfigDeclarations) error {
	if p.spec.observersConfigured || !declared.hasObservers() {
		return nil
	}
	return p.applyDeclaredObservers(rt, declared)
}

func (p *buildPlan) applyDeclaredObservers(rt *Runtime, declared frameworkConfigDeclarations) error {
	observers, err := p.resolveDeclaredObservers(rt, declared)
	if err != nil {
		return err
	}
	if observers != nil && observers.Len() > 0 {
		rt.spec.appendObserversWithLogger(func() *slog.Logger { return rt.logger }, observers.Values()...)
	}
	return nil
}

func (p *buildPlan) resolveDeclaredObservers(rt *Runtime, declared frameworkConfigDeclarations) (*collectionlist.List[Observer], error) {
	observers := collectionlist.NewListWithCapacity[Observer](1)
	if declared.observer {
		var err error
		observers, err = p.appendDeclaredObserver(rt, observers)
		if err != nil {
			return nil, err
		}
	}
	if declared.observerList {
		var err error
		observers, err = p.appendDeclaredObserverList(rt, observers)
		if err != nil {
			return nil, err
		}
	}
	return observers, nil
}

func (p *buildPlan) appendDeclaredObserver(rt *Runtime, observers *collectionlist.List[Observer]) (*collectionlist.List[Observer], error) {
	observer, err := p.resolveDeclaredObserver(rt)
	if err != nil {
		return nil, err
	}
	if observer != nil {
		observers.Add(observer)
	}
	return observers, nil
}

func (p *buildPlan) appendDeclaredObserverList(rt *Runtime, observers *collectionlist.List[Observer]) (*collectionlist.List[Observer], error) {
	resolvedObservers, err := p.resolveDeclaredObserverList(rt)
	if err != nil {
		return nil, err
	}
	if resolvedObservers != nil {
		resolvedObservers.Range(func(_ int, observer Observer) bool {
			if observer != nil {
				observers.Add(observer)
			}
			return true
		})
	}
	return observers, nil
}

func (p *buildPlan) resolveDeclaredObserver(rt *Runtime) (Observer, error) {
	observer, err := ResolveAs[Observer](rt.container)
	if err != nil {
		return nil, oops.In("dix").
			With("op", "resolve_declared_observer", "app", rt.Name(), "service", serviceNameOf[Observer]()).
			Wrapf(err, "resolve declared observer failed")
	}
	return observer, nil
}

func (p *buildPlan) resolveDeclaredObserverList(rt *Runtime) (*collectionlist.List[Observer], error) {
	observers, err := ResolveAs[*collectionlist.List[Observer]](rt.container)
	if err != nil {
		return nil, oops.In("dix").
			With("op", "resolve_declared_observer_list", "app", rt.Name(), "service", serviceNameOf[*collectionlist.List[Observer]]()).
			Wrapf(err, "resolve declared observer list failed")
	}
	return observers, nil
}
