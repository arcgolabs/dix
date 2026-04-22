package dix

import (
	"os"
	"strings"

	"github.com/arcgolabs/collectionx"
)

// ProfileManager provides utilities for working with application profiles.
type ProfileManager struct{}

// ProfileFromEnv retrieves the current profile from an environment variable.
// If the environment variable is not set or contains an invalid value,
// the default profile is returned.
//
// Example:
//
//	profile := ProfileFromEnv("APP_PROFILE", ProfileProd)
func ProfileFromEnv(envVar string, defaultProfile Profile) Profile {
	value := strings.TrimSpace(os.Getenv(envVar))
	if value == "" {
		return defaultProfile
	}
	return Profile(value)
}

// IsProfile checks if the current profile matches the given profile.
func (pm ProfileManager) IsProfile(current, target Profile) bool {
	return current == target
}

// IsDev checks if the current profile is the development profile.
func (pm ProfileManager) IsDev(profile Profile) bool {
	return profile == ProfileDev
}

// IsTest checks if the current profile is the test profile.
func (pm ProfileManager) IsTest(profile Profile) bool {
	return profile == ProfileTest
}

// IsProd checks if the current profile is the production profile.
func (pm ProfileManager) IsProd(profile Profile) bool {
	return profile == ProfileProd
}

// Profile is a helper for creating profile-aware modules.
//
// Example:
//
//	var DevOnlyModule = dix.NewModule("dev-tools",
//	    dix.WithModuleProviders(ProvideDebugHandler),
//	    dix.WithModuleProfiles(dix.ProfileDev),
//	)
//
//	var ProdOnlyModule = dix.NewModule("monitoring",
//	    dix.WithModuleProviders(ProvideMetrics),
//	    dix.WithModuleExcludeProfiles(dix.ProfileDev, dix.ProfileTest),
//	)

// ProfileFilter provides methods for filtering modules by profile.
type ProfileFilter struct {
	profile Profile
}

// NewProfileFilter creates a new profile filter for the given profile.
func NewProfileFilter(profile Profile) *ProfileFilter {
	return &ProfileFilter{profile: profile}
}

// IsActive checks if a module should be active for the current profile.
func (pf *ProfileFilter) IsActive(mod Module) bool {
	return isActiveForProfile(mod.spec, pf.profile)
}

// FilterModules returns only the modules that are active for the current profile.
func (pf *ProfileFilter) FilterModules(modules collectionx.List[Module]) collectionx.List[Module] {
	filtered, err := pf.FilterModulesE(modules)
	if err != nil {
		return collectionx.NewList[Module]()
	}
	return filtered
}

// FilterModulesE returns only the modules that are active for the current profile.
func (pf *ProfileFilter) FilterModulesE(modules collectionx.List[Module]) (collectionx.List[Module], error) {
	filtered, err := flattenModules(modules, pf.profile)
	if err != nil {
		return collectionx.NewList[Module](), err
	}
	return collectionx.MapList(filtered, func(_ int, spec *moduleSpec) Module {
		return Module{spec: spec}
	}), nil
}

// Profiles is the shared profile helper instance.
var Profiles = ProfileManager{}
