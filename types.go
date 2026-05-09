package dix

import (
	collectionlist "github.com/arcgolabs/collectionx/list"
	collectionset "github.com/arcgolabs/collectionx/set"
	"log/slog"
	"time"
)

// Profile represents an application profile (environment).
type Profile string

const (
	// ProfileDefault is the default application profile.
	ProfileDefault Profile = "default"
	// ProfileDev is the development application profile.
	ProfileDev Profile = "dev"
	// ProfileTest is the test application profile.
	ProfileTest Profile = "test"
	// ProfileProd is the production application profile.
	ProfileProd Profile = "prod"
)

// AppMeta contains application metadata.
type AppMeta struct {
	Name        string
	Version     string
	Description string
}

// AppState represents the current runtime state.
type AppState int32

const (
	// AppStateCreated indicates the runtime has been created.
	AppStateCreated AppState = iota
	// AppStateBuilt indicates the runtime has been built.
	AppStateBuilt
	// AppStateStarting indicates startup is in progress.
	AppStateStarting
	// AppStateStarted indicates startup has completed.
	AppStateStarted
	// AppStateStopped indicates shutdown has completed.
	AppStateStopped
)

// String returns the string form of the app state.
func (s AppState) String() string {
	switch s {
	case AppStateCreated:
		return "created"
	case AppStateBuilt:
		return "built"
	case AppStateStarting:
		return "starting"
	case AppStateStarted:
		return "started"
	case AppStateStopped:
		return "stopped"
	default:
		return "unknown"
	}
}

// App is an immutable application specification.
type App struct {
	spec      *appSpec
	planCache appPlanCache
}

// Runtime is a built application runtime produced from an App spec.
type Runtime struct {
	spec          *appSpec
	plan          *buildPlan
	container     *Container
	lifecycle     *lifecycleImpl
	logger        *slog.Logger
	eventLogger   EventLogger
	eventRecorder *EventRecorder
	state         AppState
	subapps       *collectionlist.List[*Runtime]
}

// Module is an immutable module specification.
type Module struct {
	spec *moduleSpec
}

type appSpec struct {
	meta                     AppMeta
	profile                  Profile
	profileConfigured        bool
	serviceNames             *serviceNamer
	modules                  *collectionlist.List[Module]
	logger                   *slog.Logger
	loggerConfigured         bool
	loggerFromContainer      func(*Container) (*slog.Logger, error)
	eventLogger              EventLogger
	eventLoggerConfigured    bool
	eventLoggerFromContainer func(*Container) (EventLogger, error)
	observers                *collectionlist.List[Observer]
	observerDispatchers      *collectionlist.List[*observerDispatcher]
	observersConfigured      bool
	subapps                  *collectionlist.List[*App]
	runStopTimeout           time.Duration
	lifecycleConcurrency     int
	eventBufferCapacity      int
	versionConfigured        bool
	descriptionConfigured    bool
	debug                    debugSettings
}

type moduleSpec struct {
	name            string
	description     string
	providers       *collectionlist.List[ProviderFunc]
	setups          *collectionlist.List[SetupFunc]
	invokes         *collectionlist.List[InvokeFunc]
	hooks           *collectionlist.List[HookFunc]
	imports         *collectionlist.List[Module]
	profiles        *collectionset.Set[Profile]
	excludeProfiles *collectionset.Set[Profile]
	disabled        bool
	tags            *collectionset.OrderedSet[string]
}

type debugSettings struct {
	scopeTree                bool
	namedServiceDependencies *collectionset.OrderedSet[string]
}

// ValidationWarningKind identifies a validation warning category.
type ValidationWarningKind string

const (
	// ValidationWarningRawProviderUndeclaredOutput warns about raw providers without declared outputs.
	ValidationWarningRawProviderUndeclaredOutput ValidationWarningKind = "raw_provider_undeclared_output"
	// ValidationWarningRawProviderUndeclaredDeps warns about raw providers without declared dependencies.
	ValidationWarningRawProviderUndeclaredDeps ValidationWarningKind = "raw_provider_undeclared_deps"
	// ValidationWarningRawInvokeUndeclaredDeps warns about raw invokes without declared dependencies.
	ValidationWarningRawInvokeUndeclaredDeps ValidationWarningKind = "raw_invoke_undeclared_deps"
	// ValidationWarningRawHookUndeclaredDeps warns about raw hooks without declared dependencies.
	ValidationWarningRawHookUndeclaredDeps ValidationWarningKind = "raw_hook_undeclared_deps"
	// ValidationWarningRawSetupUndeclaredGraph warns about raw setup graph mutations without declarations.
	ValidationWarningRawSetupUndeclaredGraph ValidationWarningKind = "raw_setup_undeclared_graph"
)

// ValidationWarning describes a non-fatal graph validation warning.
type ValidationWarning struct {
	Kind    ValidationWarningKind
	Module  string
	Label   string
	Details string
}

// ValidationReport summarizes graph validation errors and warnings.
type ValidationReport struct {
	Errors        *collectionlist.List[error]
	Warnings      *collectionlist.List[ValidationWarning]
	WarningCounts *collectionset.MultiSet[ValidationWarningKind]
	ServiceCounts *collectionset.MultiSet[string]
}
