// Package main demonstrates overriding providers during dix module setup.
package main

import (
	"fmt"

	"github.com/DaiYuANg/arcgo/logx"
	"github.com/arcgolabs/arcgo/dix"
	dixadvanced "github.com/arcgolabs/arcgo/dix/advanced"
)

type appConfig struct {
	Env string
}

func main() {
	logger, err := logx.NewDevelopment()
	if err != nil {
		panic(err)
	}

	app := dix.New(
		"override",
		dix.UseLogger(logger),
		dix.WithModule(
			dix.NewModule("override",
				dix.WithModuleProviders(
					dix.Provider0(func() appConfig { return appConfig{Env: "dev"} }),
				),
				dix.WithModuleSetups(
					dixadvanced.Override0(func() appConfig { return appConfig{Env: "prod"} }),
				),
			),
		),
	)

	err = app.Validate()
	if err != nil {
		panic(err)
	}

	rt, err := app.Build()
	if err != nil {
		panic(err)
	}

	cfg, err := dix.ResolveAs[appConfig](rt.Container())
	if err != nil {
		panic(err)
	}

	printLine("override example")
	printValues("env:", cfg.Env)
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
