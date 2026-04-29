package dix

import (
	"context"
	collectionlist "github.com/arcgolabs/collectionx/list"
	"log/slog"
	"sync"
)

type observerDispatchCall struct {
	ctx  context.Context
	emit func(context.Context, Observer)
}

type observerDispatcher struct {
	index          int
	observer       Observer
	loggerProvider func() *slog.Logger

	queue     *collectionlist.ConcurrentDeque[observerDispatchCall]
	notify    chan struct{}
	startOnce sync.Once
}

func newObserverDispatcher(index int, observer Observer, loggerProvider func() *slog.Logger) *observerDispatcher {
	return &observerDispatcher{
		index:          index,
		observer:       observer,
		loggerProvider: loggerProvider,
		queue:          collectionlist.NewConcurrentDeque[observerDispatchCall](),
		notify:         make(chan struct{}, 1),
	}
}

func (spec *appSpec) appendObserversWithLogger(loggerProvider func() *slog.Logger, observers ...Observer) {
	if spec == nil {
		return
	}
	if spec.observers == nil {
		spec.observers = collectionlist.NewList[Observer]()
	}
	if spec.observerDispatchers == nil {
		spec.observerDispatchers = collectionlist.NewList[*observerDispatcher]()
	}
	for _, observer := range observers {
		if observer == nil {
			continue
		}
		spec.observers.Add(observer)
		spec.observerDispatchers.Add(newObserverDispatcher(spec.observers.Len()-1, observer, loggerProvider))
	}
}

func (spec *appSpec) rebuildObserverDispatchers(loggerProvider func() *slog.Logger) {
	if spec == nil {
		return
	}
	spec.observerDispatchers = collectionlist.NewListWithCapacity[*observerDispatcher](spec.observers.Len())
	observers := spec.observers.Values()
	for index, observer := range observers {
		if observer == nil {
			continue
		}
		spec.observerDispatchers.Add(newObserverDispatcher(index, observer, loggerProvider))
	}
}

func (d *observerDispatcher) enqueue(ctx context.Context, emit func(context.Context, Observer)) {
	if d == nil || d.observer == nil || emit == nil {
		return
	}

	d.startOnce.Do(func() {
		go d.run()
	})
	d.queue.PushBack(observerDispatchCall{
		ctx:  contextOrBackground(ctx),
		emit: emit,
	})
	select {
	case d.notify <- struct{}{}:
	default:
	}
}

func (d *observerDispatcher) run() {
	for {
		<-d.notify
		for {
			call, ok := d.queue.PopFront()
			if !ok {
				break
			}
			d.invoke(call)
		}
	}
}

func (d *observerDispatcher) invoke(call observerDispatchCall) {
	defer func() {
		logger := d.logger()
		if recovered := recover(); recovered != nil && logger != nil {
			logger.Error("dix observer panicked", "observer_index", d.index, "panic", recovered)
		}
	}()
	call.emit(call.ctx, d.observer)
}

func (d *observerDispatcher) logger() *slog.Logger {
	if d == nil || d.loggerProvider == nil {
		return nil
	}
	return d.loggerProvider()
}
