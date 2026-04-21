package dix

// SetupFunc describes a typed setup registration.
type SetupFunc struct {
	run  func(*Container, Lifecycle) error
	meta SetupMetadata
}

func (s SetupFunc) apply(c *Container, lc Lifecycle) error {
	if s.run == nil {
		return nil
	}
	return s.run(c, lc)
}

// Setup registers a typed setup callback.
func Setup(fn func(*Container, Lifecycle) error) SetupFunc {
	return SetupWithMetadata(fn, SetupMetadata{
		Label: "Setup",
	})
}

// Setup0 registers a typed setup callback with no container, lifecycle, or DI dependencies.
func Setup0(fn func() error) SetupFunc {
	return NewSetupFunc(func(*Container, Lifecycle) error {
		return fn()
	}, SetupMetadata{
		Label: "Setup0",
	})
}

// SetupContainer registers a typed setup callback that only needs the container.
func SetupContainer(fn func(*Container) error) SetupFunc {
	return NewSetupFunc(func(c *Container, _ Lifecycle) error {
		return fn(c)
	}, SetupMetadata{
		Label: "SetupContainer",
	})
}

// SetupLifecycle registers a typed setup callback that only needs lifecycle access.
func SetupLifecycle(fn func(Lifecycle) error) SetupFunc {
	return NewSetupFunc(func(_ *Container, lc Lifecycle) error {
		return fn(lc)
	}, SetupMetadata{
		Label: "SetupLifecycle",
	})
}

// Setup1 registers a typed setup callback with one resolved dependency.
func Setup1[D1 any](fn func(D1) error) SetupFunc {
	return NewSetupFunc(func(c *Container, _ Lifecycle) error {
		d1, err := resolveDependency1[D1](c.Raw())
		if err != nil {
			return err
		}
		return fn(d1)
	}, SetupMetadata{
		Label:        "Setup1",
		Dependencies: ServiceRefs(TypedService[D1]()),
	})
}

// Setup2 registers a typed setup callback with two resolved dependencies.
func Setup2[D1, D2 any](fn func(D1, D2) error) SetupFunc {
	return NewSetupFunc(func(c *Container, _ Lifecycle) error {
		d1, d2, err := resolveDependencies2[D1, D2](c.Raw())
		if err != nil {
			return err
		}
		return fn(d1, d2)
	}, SetupMetadata{
		Label:        "Setup2",
		Dependencies: ServiceRefs(TypedService[D1](), TypedService[D2]()),
	})
}

// Setup3 registers a typed setup callback with three resolved dependencies.
func Setup3[D1, D2, D3 any](fn func(D1, D2, D3) error) SetupFunc {
	return NewSetupFunc(func(c *Container, _ Lifecycle) error {
		d1, d2, d3, err := resolveDependencies3[D1, D2, D3](c.Raw())
		if err != nil {
			return err
		}
		return fn(d1, d2, d3)
	}, SetupMetadata{
		Label:        "Setup3",
		Dependencies: ServiceRefs(TypedService[D1](), TypedService[D2](), TypedService[D3]()),
	})
}

// Setup4 registers a typed setup callback with four resolved dependencies.
func Setup4[D1, D2, D3, D4 any](fn func(D1, D2, D3, D4) error) SetupFunc {
	return NewSetupFunc(func(c *Container, _ Lifecycle) error {
		d1, d2, d3, d4, err := resolveDependencies4[D1, D2, D3, D4](c.Raw())
		if err != nil {
			return err
		}
		return fn(d1, d2, d3, d4)
	}, SetupMetadata{
		Label:        "Setup4",
		Dependencies: ServiceRefs(TypedService[D1](), TypedService[D2](), TypedService[D3](), TypedService[D4]()),
	})
}

// Setup5 registers a typed setup callback with five resolved dependencies.
func Setup5[D1, D2, D3, D4, D5 any](fn func(D1, D2, D3, D4, D5) error) SetupFunc {
	return NewSetupFunc(func(c *Container, _ Lifecycle) error {
		d1, d2, d3, d4, d5, err := resolveDependencies5[D1, D2, D3, D4, D5](c.Raw())
		if err != nil {
			return err
		}
		return fn(d1, d2, d3, d4, d5)
	}, SetupMetadata{
		Label: "Setup5",
		Dependencies: ServiceRefs(
			TypedService[D1](),
			TypedService[D2](),
			TypedService[D3](),
			TypedService[D4](),
			TypedService[D5](),
		),
	})
}

// Setup6 registers a typed setup callback with six resolved dependencies.
func Setup6[D1, D2, D3, D4, D5, D6 any](fn func(D1, D2, D3, D4, D5, D6) error) SetupFunc {
	return NewSetupFunc(func(c *Container, _ Lifecycle) error {
		d1, d2, d3, d4, d5, d6, err := resolveDependencies6[D1, D2, D3, D4, D5, D6](c.Raw())
		if err != nil {
			return err
		}
		return fn(d1, d2, d3, d4, d5, d6)
	}, SetupMetadata{
		Label: "Setup6",
		Dependencies: ServiceRefs(
			TypedService[D1](),
			TypedService[D2](),
			TypedService[D3](),
			TypedService[D4](),
			TypedService[D5](),
			TypedService[D6](),
		),
	})
}

// SetupWithMetadata registers a typed setup callback with metadata.
func SetupWithMetadata(fn func(*Container, Lifecycle) error, meta SetupMetadata) SetupFunc {
	return NewSetupFunc(fn, SetupMetadata{
		Label:         meta.Label,
		Dependencies:  meta.Dependencies,
		Provides:      meta.Provides,
		Overrides:     meta.Overrides,
		GraphMutation: meta.GraphMutation,
		Raw:           meta.Raw,
	})
}

// RawSetup registers an untyped setup callback.
func RawSetup(fn func(*Container, Lifecycle) error) SetupFunc {
	return RawSetupWithMetadata(fn, SetupMetadata{
		Label: "RawSetup",
	})
}

// RawSetupWithMetadata registers an untyped setup callback with metadata.
func RawSetupWithMetadata(fn func(*Container, Lifecycle) error, meta SetupMetadata) SetupFunc {
	meta.Raw = true
	return NewSetupFunc(fn, meta)
}
