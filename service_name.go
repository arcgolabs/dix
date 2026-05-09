package dix

import (
	"reflect"

	collectionmapping "github.com/arcgolabs/collectionx/mapping"
	typetostring "github.com/samber/go-type-to-string"
)

type serviceNamer struct {
	cache collectionmapping.ConcurrentMap[reflect.Type, string]
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
	if name, found := n.cache.Get(typ); found {
		return name
	}
	name := typetostring.GetReflectType(typ)
	actual, _ := n.cache.GetOrStore(typ, name)
	return actual
}

func serviceTypeName(typ reflect.Type) string {
	if typ == nil {
		return ""
	}
	return typetostring.GetReflectType(typ)
}
