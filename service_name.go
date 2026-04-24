package dix

import (
	"reflect"
	"sync"

	typetostring "github.com/samber/go-type-to-string"
)

type serviceNamer struct {
	cache sync.Map
}

func newServiceNamer() *serviceNamer {
	return &serviceNamer{}
}

func serviceNameOf[T any]() string {
	return serviceTypeName(reflect.TypeFor[T]())
}

func serviceNameOfSpec[T any](spec *appSpec) string {
	if spec == nil {
		return serviceNameOf[T]()
	}
	return serviceNameOfWith[T](spec.serviceNames)
}

func serviceNameOfWith[T any](n *serviceNamer) string {
	if n == nil {
		return serviceNameOf[T]()
	}
	return n.Name(reflect.TypeFor[T]())
}

func (n *serviceNamer) Name(typ reflect.Type) string {
	if typ == nil {
		return ""
	}
	if name, ok := n.cache.Load(typ); ok {
		if value, typeOK := name.(string); typeOK {
			return value
		}
	}

	name := typetostring.GetReflectType(typ)
	actual, _ := n.cache.LoadOrStore(typ, name)
	if value, ok := actual.(string); ok {
		return value
	}
	return name
}

func serviceTypeName(typ reflect.Type) string {
	if typ == nil {
		return ""
	}
	return typetostring.GetReflectType(typ)
}
