package dix

import (
	"log/slog"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/samber/oops"
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

func (p *buildPlan) resolveDeclaredObservers(rt *Runtime, declared frameworkConfigDeclarations) (collectionx.List[Observer], error) {
	observers := collectionx.NewListWithCapacity[Observer](1)
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

func (p *buildPlan) appendDeclaredObserver(rt *Runtime, observers collectionx.List[Observer]) (collectionx.List[Observer], error) {
	observer, err := p.resolveDeclaredObserver(rt)
	if err != nil {
		return nil, err
	}
	if observer != nil {
		observers.Add(observer)
	}
	return observers, nil
}

func (p *buildPlan) appendDeclaredObserverList(rt *Runtime, observers collectionx.List[Observer]) (collectionx.List[Observer], error) {
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

func (p *buildPlan) resolveDeclaredObserverList(rt *Runtime) (collectionx.List[Observer], error) {
	observers, err := ResolveAs[collectionx.List[Observer]](rt.container)
	if err != nil {
		return nil, oops.In("dix").
			With("op", "resolve_declared_observer_list", "app", rt.Name(), "service", serviceNameOf[collectionx.List[Observer]]()).
			Wrapf(err, "resolve declared observer list failed")
	}
	return observers, nil
}
