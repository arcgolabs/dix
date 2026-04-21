// Package main demonstrates transient provider behavior in dix.
package main

import (
	"fmt"

	"github.com/DaiYuANg/arcgo/logx"
	"github.com/arcgolabs/arcgo/dix"
	dixadvanced "github.com/arcgolabs/arcgo/dix/advanced"
)

func main() {
	logger, err := logx.NewDevelopment()
	if err != nil {
		panic(err)
	}

	counter := 0
	app := dix.New(
		"transient",
		dix.UseLogger(logger),
		dix.WithModule(
			dix.NewModule("transient",
				dix.Providers(
					dixadvanced.Transient(func() int {
						counter++
						return counter
					}),
				),
			),
		),
	)

	rt, err := app.Build()
	if err != nil {
		panic(err)
	}

	first, err := dix.ResolveAs[int](rt.Container())
	if err != nil {
		panic(err)
	}
	second, err := dix.ResolveAs[int](rt.Container())
	if err != nil {
		panic(err)
	}

	printLine("transient example")
	printValues("first:", first)
	printValues("second:", second)
	printValues("different:", first != second)
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
