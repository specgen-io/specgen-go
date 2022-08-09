package cmp

import "reflect"

const supportExporters = false

func retrieveUnexportedField(reflect.Value, reflect.StructField, bool) reflect.Value {
	panic("no support for forcibly accessing unexported fields")
}
