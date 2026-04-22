// Package main demonstrates a minimal dix application with health checks.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/arcgolabs/logx"
	"github.com/arcgolabs/dix"
)

type appConfig struct {
	Port int
}

type server struct {
	Logger *slog.Logger
	Config appConfig
}

func main() {
	configModule := dix.NewModule("config",
		dix.WithModuleProviders(
			dix.Provider0(func() appConfig { return appConfig{Port: 8080} }),
		),
	)

	logger, err := logx.NewDevelopment()
	if err != nil {
		panic(err)
	}
	serverModule := dix.NewModule("server",
		dix.WithModuleImports(configModule),
		dix.WithModuleProviders(
			dix.Provider2(func(logger *slog.Logger, cfg appConfig) *server {
				return &server{Logger: logger, Config: cfg}
			}),
		),
		dix.Setups(dix.SetupContainer(func(c *dix.Container) error {
			c.RegisterLivenessCheck("process", func(context.Context) error { return nil })
			c.RegisterReadinessCheck("bootstrap", func(context.Context) error {
				server, ok := dix.ResolveOptional[*server](c)
				if !ok || server == nil {
					return errors.New("server not ready")
				}
				return nil
			})
			return nil
		})),
	)

	app := dix.NewDefault(
		dix.UseProfile(dix.ProfileDev),
		dix.Version("0.5.0"),
		dix.Modules(serverModule),
		dix.UseLogger(logger),
	)

	rt, err := app.Start(context.Background())
	if err != nil {
		panic(err)
	}
	defer stopOrPanic(rt)

	printLine("basic app built and started")
	printFormat("health: %v\n", rt.CheckHealth(context.Background()).Healthy())
	printFormat("liveness: %v\n", rt.CheckLiveness(context.Background()).Healthy())
	printFormat("readiness: %v\n", rt.CheckReadiness(context.Background()).Healthy())

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", rt.HealthHandler())
	mux.HandleFunc("/livez", rt.LivenessHandler())
	mux.HandleFunc("/readyz", rt.ReadinessHandler())
	_ = mux

	time.Sleep(100 * time.Millisecond)
}

func stopOrPanic(rt *dix.Runtime) {
	if err := rt.Stop(context.Background()); err != nil {
		panic(err)
	}
}

func printLine(value any) {
	if _, err := fmt.Println(value); err != nil {
		panic(err)
	}
}

func printFormat(format string, args ...any) {
	if _, err := fmt.Printf(format, args...); err != nil {
		panic(err)
	}
}
