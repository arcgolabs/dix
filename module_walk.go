package dix

import (
	"fmt"
	"strings"

	"github.com/arcgolabs/collectionx"
	collectionset "github.com/arcgolabs/collectionx/set"
	"github.com/samber/oops"
)

type moduleVisitAction uint8

const (
	moduleVisitContinue moduleVisitAction = iota
	moduleVisitSkipChildren
	moduleVisitStop
)

type moduleVisitContext struct {
	Profile Profile
	Path    collectionx.List[string]
	Depth   int
}

type moduleVisitor interface {
	Enter(ctx moduleVisitContext, spec *moduleSpec) (moduleVisitAction, error)
	Leave(ctx moduleVisitContext, spec *moduleSpec) error
}

type moduleVisitorFuncs struct {
	enter func(ctx moduleVisitContext, spec *moduleSpec) (moduleVisitAction, error)
	leave func(ctx moduleVisitContext, spec *moduleSpec) error
}

type moduleWalkState struct {
	visited    *collectionset.Set[*moduleSpec]
	visiting   *collectionset.Set[*moduleSpec]
	knownNames collectionx.Map[string, *moduleSpec]
	stopped    bool
	path       collectionx.List[string]
	profile    Profile
	active     func(*moduleSpec, Profile) bool
	visitor    moduleVisitor
}

func (v moduleVisitorFuncs) Enter(ctx moduleVisitContext, spec *moduleSpec) (moduleVisitAction, error) {
	if v.enter == nil {
		return moduleVisitContinue, nil
	}
	return v.enter(ctx, spec)
}

func (v moduleVisitorFuncs) Leave(ctx moduleVisitContext, spec *moduleSpec) error {
	if v.leave == nil {
		return nil
	}
	return v.leave(ctx, spec)
}

// flattenModules walks active modules in dependency order and returns leaf-first results.
func flattenModules(modules collectionx.List[Module], profile Profile) (collectionx.List[*moduleSpec], error) {
	return flattenModuleList(modules, profile)
}

func flattenModuleList(modules collectionx.List[Module], profile Profile) (collectionx.List[*moduleSpec], error) {
	return flattenModuleListWithActive(modules, profile, isActiveForProfile)
}

func flattenProfileBootstrapModuleList(modules collectionx.List[Module]) (collectionx.List[*moduleSpec], error) {
	return flattenModuleListWithActive(modules, ProfileDefault, isActiveForProfileBootstrap)
}

