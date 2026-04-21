//revive:disable:file-length-limit Observer tests keep the lifecycle event matrix in one place.

package dix_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/arcgolabs/arcgo/dix"
	"github.com/stretchr/testify/require"
)

type recordingObserver struct {
	mu          sync.Mutex
	builds      []dix.BuildEvent
	starts      []dix.StartEvent
	stops       []dix.StopEvent
	health      []dix.HealthCheckEvent
	transitions []dix.StateTransitionEvent
}

func (r *recordingObserver) snapshot() ([]dix.BuildEvent, []dix.StartEvent, []dix.StopEvent, []dix.HealthCheckEvent, []dix.StateTransitionEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()

	builds := append([]dix.BuildEvent(nil), r.builds...)
	starts := append([]dix.StartEvent(nil), r.starts...)
	stops := append([]dix.StopEvent(nil), r.stops...)
	health := append([]dix.HealthCheckEvent(nil), r.health...)
	transitions := append([]dix.StateTransitionEvent(nil), r.transitions...)
	return builds, starts, stops, health, transitions
}

func (r *recordingObserver) OnBuild(_ context.Context, event dix.BuildEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.builds = append(r.builds, event)
}

func (r *recordingObserver) OnStart(_ context.Context, event dix.StartEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.starts = append(r.starts, event)
}

func (r *recordingObserver) OnStop(_ context.Context, event dix.StopEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stops = append(r.stops, event)
}

func (r *recordingObserver) OnHealthCheck(_ context.Context, event dix.HealthCheckEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.health = append(r.health, event)
}

func (r *recordingObserver) OnStateTransition(_ context.Context, event dix.StateTransitionEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.transitions = append(r.transitions, event)
}

func TestObserverReceivesLifecycleEvents(t *testing.T) {
	observer := &recordingObserver{}
	app := newObserverLifecycleApp(observer)

	rt, err := app.Build()
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if err := rt.Start(context.Background()); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	report := rt.CheckHealth(context.Background())
	if !report.Healthy() {
		t.Fatalf("expected healthy report, got %v", report.Error())
	}
	if err := rt.Stop(context.Background()); err != nil {
		t.Fatalf("stop failed: %v", err)
	}

	waitForObserverEvents(t, observer, 1, 1, 1, 4)
	assertObserverBuild(t, observer)
	assertObserverStart(t, observer)
	assertObserverHealth(t, observer)
	assertObserverStop(t, observer)
	assertObserverTransitions(t, observer)
}

func TestDIProvidedObserverReceivesLifecycleEvents(t *testing.T) {
	observer := &recordingObserver{}
	app := dix.New("observer-app",
		dix.WithModule(
			dix.NewModule("observer",
				dix.WithModuleProviders(
					dix.Provider0(func() dix.Observer { return observer }),
				),
				dix.Setups(dix.Setup(func(c *dix.Container, _ dix.Lifecycle) error {
					c.RegisterHealthCheck("db", func(context.Context) error { return nil })
					return nil
				})),
				dix.Hooks(
					dix.OnStart0(func(context.Context) error { return nil }),
					dix.OnStop0(func(context.Context) error { return nil }),
				),
			),
		),
	)

	rt, err := app.Build()
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if err := rt.Start(context.Background()); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	if !rt.CheckHealth(context.Background()).Healthy() {
		t.Fatal("expected healthy report")
	}
	if err := rt.Stop(context.Background()); err != nil {
		t.Fatalf("stop failed: %v", err)
	}

	waitForObserverEvents(t, observer, 1, 1, 1, 4)
	assertObserverBuild(t, observer)
	assertObserverStart(t, observer)
	assertObserverHealth(t, observer)
	assertObserverStop(t, observer)
	assertObserverTransitions(t, observer)
}

