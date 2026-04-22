// Package main demonstrates dix framework configuration resolved from DI.
package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/arcgolabs/dix"
)

type databaseConfig struct {
	DSN string
}

type auditObserver struct {
	mu     sync.Mutex
	builds int
	starts int
	stops  int
}

func (o *auditObserver) OnBuild(context.Context, dix.BuildEvent) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.builds++
}

func (o *auditObserver) OnStart(context.Context, dix.StartEvent) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.starts++
}

func (o *auditObserver) OnStop(context.Context, dix.StopEvent) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.stops++
}

func (*auditObserver) OnHealthCheck(context.Context, dix.HealthCheckEvent) {}
func (*auditObserver) OnStateTransition(context.Context, dix.StateTransitionEvent) {
}

type stdoutEventLogger struct{}

func (stdoutEventLogger) LogEvent(_ context.Context, event dix.Event) {
	switch e := event.(type) {
	case dix.BuildEvent:
		printFormat("event build app=%s profile=%s version=%s\n", e.Meta.Name, e.Profile, e.Meta.Version)
	case dix.StartEvent:
		printFormat("event start app=%s profile=%s\n", e.Meta.Name, e.Profile)
	case dix.StopEvent:
		printFormat("event stop app=%s profile=%s\n", e.Meta.Name, e.Profile)
	}
}

func main() {
	observer := &auditObserver{}
	configModule := dix.NewModule("framework-config",
		dix.Providers(
			dix.Provider0(func() dix.Profile {
				return dix.ProfileProd
			}),
			dix.Provider0(func() dix.AppMeta {
				return dix.AppMeta{
					Version:     "1.0.0",
					Description: "framework config resolved from DI",
				}
			}),
			dix.Provider0(func() dix.Observer {
				return observer
			}),
			dix.Provider0(func() dix.EventLogger {
				return stdoutEventLogger{}
			}),
		),
	)

	devDatabaseModule := dix.NewModule("dev-database",
		dix.UseProfiles(dix.ProfileDev),
		dix.Providers(
			dix.Provider0(func() databaseConfig {
				return databaseConfig{DSN: "sqlite://dev.db"}
			}),
		),
	)
	prodDatabaseModule := dix.NewModule("prod-database",
		dix.UseProfiles(dix.ProfileProd),
		dix.Providers(
			dix.Provider0(func() databaseConfig {
				return databaseConfig{DSN: "postgres://prod.internal/app"}
			}),
		),
	)

	app := dix.New("container-config",
		dix.Modules(configModule, devDatabaseModule, prodDatabaseModule),
	)

	rt, err := app.Start(context.Background())
	if err != nil {
		panic(err)
	}

	db, err := dix.ResolveAs[databaseConfig](rt.Container())
	if err != nil {
		panic(err)
	}

	stopOrPanic(rt)

	printFormat("runtime profile: %s\n", rt.Profile())
	printFormat("runtime version: %s\n", rt.Meta().Version)
	printFormat("selected database: %s\n", db.DSN)
	printFormat("observer counts: builds=%d starts=%d stops=%d\n", observer.builds, observer.starts, observer.stops)
}

func stopOrPanic(rt *dix.Runtime) {
	if err := rt.Stop(context.Background()); err != nil {
		panic(err)
	}
}

func printFormat(format string, args ...any) {
	if _, err := fmt.Printf(format, args...); err != nil {
		panic(err)
	}
}
