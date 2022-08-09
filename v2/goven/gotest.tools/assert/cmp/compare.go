package cmp

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/specgen-io/specgen-golang/v2/goven/github.com/google/go-cmp/cmp"
	"github.com/specgen-io/specgen-golang/v2/goven/gotest.tools/internal/format"
)

type Comparison func() Result

func DeepEqual(x, y interface{}, opts ...cmp.Option) Comparison {
	return func() (result Result) {
		defer func() {
			if panicmsg, handled := handleCmpPanic(recover()); handled {
				result = ResultFailure(panicmsg)
			}
		}()
		diff := cmp.Diff(x, y, opts...)
		if diff == "" {
			return ResultSuccess
		}
		return multiLineDiffResult(diff)
	}
}

func handleCmpPanic(r interface{}) (string, bool) {
	if r == nil {
		return "", false
	}
	panicmsg, ok := r.(string)
	if !ok {
		panic(r)
	}
	switch {
	case strings.HasPrefix(panicmsg, "cannot handle unexported field"):
		return panicmsg, true
	}
	panic(r)
}

func toResult(success bool, msg string) Result {
	if success {
		return ResultSuccess
	}
	return ResultFailure(msg)
}

type RegexOrPattern interface{}

func Regexp(re RegexOrPattern, v string) Comparison {
	match := func(re *regexp.Regexp) Result {
		return toResult(
			re.MatchString(v),
			fmt.Sprintf("value %q does not match regexp %q", v, re.String()))
	}

	return func() Result {
		switch regex := re.(type) {
		case *regexp.Regexp:
			return match(regex)
		case string:
			re, err := regexp.Compile(regex)
			if err != nil {
				return ResultFailure(err.Error())
			}
			return match(re)
		default:
			return ResultFailure(fmt.Sprintf("invalid type %T for regex pattern", regex))
		}
	}
}

func Equal(x, y interface{}) Comparison {
	return func() Result {
		switch {
		case x == y:
			return ResultSuccess
		case isMultiLineStringCompare(x, y):
			diff := format.UnifiedDiff(format.DiffConfig{A: x.(string), B: y.(string)})
			return multiLineDiffResult(diff)
		}
		return ResultFailureTemplate(`
			{{- .Data.x}} (
				{{- with callArg 0 }}{{ formatNode . }} {{end -}}
				{{- printf "%T" .Data.x -}}
			) != {{ .Data.y}} (
				{{- with callArg 1 }}{{ formatNode . }} {{end -}}
				{{- printf "%T" .Data.y -}}
			)`,
			map[string]interface{}{"x": x, "y": y})
	}
}

func isMultiLineStringCompare(x, y interface{}) bool {
	strX, ok := x.(string)
	if !ok {
		return false
	}
	strY, ok := y.(string)
	if !ok {
		return false
	}
	return strings.Contains(strX, "\n") || strings.Contains(strY, "\n")
}

func multiLineDiffResult(diff string) Result {
	return ResultFailureTemplate(`
--- {{ with callArg 0 }}{{ formatNode . }}{{else}}←{{end}}
+++ {{ with callArg 1 }}{{ formatNode . }}{{else}}→{{end}}
{{ .Data.diff }}`,
		map[string]interface{}{"diff": diff})
}

func Len(seq interface{}, expected int) Comparison {
	return func() (result Result) {
		defer func() {
			if e := recover(); e != nil {
				result = ResultFailure(fmt.Sprintf("type %T does not have a length", seq))
			}
		}()
		value := reflect.ValueOf(seq)
		length := value.Len()
		if length == expected {
			return ResultSuccess
		}
		msg := fmt.Sprintf("expected %s (length %d) to have length %d", seq, length, expected)
		return ResultFailure(msg)
	}
}

func Contains(collection interface{}, item interface{}) Comparison {
	return func() Result {
		colValue := reflect.ValueOf(collection)
		if !colValue.IsValid() {
			return ResultFailure(fmt.Sprintf("nil does not contain items"))
		}
		msg := fmt.Sprintf("%v does not contain %v", collection, item)

		itemValue := reflect.ValueOf(item)
		switch colValue.Type().Kind() {
		case reflect.String:
			if itemValue.Type().Kind() != reflect.String {
				return ResultFailure("string may only contain strings")
			}
			return toResult(
				strings.Contains(colValue.String(), itemValue.String()),
				fmt.Sprintf("string %q does not contain %q", collection, item))

		case reflect.Map:
			if itemValue.Type() != colValue.Type().Key() {
				return ResultFailure(fmt.Sprintf(
					"%v can not contain a %v key", colValue.Type(), itemValue.Type()))
			}
			return toResult(colValue.MapIndex(itemValue).IsValid(), msg)

		case reflect.Slice, reflect.Array:
			for i := 0; i < colValue.Len(); i++ {
				if reflect.DeepEqual(colValue.Index(i).Interface(), item) {
					return ResultSuccess
				}
			}
			return ResultFailure(msg)
		default:
			return ResultFailure(fmt.Sprintf("type %T does not contain items", collection))
		}
	}
}

