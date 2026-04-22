// Package main demonstrates inspecting a built dix runtime.
package main

import (
	"fmt"

	"github.com/DaiYuANg/arcgo/logx"
	"github.com/arcgolabs/dix"
	dixadvanced "github.com/arcgolabs/dix/advanced"
)

func main() {
	logger, err := logx.NewDevelopment()
	if err != nil {
		panic(err)
	}

	app := dix.New(
		"inspect",
		dix.UseLogger(logger),
		dix.WithModule(
			dix.NewModule("inspect",
				dix.WithModuleProviders(
					dix.Provider0(func() string { return "hello" }),
					dixadvanced.NamedValue("tenant.default", "public"),
				),
			),
		),
	)

	rt, err := app.Build()
	if err != nil {
		panic(err)
	}

	_, err = dix.ResolveAs[string](rt.Container())
	if err != nil {
		panic(err)
	}
	_, err = dixadvanced.ResolveNamedAs[string](rt.Container(), "tenant.default")
	if err != nil {
		panic(err)
	}

	report := dixadvanced.InspectRuntime(rt, "tenant.default")
	hasTenantDeps := false
	if report.NamedDependencies != nil {
		_, hasTenantDeps = report.NamedDependencies.Get("tenant.default")
	}
	printLine("inspect example")
	printValues("provided:", report.ProvidedServices.Len())
	printValues("invoked:", report.InvokedServices.Len())
	printValues("has tenant deps:", hasTenantDeps)
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
