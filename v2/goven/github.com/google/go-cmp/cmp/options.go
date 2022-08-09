package cmp

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/specgen-io/specgen-golang/v2/goven/github.com/google/go-cmp/cmp/internal/function"
)

type Option interface {
	filter(s *state, t reflect.Type, vx, vy reflect.Value) applicableOption
}

type applicableOption interface {
	Option

	apply(s *state, vx, vy reflect.Value)
}

type coreOption interface {
	Option
	isCore()
}

type core struct{}

func (core) isCore()	{}

type Options []Option

func (opts Options) filter(s *state, t reflect.Type, vx, vy reflect.Value) (out applicableOption) {
	for _, opt := range opts {
		switch opt := opt.filter(s, t, vx, vy); opt.(type) {
		case ignore:
			return ignore{}
		case validator:
			out = validator{}
		case *comparer, *transformer, Options:
			switch out.(type) {
			case nil:
				out = opt
			case validator:

			case *comparer, *transformer, Options:
				out = Options{out, opt}
			}
		}
	}
	return out
}

func (opts Options) apply(s *state, _, _ reflect.Value) {
	const warning = "ambiguous set of applicable options"
	const help = "consider using filters to ensure at most one Comparer or Transformer may apply"
	var ss []string
	for _, opt := range flattenOptions(nil, opts) {
		ss = append(ss, fmt.Sprint(opt))
	}
	set := strings.Join(ss, "\n\t")
	panic(fmt.Sprintf("%s at %#v:\n\t%s\n%s", warning, s.curPath, set, help))
}

func (opts Options) String() string {
	var ss []string
	for _, opt := range opts {
		ss = append(ss, fmt.Sprint(opt))
	}
	return fmt.Sprintf("Options{%s}", strings.Join(ss, ", "))
}

func FilterPath(f func(Path) bool, opt Option) Option {
	if f == nil {
		panic("invalid path filter function")
	}
	if opt := normalizeOption(opt); opt != nil {
		return &pathFilter{fnc: f, opt: opt}
	}
	return nil
}

type pathFilter struct {
	core
	fnc	func(Path) bool
	opt	Option
}

func (f pathFilter) filter(s *state, t reflect.Type, vx, vy reflect.Value) applicableOption {
	if f.fnc(s.curPath) {
		return f.opt.filter(s, t, vx, vy)
	}
	return nil
}

func (f pathFilter) String() string {
	return fmt.Sprintf("FilterPath(%s, %v)", function.NameOf(reflect.ValueOf(f.fnc)), f.opt)
}

func FilterValues(f interface{}, opt Option) Option {
	v := reflect.ValueOf(f)
	if !function.IsType(v.Type(), function.ValueFilter) || v.IsNil() {
		panic(fmt.Sprintf("invalid values filter function: %T", f))
	}
	if opt := normalizeOption(opt); opt != nil {
		vf := &valuesFilter{fnc: v, opt: opt}
		if ti := v.Type().In(0); ti.Kind() != reflect.Interface || ti.NumMethod() > 0 {
			vf.typ = ti
		}
		return vf
	}
	return nil
}

type valuesFilter struct {
	core
	typ	reflect.Type
	fnc	reflect.Value
	opt	Option
}

func (f valuesFilter) filter(s *state, t reflect.Type, vx, vy reflect.Value) applicableOption {
	if !vx.IsValid() || !vx.CanInterface() || !vy.IsValid() || !vy.CanInterface() {
		return nil
	}
	if (f.typ == nil || t.AssignableTo(f.typ)) && s.callTTBFunc(f.fnc, vx, vy) {
		return f.opt.filter(s, t, vx, vy)
	}
	return nil
}

func (f valuesFilter) String() string {
	return fmt.Sprintf("FilterValues(%s, %v)", function.NameOf(f.fnc), f.opt)
}

func Ignore() Option	{ return ignore{} }

type ignore struct{ core }

func (ignore) isFiltered() bool								{ return false }
func (ignore) filter(_ *state, _ reflect.Type, _, _ reflect.Value) applicableOption	{ return ignore{} }
func (ignore) apply(s *state, _, _ reflect.Value)					{ s.report(true, reportByIgnore) }
func (ignore) String() string								{ return "Ignore()" }

type validator struct{ core }

func (validator) filter(_ *state, _ reflect.Type, vx, vy reflect.Value) applicableOption {
	if !vx.IsValid() || !vy.IsValid() {
		return validator{}
	}
	if !vx.CanInterface() || !vy.CanInterface() {
		return validator{}
	}
	return nil
}
func (validator) apply(s *state, vx, vy reflect.Value) {

	if !vx.IsValid() || !vy.IsValid() {
		s.report(vx.IsValid() == vy.IsValid(), 0)
		return
	}

	if !vx.CanInterface() || !vy.CanInterface() {
		help := "consider using a custom Comparer; if you control the implementation of type, you can also consider using an Exporter, AllowUnexported, or cmpopts.IgnoreUnexported"
		var name string
		if t := s.curPath.Index(-2).Type(); t.Name() != "" {

			name = fmt.Sprintf("%q.%v", t.PkgPath(), t.Name())
			if _, ok := reflect.New(t).Interface().(error); ok {
				help = "consider using cmpopts.EquateErrors to compare error values"
			}
		} else {

			var pkgPath string
			for i := 0; i < t.NumField() && pkgPath == ""; i++ {
				pkgPath = t.Field(i).PkgPath
			}
			name = fmt.Sprintf("%q.(%v)", pkgPath, t.String())
		}
		panic(fmt.Sprintf("cannot handle unexported field at %#v:\n\t%v\n%s", s.curPath, name, help))
	}

	panic("not reachable")
}

