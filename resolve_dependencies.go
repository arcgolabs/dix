package dix

import "github.com/samber/do/v2"

type resolvedDependencies5[D1, D2, D3, D4, D5 any] struct {
	First  D1
	Second D2
	Third  D3
	Fourth D4
	Fifth  D5
}

type resolvedDependencies6[D1, D2, D3, D4, D5, D6 any] struct {
	First  D1
	Second D2
	Third  D3
	Fourth D4
	Fifth  D5
	Sixth  D6
}

func resolveDependency1[D1 any](injector do.Injector) (D1, error) {
	return resolveInjectorAs[D1](injector)
}

func resolveDependencies2[D1, D2 any](injector do.Injector) (D1, D2, error) {
	d1, err := resolveInjectorAs[D1](injector)
	if err != nil {
		var zeroD1 D1
		var zeroD2 D2
		return zeroD1, zeroD2, err
	}
	d2, err := resolveInjectorAs[D2](injector)
	if err != nil {
		var zeroD2 D2
		return d1, zeroD2, err
	}
	return d1, d2, nil
}

func resolveDependencies3[D1, D2, D3 any](injector do.Injector) (D1, D2, D3, error) {
	d1, d2, err := resolveDependencies2[D1, D2](injector)
	if err != nil {
		var zeroD3 D3
		return d1, d2, zeroD3, err
	}
	d3, err := resolveInjectorAs[D3](injector)
	if err != nil {
		var zeroD3 D3
		return d1, d2, zeroD3, err
	}
	return d1, d2, d3, nil
}

func resolveDependencies4[D1, D2, D3, D4 any](injector do.Injector) (D1, D2, D3, D4, error) {
	d1, d2, d3, err := resolveDependencies3[D1, D2, D3](injector)
	if err != nil {
		var zeroD4 D4
		return d1, d2, d3, zeroD4, err
	}
	d4, err := resolveInjectorAs[D4](injector)
	if err != nil {
		var zeroD4 D4
		return d1, d2, d3, zeroD4, err
	}
	return d1, d2, d3, d4, nil
}

func resolveDependencies5[D1, D2, D3, D4, D5 any](injector do.Injector) (resolvedDependencies5[D1, D2, D3, D4, D5], error) {
	d1, d2, d3, d4, err := resolveDependencies4[D1, D2, D3, D4](injector)
	if err != nil {
		return resolvedDependencies5[D1, D2, D3, D4, D5]{}, err
	}
	d5, err := resolveInjectorAs[D5](injector)
	if err != nil {
		return resolvedDependencies5[D1, D2, D3, D4, D5]{}, err
	}
	return resolvedDependencies5[D1, D2, D3, D4, D5]{
		First:  d1,
		Second: d2,
		Third:  d3,
		Fourth: d4,
		Fifth:  d5,
	}, nil
}

func resolveDependencies6[D1, D2, D3, D4, D5, D6 any](injector do.Injector) (resolvedDependencies6[D1, D2, D3, D4, D5, D6], error) {
	deps, err := resolveDependencies5[D1, D2, D3, D4, D5](injector)
	if err != nil {
		return resolvedDependencies6[D1, D2, D3, D4, D5, D6]{}, err
	}
	d6, err := resolveInjectorAs[D6](injector)
	if err != nil {
		return resolvedDependencies6[D1, D2, D3, D4, D5, D6]{}, err
	}
	return resolvedDependencies6[D1, D2, D3, D4, D5, D6]{
		First:  deps.First,
		Second: deps.Second,
		Third:  deps.Third,
		Fourth: deps.Fourth,
		Fifth:  deps.Fifth,
		Sixth:  d6,
	}, nil
}
