// Package main demonstrates grouping dix dependencies into an aggregate parameter object.
package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/DaiYuANg/arcgo/logx"
	"github.com/arcgolabs/arcgo/dix"
)

type dbConfig struct {
	DSN string
}

type repositoryParams struct {
	Logger *slog.Logger
	Cfg    dbConfig
}

type repository struct {
	params repositoryParams
}

func main() {
	logger, err := logx.NewDevelopment()
	if err != nil {
		panic(err)
	}

	module := dix.NewModule("repository",
		dix.Providers(
			dix.Provider0(func() dbConfig { return dbConfig{DSN: "postgres://demo"} }),
			dix.Provider2(func(logger *slog.Logger, cfg dbConfig) repositoryParams {
				return repositoryParams{Logger: logger, Cfg: cfg}
			}),
			dix.Provider1(func(params repositoryParams) *repository {
				return &repository{params: params}
			}),
		),
	)

	app := dix.New("aggregate-params", dix.Modules(module), dix.UseLogger(logger))
	rt, err := app.Start(context.Background())
	if err != nil {
		panic(err)
	}
	defer stopOrPanic(rt)

	repo, err := dix.ResolveAs[*repository](rt.Container())
	if err != nil {
		panic(err)
	}

	printLine("aggregate params example")
	printLine(repo.params.Cfg.DSN)
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
