package dix

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/DaiYuANg/arcgo/collectionx"
)

// Event is an internal dix framework event emitted to EventLogger implementations.
type Event interface {
	dixEvent()
}

// EventLevel is the severity level for MessageEvent.
type EventLevel string

const (
	// EventLevelDebug is the debug severity.
	EventLevelDebug EventLevel = "debug"
	// EventLevelInfo is the info severity.
	EventLevelInfo EventLevel = "info"
	// EventLevelWarn is the warn severity.
	EventLevelWarn EventLevel = "warn"
	// EventLevelError is the error severity.
	EventLevelError EventLevel = "error"
)

// EventField is a structured field attached to MessageEvent.
type EventField struct {
	Key   string
	Value any
}

// MessageEvent carries structured dix framework log messages that do not map to a higher-level lifecycle event.
type MessageEvent struct {
	Level   EventLevel
	Message string
	Fields  collectionx.List[EventField]
}

// EventLogger receives all internal dix logging events.
type EventLogger interface {
	LogEvent(context.Context, Event)
}

type eventLoggerEnabler interface {
	Enabled(context.Context, EventLevel) bool
}

func (BuildEvent) dixEvent()           {}
func (StartEvent) dixEvent()           {}
func (StopEvent) dixEvent()            {}
func (HealthCheckEvent) dixEvent()     {}
func (StateTransitionEvent) dixEvent() {}
func (MessageEvent) dixEvent()         {}

// NewSlogEventLogger adapts a slog logger to the dix EventLogger interface.
func NewSlogEventLogger(logger *slog.Logger) EventLogger {
	if logger == nil {
		logger = defaultLogger()
	}
	return &slogEventLogger{logger: logger}
}

type slogEventLogger struct {
	logger *slog.Logger
}

func (l *slogEventLogger) Enabled(ctx context.Context, level EventLevel) bool {
	if l == nil || l.logger == nil {
		return false
	}
	return l.logger.Enabled(contextOrBackground(ctx), slogLevelFromEvent(level))
}

func (l *slogEventLogger) LogEvent(ctx context.Context, event Event) {
	if l == nil || l.logger == nil || event == nil {
		return
	}

	ctx = contextOrBackground(ctx)

	switch e := event.(type) {
	case MessageEvent:
		l.logMessage(ctx, e)
	case BuildEvent:
		l.logBuild(e)
	case StartEvent:
		l.logStart(e)
	case StopEvent:
		l.logStop(e)
	case HealthCheckEvent:
		l.logHealthCheck(e)
	case StateTransitionEvent:
		l.logger.Debug("runtime state transition",
			"app", e.Meta.Name,
			"from", e.From.String(),
			"to", e.To.String(),
			"reason", e.Reason,
		)
	}
}

func (l *slogEventLogger) logBuild(event BuildEvent) {
	if event.Err != nil {
		l.logger.Error("app build failed", "app", event.Meta.Name, "profile", event.Profile, "error", event.Err)
		return
	}
	l.logger.Info("app built",
		"app", event.Meta.Name,
		"profile", event.Profile,
		"modules", event.ModuleCount,
		"providers", event.ProviderCount,
		"hooks", event.HookCount,
		"setups", event.SetupCount,
		"invokes", event.InvokeCount,
	)
}

func (l *slogEventLogger) logStart(event StartEvent) {
	if event.Err != nil {
		l.logger.Error("app start failed", "app", event.Meta.Name, "error", event.Err)
		return
	}
	l.logger.Info("app started", "app", event.Meta.Name)
}

func (l *slogEventLogger) logStop(event StopEvent) {
	if event.Err != nil {
		l.logger.Error("app stop failed", "app", event.Meta.Name, "error", event.Err)
		return
	}
	l.logger.Info("app stopped", "app", event.Meta.Name)
}

func (l *slogEventLogger) logHealthCheck(event HealthCheckEvent) {
	if event.Err != nil {
		l.logger.Warn("health check failed", "kind", event.Kind, "check", event.Name, "error", event.Err)
		return
	}
	l.logger.Debug("health check passed", "kind", event.Kind, "check", event.Name)
}

func (l *slogEventLogger) logMessage(ctx context.Context, event MessageEvent) {
	args := eventFieldArgs(event.Fields)
	switch event.Level {
	case EventLevelDebug:
		l.logger.DebugContext(ctx, event.Message, args...)
	case EventLevelInfo:
		l.logger.InfoContext(ctx, event.Message, args...)
	case EventLevelWarn:
		l.logger.WarnContext(ctx, event.Message, args...)
	case EventLevelError:
		l.logger.ErrorContext(ctx, event.Message, args...)
	default:
		l.logger.Log(ctx, slogLevelFromEvent(event.Level), event.Message, args...)
	}
}

func contextOrBackground(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func eventLoggerEnabled(ctx context.Context, logger EventLogger, level EventLevel) bool {
	if logger == nil {
		return false
	}
	if enabler, ok := logger.(eventLoggerEnabler); ok {
		return enabler.Enabled(contextOrBackground(ctx), level)
	}
	return true
}

func emitEventLogger(ctx context.Context, logger EventLogger, event Event) {
	if logger == nil || event == nil {
		return
	}
	logger.LogEvent(contextOrBackground(ctx), event)
}

func logMessageEvent(ctx context.Context, logger EventLogger, level EventLevel, message string, args ...any) {
	if !eventLoggerEnabled(ctx, logger, level) {
		return
	}
	emitEventLogger(ctx, logger, MessageEvent{
		Level:   level,
		Message: message,
		Fields:  eventFields(args...),
	})
}

func eventFields(args ...any) collectionx.List[EventField] {
	if len(args) == 0 {
		return collectionx.NewList[EventField]()
	}

	fields := collectionx.NewListWithCapacity[EventField]((len(args) + 1) / 2)
	for i := 0; i < len(args); i += 2 {
		key := fmt.Sprintf("arg_%d", i)
		if name, ok := args[i].(string); ok && name != "" {
			key = name
		}

		var value any
		if i+1 < len(args) {
			value = args[i+1]
		}

		fields.Add(EventField{Key: key, Value: value})
	}
	return fields
}

func eventFieldArgs(fields collectionx.List[EventField]) []any {
	if fields == nil || fields.Len() == 0 {
		return nil
	}
	return collectionx.FlatMapList(fields, func(_ int, field EventField) []any {
		return []any{field.Key, field.Value}
	}).Values()
}

func slogLevelFromEvent(level EventLevel) slog.Level {
	switch level {
	case EventLevelDebug:
		return slog.LevelDebug
	case EventLevelInfo:
		return slog.LevelInfo
	case EventLevelWarn:
		return slog.LevelWarn
	case EventLevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func (spec *appSpec) resolvedEventLogger() EventLogger {
	if spec == nil {
		return nil
	}
	if spec.eventLogger != nil {
		return spec.eventLogger
	}
	if spec.logger != nil {
		return NewSlogEventLogger(spec.logger)
	}
	return nil
}
