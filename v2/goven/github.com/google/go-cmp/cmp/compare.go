package cmp

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/specgen-io/specgen-golang/v2/goven/github.com/google/go-cmp/cmp/internal/diff"
	"github.com/specgen-io/specgen-golang/v2/goven/github.com/google/go-cmp/cmp/internal/function"
	"github.com/specgen-io/specgen-golang/v2/goven/github.com/google/go-cmp/cmp/internal/value"
)

func Equal(x, y interface{}, opts ...Option) bool {
	s := newState(opts)
	s.compareAny(rootStep(x, y))
	return s.result.Equal()
}

func Diff(x, y interface{}, opts ...Option) string {
	s := newState(opts)

	if len(s.reporters) == 0 {
		s.compareAny(rootStep(x, y))
		if s.result.Equal() {
			return ""
		}
		s.result = diff.Result{}
	}

	r := new(defaultReporter)
	s.reporters = append(s.reporters, reporter{r})
	s.compareAny(rootStep(x, y))
	d := r.String()
	if (d == "") != s.result.Equal() {
		panic("inconsistent difference and equality results")
	}
	return d
}

func rootStep(x, y interface{}) PathStep {
	vx := reflect.ValueOf(x)
	vy := reflect.ValueOf(y)

	var t reflect.Type
	if !vx.IsValid() || !vy.IsValid() || vx.Type() != vy.Type() {
		t = reflect.TypeOf((*interface{})(nil)).Elem()
		if vx.IsValid() {
			vvx := reflect.New(t).Elem()
			vvx.Set(vx)
			vx = vvx
		}
		if vy.IsValid() {
			vvy := reflect.New(t).Elem()
			vvy.Set(vy)
			vy = vvy
		}
	} else {
		t = vx.Type()
	}

	return &pathStep{t, vx, vy}
}

type state struct {
	result		diff.Result
	curPath		Path
	curPtrs		pointerPath
	reporters	[]reporter

	recChecker	recChecker

	dynChecker	dynChecker

	exporters	[]exporter
	opts		Options
}

func newState(opts []Option) *state {

	s := &state{opts: Options{validator{}}}
	s.curPtrs.Init()
	s.processOption(Options(opts))
	return s
}

func (s *state) processOption(opt Option) {
	switch opt := opt.(type) {
	case nil:
	case Options:
		for _, o := range opt {
			s.processOption(o)
		}
	case coreOption:
		type filtered interface {
			isFiltered() bool
		}
		if fopt, ok := opt.(filtered); ok && !fopt.isFiltered() {
			panic(fmt.Sprintf("cannot use an unfiltered option: %v", opt))
		}
		s.opts = append(s.opts, opt)
	case exporter:
		s.exporters = append(s.exporters, opt)
	case reporter:
		s.reporters = append(s.reporters, opt)
	default:
		panic(fmt.Sprintf("unknown option %T", opt))
	}
}

func (s *state) statelessCompare(step PathStep) diff.Result {

	oldResult, oldReporters := s.result, s.reporters
	s.result = diff.Result{}
	s.reporters = nil
	s.compareAny(step)
	res := s.result
	s.result, s.reporters = oldResult, oldReporters
	return res
}

func (s *state) compareAny(step PathStep) {

	s.curPath.push(step)
	defer s.curPath.pop()
	for _, r := range s.reporters {
		r.PushStep(step)
		defer r.PopStep()
	}
	s.recChecker.Check(s.curPath)

	t := step.Type()
	vx, vy := step.Values()
	if si, ok := step.(SliceIndex); ok && si.isSlice && vx.IsValid() && vy.IsValid() {
		px, py := vx.Addr(), vy.Addr()
		if eq, visited := s.curPtrs.Push(px, py); visited {
			s.report(eq, reportByCycle)
			return
		}
		defer s.curPtrs.Pop(px, py)
	}

	if s.tryOptions(t, vx, vy) {
		return
	}

	if s.tryMethod(t, vx, vy) {
		return
	}

	switch t.Kind() {
	case reflect.Bool:
		s.report(vx.Bool() == vy.Bool(), 0)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		s.report(vx.Int() == vy.Int(), 0)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		s.report(vx.Uint() == vy.Uint(), 0)
	case reflect.Float32, reflect.Float64:
		s.report(vx.Float() == vy.Float(), 0)
	case reflect.Complex64, reflect.Complex128:
		s.report(vx.Complex() == vy.Complex(), 0)
	case reflect.String:
		s.report(vx.String() == vy.String(), 0)
	case reflect.Chan, reflect.UnsafePointer:
		s.report(vx.Pointer() == vy.Pointer(), 0)
	case reflect.Func:
		s.report(vx.IsNil() && vy.IsNil(), 0)
	case reflect.Struct:
		s.compareStruct(t, vx, vy)
	case reflect.Slice, reflect.Array:
		s.compareSlice(t, vx, vy)
	case reflect.Map:
		s.compareMap(t, vx, vy)
	case reflect.Ptr:
		s.comparePtr(t, vx, vy)
	case reflect.Interface:
		s.compareInterface(t, vx, vy)
	default:
		panic(fmt.Sprintf("%v kind not handled", t.Kind()))
	}
}

