package cmp

import (
	"bytes"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/specgen-io/specgen-golang/v2/goven/github.com/google/go-cmp/cmp/internal/diff"
)

func (opts formatOptions) CanFormatDiffSlice(v *valueNode) bool {
	switch {
	case opts.DiffMode != diffUnknown:
		return false
	case v.NumDiff == 0:
		return false
	case !v.ValueX.IsValid() || !v.ValueY.IsValid():
		return false
	case v.NumIgnored > 0:
		return false
	case v.NumTransformed > 0:
		return false
	case v.NumCompared > 1:
		return false
	case v.NumCompared == 1 && v.Type.Name() != "":

		return false
	}

	t := v.Type
	vx, vy := v.ValueX, v.ValueY
	if t.Kind() == reflect.Interface && !vx.IsNil() && !vy.IsNil() && vx.Elem().Type() == vy.Elem().Type() {
		vx, vy = vx.Elem(), vy.Elem()
		t = vx.Type()
	}

	switch t.Kind() {
	case reflect.String:
	case reflect.Array, reflect.Slice:

		switch t.Elem().Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
			reflect.Bool, reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
		default:
			return false
		}

		if t.Kind() == reflect.Slice && (vx.Len() == 0 || vy.Len() == 0) {
			return false
		}

		if v.NumDiff > v.NumSame {
			return true
		}
	default:
		return false
	}

	const minLength = 32
	return vx.Len() >= minLength && vy.Len() >= minLength
}

