// Package main demonstrates collection contributions in dix.
package main

import (
	"context"
	"fmt"
	"sort"

	"github.com/arcgolabs/collectionx"
	"github.com/arcgolabs/dix"
)

type appConfig struct {
	Tenant string
}

type endpoint interface {
	Name() string
	Route() string
}

type healthEndpoint struct{}

func (e *healthEndpoint) Name() string {
	return "health"
}

func (e *healthEndpoint) Route() string {
	return "GET /healthz"
}

type tenantEndpoint struct {
	tenant string
}

func (e *tenantEndpoint) Name() string {
	return "tenant"
}

func (e *tenantEndpoint) Route() string {
	return "GET /tenants/" + e.tenant
}

type debugEndpoint struct {
	tenant string
}

func (e *debugEndpoint) Name() string {
	return "debug"
}

func (e *debugEndpoint) Route() string {
	return "GET /debug/" + e.tenant
}

type router struct {
	endpoints []endpoint
	byName    collectionx.OrderedMap[string, endpoint]
}

func newRouter(endpoints []endpoint, byName collectionx.OrderedMap[string, endpoint]) *router {
	return &router{endpoints: endpoints, byName: byName}
}

func (r *router) routes() []string {
	routes := make([]string, 0, len(r.endpoints))
	for _, endpoint := range r.endpoints {
		routes = append(routes, endpoint.Route())
	}
	return routes
}

func main() {
	configModule := dix.NewModule("config",
		dix.Providers(
			dix.Provider0(func() appConfig {
				return appConfig{Tenant: "acme"}
			}),
		),
	)

	endpointModule := dix.NewModule("endpoints",
		dix.Imports(configModule),
		dix.Providers(
			dix.Provider0(func() *healthEndpoint {
				return &healthEndpoint{}
			}, dix.As[endpoint](), dix.Into[endpoint](dix.Key("health"), dix.Order(-100))),
			dix.Provider1(func(cfg appConfig) *tenantEndpoint {
				return &tenantEndpoint{tenant: cfg.Tenant}
			}, dix.Into[endpoint](dix.Key("tenant"), dix.Order(10))),
		),
	)

	pluginModule := dix.NewModule("plugin",
		dix.Imports(configModule),
		dix.Providers(
			dix.Contribute1[endpoint, appConfig](func(cfg appConfig) endpoint {
				return &debugEndpoint{tenant: cfg.Tenant}
			}, dix.Key("debug"), dix.Order(50)),
		),
	)

	serverModule := dix.NewModule("server",
		dix.Imports(endpointModule, pluginModule),
		dix.Providers(
			dix.Provider2(newRouter),
		),
	)

	app := dix.New("collection-contributions", dix.Modules(serverModule))
	rt, err := app.Start(context.Background())
	if err != nil {
		panic(err)
	}
	defer stopOrPanic(rt)

	routerValue, err := dix.ResolveAs[*router](rt.Container())
	if err != nil {
		panic(err)
	}
	printValues("router routes:", routerValue.routes())
	printValues("ordered keys:", routerValue.byName.Keys())

	endpointList, err := dix.ResolveAs[collectionx.List[endpoint]](rt.Container())
	if err != nil {
		panic(err)
	}
	printValues("collectionx list length:", endpointList.Len())

	endpointMap, err := dix.ResolveAs[map[string]endpoint](rt.Container())
	if err != nil {
		panic(err)
	}
	printValues("plain map keys:", sortedKeys(endpointMap))

	primary, err := dix.ResolveAs[endpoint](rt.Container())
	if err != nil {
		panic(err)
	}
	printValues("unique endpoint alias:", primary.Route())
}

func sortedKeys(values map[string]endpoint) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
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
