package dix

import (
	"context"
	"time"

	collectionlist "github.com/arcgolabs/collectionx/list"
)

const defaultEventRecorderCapacity = 128

// EventRecord is one timestamped framework event captured by EventRecorder.
type EventRecord struct {
	At    time.Time
	Event Event
}

// EventRecorder stores recent framework events in a concurrent ring buffer.
type EventRecorder struct {
	buffer *collectionlist.ConcurrentRingBuffer[EventRecord]
}

// NewEventRecorder creates an event recorder with the requested capacity.
func NewEventRecorder(capacity int) *EventRecorder {
	if capacity <= 0 {
		capacity = defaultEventRecorderCapacity
	}
	return &EventRecorder{buffer: collectionlist.NewConcurrentRingBuffer[EventRecord](capacity)}
}

// LogEvent records an EventLogger event.
func (r *EventRecorder) LogEvent(_ context.Context, event Event) {
	r.record(event)
}

// OnBuild records a build event.
func (r *EventRecorder) OnBuild(_ context.Context, event BuildEvent) {
	r.record(event)
}

// OnStart records a start event.
func (r *EventRecorder) OnStart(_ context.Context, event StartEvent) {
	r.record(event)
}

// OnStop records a stop event.
func (r *EventRecorder) OnStop(_ context.Context, event StopEvent) {
	r.record(event)
}

// OnHealthCheck records a health check event.
func (r *EventRecorder) OnHealthCheck(_ context.Context, event HealthCheckEvent) {
	r.record(event)
}

// OnStateTransition records a state transition event.
func (r *EventRecorder) OnStateTransition(_ context.Context, event StateTransitionEvent) {
	r.record(event)
}

// OnProvider records a provider diagnostic event.
func (r *EventRecorder) OnProvider(_ context.Context, event ProviderEvent) {
	r.record(event)
}

// OnResolve records a resolve diagnostic event.
func (r *EventRecorder) OnResolve(_ context.Context, event ResolveEvent) {
	r.record(event)
}

// OnLifecycleHook records a lifecycle hook diagnostic event.
func (r *EventRecorder) OnLifecycleHook(_ context.Context, event LifecycleHookEvent) {
	r.record(event)
}

// Snapshot returns a point-in-time ring-buffer snapshot.
func (r *EventRecorder) Snapshot() *collectionlist.RingBuffer[EventRecord] {
	if r == nil || r.buffer == nil {
		return collectionlist.NewRingBuffer[EventRecord](defaultEventRecorderCapacity)
	}
	return r.buffer.Snapshot()
}

// Events returns recent events in FIFO order.
func (r *EventRecorder) Events() *collectionlist.List[EventRecord] {
	if r == nil || r.buffer == nil {
		return collectionlist.NewList[EventRecord]()
	}
	values := r.buffer.Values()
	return collectionlist.NewListWithCapacity[EventRecord](len(values), values...)
}

// Len returns the number of currently buffered events.
func (r *EventRecorder) Len() int {
	if r == nil || r.buffer == nil {
		return 0
	}
	return r.buffer.Len()
}

// Capacity returns the configured ring buffer capacity.
func (r *EventRecorder) Capacity() int {
	if r == nil || r.buffer == nil {
		return 0
	}
	return r.buffer.Capacity()
}

// Clear removes all buffered events.
func (r *EventRecorder) Clear() {
	if r == nil || r.buffer == nil {
		return
	}
	r.buffer.Clear()
}

func (r *EventRecorder) record(event Event) {
	if r == nil || r.buffer == nil || event == nil {
		return
	}
	r.buffer.Push(EventRecord{
		At:    time.Now(),
		Event: event,
	})
}

type recordingEventLogger struct {
	recorder *EventRecorder
	next     EventLogger
}

func newRecordingEventLogger(recorder *EventRecorder, next EventLogger) EventLogger {
	if recorder == nil {
		return next
	}
	if current, ok := next.(recordingEventLogger); ok && current.recorder == recorder {
		return next
	}
	return recordingEventLogger{recorder: recorder, next: next}
}

func (l recordingEventLogger) Enabled(ctx context.Context, level EventLevel) bool {
	if l.next == nil {
		return true
	}
	return eventLoggerEnabled(ctx, l.next, level)
}

func (l recordingEventLogger) LogEvent(ctx context.Context, event Event) {
	l.recorder.record(event)
	emitEventLogger(ctx, l.next, event)
}