func (s *state) tryOptions(t reflect.Type, vx, vy reflect.Value) bool {

	if opt := s.opts.filter(s, t, vx, vy); opt != nil {
		opt.apply(s, vx, vy)
		return true
	}
	return false
}

func (s *state) tryMethod(t reflect.Type, vx, vy reflect.Value) bool {

	m, ok := t.MethodByName("Equal")
	if !ok || !function.IsType(m.Type, function.EqualAssignable) {
		return false
	}

	eq := s.callTTBFunc(m.Func, vx, vy)
	s.report(eq, reportByMethod)
	return true
}

func (s *state) callTRFunc(f, v reflect.Value, step Transform) reflect.Value {
	if !s.dynChecker.Next() {
		return f.Call([]reflect.Value{v})[0]
	}

	c := make(chan reflect.Value)
	go detectRaces(c, f, v)
	got := <-c
	want := f.Call([]reflect.Value{v})[0]
	if step.vx, step.vy = got, want; !s.statelessCompare(step).Equal() {

		if step.vx, step.vy = want, want; !s.statelessCompare(step).Equal() {
			return want
		}
		panic(fmt.Sprintf("non-deterministic function detected: %s", function.NameOf(f)))
	}
	return want
}

func (s *state) callTTBFunc(f, x, y reflect.Value) bool {
	if !s.dynChecker.Next() {
		return f.Call([]reflect.Value{x, y})[0].Bool()
	}

	c := make(chan reflect.Value)
	go detectRaces(c, f, y, x)
	got := <-c
	want := f.Call([]reflect.Value{x, y})[0].Bool()
	if !got.IsValid() || got.Bool() != want {
		panic(fmt.Sprintf("non-deterministic or non-symmetric function detected: %s", function.NameOf(f)))
	}
	return want
}

func detectRaces(c chan<- reflect.Value, f reflect.Value, vs ...reflect.Value) {
	var ret reflect.Value
	defer func() {
		recover()
		c <- ret
	}()
	ret = f.Call(vs)[0]
}

func (s *state) compareStruct(t reflect.Type, vx, vy reflect.Value) {
	var addr bool
	var vax, vay reflect.Value

	var mayForce, mayForceInit bool
	step := StructField{&structField{}}
	for i := 0; i < t.NumField(); i++ {
		step.typ = t.Field(i).Type
		step.vx = vx.Field(i)
		step.vy = vy.Field(i)
		step.name = t.Field(i).Name
		step.idx = i
		step.unexported = !isExported(step.name)
		if step.unexported {
			if step.name == "_" {
				continue
			}

			if !vax.IsValid() || !vay.IsValid() {

				addr = vx.CanAddr() || vy.CanAddr()
				vax = makeAddressable(vx)
				vay = makeAddressable(vy)
			}
			if !mayForceInit {
				for _, xf := range s.exporters {
					mayForce = mayForce || xf(t)
				}
				mayForceInit = true
			}
			step.mayForce = mayForce
			step.paddr = addr
			step.pvx = vax
			step.pvy = vay
			step.field = t.Field(i)
		}
		s.compareAny(step)
	}
}

func (s *state) compareSlice(t reflect.Type, vx, vy reflect.Value) {
	isSlice := t.Kind() == reflect.Slice
	if isSlice && (vx.IsNil() || vy.IsNil()) {
		s.report(vx.IsNil() && vy.IsNil(), 0)
		return
	}

	step := SliceIndex{&sliceIndex{pathStep: pathStep{typ: t.Elem()}, isSlice: isSlice}}
	withIndexes := func(ix, iy int) SliceIndex {
		if ix >= 0 {
			step.vx, step.xkey = vx.Index(ix), ix
		} else {
			step.vx, step.xkey = reflect.Value{}, -1
		}
		if iy >= 0 {
			step.vy, step.ykey = vy.Index(iy), iy
		} else {
			step.vy, step.ykey = reflect.Value{}, -1
		}
		return step
	}

	var indexesX, indexesY []int
	var ignoredX, ignoredY []bool
	for ix := 0; ix < vx.Len(); ix++ {
		ignored := s.statelessCompare(withIndexes(ix, -1)).NumDiff == 0
		if !ignored {
			indexesX = append(indexesX, ix)
		}
		ignoredX = append(ignoredX, ignored)
	}
	for iy := 0; iy < vy.Len(); iy++ {
		ignored := s.statelessCompare(withIndexes(-1, iy)).NumDiff == 0
		if !ignored {
			indexesY = append(indexesY, iy)
		}
		ignoredY = append(ignoredY, ignored)
	}

	edits := diff.Difference(len(indexesX), len(indexesY), func(ix, iy int) diff.Result {
		return s.statelessCompare(withIndexes(indexesX[ix], indexesY[iy]))
	})

	var ix, iy int
	for ix < vx.Len() || iy < vy.Len() {
		var e diff.EditType
		switch {
		case ix < len(ignoredX) && ignoredX[ix]:
			e = diff.UniqueX
		case iy < len(ignoredY) && ignoredY[iy]:
			e = diff.UniqueY
		default:
			e, edits = edits[0], edits[1:]
		}
		switch e {
		case diff.UniqueX:
			s.compareAny(withIndexes(ix, -1))
			ix++
		case diff.UniqueY:
			s.compareAny(withIndexes(-1, iy))
			iy++
		default:
			s.compareAny(withIndexes(ix, iy))
			ix++
			iy++
		}
	}
}

