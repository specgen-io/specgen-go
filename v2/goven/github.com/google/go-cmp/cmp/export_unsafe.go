package cmp

import (
	"reflect"
	"unsafe"
)

const supportExporters = true

func retrieveUnexportedField(v reflect.Value, f reflect.StructField, addr bool) reflect.Value {
	ve := reflect.NewAt(f.Type, unsafe.Pointer(uintptr(unsafe.Pointer(v.UnsafeAddr()))+f.Offset)).Elem()
	if !addr {

		if ve.Kind() == reflect.Interface && ve.IsNil() {
			return reflect.Zero(f.Type)
		}
		return reflect.ValueOf(ve.Interface()).Convert(f.Type)
	}
	return ve
}
