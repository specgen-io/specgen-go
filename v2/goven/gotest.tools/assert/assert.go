package assert

import (
	"fmt"
	"go/ast"
	"go/token"

	gocmp "github.com/specgen-io/specgen-golang/v2/goven/github.com/google/go-cmp/cmp"
	"github.com/specgen-io/specgen-golang/v2/goven/gotest.tools/assert/cmp"
	"github.com/specgen-io/specgen-golang/v2/goven/gotest.tools/internal/format"
	"github.com/specgen-io/specgen-golang/v2/goven/gotest.tools/internal/source"
)

type BoolOrComparison interface{}

type TestingT interface {
	FailNow()
	Fail()
	Log(args ...interface{})
}

type helperT interface {
	Helper()
}

const failureMessage = "assertion failed: "

func assert(
	t TestingT,
	failer func(),
	argSelector argSelector,
	comparison BoolOrComparison,
	msgAndArgs ...interface{},
) bool {
	if ht, ok := t.(helperT); ok {
		ht.Helper()
	}
	var success bool
	switch check := comparison.(type) {
	case bool:
		if check {
			return true
		}
		logFailureFromBool(t, msgAndArgs...)

	case func() (success bool, message string):
		success = runCompareFunc(t, check, msgAndArgs...)

	case nil:
		return true

	case error:
		msg := "error is not nil: "
		t.Log(format.WithCustomMessage(failureMessage+msg+check.Error(), msgAndArgs...))

	case cmp.Comparison:
		success = runComparison(t, argSelector, check, msgAndArgs...)

	case func() cmp.Result:
		success = runComparison(t, argSelector, check, msgAndArgs...)

	default:
		t.Log(fmt.Sprintf("invalid Comparison: %v (%T)", check, check))
	}

	if success {
		return true
	}
	failer()
	return false
}

func runCompareFunc(
	t TestingT,
	f func() (success bool, message string),
	msgAndArgs ...interface{},
) bool {
	if ht, ok := t.(helperT); ok {
		ht.Helper()
	}
	if success, message := f(); !success {
		t.Log(format.WithCustomMessage(failureMessage+message, msgAndArgs...))
		return false
	}
	return true
}

func logFailureFromBool(t TestingT, msgAndArgs ...interface{}) {
	if ht, ok := t.(helperT); ok {
		ht.Helper()
	}
	const stackIndex = 3
	const comparisonArgPos = 1
	args, err := source.CallExprArgs(stackIndex)
	if err != nil {
		t.Log(err.Error())
		return
	}

	msg, err := boolFailureMessage(args[comparisonArgPos])
	if err != nil {
		t.Log(err.Error())
		msg = "expression is false"
	}

	t.Log(format.WithCustomMessage(failureMessage+msg, msgAndArgs...))
}

func boolFailureMessage(expr ast.Expr) (string, error) {
	if binaryExpr, ok := expr.(*ast.BinaryExpr); ok && binaryExpr.Op == token.NEQ {
		x, err := source.FormatNode(binaryExpr.X)
		if err != nil {
			return "", err
		}
		y, err := source.FormatNode(binaryExpr.Y)
		if err != nil {
			return "", err
		}
		return x + " is " + y, nil
	}

	if unaryExpr, ok := expr.(*ast.UnaryExpr); ok && unaryExpr.Op == token.NOT {
		x, err := source.FormatNode(unaryExpr.X)
		if err != nil {
			return "", err
		}
		return x + " is true", nil
	}

	formatted, err := source.FormatNode(expr)
	if err != nil {
		return "", err
	}
	return "expression is false: " + formatted, nil
}

func Assert(t TestingT, comparison BoolOrComparison, msgAndArgs ...interface{}) {
	if ht, ok := t.(helperT); ok {
		ht.Helper()
	}
	assert(t, t.FailNow, argsFromComparisonCall, comparison, msgAndArgs...)
}

func Check(t TestingT, comparison BoolOrComparison, msgAndArgs ...interface{}) bool {
	if ht, ok := t.(helperT); ok {
		ht.Helper()
	}
	return assert(t, t.Fail, argsFromComparisonCall, comparison, msgAndArgs...)
}

func NilError(t TestingT, err error, msgAndArgs ...interface{}) {
	if ht, ok := t.(helperT); ok {
		ht.Helper()
	}
	assert(t, t.FailNow, argsAfterT, err, msgAndArgs...)
}

func Equal(t TestingT, x, y interface{}, msgAndArgs ...interface{}) {
	if ht, ok := t.(helperT); ok {
		ht.Helper()
	}
	assert(t, t.FailNow, argsAfterT, cmp.Equal(x, y), msgAndArgs...)
}

func DeepEqual(t TestingT, x, y interface{}, opts ...gocmp.Option) {
	if ht, ok := t.(helperT); ok {
		ht.Helper()
	}
	assert(t, t.FailNow, argsAfterT, cmp.DeepEqual(x, y, opts...))
}

func Error(t TestingT, err error, message string, msgAndArgs ...interface{}) {
	if ht, ok := t.(helperT); ok {
		ht.Helper()
	}
	assert(t, t.FailNow, argsAfterT, cmp.Error(err, message), msgAndArgs...)
}

func ErrorContains(t TestingT, err error, substring string, msgAndArgs ...interface{}) {
	if ht, ok := t.(helperT); ok {
		ht.Helper()
	}
	assert(t, t.FailNow, argsAfterT, cmp.ErrorContains(err, substring), msgAndArgs...)
}

func ErrorType(t TestingT, err error, expected interface{}, msgAndArgs ...interface{}) {
	if ht, ok := t.(helperT); ok {
		ht.Helper()
	}
	assert(t, t.FailNow, argsAfterT, cmp.ErrorType(err, expected), msgAndArgs...)
}
