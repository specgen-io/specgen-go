package value

import (
	"reflect"
	"unsafe"
)

type Pointer struct {
	p	unsafe.Pointer
	t	reflect.Type
}

func PointerOf(v reflect.Value) Pointer {

	return Pointer{unsafe.Pointer(v.Pointer()), v.Type()}
}

func (p Pointer) IsNil() bool {
	return p.p == nil
}

func (p Pointer) Uintptr() uintptr {
	return uintptr(p.p)
}