func TestDIProvidedObserverListReceivesLifecycleEvents(t *testing.T) {
	first := &recordingObserver{}
	second := &recordingObserver{}
	app := dix.New("di-observer-list",
		dix.WithModule(
			dix.NewModule("observer",
				dix.Providers(
					dix.Provider0(func() collectionx.List[dix.Observer] {
						return collectionx.NewList[dix.Observer](first, second)
					}),
				),
			),
		),
	)

	_, err := app.Build()
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}

	waitForObserverEvents(t, first, 0, 0, 0, 1)
	waitForObserverEvents(t, second, 0, 0, 0, 1)
	firstBuilds, _, _, _, _ := first.snapshot()
	secondBuilds, _, _, _, _ := second.snapshot()
	if len(firstBuilds) != 1 {
		t.Fatalf("expected first observer to receive build event, got %d", len(firstBuilds))
	}
	if len(secondBuilds) != 1 {
		t.Fatalf("expected second observer to receive build event, got %d", len(secondBuilds))
	}
}

func TestExplicitObserverTakesPriorityOverDIProvidedObserver(t *testing.T) {
	explicitObserver := &recordingObserver{}
	diObserver := &recordingObserver{}
	app := dix.New("explicit-observer",
		dix.WithObserver(explicitObserver),
		dix.WithModule(
			dix.NewModule("observer",
				dix.WithModuleProviders(
					dix.Provider0(func() dix.Observer { return diObserver }),
				),
			),
		),
	)

	_, err := app.Build()
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}

	waitForObserverEvents(t, explicitObserver, 0, 0, 0, 1)
	explicitBuilds, _, _, _, _ := explicitObserver.snapshot()
	diBuilds, _, _, _, _ := diObserver.snapshot()
	if len(explicitBuilds) != 1 {
		t.Fatalf("expected explicit observer to receive build event, got %d", len(explicitBuilds))
	}
	if len(diBuilds) != 0 {
		t.Fatalf("expected DI observer to be ignored, got %d build events", len(diBuilds))
	}
}

func newObserverLifecycleApp(observer dix.Observer) *dix.App {
	return dix.New("observer-app",
		dix.WithObserver(observer),
		dix.WithModule(
			dix.NewModule("health",
				dix.Setups(dix.Setup(func(c *dix.Container, _ dix.Lifecycle) error {
					c.RegisterHealthCheck("db", func(context.Context) error { return nil })
					return nil
				})),
				dix.Hooks(
					dix.OnStart0(func(context.Context) error { return nil }),
					dix.OnStop0(func(context.Context) error { return nil }),
				),
			),
		),
	)
}

func assertObserverBuild(t *testing.T, observer *recordingObserver) {
	t.Helper()
	builds, _, _, _, _ := observer.snapshot()
	if len(builds) != 1 {
		t.Fatalf("expected 1 build event, got %d", len(builds))
	}
	build := builds[0]
	if build.Meta.Name != "observer-app" {
		t.Fatalf("expected build app name observer-app, got %q", build.Meta.Name)
	}
	if build.ModuleCount != 1 || build.SetupCount != 1 || build.HookCount != 2 {
		t.Fatalf("unexpected build counts: %+v", build)
	}
	if build.Err != nil {
		t.Fatalf("expected successful build event, got %v", build.Err)
	}
}

func assertObserverStart(t *testing.T, observer *recordingObserver) {
	t.Helper()
	_, starts, _, _, _ := observer.snapshot()
	if len(starts) != 1 {
		t.Fatalf("expected 1 start event, got %d", len(starts))
	}
	start := starts[0]
	if start.StartHookCount != 1 || start.StartedHookCount != 1 {
		t.Fatalf("unexpected start counts: %+v", start)
	}
	if start.Err != nil {
		t.Fatalf("expected successful start event, got %v", start.Err)
	}
}

func assertObserverHealth(t *testing.T, observer *recordingObserver) {
	t.Helper()
	_, _, _, healthEvents, _ := observer.snapshot()
	if len(healthEvents) != 1 {
		t.Fatalf("expected 1 health event, got %d", len(healthEvents))
	}
	health := healthEvents[0]
	if health.Kind != dix.HealthKindGeneral || health.Name != "db" {
		t.Fatalf("unexpected health event: %+v", health)
	}
	if health.Err != nil {
		t.Fatalf("expected successful health event, got %v", health.Err)
	}
}

