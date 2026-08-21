package dix_test

import (
	"testing"

	collectionlist "github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/dix"
	"github.com/stretchr/testify/require"
)

func TestGenericContainerMethodsResolveAndProvide(t *testing.T) {
	module := dix.NewModuleBuilder("generic-api").
		Provide(func() string { return "provided" }).
		Build()

	runtime, err := dix.New("generic-api", dix.Modules(module)).Build()
	require.NoError(t, err)

	container := runtime.Container()
	value, err := container.Resolve[string]()
	require.NoError(t, err)
	require.Equal(t, "provided", value)

	key := dix.NamedService[int]("answer")
	key.Provide(container, func() int { return 42 })

	answer, err := key.Resolve(container)
	require.NoError(t, err)
	require.Equal(t, 42, answer)
}

func TestGenericModuleBuilderRegistersTypedContributions(t *testing.T) {
	module := dix.NewModuleBuilder("generic-contributions").
		Contribute(func() string { return "first" }, dix.Key("first")).
		Contribute(func() string { return "second" }, dix.Key("second")).
		Build()

	runtime, err := dix.New("generic-contributions", dix.Modules(module)).Build()
	require.NoError(t, err)

	values, err := runtime.Container().Resolve[*collectionlist.List[string]]()
	require.NoError(t, err)
	require.Equal(t, []string{"first", "second"}, values.Values())
}
