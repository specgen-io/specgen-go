package cmp

import (
	"bytes"
	"fmt"
	"math/rand"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/specgen-io/specgen-golang/v2/goven/github.com/google/go-cmp/cmp/internal/flags"
)

var randBool = rand.New(rand.NewSource(time.Now().Unix())).Intn(2) == 0

const maxColumnLength = 80

type indentMode int

func (n indentMode) appendIndent(b []byte, d diffMode) []byte {

	if flags.Deterministic || randBool {

		switch d {
		case diffUnknown, diffIdentical:
			b = append(b, "  "...)
		case diffRemoved:
			b = append(b, "- "...)
		case diffInserted:
			b = append(b, "+ "...)
		}
	} else {

		switch d {
		case diffUnknown, diffIdentical:
			b = append(b, "  "...)
		case diffRemoved:
			b = append(b, "- "...)
		case diffInserted:
			b = append(b, "+ "...)
		}
	}
	return repeatCount(n).appendChar(b, '\t')
}

type repeatCount int

func (n repeatCount) appendChar(b []byte, c byte) []byte {
	for ; n > 0; n-- {
		b = append(b, c)
	}
	return b
}

type textNode interface {
	Len() int

	Equal(textNode) bool

	String() string

	formatCompactTo([]byte, diffMode) ([]byte, textNode)

	formatExpandedTo([]byte, diffMode, indentMode) []byte
}

type textWrap struct {
	Prefix		string
	Value		textNode
	Suffix		string
	Metadata	interface{}
}

func (s *textWrap) Len() int {
	return len(s.Prefix) + s.Value.Len() + len(s.Suffix)
}
func (s1 *textWrap) Equal(s2 textNode) bool {
	if s2, ok := s2.(*textWrap); ok {
		return s1.Prefix == s2.Prefix && s1.Value.Equal(s2.Value) && s1.Suffix == s2.Suffix
	}
	return false
}
func (s *textWrap) String() string {
	var d diffMode
	var n indentMode
	_, s2 := s.formatCompactTo(nil, d)
	b := n.appendIndent(nil, d)
	b = s2.formatExpandedTo(b, d, n)
	b = append(b, '\n')
	return string(b)
}
func (s *textWrap) formatCompactTo(b []byte, d diffMode) ([]byte, textNode) {
	n0 := len(b)
	b = append(b, s.Prefix...)
	b, s.Value = s.Value.formatCompactTo(b, d)
	b = append(b, s.Suffix...)
	if _, ok := s.Value.(textLine); ok {
		return b, textLine(b[n0:])
	}
	return b, s
}
func (s *textWrap) formatExpandedTo(b []byte, d diffMode, n indentMode) []byte {
	b = append(b, s.Prefix...)
	b = s.Value.formatExpandedTo(b, d, n)
	b = append(b, s.Suffix...)
	return b
}

type textList []textRecord
type textRecord struct {
	Diff		diffMode
	Key		string
	Value		textNode
	ElideComma	bool
	Comment		fmt.Stringer
}

func (s *textList) AppendEllipsis(ds diffStats) {
	hasStats := !ds.IsZero()
	if len(*s) == 0 || !(*s)[len(*s)-1].Value.Equal(textEllipsis) {
		if hasStats {
			*s = append(*s, textRecord{Value: textEllipsis, ElideComma: true, Comment: ds})
		} else {
			*s = append(*s, textRecord{Value: textEllipsis, ElideComma: true})
		}
		return
	}
	if hasStats {
		(*s)[len(*s)-1].Comment = (*s)[len(*s)-1].Comment.(diffStats).Append(ds)
	}
}

func (s textList) Len() (n int) {
	for i, r := range s {
		n += len(r.Key)
		if r.Key != "" {
			n += len(": ")
		}
		n += r.Value.Len()
		if i < len(s)-1 {
			n += len(", ")
		}
	}
	return n
}

func (s1 textList) Equal(s2 textNode) bool {
	if s2, ok := s2.(textList); ok {
		if len(s1) != len(s2) {
			return false
		}
		for i := range s1 {
			r1, r2 := s1[i], s2[i]
			if !(r1.Diff == r2.Diff && r1.Key == r2.Key && r1.Value.Equal(r2.Value) && r1.Comment == r2.Comment) {
				return false
			}
		}
		return true
	}
	return false
}

func (s textList) String() string {
	return (&textWrap{Prefix: "{", Value: s, Suffix: "}"}).String()
}

func (s textList) formatCompactTo(b []byte, d diffMode) ([]byte, textNode) {
	s = append(textList(nil), s...)

	n0 := len(b)
	var multiLine bool
	for i, r := range s {
		if r.Diff == diffInserted || r.Diff == diffRemoved {
			multiLine = true
		}
		b = append(b, r.Key...)
		if r.Key != "" {
			b = append(b, ": "...)
		}
		b, s[i].Value = r.Value.formatCompactTo(b, d|r.Diff)
		if _, ok := s[i].Value.(textLine); !ok {
			multiLine = true
		}
		if r.Comment != nil {
			multiLine = true
		}
		if i < len(s)-1 {
			b = append(b, ", "...)
		}
	}

	if (d == diffInserted || d == diffRemoved) && len(b[n0:]) > maxColumnLength {
		multiLine = true
	}
	if !multiLine {
		return b, textLine(b[n0:])
	}
	return b, s
}