func flattenModuleListWithActive(
	modules collectionx.List[Module],
	profile Profile,
	active func(*moduleSpec, Profile) bool,
) (collectionx.List[*moduleSpec], error) {
	capacity := 8
	if modules != nil && modules.Len() > 0 {
		if c := modules.Len() * 2; c > capacity {
			capacity = c
		}
	}
	result := collectionx.NewListWithCapacity[*moduleSpec](capacity)

	err := walkModuleListWithActive(modules, profile, active, moduleVisitorFuncs{
		leave: func(_ moduleVisitContext, spec *moduleSpec) error {
			result.Add(spec)
			return nil
		},
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func walkModules(modules collectionx.List[Module], profile Profile, visitor moduleVisitor) error {
	return walkModuleList(modules, profile, visitor)
}

func walkModuleList(modules collectionx.List[Module], profile Profile, visitor moduleVisitor) error {
	return walkModuleListWithActive(modules, profile, isActiveForProfile, visitor)
}

func walkModuleListWithActive(
	modules collectionx.List[Module],
	profile Profile,
	active func(*moduleSpec, Profile) bool,
	visitor moduleVisitor,
) error {
	if modules == nil {
		return nil
	}

	state := newModuleWalkState(profile, active, visitor)
	return state.walkAll(modules)
}

func newModuleWalkState(profile Profile, active func(*moduleSpec, Profile) bool, visitor moduleVisitor) *moduleWalkState {
	if active == nil {
		active = isActiveForProfile
	}
	return &moduleWalkState{
		visited:    collectionset.NewSetWithCapacity[*moduleSpec](16),
		visiting:   collectionset.NewSetWithCapacity[*moduleSpec](8),
		knownNames: collectionx.NewMapWithCapacity[string, *moduleSpec](8),
		path:       collectionx.NewListWithCapacity[string](8),
		profile:    profile,
		active:     active,
		visitor:    visitor,
	}
}

func (s *moduleWalkState) walkAll(modules collectionx.List[Module]) error {
	var walkErr error
	modules.Range(func(_ int, mod Module) bool {
		walkErr = s.walk(mod.spec)
		return walkErr == nil && !s.stopped
	})
	return walkErr
}

func (s *moduleWalkState) walk(spec *moduleSpec) error {
	if s.shouldSkip(spec) {
		return nil
	}

	key, alreadyVisited, err := s.beginVisit(spec)
	if err != nil || alreadyVisited {
		return err
	}

	ctx := s.currentContext()
	action, err := s.enterModule(key, ctx, spec)
	if err != nil {
		s.abortVisit(spec)
		return err
	}
	if err := s.handleVisitAction(spec, action); err != nil {
		s.abortVisit(spec)
		return err
	}

	return s.finishVisit(key, ctx, spec)
}

func (s *moduleWalkState) shouldSkip(spec *moduleSpec) bool {
	return s.stopped || spec == nil || spec.disabled || !s.active(spec, s.profile)
}

func (s *moduleWalkState) beginVisit(spec *moduleSpec) (string, bool, error) {
	key := moduleKey(spec)
	if spec.name != "" {
		if known, ok := s.knownNames.Get(spec.name); ok && known != spec {
			return "", false, oops.In("dix").
				With("op", "begin_module_visit", "module", spec.name).
				Errorf("duplicate module name detected: %s", spec.name)
		}
		s.knownNames.Set(spec.name, spec)
	}
	if s.visited.Contains(spec) {
		return key, true, nil
	}
	if s.visiting.Contains(spec) {
		return "", false, oops.In("dix").
			With("op", "begin_module_visit", "module", key, "path", formatModulePath(s.path)).
			Errorf("module import cycle detected: %s -> %s", formatModulePath(s.path), key)
	}

	s.path.Add(key)
	s.visiting.Add(spec)
	return key, false, nil
}

func (s *moduleWalkState) currentContext() moduleVisitContext {
	return moduleVisitContext{
		Profile: s.profile,
		Path:    s.path,
		Depth:   s.path.Len() - 1,
	}
}

func (s *moduleWalkState) enterModule(key string, ctx moduleVisitContext, spec *moduleSpec) (moduleVisitAction, error) {
	action, err := s.visitor.Enter(ctx, spec)
	if err != nil {
		return 0, oops.In("dix").
			With("op", "enter_module", "module", key, "depth", ctx.Depth).
			Wrapf(err, "enter module %s", key)
	}
	return action, nil
}

func (s *moduleWalkState) handleVisitAction(spec *moduleSpec, action moduleVisitAction) error {
	switch action {
	case moduleVisitContinue:
		return s.walkChildren(spec)
	case moduleVisitSkipChildren:
		return nil
	case moduleVisitStop:
		s.stopped = true
		return nil
	}
	return nil
}

func (s *moduleWalkState) walkChildren(spec *moduleSpec) error {
	var childErr error
	spec.imports.Range(func(_ int, imported Module) bool {
		childErr = s.walk(imported.spec)
		return childErr == nil && !s.stopped
	})
	return childErr
}

func (s *moduleWalkState) abortVisit(spec *moduleSpec) {
	s.visiting.Remove(spec)
	_, _ = s.path.RemoveAt(s.path.Len() - 1)
}

func (s *moduleWalkState) finishVisit(key string, ctx moduleVisitContext, spec *moduleSpec) error {
	s.visiting.Remove(spec)
	s.visited.Add(spec)
	leaveErr := s.visitor.Leave(ctx, spec)
	_, _ = s.path.RemoveAt(s.path.Len() - 1)
	if leaveErr != nil {
		return oops.In("dix").
			With("op", "leave_module", "module", key, "depth", ctx.Depth).
			Wrapf(leaveErr, "leave module %s", key)
	}
	return nil
}

func moduleKey(spec *moduleSpec) string {
	if spec == nil {
		return "<nil>"
	}
	if spec.name != "" {
		return spec.name
	}
	return fmt.Sprintf("<anonymous:%p>", spec)
}

func formatModulePath(path collectionx.List[string]) string {
	if path.IsEmpty() {
		return "<root>"
	}
	return strings.Join(path.Values(), " -> ")
}

func isActiveForProfile(spec *moduleSpec, profile Profile) bool {
	if spec == nil || spec.disabled {
		return false
	}
	if spec.excludeProfiles.Contains(profile) {
		return false
	}
	if spec.profiles.IsEmpty() {
		return true
	}
	return spec.profiles.Contains(profile)
}

func isActiveForProfileBootstrap(spec *moduleSpec, _ Profile) bool {
	if spec == nil || spec.disabled {
		return false
	}
	return spec.profiles.IsEmpty() && spec.excludeProfiles.IsEmpty()
}