func (opts formatOptions) FormatDiffSlice(v *valueNode) textNode {
	assert(opts.DiffMode == diffUnknown)
	t, vx, vy := v.Type, v.ValueX, v.ValueY
	if t.Kind() == reflect.Interface {
		vx, vy = vx.Elem(), vy.Elem()
		t = vx.Type()
		opts = opts.WithTypeMode(emitType)
	}

	var sx, sy string
	var ssx, ssy []string
	var isString, isMostlyText, isPureLinedText, isBinary bool
	switch {
	case t.Kind() == reflect.String:
		sx, sy = vx.String(), vy.String()
		isString = true
	case t.Kind() == reflect.Slice && t.Elem() == reflect.TypeOf(byte(0)):
		sx, sy = string(vx.Bytes()), string(vy.Bytes())
		isString = true
	case t.Kind() == reflect.Array:

		vx2, vy2 := reflect.New(t).Elem(), reflect.New(t).Elem()
		vx2.Set(vx)
		vy2.Set(vy)
		vx, vy = vx2, vy2
	}
	if isString {
		var numTotalRunes, numValidRunes, numLines, lastLineIdx, maxLineLen int
		for i, r := range sx + sy {
			numTotalRunes++
			if (unicode.IsPrint(r) || unicode.IsSpace(r)) && r != utf8.RuneError {
				numValidRunes++
			}
			if r == '\n' {
				if maxLineLen < i-lastLineIdx {
					maxLineLen = i - lastLineIdx
				}
				lastLineIdx = i + 1
				numLines++
			}
		}
		isPureText := numValidRunes == numTotalRunes
		isMostlyText = float64(numValidRunes) > math.Floor(0.90*float64(numTotalRunes))
		isPureLinedText = isPureText && numLines >= 4 && maxLineLen <= 1024
		isBinary = !isMostlyText

		if isPureLinedText {
			ssx = strings.Split(sx, "\n")
			ssy = strings.Split(sy, "\n")
			esLines := diff.Difference(len(ssx), len(ssy), func(ix, iy int) diff.Result {
				return diff.BoolResult(ssx[ix] == ssy[iy])
			})
			esBytes := diff.Difference(len(sx), len(sy), func(ix, iy int) diff.Result {
				return diff.BoolResult(sx[ix] == sy[iy])
			})
			efficiencyLines := float64(esLines.Dist()) / float64(len(esLines))
			efficiencyBytes := float64(esBytes.Dist()) / float64(len(esBytes))
			isPureLinedText = efficiencyLines < 4*efficiencyBytes
		}
	}

	var list textList
	var delim string
	switch {

	case isPureLinedText:
		list = opts.formatDiffSlice(
			reflect.ValueOf(ssx), reflect.ValueOf(ssy), 1, "line",
			func(v reflect.Value, d diffMode) textRecord {
				s := formatString(v.Index(0).String())
				return textRecord{Diff: d, Value: textLine(s)}
			},
		)
		delim = "\n"

		isTripleQuoted := true
		prevRemoveLines := map[string]bool{}
		prevInsertLines := map[string]bool{}
		var list2 textList
		list2 = append(list2, textRecord{Value: textLine(`"""`), ElideComma: true})
		for _, r := range list {
			if !r.Value.Equal(textEllipsis) {
				line, _ := strconv.Unquote(string(r.Value.(textLine)))
				line = strings.TrimPrefix(strings.TrimSuffix(line, "\r"), "\r")
				normLine := strings.Map(func(r rune) rune {
					if unicode.IsSpace(r) {
						return -1
					}
					return r
				}, line)
				isPrintable := func(r rune) bool {
					return unicode.IsPrint(r) || r == '\t'
				}
				isTripleQuoted = !strings.HasPrefix(line, `"""`) && !strings.HasPrefix(line, "...") && strings.TrimFunc(line, isPrintable) == ""
				switch r.Diff {
				case diffRemoved:
					isTripleQuoted = isTripleQuoted && !prevInsertLines[normLine]
					prevRemoveLines[normLine] = true
				case diffInserted:
					isTripleQuoted = isTripleQuoted && !prevRemoveLines[normLine]
					prevInsertLines[normLine] = true
				}
				if !isTripleQuoted {
					break
				}
				r.Value = textLine(line)
				r.ElideComma = true
			}
			if !(r.Diff == diffRemoved || r.Diff == diffInserted) {
				prevRemoveLines = map[string]bool{}
				prevInsertLines = map[string]bool{}
			}
			list2 = append(list2, r)
		}
		if r := list2[len(list2)-1]; r.Diff == diffIdentical && len(r.Value.(textLine)) == 0 {
			list2 = list2[:len(list2)-1]
		}
		list2 = append(list2, textRecord{Value: textLine(`"""`), ElideComma: true})
		if isTripleQuoted {
			var out textNode = &textWrap{Prefix: "(", Value: list2, Suffix: ")"}
			switch t.Kind() {
			case reflect.String:
				if t != reflect.TypeOf(string("")) {
					out = opts.FormatType(t, out)
				}
			case reflect.Slice:

				opts = opts.WithTypeMode(emitType)
				out = opts.FormatType(t, out)
			}
			return out
		}

	case isMostlyText:
		list = opts.formatDiffSlice(
			reflect.ValueOf(sx), reflect.ValueOf(sy), 64, "byte",
			func(v reflect.Value, d diffMode) textRecord {
				s := formatString(v.String())
				return textRecord{Diff: d, Value: textLine(s)}
			},
		)

	case isBinary:
		list = opts.formatDiffSlice(
			reflect.ValueOf(sx), reflect.ValueOf(sy), 16, "byte",
			func(v reflect.Value, d diffMode) textRecord {
				var ss []string
				for i := 0; i < v.Len(); i++ {
					ss = append(ss, formatHex(v.Index(i).Uint()))
				}
				s := strings.Join(ss, ", ")
				comment := commentString(fmt.Sprintf("%c|%v|", d, formatASCII(v.String())))
				return textRecord{Diff: d, Value: textLine(s), Comment: comment}
			},
		)

	default:
		var chunkSize int
		if t.Elem().Kind() == reflect.Bool {
			chunkSize = 16
		} else {
			switch t.Elem().Bits() {
			case 8:
				chunkSize = 16
			case 16:
				chunkSize = 12
			case 32:
				chunkSize = 8
			default:
				chunkSize = 8
			}
		}
		list = opts.formatDiffSlice(
			vx, vy, chunkSize, t.Elem().Kind().String(),
			func(v reflect.Value, d diffMode) textRecord {
				var ss []string
				for i := 0; i < v.Len(); i++ {
					switch t.Elem().Kind() {
					case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
						ss = append(ss, fmt.Sprint(v.Index(i).Int()))
					case reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64:
						ss = append(ss, fmt.Sprint(v.Index(i).Uint()))
					case reflect.Uint8, reflect.Uintptr:
						ss = append(ss, formatHex(v.Index(i).Uint()))
					case reflect.Bool, reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
						ss = append(ss, fmt.Sprint(v.Index(i).Interface()))
					}
				}
				s := strings.Join(ss, ", ")
				return textRecord{Diff: d, Value: textLine(s)}
			},
		)
	}

	var out textNode = &textWrap{Prefix: "{", Value: list, Suffix: "}"}
	if !isMostlyText {

		if t.Kind() == reflect.String {
			opts = opts.WithTypeMode(emitType)
		}
		return opts.FormatType(t, out)
	}
	switch t.Kind() {
	case reflect.String:
		out = &textWrap{Prefix: "strings.Join(", Value: out, Suffix: fmt.Sprintf(", %q)", delim)}
		if t != reflect.TypeOf(string("")) {
			out = opts.FormatType(t, out)
		}
	case reflect.Slice:
		out = &textWrap{Prefix: "bytes.Join(", Value: out, Suffix: fmt.Sprintf(", %q)", delim)}
		if t != reflect.TypeOf([]byte(nil)) {
			out = opts.FormatType(t, out)
		}
	}
	return out
}

