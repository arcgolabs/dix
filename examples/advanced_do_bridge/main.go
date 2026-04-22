// Package main demonstrates bridging a `do` injector into a dix module.
package main

import (
	"context"
	"fmt"

	"github.com/arcgolabs/logx"
	"github.com/arcgolabs/dix"
	dixadvanced "github.com/arcgolabs/dix/advanced"
	do "github.com/samber/do/v2"
)

type namedValue string

func main() {
	logger, err := logx.NewDevelopment()
	if err != nil {
		panic(err)
	}

	module := dix.NewModule("advanced-bridge",
		dix.WithModuleSetups(
			dixadvanced.DoSetup(func(raw do.Injector) error {
				do.ProvideNamedValue(raw, "tenant.default", namedValue("public"))
				return nil
			}),
		),
	)

	app := dix.New(
		"advanced-do-bridge",
		dix.WithDebugScopeTree(true),
		dix.WithDebugNamedServiceDependencies("tenant.default"),
		dix.UseLogger(logger),
		dix.WithModule(module),
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

	value, err := dixadvanced.ResolveNamedAs[namedValue](rt.Container(), "tenant.default")
	if err != nil {
		panic(err)
	}

	printLine("advanced do bridge example")
	printLine(value)
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
