// Package main demonstrates scoped runtime values in dix.
package main

import (
	"context"
	"fmt"

	"github.com/DaiYuANg/arcgo/logx"
	"github.com/arcgolabs/arcgo/dix"
	dixadvanced "github.com/arcgolabs/arcgo/dix/advanced"
	do "github.com/samber/do/v2"
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

	requestScope := dixadvanced.Scope(rt, "request-42", func(injector do.Injector) {
		dixadvanced.ProvideScopedValue(injector, requestContext{RequestID: "req-42"})
		dixadvanced.ProvideScoped2(injector, func(cfg appConfig, req requestContext) scopedService {
			return scopedService{Config: cfg, Request: req}
		})
	})

	service, err := dixadvanced.ResolveScopedAs[scopedService](requestScope)
	if err != nil {
		panic(err)
	}

	_, rootCanResolveRequest := dixadvanced.ResolveRuntimeAs[requestContext](rt)

	printLine("runtime scope example")
	printLine(service.Config.Name)
	printLine(service.Request.RequestID)
	printValues("root sees request context:", rootCanResolveRequest == nil)
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
