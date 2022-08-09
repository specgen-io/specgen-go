package diff

var debug debugger

type debugger struct{}

func (debugger) Begin(_, _ int, f EqualFunc, _, _ *EditScript) EqualFunc {
	return f
}
func (debugger) Update()	{}
func (debugger) Finish()	{}