func formatASCII(s string) string {
	b := bytes.Repeat([]byte{'.'}, len(s))
	for i := 0; i < len(s); i++ {
		if ' ' <= s[i] && s[i] <= '~' {
			b[i] = s[i]
		}
	}
	return string(b)
}

func (opts formatOptions) formatDiffSlice(
	vx, vy reflect.Value, chunkSize int, name string,
	makeRec func(reflect.Value, diffMode) textRecord,
) (list textList) {
	eq := func(ix, iy int) bool {
		return vx.Index(ix).Interface() == vy.Index(iy).Interface()
	}
	es := diff.Difference(vx.Len(), vy.Len(), func(ix, iy int) diff.Result {
		return diff.BoolResult(eq(ix, iy))
	})

	appendChunks := func(v reflect.Value, d diffMode) int {
		n0 := v.Len()
		for v.Len() > 0 {
			n := chunkSize
			if n > v.Len() {
				n = v.Len()
			}
			list = append(list, makeRec(v.Slice(0, n), d))
			v = v.Slice(n, v.Len())
		}
		return n0 - v.Len()
	}

	var numDiffs int
	maxLen := -1
	if opts.LimitVerbosity {
		maxLen = (1 << opts.verbosity()) << 2
		opts.VerbosityLevel--
	}

	groups := coalesceAdjacentEdits(name, es)
	groups = coalesceInterveningIdentical(groups, chunkSize/4)
	groups = cleanupSurroundingIdentical(groups, eq)
	maxGroup := diffStats{Name: name}
	for i, ds := range groups {
		if maxLen >= 0 && numDiffs >= maxLen {
			maxGroup = maxGroup.Append(ds)
			continue
		}

		if ds.NumDiff() == 0 {

			var numLo, numHi int
			numEqual := ds.NumIgnored + ds.NumIdentical
			for numLo < chunkSize*numContextRecords && numLo+numHi < numEqual && i != 0 {
				numLo++
			}
			for numHi < chunkSize*numContextRecords && numLo+numHi < numEqual && i != len(groups)-1 {
				numHi++
			}
			if numEqual-(numLo+numHi) <= chunkSize && ds.NumIgnored == 0 {
				numHi = numEqual - numLo
			}

			appendChunks(vx.Slice(0, numLo), diffIdentical)
			if numEqual > numLo+numHi {
				ds.NumIdentical -= numLo + numHi
				list.AppendEllipsis(ds)
			}
			appendChunks(vx.Slice(numEqual-numHi, numEqual), diffIdentical)
			vx = vx.Slice(numEqual, vx.Len())
			vy = vy.Slice(numEqual, vy.Len())
			continue
		}

		len0 := len(list)
		nx := appendChunks(vx.Slice(0, ds.NumIdentical+ds.NumRemoved+ds.NumModified), diffRemoved)
		vx = vx.Slice(nx, vx.Len())
		ny := appendChunks(vy.Slice(0, ds.NumIdentical+ds.NumInserted+ds.NumModified), diffInserted)
		vy = vy.Slice(ny, vy.Len())
		numDiffs += len(list) - len0
	}
	if maxGroup.IsZero() {
		assert(vx.Len() == 0 && vy.Len() == 0)
	} else {
		list.AppendEllipsis(maxGroup)
	}
	return list
}

