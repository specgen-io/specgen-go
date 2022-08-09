package diff

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	updateDelay	= 100 * time.Millisecond
	finishDelay	= 500 * time.Millisecond
	ansiTerminal	= true
)

var debug debugger

type debugger struct {
	sync.Mutex
	p1, p2			EditScript
	fwdPath, revPath	*EditScript
	grid			[]byte
	lines			int
}

func (dbg *debugger) Begin(nx, ny int, f EqualFunc, p1, p2 *EditScript) EqualFunc {
	dbg.Lock()
	dbg.fwdPath, dbg.revPath = p1, p2
	top := "┌─" + strings.Repeat("──", nx) + "┐\n"
	row := "│ " + strings.Repeat("· ", nx) + "│\n"
	btm := "└─" + strings.Repeat("──", nx) + "┘\n"
	dbg.grid = []byte(top + strings.Repeat(row, ny) + btm)
	dbg.lines = strings.Count(dbg.String(), "\n")
	fmt.Print(dbg)

	return func(ix, iy int) (r Result) {
		cell := dbg.grid[len(top)+iy*len(row):][len("│ ")+len("· ")*ix:][:len("·")]
		for i := range cell {
			cell[i] = 0
		}
		switch r = f(ix, iy); {
		case r.Equal():
			cell[0] = '\\'
		case r.Similar():
			cell[0] = 'X'
		default:
			cell[0] = '#'
		}
		return
	}
}

func (dbg *debugger) Update() {
	dbg.print(updateDelay)
}

func (dbg *debugger) Finish() {
	dbg.print(finishDelay)
	dbg.Unlock()
}

func (dbg *debugger) String() string {
	dbg.p1, dbg.p2 = *dbg.fwdPath, dbg.p2[:0]
	for i := len(*dbg.revPath) - 1; i >= 0; i-- {
		dbg.p2 = append(dbg.p2, (*dbg.revPath)[i])
	}
	return fmt.Sprintf("%s[%v|%v]\n\n", dbg.grid, dbg.p1, dbg.p2)
}

func (dbg *debugger) print(d time.Duration) {
	if ansiTerminal {
		fmt.Printf("\x1b[%dA", dbg.lines)
	}
	fmt.Print(dbg)
	time.Sleep(d)
}
