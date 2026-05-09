// Package main demonstrates alias binding and named alias resolution in dix.
package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/arcgolabs/dix"
	dixadvanced "github.com/arcgolabs/dix/advanced"
	"github.com/arcgolabs/logx"
)

type greeter interface {
	Greet() string
}

type englishGreeter struct {
	logger *slog.Logger
}

func (g *englishGreeter) Greet() string {
	g.logger.Info("greet invoked", "lang", "en")
	return "hello"
}

func main() {
	logger, err := logx.NewDevelopment()
	if err != nil {
		panic(err)
	}

	serviceModule := dix.NewModule("greeter",
		dix.Providers(
			dix.Provider1(func(logger *slog.Logger) *englishGreeter {
				return &englishGreeter{logger: logger}
			}),
			dixadvanced.Named(dix.NamedService[string]("locale.default"), "en-US"),
			dixadvanced.NamedProvider1(dix.NamedService[*englishGreeter]("greeter.en"), func(logger *slog.Logger) *englishGreeter {
				return &englishGreeter{logger: logger}
			}),
		),
		dix.Setups(
			dixadvanced.Alias[*englishGreeter, greeter](),
			dixadvanced.NamedAlias(
				dix.NamedService[*englishGreeter]("greeter.en"),
				dix.NamedService[greeter]("greeter.en.alias"),
			),
		),
	)

	app := dix.New("named-alias", dix.Modules(serviceModule), dix.UseLogger(logger))
	rt, err := app.Start(context.Background())
	if err != nil {
		panic(err)
	}
	defer stopOrPanic(rt)

	locale, err := dix.ResolveKey(rt.Container(), dix.NamedService[string]("locale.default"))
	if err != nil {
		panic(err)
	}
	printValues("locale:", locale)

	greeterValue, err := dix.ResolveAs[greeter](rt.Container())
	if err != nil {
		panic(err)
	}
	printValues("implicit/assignable alias:", greeterValue.Greet())

	namedAlias, err := dix.ResolveKey(rt.Container(), dix.NamedService[greeter]("greeter.en.alias"))
	if err != nil {
		panic(err)
	}
	printValues("named explicit alias:", namedAlias.Greet())
}

func stopOrPanic(rt *dix.Runtime) {
	if err := rt.Stop(context.Background()); err != nil {
		panic(err)
	}
}

func printValues(values ...any) {
	if _, err := fmt.Println(values...); err != nil {
		panic(err)
	}
}
