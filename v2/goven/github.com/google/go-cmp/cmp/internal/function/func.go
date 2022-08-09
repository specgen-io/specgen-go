package function

import (
	"reflect"
	"regexp"
	"runtime"
	"strings"
)

type funcType int

const (
	_	funcType	= iota

	tbFunc
	ttbFunc
	trbFunc
	tibFunc
	trFunc

	Equal			= ttbFunc
	EqualAssignable		= tibFunc
	Transformer		= trFunc
	ValueFilter		= ttbFunc
	Less			= ttbFunc
	ValuePredicate		= tbFunc
	KeyValuePredicate	= trbFunc
)

var boolType = reflect.TypeOf(true)

func IsType(t reflect.Type, ft funcType) bool {
	if t == nil || t.Kind() != reflect.Func || t.IsVariadic() {
		return false
	}
	ni, no := t.NumIn(), t.NumOut()
	switch ft {
	case tbFunc:
		if ni == 1 && no == 1 && t.Out(0) == boolType {
			return true
		}
	case ttbFunc:
		if ni == 2 && no == 1 && t.In(0) == t.In(1) && t.Out(0) == boolType {
			return true
		}
	case trbFunc:
		if ni == 2 && no == 1 && t.Out(0) == boolType {
			return true
		}
	case tibFunc:
		if ni == 2 && no == 1 && t.In(0).AssignableTo(t.In(1)) && t.Out(0) == boolType {
			return true
		}
	case trFunc:
		if ni == 1 && no == 1 {
			return true
		}
	}
	return false
}

var lastIdentRx = regexp.MustCompile(`[_\p{L}][_\p{L}\p{N}]*$`)

func NameOf(v reflect.Value) string {
	fnc := runtime.FuncForPC(v.Pointer())
	if fnc == nil {
		return "<unknown>"
	}
	fullName := fnc.Name()

	fullName = strings.TrimSuffix(fullName, "-fm")

	var name string
	for len(fullName) > 0 {
		inParen := strings.HasSuffix(fullName, ")")
		fullName = strings.TrimSuffix(fullName, ")")

		s := lastIdentRx.FindString(fullName)
		if s == "" {
			break
		}
		name = s + "." + name
		fullName = strings.TrimSuffix(fullName, s)

		if i := strings.LastIndexByte(fullName, '('); inParen && i >= 0 {
			fullName = fullName[:i]
		}
		fullName = strings.TrimSuffix(fullName, ".")
	}
	return strings.TrimSuffix(name, ".")
}