func coalesceAdjacentEdits(name string, es diff.EditScript) (groups []diffStats) {
	var prevMode byte
	lastStats := func(mode byte) *diffStats {
		if prevMode != mode {
			groups = append(groups, diffStats{Name: name})
			prevMode = mode
		}
		return &groups[len(groups)-1]
	}
	for _, e := range es {
		switch e {
		case diff.Identity:
			lastStats('=').NumIdentical++
		case diff.UniqueX:
			lastStats('!').NumRemoved++
		case diff.UniqueY:
			lastStats('!').NumInserted++
		case diff.Modified:
			lastStats('!').NumModified++
		}
	}
	return groups
}

func coalesceInterveningIdentical(groups []diffStats, windowSize int) []diffStats {
	groups, groupsOrig := groups[:0], groups
	for i, ds := range groupsOrig {
		if len(groups) >= 2 && ds.NumDiff() > 0 {
			prev := &groups[len(groups)-2]
			curr := &groups[len(groups)-1]
			next := &groupsOrig[i]
			hadX, hadY := prev.NumRemoved > 0, prev.NumInserted > 0
			hasX, hasY := next.NumRemoved > 0, next.NumInserted > 0
			if ((hadX || hasX) && (hadY || hasY)) && curr.NumIdentical <= windowSize {
				*prev = prev.Append(*curr).Append(*next)
				groups = groups[:len(groups)-1]
				continue
			}
		}
		groups = append(groups, ds)
	}
	return groups
}

func cleanupSurroundingIdentical(groups []diffStats, eq func(i, j int) bool) []diffStats {
	var ix, iy int
	for i, ds := range groups {

		if ds.NumDiff() == 0 {
			ix += ds.NumIdentical
			iy += ds.NumIdentical
			continue
		}

		nx := ds.NumIdentical + ds.NumRemoved + ds.NumModified
		ny := ds.NumIdentical + ds.NumInserted + ds.NumModified
		var numLeadingIdentical, numTrailingIdentical int
		for j := 0; j < nx && j < ny && eq(ix+j, iy+j); j++ {
			numLeadingIdentical++
		}
		for j := 0; j < nx && j < ny && eq(ix+nx-1-j, iy+ny-1-j); j++ {
			numTrailingIdentical++
		}
		if numIdentical := numLeadingIdentical + numTrailingIdentical; numIdentical > 0 {
			if numLeadingIdentical > 0 {

				if i-1 >= 0 {
					groups[i-1].NumIdentical += numLeadingIdentical
				} else {

					defer func() {
						groups = append([]diffStats{{Name: groups[0].Name, NumIdentical: numLeadingIdentical}}, groups...)
					}()
				}

				ix += numLeadingIdentical
				iy += numLeadingIdentical
			}
			if numTrailingIdentical > 0 {

				if i+1 < len(groups) {
					groups[i+1].NumIdentical += numTrailingIdentical
				} else {

					defer func() {
						groups = append(groups, diffStats{Name: groups[len(groups)-1].Name, NumIdentical: numTrailingIdentical})
					}()
				}

			}

			nx -= numIdentical
			ny -= numIdentical
			groups[i] = diffStats{Name: ds.Name, NumRemoved: nx, NumInserted: ny}
		}
		ix += nx
		iy += ny
	}
	return groups
}
