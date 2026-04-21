package dix

// InvokeFunc describes a typed invoke registration.
type InvokeFunc struct {
	run  func(*Container) error
	meta InvokeMetadata
}

func (i InvokeFunc) apply(c *Container) error {
	if i.run == nil {
		return nil
	}
	return i.run(c)
}

// RawInvoke registers an untyped invoke callback.
func RawInvoke(fn func(*Container) error) InvokeFunc {
	return RawInvokeWithMetadata(fn, InvokeMetadata{
		Label: "RawInvoke",
	})
}

// RawInvokeWithMetadata registers an untyped invoke callback with metadata.
func RawInvokeWithMetadata(fn func(*Container) error, meta InvokeMetadata) InvokeFunc {
	return NewInvokeFunc(fn, InvokeMetadata{
		Label:        meta.Label,
		Dependencies: meta.Dependencies,
		Raw:          true,
	})
}

// Invoke registers an invoke callback with no dependencies.
func Invoke(fn func()) InvokeFunc {
	return Invoke0(fn)
}

// Invoke0 registers an invoke callback with no dependencies.
func Invoke0(fn func()) InvokeFunc {
	return NewInvokeFunc(func(*Container) error {
		fn()
		return nil
	}, InvokeMetadata{Label: "Invoke0"})
}

// Invoke1 registers an invoke callback with one dependency.
func Invoke1[T any](fn func(T)) InvokeFunc {
	return NewInvokeFunc(
		func(c *Container) error { return dixInvoke1(c, fn) },
		InvokeMetadata{
			Label:        "Invoke1",
			Dependencies: ServiceRefs(TypedService[T]()),
		},
	)
}

// Invoke2 registers an invoke callback with two dependencies.
func Invoke2[T1, T2 any](fn func(T1, T2)) InvokeFunc {
	return NewInvokeFunc(
		func(c *Container) error { return dixInvoke2(c, fn) },
		InvokeMetadata{
			Label:        "Invoke2",
			Dependencies: ServiceRefs(TypedService[T1](), TypedService[T2]()),
		},
	)
}

// Invoke3 registers an invoke callback with three dependencies.
func Invoke3[T1, T2, T3 any](fn func(T1, T2, T3)) InvokeFunc {
	return NewInvokeFunc(
		func(c *Container) error { return dixInvoke3(c, fn) },
		InvokeMetadata{
			Label:        "Invoke3",
			Dependencies: ServiceRefs(TypedService[T1](), TypedService[T2](), TypedService[T3]()),
		},
	)
}

// Invoke4 registers an invoke callback with four dependencies.
func Invoke4[T1, T2, T3, T4 any](fn func(T1, T2, T3, T4)) InvokeFunc {
	return NewInvokeFunc(
		func(c *Container) error { return dixInvoke4(c, fn) },
		InvokeMetadata{
			Label:        "Invoke4",
			Dependencies: ServiceRefs(TypedService[T1](), TypedService[T2](), TypedService[T3](), TypedService[T4]()),
		},
	)
}

// Invoke5 registers an invoke callback with five dependencies.
func Invoke5[T1, T2, T3, T4, T5 any](fn func(T1, T2, T3, T4, T5)) InvokeFunc {
	return NewInvokeFunc(
		func(c *Container) error { return dixInvoke5(c, fn) },
		InvokeMetadata{
			Label: "Invoke5",
			Dependencies: ServiceRefs(
				TypedService[T1](),
				TypedService[T2](),
				TypedService[T3](),
				TypedService[T4](),
				TypedService[T5](),
			),
		},
	)
}

// Invoke6 registers an invoke callback with six dependencies.
func Invoke6[T1, T2, T3, T4, T5, T6 any](fn func(T1, T2, T3, T4, T5, T6)) InvokeFunc {
	return NewInvokeFunc(
		func(c *Container) error { return dixInvoke6(c, fn) },
		InvokeMetadata{
			Label: "Invoke6",
			Dependencies: ServiceRefs(
				TypedService[T1](),
				TypedService[T2](),
				TypedService[T3](),
				TypedService[T4](),
				TypedService[T5](),
				TypedService[T6](),
			),
		},
	)
}

func dixInvoke1[T any](c *Container, fn func(T)) error {
	t, err := resolveDependency1[T](c.Raw())
	if err != nil {
		return err
	}
	fn(t)
	return nil
}

func dixInvoke2[T1, T2 any](c *Container, fn func(T1, T2)) error {
	t1, t2, err := resolveDependencies2[T1, T2](c.Raw())
	if err != nil {
		return err
	}
	fn(t1, t2)
	return nil
}

func dixInvoke3[T1, T2, T3 any](c *Container, fn func(T1, T2, T3)) error {
	t1, t2, t3, err := resolveDependencies3[T1, T2, T3](c.Raw())
	if err != nil {
		return err
	}
	fn(t1, t2, t3)
	return nil
}

func dixInvoke4[T1, T2, T3, T4 any](c *Container, fn func(T1, T2, T3, T4)) error {
	t1, t2, t3, t4, err := resolveDependencies4[T1, T2, T3, T4](c.Raw())
	if err != nil {
		return err
	}
	fn(t1, t2, t3, t4)
	return nil
}

func dixInvoke5[T1, T2, T3, T4, T5 any](c *Container, fn func(T1, T2, T3, T4, T5)) error {
	t1, t2, t3, t4, t5, err := resolveDependencies5[T1, T2, T3, T4, T5](c.Raw())
	if err != nil {
		return err
	}
	fn(t1, t2, t3, t4, t5)
	return nil
}

func dixInvoke6[T1, T2, T3, T4, T5, T6 any](c *Container, fn func(T1, T2, T3, T4, T5, T6)) error {
	t1, t2, t3, t4, t5, t6, err := resolveDependencies6[T1, T2, T3, T4, T5, T6](c.Raw())
	if err != nil {
		return err
	}
	fn(t1, t2, t3, t4, t5, t6)
	return nil
}
