// Package service provides business services for the backend example.
package service

import (
	"github.com/DaiYuANg/arcgo/examples/dix/backend/event"
	"github.com/DaiYuANg/arcgo/examples/dix/backend/repo"
	"github.com/arcgolabs/arcgo/dix"
)

// Module wires the backend example services.
var Module = dix.NewModule("service",
	dix.WithModuleImports(repo.Module, event.Module),
	dix.WithModuleProviders(
		dix.Provider3(NewUserService),
	),
)
