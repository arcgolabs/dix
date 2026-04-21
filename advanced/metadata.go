package advanced

import (
	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/arcgolabs/arcgo/dix"
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
	dependencies collectionx.List[dix.ServiceRef],
	provides collectionx.List[dix.ServiceRef],
	overrides collectionx.List[dix.ServiceRef],
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
