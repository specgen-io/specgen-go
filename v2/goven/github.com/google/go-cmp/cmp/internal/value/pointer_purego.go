package value

import "reflect"

type Pointer struct {
	p	uintptr
	t	reflect.Type
}

func PointerOf(v reflect.Value) Pointer {

	return Pointer{v.Pointer(), v.Type()}
}

func (p Pointer) IsNil() bool {
	return p.p == 0
}

func (p Pointer) Uintptr() uintptr {
	return p.p
}
