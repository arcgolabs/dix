package dix

import "github.com/samber/do/v2"

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

//nolint:gocritic // Typed DI helpers intentionally return each dependency plus an error for generated hook signatures.
func resolveDependencies5[D1, D2, D3, D4, D5 any](injector do.Injector) (D1, D2, D3, D4, D5, error) {
	d1, d2, d3, d4, err := resolveDependencies4[D1, D2, D3, D4](injector)
	if err != nil {
		var zeroD5 D5
		return d1, d2, d3, d4, zeroD5, err
	}
	d5, err := resolveInjectorAs[D5](injector)
	if err != nil {
		var zeroD5 D5
		return d1, d2, d3, d4, zeroD5, err
	}
	return d1, d2, d3, d4, d5, nil
}

//nolint:gocritic // Typed DI helpers intentionally return each dependency plus an error for generated hook signatures.
func resolveDependencies6[D1, D2, D3, D4, D5, D6 any](injector do.Injector) (D1, D2, D3, D4, D5, D6, error) {
	d1, d2, d3, d4, d5, err := resolveDependencies5[D1, D2, D3, D4, D5](injector)
	if err != nil {
		var zeroD6 D6
		return d1, d2, d3, d4, d5, zeroD6, err
	}
	d6, err := resolveInjectorAs[D6](injector)
	if err != nil {
		var zeroD6 D6
		return d1, d2, d3, d4, d5, zeroD6, err
	}
	return d1, d2, d3, d4, d5, d6, nil
}
