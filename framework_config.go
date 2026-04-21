package dix

import (
	"context"
	"log/slog"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/samber/do/v2"
	"github.com/samber/oops"
)

type frameworkConfigDeclarations struct {
	slogLogger   bool
	eventLogger  bool
	appMeta      bool
	profile      bool
	observer     bool
	observerList bool
}

func (p *buildPlan) prepareFrameworkConfig(ctx context.Context, rt *Runtime) (bool, error) {
	if rt == nil || rt.container == nil {
		return false, oops.In("dix").
			With("op", "prepare_framework_config").
			New("runtime container is nil")
	}

	declared := p.frameworkConfigDeclarations()
	if !declared.needsProviderRegistration(p.spec) {
		return false, nil
	}

	p.registerProviders(ctx, rt, false)

	if declared.profile {
		p.applyRuntimeProfile(rt)
	}
	if err := p.applyFrameworkLogger(rt, declared); err != nil {
		return true, err
	}
	if err := p.applyFrameworkEventLogger(rt, declared); err != nil {
		return true, err
	}
	if err := p.applyFrameworkAppMeta(rt, declared); err != nil {
		return true, err
	}
	if err := p.applyFrameworkObservers(rt, declared); err != nil {
		return true, err
	}

	return true, nil
}

func (p *buildPlan) frameworkConfigDeclarations() frameworkConfigDeclarations {
	if p == nil {
		return frameworkConfigDeclarations{}
	}
	return frameworkConfigDeclarations{
		slogLogger:   p.declaresProviderOutput(TypedService[*slog.Logger]()),
		eventLogger:  p.declaresProviderOutput(TypedService[EventLogger]()),
		appMeta:      p.declaresProviderOutput(TypedService[AppMeta]()),
		profile:      p.declaresProviderOutput(TypedService[Profile]()),
		observer:     p.declaresProviderOutput(TypedService[Observer]()),
		observerList: p.declaresProviderOutput(TypedService[collectionx.List[Observer]]()),
	}
}

func (p *buildPlan) registerRuntimeCoreServices(rt *Runtime) {
	if rt == nil || rt.container == nil || rt.spec == nil {
		return
	}
	if !p.declaresProviderOutput(TypedService[*slog.Logger]()) {
		ProvideValueT[*slog.Logger](rt.container, rt.logger)
	}
	if !p.declaresProviderOutput(TypedService[AppMeta]()) {
		ProvideValueT[AppMeta](rt.container, rt.spec.meta)
	}
	if !p.declaresProviderOutput(TypedService[Profile]()) {
		ProvideValueT[Profile](rt.container, rt.spec.profile)
	}
}

func (d frameworkConfigDeclarations) needsProviderRegistration(spec *appSpec) bool {
	if spec == nil {
		return false
	}
	return spec.eventLoggerFromContainer != nil ||
		spec.loggerFromContainer != nil ||
		d.slogLogger ||
		(!spec.eventLoggerConfigured && d.eventLogger) ||
		d.appMeta ||
		d.profile ||
		(!spec.observersConfigured && d.hasObservers())
}

func (d frameworkConfigDeclarations) hasObservers() bool {
	return d.observer || d.observerList
}

func (p *buildPlan) applyFrameworkLogger(rt *Runtime, declared frameworkConfigDeclarations) error {
	switch {
	case p.spec.loggerConfigured:
		p.applyConfiguredLogger(rt)
	case p.spec.loggerFromContainer != nil:
		return p.applyResolvedLogger(rt)
	case declared.slogLogger:
		return p.applyDeclaredSlogLogger(rt)
	}
	return nil
}

func (p *buildPlan) applyFrameworkEventLogger(rt *Runtime, declared frameworkConfigDeclarations) error {
	switch {
	case p.spec.eventLoggerFromContainer != nil:
		return p.applyResolvedEventLogger(rt)
	case !p.spec.eventLoggerConfigured && declared.eventLogger:
		return p.applyDeclaredEventLogger(rt)
	}
	return nil
}

func (p *buildPlan) applyFrameworkAppMeta(rt *Runtime, declared frameworkConfigDeclarations) error {
	if !declared.appMeta {
		return nil
	}
	return p.applyDeclaredAppMeta(rt)
}

func (p *buildPlan) applyResolvedEventLogger(rt *Runtime) error {
	resolvedEventLogger, err := p.resolveFrameworkEventLogger(rt)
	if err != nil {
		return err
	}
	applyRuntimeEventLogger(rt, resolvedEventLogger)
	return nil
}

func (p *buildPlan) applyDeclaredEventLogger(rt *Runtime) error {
	resolvedEventLogger, err := ResolveAs[EventLogger](rt.container)
	if err != nil {
		return oops.In("dix").
			With("op", "resolve_declared_event_logger", "app", rt.Name(), "service", serviceNameOf[EventLogger]()).
			Wrapf(err, "resolve declared event logger failed")
	}
	if resolvedEventLogger == nil {
		return oops.In("dix").
			With("op", "resolve_declared_event_logger", "app", rt.Name(), "service", serviceNameOf[EventLogger]()).
			New("resolve declared event logger failed: provider returned nil event logger")
	}
	applyRuntimeEventLogger(rt, resolvedEventLogger)
	return nil
}

func (p *buildPlan) applyDeclaredAppMeta(rt *Runtime) error {
	resolvedMeta, err := ResolveAs[AppMeta](rt.container)
	if err != nil {
		return oops.In("dix").
			With("op", "resolve_declared_app_meta", "app", rt.Name(), "service", serviceNameOf[AppMeta]()).
			Wrapf(err, "resolve declared app meta failed")
	}

	meta := rt.Meta()
	if meta.Name == "" {
		meta.Name = resolvedMeta.Name
	}
	if !p.spec.versionConfigured {
		meta.Version = resolvedMeta.Version
	}
	if !p.spec.descriptionConfigured {
		meta.Description = resolvedMeta.Description
	}

	rt.spec.meta = meta
	do.OverrideNamedValue(rt.container.Raw(), serviceNameOf[AppMeta](), meta)
	return nil
}

func (p *buildPlan) applyRuntimeProfile(rt *Runtime) {
	if rt == nil || rt.container == nil || rt.spec == nil {
		return
	}
	do.OverrideNamedValue(rt.container.Raw(), serviceNameOf[Profile](), rt.spec.profile)
}

func (p *buildPlan) applyResolvedLogger(rt *Runtime) error {
	resolvedLogger, err := p.resolveFrameworkLogger(rt)
	if err != nil {
		return err
	}
	p.applyRuntimeLogger(rt, resolvedLogger)
	do.OverrideNamedValue(rt.container.Raw(), serviceNameOf[*slog.Logger](), resolvedLogger)
	return nil
}

func (p *buildPlan) applyConfiguredLogger(rt *Runtime) {
	if rt == nil || rt.container == nil || rt.spec == nil || rt.spec.logger == nil {
		return
	}
	p.applyRuntimeLogger(rt, rt.spec.logger)
	do.OverrideNamedValue(rt.container.Raw(), serviceNameOf[*slog.Logger](), rt.spec.logger)
}

func (p *buildPlan) applyRuntimeLogger(rt *Runtime, logger *slog.Logger) {
	if rt == nil || logger == nil {
		return
	}
	rt.logger = logger
	rt.container.logger = logger
	rt.lifecycle.logger = logger
	if p == nil || p.spec == nil || p.spec.eventLogger == nil {
		applyRuntimeEventLogger(rt, NewSlogEventLogger(logger))
	}
}