func assertObserverStop(t *testing.T, observer *recordingObserver) {
	t.Helper()
	_, _, stops, _, _ := observer.snapshot()
	if len(stops) != 1 {
		t.Fatalf("expected 1 stop event, got %d", len(stops))
	}
	stop := stops[0]
	if stop.StopHookCount != 1 {
		t.Fatalf("unexpected stop counts: %+v", stop)
	}
	if stop.Err != nil {
		t.Fatalf("expected successful stop event, got %v", stop.Err)
	}
}

func assertObserverTransitions(t *testing.T, observer *recordingObserver) {
	t.Helper()
	_, _, _, _, transitions := observer.snapshot()
	if len(transitions) != 4 {
		t.Fatalf("expected 4 transitions, got %d", len(transitions))
	}
	expected := []struct {
		from dix.AppState
		to   dix.AppState
	}{
		{from: dix.AppStateCreated, to: dix.AppStateBuilt},
		{from: dix.AppStateBuilt, to: dix.AppStateStarting},
		{from: dix.AppStateStarting, to: dix.AppStateStarted},
		{from: dix.AppStateStarted, to: dix.AppStateStopped},
	}
	for index, transition := range expected {
		got := transitions[index]
		if got.From != transition.from || got.To != transition.to {
			t.Fatalf("unexpected transition at %d: %+v", index, got)
		}
	}
}

func TestObserverReceivesBuildFailureEvent(t *testing.T) {
	observer := &recordingObserver{}
	app := dix.New("observer-build-failure",
		dix.WithObserver(observer),
		dix.WithModule(
			dix.NewModule("broken",
				dix.WithModuleInvokes(dix.RawInvoke(func(*dix.Container) error {
					return errors.New("boom")
				})),
			),
		),
	)

	_, err := app.Build()
	if err == nil {
		t.Fatal("expected build failure")
	}

	waitForObserverEvents(t, observer, 0, 0, 0, 0)
	builds, _, _, _, _ := observer.snapshot()
	if len(builds) != 1 {
		t.Fatalf("expected 1 build event, got %d", len(builds))
	}
	if builds[0].Err == nil {
		t.Fatal("expected build event error to be set")
	}
}

type blockingObserver struct {
	invoked chan struct{}
	release chan struct{}
	done    chan struct{}
}

func newBlockingObserver() *blockingObserver {
	return &blockingObserver{
		invoked: make(chan struct{}),
		release: make(chan struct{}),
		done:    make(chan struct{}),
	}
}

func (o *blockingObserver) OnBuild(context.Context, dix.BuildEvent) {
	close(o.invoked)
	<-o.release
	close(o.done)
}

func (*blockingObserver) OnStart(context.Context, dix.StartEvent)                     {}
func (*blockingObserver) OnStop(context.Context, dix.StopEvent)                       {}
func (*blockingObserver) OnHealthCheck(context.Context, dix.HealthCheckEvent)         {}
func (*blockingObserver) OnStateTransition(context.Context, dix.StateTransitionEvent) {}

func TestObserverDispatchIsNonBlocking(t *testing.T) {
	observer := newBlockingObserver()
	app := dix.New("observer-non-blocking", dix.WithObserver(observer))

	errCh := make(chan error, 1)
	go func() {
		_, err := app.Build()
		errCh <- err
	}()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("build failed: %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("build blocked on observer dispatch")
	}

	select {
	case <-observer.invoked:
	case <-time.After(time.Second):
		t.Fatal("expected observer to receive build event")
	}

	close(observer.release)

	select {
	case <-observer.done:
	case <-time.After(time.Second):
		t.Fatal("observer did not finish after release")
	}
}

func waitForObserverEvents(t *testing.T, observer *recordingObserver, starts, stops, health, transitions int) {
	t.Helper()
	require.Eventually(t, func() bool {
		currentBuilds, currentStarts, currentStops, currentHealth, currentTransitions := observer.snapshot()
		return len(currentBuilds) >= 1 &&
			len(currentStarts) >= starts &&
			len(currentStops) >= stops &&
			len(currentHealth) >= health &&
			len(currentTransitions) >= transitions
	}, time.Second, 10*time.Millisecond)
}
