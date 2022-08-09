package cmp

import "reflect"

type valueNode struct {
	parent	*valueNode

	Type	reflect.Type
	ValueX	reflect.Value
	ValueY	reflect.Value

	NumSame	int

	NumDiff	int

	NumIgnored	int

	NumCompared	int

	NumTransformed	int

	NumChildren	int

	MaxDepth	int

	Records	[]reportRecord

	Value	*valueNode

	TransformerName	string
}
type reportRecord struct {
	Key	reflect.Value
	Value	*valueNode
}

func (parent *valueNode) PushStep(ps PathStep) (child *valueNode) {
	vx, vy := ps.Values()
	child = &valueNode{parent: parent, Type: ps.Type(), ValueX: vx, ValueY: vy}
	switch s := ps.(type) {
	case StructField:
		assert(parent.Value == nil)
		parent.Records = append(parent.Records, reportRecord{Key: reflect.ValueOf(s.Name()), Value: child})
	case SliceIndex:
		assert(parent.Value == nil)
		parent.Records = append(parent.Records, reportRecord{Value: child})
	case MapIndex:
		assert(parent.Value == nil)
		parent.Records = append(parent.Records, reportRecord{Key: s.Key(), Value: child})
	case Indirect:
		assert(parent.Value == nil && parent.Records == nil)
		parent.Value = child
	case TypeAssertion:
		assert(parent.Value == nil && parent.Records == nil)
		parent.Value = child
	case Transform:
		assert(parent.Value == nil && parent.Records == nil)
		parent.Value = child
		parent.TransformerName = s.Name()
		parent.NumTransformed++
	default:
		assert(parent == nil)
	}
	return child
}

func (r *valueNode) Report(rs Result) {
	assert(r.MaxDepth == 0)

	if rs.ByIgnore() {
		r.NumIgnored++
	} else {
		if rs.Equal() {
			r.NumSame++
		} else {
			r.NumDiff++
		}
	}
	assert(r.NumSame+r.NumDiff+r.NumIgnored == 1)

	if rs.ByMethod() {
		r.NumCompared++
	}
	if rs.ByFunc() {
		r.NumCompared++
	}
	assert(r.NumCompared <= 1)
}

func (child *valueNode) PopStep() (parent *valueNode) {
	if child.parent == nil {
		return nil
	}
	parent = child.parent
	parent.NumSame += child.NumSame
	parent.NumDiff += child.NumDiff
	parent.NumIgnored += child.NumIgnored
	parent.NumCompared += child.NumCompared
	parent.NumTransformed += child.NumTransformed
	parent.NumChildren += child.NumChildren + 1
	if parent.MaxDepth < child.MaxDepth+1 {
		parent.MaxDepth = child.MaxDepth + 1
	}
	return parent
}
