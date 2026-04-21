package dix

import (
	"log/slog"

	"github.com/samber/oops"
)

func applyRuntimeEventLogger(rt *Runtime, logger EventLogger) {
	if rt == nil || logger == nil {
		return
	}
	rt.eventLogger = logger
	rt.container.eventLogger = logger
	rt.lifecycle.eventLogger = logger
}

func (p *buildPlan) applyDeclaredSlogLogger(rt *Runtime) error {
	logger, err := ResolveAs[*slog.Logger](rt.container)
	if err != nil {
		return oops.In("dix").
			With("op", "resolve_declared_slog_logger", "app", rt.Name(), "service", serviceNameOf[*slog.Logger]()).
			Wrapf(err, "resolve declared slog logger failed")
	}
	if logger == nil {
		return oops.In("dix").
			With("op", "resolve_declared_slog_logger", "app", rt.Name(), "service", serviceNameOf[*slog.Logger]()).
			New("resolve declared slog logger failed: provider returned nil logger")
	}
	p.applyRuntimeLogger(rt, logger)
	return nil
}

func (p *buildPlan) resolveFrameworkLogger(rt *Runtime) (*slog.Logger, error) {
	if p == nil || p.spec == nil || rt == nil || p.spec.loggerFromContainer == nil {
		return nil, oops.In("dix").
			With("op", "resolve_framework_logger").
			New("resolve framework logger failed: resolver is not configured")
	}

	logger, err := p.spec.loggerFromContainer(rt.container)
	if err != nil {
		return nil, oops.In("dix").
			With("op", "resolve_framework_logger", "app", rt.Name()).
			Wrapf(err, "resolve framework logger failed")
	}
	if logger == nil {
		return nil, oops.In("dix").
			With("op", "resolve_framework_logger", "app", rt.Name()).
			New("resolve framework logger failed: resolver returned nil logger")
	}

	return logger, nil
}

func (p *buildPlan) resolveFrameworkEventLogger(rt *Runtime) (EventLogger, error) {
	if p == nil || p.spec == nil || rt == nil || p.spec.eventLoggerFromContainer == nil {
		return nil, oops.In("dix").
			With("op", "resolve_framework_event_logger").
			New("resolve framework event logger failed: resolver is not configured")
	}

	logger, err := p.spec.eventLoggerFromContainer(rt.container)
	if err != nil {
		return nil, oops.In("dix").
			With("op", "resolve_framework_event_logger", "app", rt.Name()).
			Wrapf(err, "resolve framework event logger failed")
	}
	if logger == nil {
		return nil, oops.In("dix").
			With("op", "resolve_framework_event_logger", "app", rt.Name()).
			New("resolve framework event logger failed: resolver returned nil event logger")
	}

	return logger, nil
}
