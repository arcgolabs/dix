package dix

func (l *slogEventLogger) logProvider(event ProviderEvent) {
	args := []any{
		"app", event.Meta.Name,
		"profile", event.Profile,
		"module", event.Module,
		"label", event.Label,
		"service", event.Service,
		"op", event.Operation,
		"duration", event.Duration,
	}
	if event.Err != nil {
		l.logger.Error("provider operation failed", append(args, "error", event.Err)...)
		return
	}
	l.logger.Debug("provider operation completed", args...)
}

func (l *slogEventLogger) logResolve(event ResolveEvent) {
	args := []any{
		"app", event.Meta.Name,
		"profile", event.Profile,
		"service", event.Service,
		"op", event.Operation,
		"duration", event.Duration,
	}
	if event.Err != nil {
		l.logger.Error("service resolution failed", append(args, "error", event.Err)...)
		return
	}
	l.logger.Debug("service resolved", args...)
}

func (l *slogEventLogger) logLifecycleHook(event LifecycleHookEvent) {
	args := []any{
		"app", event.Meta.Name,
		"profile", event.Profile,
		"kind", string(event.Kind),
		"name", event.Name,
		"label", event.Label,
		"priority", event.Priority,
		"parallel", event.Parallel,
		"timeout", event.Timeout,
		"sequence", event.Sequence,
		"duration", event.Duration,
	}
	if event.Err != nil {
		l.logger.Error("lifecycle hook failed", append(args, "error", event.Err)...)
		return
	}
	l.logger.Debug("lifecycle hook completed", args...)
}
