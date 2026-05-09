// Package main demonstrates scoped runtime values in dix.
package main

import (
	"context"
	"fmt"

	"github.com/arcgolabs/dix"
	"github.com/arcgolabs/logx"
)

type appConfig struct {
	Name string
}

type requestContext struct {
	RequestID string
}

type scopedService struct {
	Config  appConfig
	Request requestContext
}

func main() {
	logger, err := logx.NewDevelopment()
	if err != nil {
		panic(err)
	}

	app := dix.New(
		"runtime-scope",
		dix.UseLogger(logger),
		dix.WithModule(
			dix.NewModule("root",
				dix.WithModuleProviders(
					dix.Provider0(func() appConfig {
						return appConfig{Name: "arcgo"}
					}),
				),
			),
		),
	)

	rt, err := app.Build()
	if err != nil {
		panic(err)
	}
	err = rt.Start(context.Background())
	if err != nil {
		panic(err)
	}
	defer stopOrPanic(rt)

	requestScope, err := rt.Scope("request-42", dix.ScopeFunc(func(c *dix.Container) {
		dix.ProvideValueT(c, requestContext{RequestID: "req-42"})
		dix.Provide2T(c, func(cfg appConfig, req requestContext) scopedService {
			return scopedService{Config: cfg, Request: req}
		})
	}))
	if err != nil {
		panic(err)
	}

	service, err := dix.ResolveAs[scopedService](requestScope)
	if err != nil {
		panic(err)
	}

	_, rootCanResolveRequest := dix.ResolveOptionalAs[requestContext](rt.Container())

	printLine("runtime scope example")
	printLine(service.Config.Name)
	printLine(service.Request.RequestID)
	printValues("root sees request context:", rootCanResolveRequest)
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

func printValues(values ...any) {
	if _, err := fmt.Println(values...); err != nil {
		panic(err)
	}
}