const identRx = `[_\p{L}][_\p{L}\p{N}]*`

var identsRx = regexp.MustCompile(`^` + identRx + `(\.` + identRx + `)*$`)

func Transformer(name string, f interface{}) Option {
	v := reflect.ValueOf(f)
	if !function.IsType(v.Type(), function.Transformer) || v.IsNil() {
		panic(fmt.Sprintf("invalid transformer function: %T", f))
	}
	if name == "" {
		name = function.NameOf(v)
		if !identsRx.MatchString(name) {
			name = "λ"
		}
	} else if !identsRx.MatchString(name) {
		panic(fmt.Sprintf("invalid name: %q", name))
	}
	tr := &transformer{name: name, fnc: reflect.ValueOf(f)}
	if ti := v.Type().In(0); ti.Kind() != reflect.Interface || ti.NumMethod() > 0 {
		tr.typ = ti
	}
	return tr
}

type transformer struct {
	core
	name	string
	typ	reflect.Type
	fnc	reflect.Value
}

func (tr *transformer) isFiltered() bool	{ return tr.typ != nil }

func (tr *transformer) filter(s *state, t reflect.Type, _, _ reflect.Value) applicableOption {
	for i := len(s.curPath) - 1; i >= 0; i-- {
		if t, ok := s.curPath[i].(Transform); !ok {
			break
		} else if tr == t.trans {
			return nil
		}
	}
	if tr.typ == nil || t.AssignableTo(tr.typ) {
		return tr
	}
	return nil
}

func (tr *transformer) apply(s *state, vx, vy reflect.Value) {
	step := Transform{&transform{pathStep{typ: tr.fnc.Type().Out(0)}, tr}}
	vvx := s.callTRFunc(tr.fnc, vx, step)
	vvy := s.callTRFunc(tr.fnc, vy, step)
	step.vx, step.vy = vvx, vvy
	s.compareAny(step)
}

func (tr transformer) String() string {
	return fmt.Sprintf("Transformer(%s, %s)", tr.name, function.NameOf(tr.fnc))
}

func Comparer(f interface{}) Option {
	v := reflect.ValueOf(f)
	if !function.IsType(v.Type(), function.Equal) || v.IsNil() {
		panic(fmt.Sprintf("invalid comparer function: %T", f))
	}
	cm := &comparer{fnc: v}
	if ti := v.Type().In(0); ti.Kind() != reflect.Interface || ti.NumMethod() > 0 {
		cm.typ = ti
	}
	return cm
}

type comparer struct {
	core
	typ	reflect.Type
	fnc	reflect.Value
}

func (cm *comparer) isFiltered() bool	{ return cm.typ != nil }

func (cm *comparer) filter(_ *state, t reflect.Type, _, _ reflect.Value) applicableOption {
	if cm.typ == nil || t.AssignableTo(cm.typ) {
		return cm
	}
	return nil
}

func (cm *comparer) apply(s *state, vx, vy reflect.Value) {
	eq := s.callTTBFunc(cm.fnc, vx, vy)
	s.report(eq, reportByFunc)
}

func (cm comparer) String() string {
	return fmt.Sprintf("Comparer(%s)", function.NameOf(cm.fnc))
}

func Exporter(f func(reflect.Type) bool) Option {
	if !supportExporters {
		panic("Exporter is not supported on purego builds")
	}
	return exporter(f)
}

type exporter func(reflect.Type) bool

func (exporter) filter(_ *state, _ reflect.Type, _, _ reflect.Value) applicableOption {
	panic("not implemented")
}

func AllowUnexported(types ...interface{}) Option {
	m := make(map[reflect.Type]bool)
	for _, typ := range types {
		t := reflect.TypeOf(typ)
		if t.Kind() != reflect.Struct {
			panic(fmt.Sprintf("invalid struct type: %T", typ))
		}
		m[t] = true
	}
	return exporter(func(t reflect.Type) bool { return m[t] })
}

type Result struct {
	_	[0]func()
	flags	resultFlags
}

func (r Result) Equal() bool {
	return r.flags&(reportEqual|reportByIgnore) != 0
}

func (r Result) ByIgnore() bool {
	return r.flags&reportByIgnore != 0
}

func (r Result) ByMethod() bool {
	return r.flags&reportByMethod != 0
}

func (r Result) ByFunc() bool {
	return r.flags&reportByFunc != 0
}

func (r Result) ByCycle() bool {
	return r.flags&reportByCycle != 0
}

type resultFlags uint

const (
	_	resultFlags	= (1 << iota) / 2

	reportEqual
	reportUnequal
	reportByIgnore
	reportByMethod
	reportByFunc
	reportByCycle
)

func Reporter(r interface {
	PushStep(PathStep)

	Report(Result)

	PopStep()
}) Option {
	return reporter{r}
}

type reporter struct{ reporterIface }
type reporterIface interface {
	PushStep(PathStep)
	Report(Result)
	PopStep()
}

func (reporter) filter(_ *state, _ reflect.Type, _, _ reflect.Value) applicableOption {
	panic("not implemented")
}

func normalizeOption(src Option) Option {
	switch opts := flattenOptions(nil, Options{src}); len(opts) {
	case 0:
		return nil
	case 1:
		return opts[0]
	default:
		return opts
	}
}

func flattenOptions(dst, src Options) Options {
	for _, opt := range src {
		switch opt := opt.(type) {
		case nil:
			continue
		case Options:
			dst = flattenOptions(dst, opt)
		case coreOption:
			dst = append(dst, opt)
		default:
			panic(fmt.Sprintf("invalid option type: %T", opt))
		}
	}
	return dst
}