func (s textList) formatExpandedTo(b []byte, d diffMode, n indentMode) []byte {
	alignKeyLens := s.alignLens(
		func(r textRecord) bool {
			_, isLine := r.Value.(textLine)
			return r.Key == "" || !isLine
		},
		func(r textRecord) int { return utf8.RuneCountInString(r.Key) },
	)
	alignValueLens := s.alignLens(
		func(r textRecord) bool {
			_, isLine := r.Value.(textLine)
			return !isLine || r.Value.Equal(textEllipsis) || r.Comment == nil
		},
		func(r textRecord) int { return utf8.RuneCount(r.Value.(textLine)) },
	)

	var isSimple bool
	for _, r := range s {
		_, isLine := r.Value.(textLine)
		isSimple = r.Diff == 0 && r.Key == "" && isLine && r.Comment == nil
		if !isSimple {
			break
		}
	}
	if isSimple {
		n++
		var batch []byte
		emitBatch := func() {
			if len(batch) > 0 {
				b = n.appendIndent(append(b, '\n'), d)
				b = append(b, bytes.TrimRight(batch, " ")...)
				batch = batch[:0]
			}
		}
		for _, r := range s {
			line := r.Value.(textLine)
			if len(batch)+len(line)+len(", ") > maxColumnLength {
				emitBatch()
			}
			batch = append(batch, line...)
			batch = append(batch, ", "...)
		}
		emitBatch()
		n--
		return n.appendIndent(append(b, '\n'), d)
	}

	n++
	for i, r := range s {
		b = n.appendIndent(append(b, '\n'), d|r.Diff)
		if r.Key != "" {
			b = append(b, r.Key+": "...)
		}
		b = alignKeyLens[i].appendChar(b, ' ')

		b = r.Value.formatExpandedTo(b, d|r.Diff, n)
		if !r.ElideComma {
			b = append(b, ',')
		}
		b = alignValueLens[i].appendChar(b, ' ')

		if r.Comment != nil {
			b = append(b, " // "+r.Comment.String()...)
		}
	}
	n--

	return n.appendIndent(append(b, '\n'), d)
}

func (s textList) alignLens(
	skipFunc func(textRecord) bool,
	lenFunc func(textRecord) int,
) []repeatCount {
	var startIdx, endIdx, maxLen int
	lens := make([]repeatCount, len(s))
	for i, r := range s {
		if skipFunc(r) {
			for j := startIdx; j < endIdx && j < len(s); j++ {
				lens[j] = repeatCount(maxLen - lenFunc(s[j]))
			}
			startIdx, endIdx, maxLen = i+1, i+1, 0
		} else {
			if maxLen < lenFunc(r) {
				maxLen = lenFunc(r)
			}
			endIdx = i + 1
		}
	}
	for j := startIdx; j < endIdx && j < len(s); j++ {
		lens[j] = repeatCount(maxLen - lenFunc(s[j]))
	}
	return lens
}

type textLine []byte

var (
	textNil		= textLine("nil")
	textEllipsis	= textLine("...")
)

func (s textLine) Len() int {
	return len(s)
}
func (s1 textLine) Equal(s2 textNode) bool {
	if s2, ok := s2.(textLine); ok {
		return bytes.Equal([]byte(s1), []byte(s2))
	}
	return false
}
func (s textLine) String() string {
	return string(s)
}
func (s textLine) formatCompactTo(b []byte, d diffMode) ([]byte, textNode) {
	return append(b, s...), s
}
func (s textLine) formatExpandedTo(b []byte, _ diffMode, _ indentMode) []byte {
	return append(b, s...)
}

type diffStats struct {
	Name		string
	NumIgnored	int
	NumIdentical	int
	NumRemoved	int
	NumInserted	int
	NumModified	int
}

func (s diffStats) IsZero() bool {
	s.Name = ""
	return s == diffStats{}
}

func (s diffStats) NumDiff() int {
	return s.NumRemoved + s.NumInserted + s.NumModified
}

func (s diffStats) Append(ds diffStats) diffStats {
	assert(s.Name == ds.Name)
	s.NumIgnored += ds.NumIgnored
	s.NumIdentical += ds.NumIdentical
	s.NumRemoved += ds.NumRemoved
	s.NumInserted += ds.NumInserted
	s.NumModified += ds.NumModified
	return s
}

func (s diffStats) String() string {
	var ss []string
	var sum int
	labels := [...]string{"ignored", "identical", "removed", "inserted", "modified"}
	counts := [...]int{s.NumIgnored, s.NumIdentical, s.NumRemoved, s.NumInserted, s.NumModified}
	for i, n := range counts {
		if n > 0 {
			ss = append(ss, fmt.Sprintf("%d %v", n, labels[i]))
		}
		sum += n
	}

	name := s.Name
	if sum > 1 {
		name += "s"
		if strings.HasSuffix(name, "ys") {
			name = name[:len(name)-2] + "ies"
		}
	}

	switch n := len(ss); n {
	case 0:
		return ""
	case 1, 2:
		return strings.Join(ss, " and ") + " " + name
	default:
		return strings.Join(ss[:n-1], ", ") + ", and " + ss[n-1] + " " + name
	}
}

type commentString string

func (s commentString) String() string	{ return string(s) }
