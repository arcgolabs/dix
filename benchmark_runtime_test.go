package dix_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/arcgolabs/dix"
	dixadvanced "github.com/arcgolabs/dix/advanced"
	"github.com/samber/do/v2"
)

func BenchmarkAppBuild(b *testing.B) {
	app := newBenchmarkApp()

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		rt, err := app.Build()
		if err != nil {
			b.Fatal(err)
		}
		benchHandlerSink, err = dix.ResolveAs[*benchHandler](rt.Container())
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAppBuildParallel(b *testing.B) {
	app := newBenchmarkApp()

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rt, err := app.Build()
			if err != nil {
				b.Fatal(err)
			}
			handler, err := dix.ResolveAs[*benchHandler](rt.Container())
			if err != nil {
				b.Fatal(err)
			}
			benchHandlerSink = handler
		}
	})
}

func BenchmarkRuntimeResolve(b *testing.B) {
	rt := buildBenchmarkRuntime(b)

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		service, err := dix.ResolveAs[*benchService](rt.Container())
		if err != nil {
			b.Fatal(err)
		}
		benchServiceSink = service
	}
}

func BenchmarkRuntimeResolveParallel(b *testing.B) {
	rt := buildBenchmarkRuntime(b)

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			service, err := dix.ResolveAs[*benchService](rt.Container())
			if err != nil {
				b.Fatal(err)
			}
			benchServiceSink = service
		}
	})
}

func BenchmarkRuntimeStartStop(b *testing.B) {
	app := newBenchmarkApp()
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		rt, err := app.Build()
		if err != nil {
			b.Fatal(err)
		}
		if err := rt.Start(ctx); err != nil {
			b.Fatal(err)
		}
		if err := rt.Stop(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAdvancedResolveNamed(b *testing.B) {
	rt := buildAdvancedBenchmarkRuntime(b)

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		value, err := dixadvanced.ResolveNamedAs[string](rt.Container(), "locale.default")
		if err != nil {
			b.Fatal(err)
		}
		benchNamedValueSink = value
	}
}

func BenchmarkAdvancedResolveAssignable(b *testing.B) {
	rt := buildAdvancedBenchmarkRuntime(b)

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		greeter, err := dixadvanced.ResolveAssignableAs[benchGreeter](rt.Container())
		if err != nil {
			b.Fatal(err)
		}
		benchGreeterSink = greeter
	}
}

func BenchmarkAdvancedResolveTypedAlias(b *testing.B) {
	rt := buildAdvancedBenchmarkRuntime(b)

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		greeter, err := dix.ResolveAs[benchGreeter](rt.Container())
		if err != nil {
			b.Fatal(err)
		}
		benchGreeterSink = greeter
	}
}

func BenchmarkAdvancedResolveTransient(b *testing.B) {
	rt := buildAdvancedBenchmarkRuntime(b)

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		token, err := dix.ResolveAs[*benchTransientToken](rt.Container())
		if err != nil {
			b.Fatal(err)
		}
		benchTransientTokenSink = token
	}
}

func BenchmarkAdvancedInspectRuntime(b *testing.B) {
	rt := buildAdvancedBenchmarkRuntime(b)

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		benchInspectionSink = dixadvanced.InspectRuntime(rt, "locale.default", "greeter.en.alias")
	}
}

func BenchmarkAdvancedExplainScopeTree(b *testing.B) {
	rt := buildAdvancedBenchmarkRuntime(b)

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		benchInspectionSink.ScopeTree = dixadvanced.ExplainScopeTree(rt)
	}
}

func BenchmarkAdvancedListProvidedServices(b *testing.B) {
	rt := buildAdvancedBenchmarkRuntime(b)

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		benchInspectionSink.ProvidedServices = dixadvanced.ListProvidedServices(rt)
	}
}

func BenchmarkAdvancedExplainNamedDependencies(b *testing.B) {
	rt := buildAdvancedBenchmarkRuntime(b)

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		benchInspectionSink.NamedDependencies = dixadvanced.ExplainNamedDependencies(rt, "locale.default", "greeter.en.alias")
	}
}

func BenchmarkAdvancedInspectRuntimeLight(b *testing.B) {
	rt := buildAdvancedBenchmarkRuntime(b)
	opts := dixadvanced.InspectOptions{
		IncludeNamedDeps: true,
	}

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		benchInspectionSink = dixadvanced.InspectRuntimeWithOptions(rt, opts, "locale.default", "greeter.en.alias")
	}
}

func BenchmarkServiceNameOf(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = dix.TypedService[*benchService]().Name
	}
}

func BenchmarkResolveAsDirectDo(b *testing.B) {
	rt := buildBenchmarkRuntime(b)
	raw := rt.Container().Raw()
	name := dix.TypedService[*benchService]().Name

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		service, err := do.InvokeNamed[*benchService](raw, name)
		if err != nil {
			b.Fatal(err)
		}
		benchServiceSink = service
	}
}

func BenchmarkAdvancedScopeResolve(b *testing.B) {
	rt := buildAdvancedBenchmarkRuntime(b)
	names := make([]string, b.N)
	for i := range b.N {
		names[i] = "request-bench-" + strconv.Itoa(i)
	}
	scopePackage := func(injector do.Injector) {
		dixadvanced.ProvideScopedValue(injector, benchRequestContext{RequestID: "req-42"})
		dixadvanced.ProvideScoped2(injector, func(service *benchService, req benchRequestContext) benchScopedHandler {
			return benchScopedHandler{service: service, req: req}
		})
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := range b.N {
		scope := dixadvanced.Scope(rt, names[i], scopePackage)
		handler, err := dixadvanced.ResolveScopedAs[benchScopedHandler](scope)
		if err != nil {
			b.Fatal(err)
		}
		benchScopedHandlerSink = handler
	}
}