func (s *state) compareMap(t reflect.Type, vx, vy reflect.Value) {
	if vx.IsNil() || vy.IsNil() {
		s.report(vx.IsNil() && vy.IsNil(), 0)
		return
	}

	if eq, visited := s.curPtrs.Push(vx, vy); visited {
		s.report(eq, reportByCycle)
		return
	}
	defer s.curPtrs.Pop(vx, vy)

	step := MapIndex{&mapIndex{pathStep: pathStep{typ: t.Elem()}}}
	for _, k := range value.SortKeys(append(vx.MapKeys(), vy.MapKeys()...)) {
		step.vx = vx.MapIndex(k)
		step.vy = vy.MapIndex(k)
		step.key = k
		if !step.vx.IsValid() && !step.vy.IsValid() {

			const help = "consider providing a Comparer to compare the map"
			panic(fmt.Sprintf("%#v has map key with NaNs\n%s", s.curPath, help))
		}
		s.compareAny(step)
	}
}

func (s *state) comparePtr(t reflect.Type, vx, vy reflect.Value) {
	if vx.IsNil() || vy.IsNil() {
		s.report(vx.IsNil() && vy.IsNil(), 0)
		return
	}

	if eq, visited := s.curPtrs.Push(vx, vy); visited {
		s.report(eq, reportByCycle)
		return
	}
	defer s.curPtrs.Pop(vx, vy)

	vx, vy = vx.Elem(), vy.Elem()
	s.compareAny(Indirect{&indirect{pathStep{t.Elem(), vx, vy}}})
}

func (s *state) compareInterface(t reflect.Type, vx, vy reflect.Value) {
	if vx.IsNil() || vy.IsNil() {
		s.report(vx.IsNil() && vy.IsNil(), 0)
		return
	}
	vx, vy = vx.Elem(), vy.Elem()
	if vx.Type() != vy.Type() {
		s.report(false, 0)
		return
	}
	s.compareAny(TypeAssertion{&typeAssertion{pathStep{vx.Type(), vx, vy}}})
}

func (s *state) report(eq bool, rf resultFlags) {
	if rf&reportByIgnore == 0 {
		if eq {
			s.result.NumSame++
			rf |= reportEqual
		} else {
			s.result.NumDiff++
			rf |= reportUnequal
		}
	}
	for _, r := range s.reporters {
		r.Report(Result{flags: rf})
	}
}

type recChecker struct{ next int }

func (rc *recChecker) Check(p Path) {
	const minLen = 1 << 16
	if rc.next == 0 {
		rc.next = minLen
	}
	if len(p) < rc.next {
		return
	}
	rc.next <<= 1

	var ss []string
	m := map[Option]int{}
	for _, ps := range p {
		if t, ok := ps.(Transform); ok {
			t := t.Option()
			if m[t] == 1 {
				tf := t.(*transformer).fnc.Type()
				ss = append(ss, fmt.Sprintf("%v: %v => %v", t, tf.In(0), tf.Out(0)))
			}
			m[t]++
		}
	}
	if len(ss) > 0 {
		const warning = "recursive set of Transformers detected"
		const help = "consider using cmpopts.AcyclicTransformer"
		set := strings.Join(ss, "\n\t")
		panic(fmt.Sprintf("%s:\n\t%s\n%s", warning, set, help))
	}
}

type dynChecker struct{ curr, next int }

func (dc *dynChecker) Next() bool {
	ok := dc.curr == dc.next
	if ok {
		dc.curr = 0
		dc.next++
	}
	dc.curr++
	return ok
}

func makeAddressable(v reflect.Value) reflect.Value {
	if v.CanAddr() {
		return v
	}
	vc := reflect.New(v.Type()).Elem()
	vc.Set(v)
	return vc
}
