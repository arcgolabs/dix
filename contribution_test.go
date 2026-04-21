package dix_test

import (
	"testing"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/arcgolabs/arcgo/dix"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type contributionEndpoint interface {
	Name() string
}

type alphaContributionEndpoint struct{}

func (alphaContributionEndpoint) Name() string { return "alpha" }

type betaContributionEndpoint struct{}

func (betaContributionEndpoint) Name() string { return "beta" }

type contributionServer struct {
	names []string
}

func TestProviderIntoInjectsCollectionxListByRole(t *testing.T) {
	app := dix.New("contribution-list-by-role",
		dix.WithModules(
			dix.NewModule("endpoints",
				dix.Providers(
					dix.Provider0(func() *alphaContributionEndpoint {
						return &alphaContributionEndpoint{}
					}, dix.Into[contributionEndpoint](dix.Order(20))),
					dix.Provider0(func() *betaContributionEndpoint {
						return &betaContributionEndpoint{}
					}, dix.Into[contributionEndpoint](dix.Order(10))),
					dix.Provider1(func(endpoints collectionx.List[contributionEndpoint]) *contributionServer {
						names := collectionx.MapList(endpoints, func(_ int, endpoint contributionEndpoint) string {
							return endpoint.Name()
						})
						return &contributionServer{names: names.Values()}
					}),
				),
			),
		),
	)

	rt, err := app.Build()
	require.NoError(t, err)

	server, err := dix.ResolveAs[*contributionServer](rt.Container())
	require.NoError(t, err)
	assert.Equal(t, []string{"beta", "alpha"}, server.names)
}

func TestContributeInjectsCollectionxList(t *testing.T) {
	app := dix.New("contribution-list",
		dix.WithModules(
			dix.NewModule("endpoints",
				dix.Providers(
					dix.Contribute0[contributionEndpoint](func() contributionEndpoint {
						return &alphaContributionEndpoint{}
					}),
					dix.Contribute0[contributionEndpoint](func() contributionEndpoint {
						return &betaContributionEndpoint{}
					}),
					dix.Provider1(func(endpoints collectionx.List[contributionEndpoint]) *contributionServer {
						names := collectionx.MapList(endpoints, func(_ int, endpoint contributionEndpoint) string {
							return endpoint.Name()
						})
						return &contributionServer{names: names.Values()}
					}),
				),
			),
		),
	)

	rt, err := app.Build()
	require.NoError(t, err)

	server, err := dix.ResolveAs[*contributionServer](rt.Container())
	require.NoError(t, err)
	assert.Equal(t, []string{"alpha", "beta"}, server.names)
}

func TestContributeInjectsOrderedMapByKey(t *testing.T) {
	app := dix.New("contribution-map",
		dix.WithModules(
			dix.NewModule("endpoints",
				dix.Providers(
					dix.Contribute0[contributionEndpoint](func() contributionEndpoint {
						return &alphaContributionEndpoint{}
					}, dix.Key("alpha")),
					dix.Contribute0[contributionEndpoint](func() contributionEndpoint {
						return &betaContributionEndpoint{}
					}, dix.Key("beta")),
					dix.Provider1(func(endpoints collectionx.OrderedMap[string, contributionEndpoint]) *contributionServer {
						names := collectionx.NewListWithCapacity[string](endpoints.Len())
						endpoints.Range(func(key string, endpoint contributionEndpoint) bool {
							names.Add(key + ":" + endpoint.Name())
							return true
						})
						return &contributionServer{names: names.Values()}
					}),
				),
			),
		),
	)

	rt, err := app.Build()
	require.NoError(t, err)

	server, err := dix.ResolveAs[*contributionServer](rt.Container())
	require.NoError(t, err)
	assert.Equal(t, []string{"alpha:alpha", "beta:beta"}, server.names)
}

func TestProviderAsInjectsAlias(t *testing.T) {
	app := dix.New("provider-as",
		dix.WithModules(
			dix.NewModule("endpoints",
				dix.Providers(
					dix.Provider0(func() *alphaContributionEndpoint {
						return &alphaContributionEndpoint{}
					}, dix.As[contributionEndpoint]()),
					dix.Provider1(func(endpoint contributionEndpoint) string {
						return endpoint.Name()
					}),
				),
			),
		),
	)

	rt, err := app.Build()
	require.NoError(t, err)

	name, err := dix.ResolveAs[string](rt.Container())
	require.NoError(t, err)
	assert.Equal(t, "alpha", name)
}
