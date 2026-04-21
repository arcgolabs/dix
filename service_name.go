package dix

import (
	"reflect"
	"sync"

	typetostring "github.com/samber/go-type-to-string"
)

var serviceNameCache sync.Map

func serviceNameOf[T any]() string {
	typ := reflect.TypeFor[T]()
	if name, ok := serviceNameCache.Load(typ); ok {
		if value, typeOK := name.(string); typeOK {
			return value
		}
	}

	name := typetostring.GetReflectType(typ)
	actual, _ := serviceNameCache.LoadOrStore(typ, name)
	if value, ok := actual.(string); ok {
		return value
	}
	return name
}
