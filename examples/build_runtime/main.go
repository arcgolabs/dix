// Package main demonstrates building multiple runtimes from one dix app definition.
package main

import (
	"context"
	"fmt"

	"github.com/arcgolabs/logx"
	"github.com/arcgolabs/dix"
)

type greeting struct {
	Message string
}

func main() {
	logger, err := logx.NewDevelopment()
	if err != nil {
		panic(err)
	}

	module := dix.NewModule("greeting",
		dix.WithModuleProvider(dix.Provider0(func() greeting {
			return greeting{Message: "hello runtime"}
		})),
	)

	app := dix.New(
		"build-runtime",
		dix.WithVersion("0.6.0"),
		dix.UseLogger(logger),
		dix.WithModule(module),
	)

	first, err := app.Build()
	if err != nil {
		panic(err)
	}
	second, err := app.Build()
	if err != nil {
		panic(err)
	}

	greetingValue, err := dix.ResolveAs[greeting](first.Container())
	if err != nil {
		panic(err)
	}

	err = first.Start(context.Background())
	if err != nil {
		panic(err)
	}
	defer stopOrPanic(first)

	printLine("build runtime example")
	printValues("app name:", app.Name())
	printValues("first runtime started:", first.State() == dix.AppStateStarted)
	printValues("second runtime built:", second.State() == dix.AppStateBuilt)
	printValues("independent runtimes:", first != second)
	printLine(greetingValue.Message)
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
