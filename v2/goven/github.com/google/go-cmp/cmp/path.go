package cmp

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/specgen-io/specgen-golang/v2/goven/github.com/google/go-cmp/cmp/internal/value"
)

type Path []PathStep

type PathStep interface {
	String() string

	Type() reflect.Type

	Values() (vx, vy reflect.Value)
}

var (
	_	PathStep	= StructField{}
	_	PathStep	= SliceIndex{}
	_	PathStep	= MapIndex{}
	_	PathStep	= Indirect{}
	_	PathStep	= TypeAssertion{}
	_	PathStep	= Transform{}
)

func (pa *Path) push(s PathStep) {
	*pa = append(*pa, s)
}

func (pa *Path) pop() {
	*pa = (*pa)[:len(*pa)-1]
}

func (pa Path) Last() PathStep {
	return pa.Index(-1)
}

func (pa Path) Index(i int) PathStep {
	if i < 0 {
		i = len(pa) + i
	}
	if i < 0 || i >= len(pa) {
		return pathStep{}
	}
	return pa[i]
}

func (pa Path) String() string {
	var ss []string
	for _, s := range pa {
		if _, ok := s.(StructField); ok {
			ss = append(ss, s.String())
		}
	}
	return strings.TrimPrefix(strings.Join(ss, ""), ".")
}

func (pa Path) GoString() string {
	var ssPre, ssPost []string
	var numIndirect int
	for i, s := range pa {
		var nextStep PathStep
		if i+1 < len(pa) {
			nextStep = pa[i+1]
		}
		switch s := s.(type) {
		case Indirect:
			numIndirect++
			pPre, pPost := "(", ")"
			switch nextStep.(type) {
			case Indirect:
				continue
			case StructField:
				numIndirect--
			case nil:
				pPre, pPost = "", ""
			}
			if numIndirect > 0 {
				ssPre = append(ssPre, pPre+strings.Repeat("*", numIndirect))
				ssPost = append(ssPost, pPost)
			}
			numIndirect = 0
			continue
		case Transform:
			ssPre = append(ssPre, s.trans.name+"(")
			ssPost = append(ssPost, ")")
			continue
		}
		ssPost = append(ssPost, s.String())
	}
	for i, j := 0, len(ssPre)-1; i < j; i, j = i+1, j-1 {
		ssPre[i], ssPre[j] = ssPre[j], ssPre[i]
	}
	return strings.Join(ssPre, "") + strings.Join(ssPost, "")
}

type pathStep struct {
	typ	reflect.Type
	vx, vy	reflect.Value
}

func (ps pathStep) Type() reflect.Type			{ return ps.typ }
func (ps pathStep) Values() (vx, vy reflect.Value)	{ return ps.vx, ps.vy }
func (ps pathStep) String() string {
	if ps.typ == nil {
		return "<nil>"
	}
	s := ps.typ.String()
	if s == "" || strings.ContainsAny(s, "{}\n") {
		return "root"
	}
	return fmt.Sprintf("{%s}", s)
}

type StructField struct{ *structField }
type structField struct {
	pathStep
	name	string
	idx	int

	unexported	bool
	mayForce	bool
	paddr		bool
	pvx, pvy	reflect.Value
	field		reflect.StructField
}

func (sf StructField) Type() reflect.Type	{ return sf.typ }
func (sf StructField) Values() (vx, vy reflect.Value) {
	if !sf.unexported {
		return sf.vx, sf.vy
	}

	if sf.mayForce {
		vx = retrieveUnexportedField(sf.pvx, sf.field, sf.paddr)
		vy = retrieveUnexportedField(sf.pvy, sf.field, sf.paddr)
		return vx, vy
	}
	return sf.vx, sf.vy
}
func (sf StructField) String() string	{ return fmt.Sprintf(".%s", sf.name) }

func (sf StructField) Name() string	{ return sf.name }

func (sf StructField) Index() int	{ return sf.idx }

type SliceIndex struct{ *sliceIndex }
type sliceIndex struct {
	pathStep
	xkey, ykey	int
	isSlice		bool
}

func (si SliceIndex) Type() reflect.Type		{ return si.typ }
func (si SliceIndex) Values() (vx, vy reflect.Value)	{ return si.vx, si.vy }
func (si SliceIndex) String() string {
	switch {
	case si.xkey == si.ykey:
		return fmt.Sprintf("[%d]", si.xkey)
	case si.ykey == -1:

		return fmt.Sprintf("[%d->?]", si.xkey)
	case si.xkey == -1:

		return fmt.Sprintf("[?->%d]", si.ykey)
	default:

		return fmt.Sprintf("[%d->%d]", si.xkey, si.ykey)
	}
}

func (si SliceIndex) Key() int {
	if si.xkey != si.ykey {
		return -1
	}
	return si.xkey
}

func (si SliceIndex) SplitKeys() (ix, iy int)	{ return si.xkey, si.ykey }

type MapIndex struct{ *mapIndex }
type mapIndex struct {
	pathStep
	key	reflect.Value
}

func (mi MapIndex) Type() reflect.Type			{ return mi.typ }
func (mi MapIndex) Values() (vx, vy reflect.Value)	{ return mi.vx, mi.vy }
func (mi MapIndex) String() string			{ return fmt.Sprintf("[%#v]", mi.key) }

func (mi MapIndex) Key() reflect.Value	{ return mi.key }

type Indirect struct{ *indirect }
type indirect struct {
	pathStep
}

func (in Indirect) Type() reflect.Type			{ return in.typ }
func (in Indirect) Values() (vx, vy reflect.Value)	{ return in.vx, in.vy }
func (in Indirect) String() string			{ return "*" }

type TypeAssertion struct{ *typeAssertion }
type typeAssertion struct {
	pathStep
}

func (ta TypeAssertion) Type() reflect.Type		{ return ta.typ }
func (ta TypeAssertion) Values() (vx, vy reflect.Value)	{ return ta.vx, ta.vy }
func (ta TypeAssertion) String() string			{ return fmt.Sprintf(".(%v)", ta.typ) }

type Transform struct{ *transform }
type transform struct {
	pathStep
	trans	*transformer
}

func (tf Transform) Type() reflect.Type			{ return tf.typ }
func (tf Transform) Values() (vx, vy reflect.Value)	{ return tf.vx, tf.vy }
func (tf Transform) String() string			{ return fmt.Sprintf("%s()", tf.trans.name) }

func (tf Transform) Name() string	{ return tf.trans.name }

func (tf Transform) Func() reflect.Value	{ return tf.trans.fnc }

func (tf Transform) Option() Option	{ return tf.trans }

type pointerPath struct {
	mx	map[value.Pointer]value.Pointer

	my	map[value.Pointer]value.Pointer
}

func (p *pointerPath) Init() {
	p.mx = make(map[value.Pointer]value.Pointer)
	p.my = make(map[value.Pointer]value.Pointer)
}

func (p pointerPath) Push(vx, vy reflect.Value) (equal, visited bool) {
	px := value.PointerOf(vx)
	py := value.PointerOf(vy)
	_, ok1 := p.mx[px]
	_, ok2 := p.my[py]
	if ok1 || ok2 {
		equal = p.mx[px] == py && p.my[py] == px
		return equal, true
	}
	p.mx[px] = py
	p.my[py] = px
	return false, false
}

func (p pointerPath) Pop(vx, vy reflect.Value) {
	delete(p.mx, value.PointerOf(vx))
	delete(p.my, value.PointerOf(vy))
}

func isExported(id string) bool {
	r, _ := utf8.DecodeRuneInString(id)
	return unicode.IsUpper(r)
}
