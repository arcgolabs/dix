package dix

import (
	"errors"

	"github.com/samber/oops"
)

// ScopePackage configures a child runtime scope.
type ScopePackage func(*Container) error

// ScopeFunc adapts a no-error scope callback into a ScopePackage.
func ScopeFunc(fn func(*Container)) ScopePackage {
	return func(c *Container) error {
		if fn != nil {
			fn(c)
		}
		return nil
	}
}

// Scope creates a named child scope from the runtime container.
func (r *Runtime) Scope(name string, packages ...ScopePackage) (*Container, error) {
	if r == nil || r.container == nil {
		return nil, oops.In("dix").
			With("op", "runtime_scope", "scope", name).
			New("runtime is nil")
	}
	return r.container.Scope(name, packages...)
}

// Scope creates a named child scope from this container.
func (c *Container) Scope(name string, packages ...ScopePackage) (*Container, error) {
	child, err := newChildContainer(c, name, c.logger)
	if err != nil {
		return nil, oops.In("dix").
			With("op", "container_scope", "scope", name).
			Wrapf(err, "create scope failed")
	}
	if err := applyScopePackages(child, packages...); err != nil {
		return nil, err
	}
	return child, nil
}

func applyScopePackages(scope *Container, packages ...ScopePackage) error {
	errs := collectScopePackageErrors(packages, scope)
	if len(errs) == 0 {
		return nil
	}
	return errors.Join(errs...)
}

func collectScopePackageErrors(packages []ScopePackage, scope *Container) []error {
	errs := make([]error, 0, 1)
	for _, pkg := range packages {
		if pkg == nil {
			continue
		}
		if err := pkg(scope); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}
