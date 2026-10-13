package exporttoporobot

import (
	"fmt"
	"math"
	"pockettopo-exporter/internal/nativefmt"
	"strconv"
	"strings"
	"time"
)

type Record struct {
	Kind                         string
	Series, Index, Line, EndLine int
	Keys                         []string
}
type writer struct {
	b           []byte
	line, limit int
	failed      bool
	records     []Record
}

func (w *writer) write(s string) {
	b := encode(strings.Join([]string{s, "\r\n"}, ""))
	if len(b) > w.limit-len(w.b) {
		w.failed = true
		return
	}
	w.b = append(w.b, b...)
	w.line += strings.Count(s, "\n") + 1
}
func header(ser, num, code, trip int) string {
	prefix := " "
	suffix := " "
	if trip < 0 {
		prefix = "("
		suffix = ""
		trip = -trip
	}
	return fmt.Sprintf("%s%5d%6d%4d%4d%4d%s", prefix, ser, num, code, 1, trip, suffix)
}
func single(v float32, places int) string {
	f := float64(v)
	if math.IsNaN(f) {
		return "NaN"
	}
	if math.IsInf(f, 1) {
		return "Infinity"
	}
	if math.IsInf(f, -1) {
		return "-Infinity"
	}
	s := strconv.FormatFloat(math.Abs(f), 'e', 6, 64)
	e, _ := strconv.Atoi(s[strings.IndexByte(s, 'e')+1:])
	p := 6 - e
	scale := float64(1)
	if p < 0 {
		scale = math.Pow10(-p)
		p = 0
	}
	rounded, _ := strconv.ParseFloat(nativefmt.Fixed(f/scale, p), 64)
	return nativefmt.Fixed(rounded*scale, places)
}
func (w *writer) comment(ser, num int, s *string, n int) int {
	if s == nil {
		return n
	}
	budget := 254
	for _, part := range strings.FieldsFunc(*s, func(r rune) bool { return r == '\r' || r == '\n' }) {
		if budget <= 0 {
			break
		}
		text := limit(trim(part), budget)
		if text == "" {
			continue
		}
		budget -= len(units(text)) + 2
		prefix := " "
		for len(units(text)) > 55 {
			w.write(strings.Join([]string{header(ser, num, n, -1), prefix, limit(text, 55)}, ""))
			n++
			text = string(utf16Tail(text, 55))
			prefix = "-"
		}
		w.write(strings.Join([]string{header(ser, num, n, -1), prefix, text}, ""))
		n++
	}
	return n
}
func utf16Tail(s string, n int) string { return stringFromUnits(units(s)[n:]) }
func tripName(s string) string {
	s = limit(s, 12)
	return strings.Join([]string{s, strings.Repeat(" ", 14-len(units(s)))}, "")
}
func (w *writer) record(kind string, ser, index, start int, keys []string) {
	if len(keys) == 0 {
		keys = []string{strings.Join([]string{"generated:", kind}, "")}
	}
	w.records = append(w.records, Record{kind, ser, index, start, w.line - 1, append([]string{}, keys...)})
}
func (m *model) write(at time.Time, byteLimit int) ([]byte, []Record, error) {
	w := writer{line: 1, limit: byteLimit, records: []Record{}}
	for i := 1; i < len(m.refs); i++ {
		r := m.refs[i]
		if r != nil {
			if r.name == nil {
				return nil, nil, fmt.Errorf("native_failure (Toporobot null reference name)")
			}
			start := w.line
			w.write(strings.Join([]string{header(-6, i, 1, 1), limit(*r.name, 55)}, ""))
			w.record("reference-name", -6, i, start, r.keys)
		}
	}
	for i := 1; i < len(m.refs); i++ {
		r := m.refs[i]
		if r != nil {
			start := w.line
			w.write(fmt.Sprintf("%s %10s  %10s  %10s%6d%6d", header(-5, i, 1, 1), nativefmt.Fixed(r.e, 2), nativefmt.Fixed(r.n, 2), nativefmt.Fixed(r.h, 2), r.ser, r.num))
			w.comment(-5, i, r.comment, 1)
			w.record("reference", -5, i, start, r.keys)
		}
	}
	if len(m.refs) > 0 {
		for i := 1; i <= 2; i++ {
			start := w.line
			w.write(fmt.Sprintf("%s%02d/%02d/%02d %02d:%02d:%02d  Guest", header(-4, i, 1, 1), at.Year()%100, at.Month(), at.Day(), at.Hour(), at.Minute(), at.Second()))
			w.record("writer-time", -4, i, start, []string{"metadata:export-time"})
		}
		start := w.line
		w.write(header(-3, 1, 1, 1))
		w.record("separator", -3, 1, start, []string{"metadata:separator"})
	}
	for i := 1; i < len(m.trips); i++ {
		t := m.trips[i]
		if t == nil {
			continue
		}
		start := w.line
		date := fmt.Sprintf("%s%s%s  ", decimal(t.day, 2), decimal(t.month, 2), decimal(t.year, 4))
		if t.year >= 1920 && t.year < 2020 {
			date = fmt.Sprintf("%s/%s/%s  ", decimal(t.day, 2), decimal(t.month, 2), decimal(t.year%100, 2))
		}
		automatic := 0
		if t.auto {
			automatic = 1
		}
		w.write(fmt.Sprintf("%s%s%s%s%d %7s%4d%4d", header(-2, i, 1, 1), date, tripName(t.name1), tripName(t.name2), automatic, single(t.azi, 2), int(math.RoundToEven(float64(t.incl*10))), t.color))
		n := 1
		for j, name := range []string{t.name1, t.name2} {
			if len(units(name)) > 12 {
				w.write(fmt.Sprintf("%s-%s¸%d", header(-2, i, n, -1), limit(utf16Tail(name, 12), 53), j+1))
				n++
			}
		}
		w.comment(-2, i, t.comment, n)
		w.record("trip", -2, i, start, t.keys)
	}
	for i := 1; i < len(m.codes); i++ {
		c := m.codes[i]
		if c == nil {
			continue
		}
		start := w.line
		w.write(fmt.Sprintf("%s %s.00  %s.00 %7s %7s %7s %7s %7s", header(-1, i, 1, 1), decimal(c.aziUnit, 3), decimal(c.inclUnit, 3), single(c.deltaLen, 2), single(c.deltaAzi, 2), single(c.deltaIncl, 2), single(c.ratio, 2), single(c.limit, 2)))
		w.comment(-1, i, c.comment, 1)
		w.record("code", -1, i, start, c.keys)
	}
	for i := 1; i < len(m.series); i++ {
		s := m.series[i]
		if s == nil {
			continue
		}
		start := w.line
		name := ""
		if s.name != nil {
			name = limit(*s.name, 55)
		}
		w.write(strings.Join([]string{header(i, -2, 1, 1), name}, ""))
		w.write(fmt.Sprintf("%s%7d %7d %7d %7d %7d %7d %7d", header(i, -1, 1, 1), s.startSer, s.startNum, s.endSer, s.endNum, s.length, s.chance, s.obst))
		w.comment(i, -1, s.comment, 1)
		w.record("series", i, -1, start, s.keys)
		for j, t := range s.stations {
			if t == nil {
				continue
			}
			start = w.line
			w.write(fmt.Sprintf("%s%7s %7s %7s %7s %7s %7s %7s", header(i, j, t.code, t.trip), single(t.dist, 2), single(t.az, 2), single(t.incl, 2), single(t.left, 2), single(t.right, 2), single(t.up, 2), single(t.down, 2)))
			n := w.comment(i, j, t.comment, 1)
			w.record("station", i, j, start, t.keys)
			for _, l := range t.lines {
				start = w.line
				w.write(fmt.Sprintf("%s [%s %s %s]", header(i, j, n, -1), single(l.dist, 2), single(l.az, 1), single(l.incl, 1)))
				w.record("splay", i, j, start, l.keys)
				n++
			}
		}
	}
	if w.failed {
		return nil, nil, fmt.Errorf("resource_limit (Toporobot output bytes)")
	}
	return w.b, w.records, nil
}

func decimal(value, digits int) string {
	s := strconv.Itoa(value)
	sign := ""
	if strings.HasPrefix(s, "-") {
		sign = "-"
		s = s[1:]
	}
	return strings.Join([]string{sign, strings.Repeat("0", max(0, digits-len(s))), s}, "")
}
