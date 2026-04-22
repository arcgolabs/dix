// Package repo provides persistence services for the backend example.
package repo

import (
	"github.com/arcgolabs/dix"
	"github.com/arcgolabs/dix/examples/backend/db"
)

// Module wires the backend example repositories.
var Module = dix.NewModule("repo",
	dix.WithModuleImports(db.Module),
	dix.WithModuleProviders(
		dix.Provider2(NewUserRepository),
	),
)
