// Package main demonstrates dix validation and build failures.
package main

import (
	"fmt"

	"github.com/arcgolabs/logx"
	"github.com/arcgolabs/dix"
)

type missingDependency struct {
	Name string
}

type needsMissingDependency struct {
	Name string
}

func main() {
	logger, err := logx.NewDevelopment()
	if err != nil {
		panic(err)
	}

	app := dix.New(
		"build-failure",
		dix.UseLogger(logger),
		dix.WithModule(
			dix.NewModule("broken",
				dix.WithModuleProviders(
					dix.Provider1(func(dep missingDependency) needsMissingDependency {
						return needsMissingDependency(dep)
					}),
				),
				dix.WithModuleInvokes(
					dix.Invoke1(func(value needsMissingDependency) {
						printLine(value.Name)
					}),
				),
			),
		),
	)

	err = app.Validate()
	if err != nil {
		printValues("validate error:", err)
	} else {
		printLine("validate error: <nil>")
	}

	_, err = app.Build()
	printValues("build error:", err != nil)
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
