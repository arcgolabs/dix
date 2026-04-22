// Package service provides business services for the backend example.
package service

import (
	"github.com/arcgolabs/dix"
	"github.com/arcgolabs/dix/examples/backend/event"
	"github.com/arcgolabs/dix/examples/backend/repo"
)

// Module wires the backend example services.
var Module = dix.NewModule("service",
	dix.WithModuleImports(repo.Module, event.Module),
	dix.WithModuleProviders(
		dix.Provider3(NewUserService),
	),
)
