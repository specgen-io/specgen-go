package cmp

type defaultReporter struct {
	root	*valueNode
	curr	*valueNode
}

func (r *defaultReporter) PushStep(ps PathStep) {
	r.curr = r.curr.PushStep(ps)
	if r.root == nil {
		r.root = r.curr
	}
}
func (r *defaultReporter) Report(rs Result) {
	r.curr.Report(rs)
}
func (r *defaultReporter) PopStep() {
	r.curr = r.curr.PopStep()
}

func (r *defaultReporter) String() string {
	assert(r.root != nil && r.curr == nil)
	if r.root.NumDiff == 0 {
		return ""
	}
	ptrs := new(pointerReferences)
	text := formatOptions{}.FormatDiff(r.root, ptrs)
	resolveReferences(text)
	return text.String()
}

func assert(ok bool) {
	if !ok {
		panic("assertion failure")
	}
}
