package diff

import (
	"math/rand"
	"time"

	"github.com/specgen-io/specgen-golang/v2/goven/github.com/google/go-cmp/cmp/internal/flags"
)

type EditType uint8

const (
	Identity	EditType	= iota

	UniqueX

	UniqueY

	Modified
)

type EditScript []EditType

func (es EditScript) String() string {
	b := make([]byte, len(es))
	for i, e := range es {
		switch e {
		case Identity:
			b[i] = '.'
		case UniqueX:
			b[i] = 'X'
		case UniqueY:
			b[i] = 'Y'
		case Modified:
			b[i] = 'M'
		default:
			panic("invalid edit-type")
		}
	}
	return string(b)
}

func (es EditScript) stats() (s struct{ NI, NX, NY, NM int }) {
	for _, e := range es {
		switch e {
		case Identity:
			s.NI++
		case UniqueX:
			s.NX++
		case UniqueY:
			s.NY++
		case Modified:
			s.NM++
		default:
			panic("invalid edit-type")
		}
	}
	return
}

func (es EditScript) Dist() int	{ return len(es) - es.stats().NI }

func (es EditScript) LenX() int	{ return len(es) - es.stats().NY }

func (es EditScript) LenY() int	{ return len(es) - es.stats().NX }

type EqualFunc func(ix int, iy int) Result

type Result struct{ NumSame, NumDiff int }

func BoolResult(b bool) Result {
	if b {
		return Result{NumSame: 1}
	} else {
		return Result{NumDiff: 2}
	}
}

func (r Result) Equal() bool	{ return r.NumDiff == 0 }

func (r Result) Similar() bool {

	return r.NumSame+1 >= r.NumDiff
}

var randBool = rand.New(rand.NewSource(time.Now().Unix())).Intn(2) == 0

func Difference(nx, ny int, f EqualFunc) (es EditScript) {

	fwdPath := path{+1, point{0, 0}, make(EditScript, 0, (nx+ny)/2)}
	revPath := path{-1, point{nx, ny}, make(EditScript, 0)}
	fwdFrontier := fwdPath.point
	revFrontier := revPath.point

	searchBudget := 4 * (nx + ny)

	f = debug.Begin(nx, ny, f, &fwdPath.es, &revPath.es)

	if flags.Deterministic || randBool {
		goto forwardSearch
	} else {
		goto reverseSearch
	}

forwardSearch:
	{

		if fwdFrontier.X >= revFrontier.X || fwdFrontier.Y >= revFrontier.Y || searchBudget == 0 {
			goto finishSearch
		}
		for stop1, stop2, i := false, false, 0; !(stop1 && stop2) && searchBudget > 0; i++ {

			z := zigzag(i)
			p := point{fwdFrontier.X + z, fwdFrontier.Y - z}
			switch {
			case p.X >= revPath.X || p.Y < fwdPath.Y:
				stop1 = true
			case p.Y >= revPath.Y || p.X < fwdPath.X:
				stop2 = true
			case f(p.X, p.Y).Equal():

				fwdPath.connect(p, f)
				fwdPath.append(Identity)

				for fwdPath.X < revPath.X && fwdPath.Y < revPath.Y {
					if !f(fwdPath.X, fwdPath.Y).Equal() {
						break
					}
					fwdPath.append(Identity)
				}
				fwdFrontier = fwdPath.point
				stop1, stop2 = true, true
			default:
				searchBudget--
			}
			debug.Update()
		}

		if revPath.X-fwdFrontier.X >= revPath.Y-fwdFrontier.Y {
			fwdFrontier.X++
		} else {
			fwdFrontier.Y++
		}
		goto reverseSearch
	}

reverseSearch:
	{

		if fwdFrontier.X >= revFrontier.X || fwdFrontier.Y >= revFrontier.Y || searchBudget == 0 {
			goto finishSearch
		}
		for stop1, stop2, i := false, false, 0; !(stop1 && stop2) && searchBudget > 0; i++ {

			z := zigzag(i)
			p := point{revFrontier.X - z, revFrontier.Y + z}
			switch {
			case fwdPath.X >= p.X || revPath.Y < p.Y:
				stop1 = true
			case fwdPath.Y >= p.Y || revPath.X < p.X:
				stop2 = true
			case f(p.X-1, p.Y-1).Equal():

				revPath.connect(p, f)
				revPath.append(Identity)

				for fwdPath.X < revPath.X && fwdPath.Y < revPath.Y {
					if !f(revPath.X-1, revPath.Y-1).Equal() {
						break
					}
					revPath.append(Identity)
				}
				revFrontier = revPath.point
				stop1, stop2 = true, true
			default:
				searchBudget--
			}
			debug.Update()
		}

		if revFrontier.X-fwdPath.X >= revFrontier.Y-fwdPath.Y {
			revFrontier.X--
		} else {
			revFrontier.Y--
		}
		goto forwardSearch
	}

finishSearch:

	fwdPath.connect(revPath.point, f)
	for i := len(revPath.es) - 1; i >= 0; i-- {
		t := revPath.es[i]
		revPath.es = revPath.es[:i]
		fwdPath.append(t)
	}
	debug.Finish()
	return fwdPath.es
}

type path struct {
	dir	int
	point
	es	EditScript
}

func (p *path) connect(dst point, f EqualFunc) {
	if p.dir > 0 {

		for dst.X > p.X && dst.Y > p.Y {
			switch r := f(p.X, p.Y); {
			case r.Equal():
				p.append(Identity)
			case r.Similar():
				p.append(Modified)
			case dst.X-p.X >= dst.Y-p.Y:
				p.append(UniqueX)
			default:
				p.append(UniqueY)
			}
		}
		for dst.X > p.X {
			p.append(UniqueX)
		}
		for dst.Y > p.Y {
			p.append(UniqueY)
		}
	} else {

		for p.X > dst.X && p.Y > dst.Y {
			switch r := f(p.X-1, p.Y-1); {
			case r.Equal():
				p.append(Identity)
			case r.Similar():
				p.append(Modified)
			case p.Y-dst.Y >= p.X-dst.X:
				p.append(UniqueY)
			default:
				p.append(UniqueX)
			}
		}
		for p.X > dst.X {
			p.append(UniqueX)
		}
		for p.Y > dst.Y {
			p.append(UniqueY)
		}
	}
}

func (p *path) append(t EditType) {
	p.es = append(p.es, t)
	switch t {
	case Identity, Modified:
		p.add(p.dir, p.dir)
	case UniqueX:
		p.add(p.dir, 0)
	case UniqueY:
		p.add(0, p.dir)
	}
	debug.Update()
}

type point struct{ X, Y int }

func (p *point) add(dx, dy int)	{ p.X += dx; p.Y += dy }

func zigzag(x int) int {
	if x&1 != 0 {
		x = ^x
	}
	return x >> 1
}