func Panics(f func()) Comparison {
	return func() (result Result) {
		defer func() {
			if err := recover(); err != nil {
				result = ResultSuccess
			}
		}()
		f()
		return ResultFailure("did not panic")
	}
}

func Error(err error, message string) Comparison {
	return func() Result {
		switch {
		case err == nil:
			return ResultFailure("expected an error, got nil")
		case err.Error() != message:
			return ResultFailure(fmt.Sprintf(
				"expected error %q, got %s", message, formatErrorMessage(err)))
		}
		return ResultSuccess
	}
}

func ErrorContains(err error, substring string) Comparison {
	return func() Result {
		switch {
		case err == nil:
			return ResultFailure("expected an error, got nil")
		case !strings.Contains(err.Error(), substring):
			return ResultFailure(fmt.Sprintf(
				"expected error to contain %q, got %s", substring, formatErrorMessage(err)))
		}
		return ResultSuccess
	}
}

func formatErrorMessage(err error) string {
	if _, ok := err.(interface {
		Cause() error
	}); ok {
		return fmt.Sprintf("%q\n%+v", err, err)
	}

	return fmt.Sprintf("%q", err)
}

func Nil(obj interface{}) Comparison {
	msgFunc := func(value reflect.Value) string {
		return fmt.Sprintf("%v (type %s) is not nil", reflect.Indirect(value), value.Type())
	}
	return isNil(obj, msgFunc)
}

func isNil(obj interface{}, msgFunc func(reflect.Value) string) Comparison {
	return func() Result {
		if obj == nil {
			return ResultSuccess
		}
		value := reflect.ValueOf(obj)
		kind := value.Type().Kind()
		if kind >= reflect.Chan && kind <= reflect.Slice {
			if value.IsNil() {
				return ResultSuccess
			}
			return ResultFailure(msgFunc(value))
		}

		return ResultFailure(fmt.Sprintf("%v (type %s) can not be nil", value, value.Type()))
	}
}

func ErrorType(err error, expected interface{}) Comparison {
	return func() Result {
		switch expectedType := expected.(type) {
		case func(error) bool:
			return cmpErrorTypeFunc(err, expectedType)
		case reflect.Type:
			if expectedType.Kind() == reflect.Interface {
				return cmpErrorTypeImplementsType(err, expectedType)
			}
			return cmpErrorTypeEqualType(err, expectedType)
		case nil:
			return ResultFailure(fmt.Sprintf("invalid type for expected: nil"))
		}

		expectedType := reflect.TypeOf(expected)
		switch {
		case expectedType.Kind() == reflect.Struct, isPtrToStruct(expectedType):
			return cmpErrorTypeEqualType(err, expectedType)
		case isPtrToInterface(expectedType):
			return cmpErrorTypeImplementsType(err, expectedType.Elem())
		}
		return ResultFailure(fmt.Sprintf("invalid type for expected: %T", expected))
	}
}

func cmpErrorTypeFunc(err error, f func(error) bool) Result {
	if f(err) {
		return ResultSuccess
	}
	actual := "nil"
	if err != nil {
		actual = fmt.Sprintf("%s (%T)", err, err)
	}
	return ResultFailureTemplate(`error is {{ .Data.actual }}
		{{- with callArg 1 }}, not {{ formatNode . }}{{end -}}`,
		map[string]interface{}{"actual": actual})
}

func cmpErrorTypeEqualType(err error, expectedType reflect.Type) Result {
	if err == nil {
		return ResultFailure(fmt.Sprintf("error is nil, not %s", expectedType))
	}
	errValue := reflect.ValueOf(err)
	if errValue.Type() == expectedType {
		return ResultSuccess
	}
	return ResultFailure(fmt.Sprintf("error is %s (%T), not %s", err, err, expectedType))
}

func cmpErrorTypeImplementsType(err error, expectedType reflect.Type) Result {
	if err == nil {
		return ResultFailure(fmt.Sprintf("error is nil, not %s", expectedType))
	}
	errValue := reflect.ValueOf(err)
	if errValue.Type().Implements(expectedType) {
		return ResultSuccess
	}
	return ResultFailure(fmt.Sprintf("error is %s (%T), not %s", err, err, expectedType))
}

func isPtrToInterface(typ reflect.Type) bool {
	return typ.Kind() == reflect.Ptr && typ.Elem().Kind() == reflect.Interface
}

func isPtrToStruct(typ reflect.Type) bool {
	return typ.Kind() == reflect.Ptr && typ.Elem().Kind() == reflect.Struct
}
