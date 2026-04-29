package advanced

import (
	collectionlist "github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/dix"
)

func newProvider(
	label string,
	output dix.ServiceRef,
	register func(*dix.Container),
	deps ...dix.ServiceRef,
) dix.ProviderFunc {
	return dix.NewProviderFunc(register, dix.ProviderMetadata{
		Label:        label,
		Output:       output,
		Dependencies: dix.ServiceRefs(deps...),
	})
}

func newSetup(
	label string,
	run func(*dix.Container) error,
	dependencies *collectionlist.List[dix.ServiceRef],
	provides *collectionlist.List[dix.ServiceRef],
	overrides *collectionlist.List[dix.ServiceRef],
) dix.SetupFunc {
	return dix.NewSetupFunc(func(c *dix.Container, _ dix.Lifecycle) error {
		return run(c)
	}, dix.SetupMetadata{
		Label:         label,
		Dependencies:  dependencies,
		Provides:      provides,
		Overrides:     overrides,
		GraphMutation: false,
		Raw:           false,
	})
}
