package dix_test

import (
	"context"
	"testing"

	"github.com/arcgolabs/dix"
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
