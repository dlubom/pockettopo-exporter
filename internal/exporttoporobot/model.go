// Package exporttoporobot writes and merges the original PocketTopo Toporobot profile.
package exporttoporobot

import (
	_ "embed"
	"fmt"
	"math"
	"strings"
	"unicode/utf16"
)

// The exhaustive BMP map is captured from the original MacEncoding class.
//
//go:embed testdata/mac.bin
var mac []byte

//go:embed testdata/trim.bin
var whitespace []byte

type reference struct {
	name     *string
	e, n, h  float64
	ser, num int
	comment  *string
	keys     []string
}
type trip struct {
	day, month, year int
	name1, name2     string
	auto             bool
	azi, incl        float32
	color            int
	comment          *string
	keys             []string
}
type code struct {
	aziUnit, inclUnit                           int
	deltaLen, deltaAzi, deltaIncl, ratio, limit float32
	comment                                     *string
	keys                                        []string
}
type inline struct {
	dist, az, incl float32
	keys           []string
}
type station struct {
	trip, code                            int
	dist, az, incl, left, right, up, down float32
	comment                               *string
	lines                                 []inline
	keys                                  []string
}
type series struct {
	name                                                     *string
	startSer, startNum, endSer, endNum, length, chance, obst int
	stations                                                 []*station
	comment                                                  *string
	keys                                                     []string
}
type model struct {
	refs     []*reference
	trips    []*trip
	codes    []*code
	series   []*series
	work     int
	maxWork  int
	slots    int
	maxSlots int
}

func newModel() *model {
	m := new(model)
	m.maxWork = 10_000_000
	m.maxSlots = 2_000_000
	return m
}
func (m *model) tick() error {
	m.work++
	if m.work > m.maxWork {
		return fmt.Errorf("resource_limit (Toporobot work)")
	}
	return nil
}
func grow[T any](m *model, a []*T, n int) ([]*T, error) {
	if n < 0 {
		return nil, fmt.Errorf("native_failure (negative Toporobot index)")
	}
	if n > 1000000 || n+1-len(a) > m.maxSlots-m.slots {
		return nil, fmt.Errorf("resource_limit (Toporobot indexes)")
	}
	if n >= len(a) {
		extra := n + 1 - len(a)
		m.slots += extra
		a = append(a, make([]*T, extra)...)
	}
	return a, nil
}
func (m *model) addSeries(n int) (*series, error) {
	a, e := grow(m, m.series, n)
	if e != nil {
		return nil, e
	}
	m.series = a
	if a[n] == nil {
		a[n] = new(series)
	}
	return a[n], nil
}
func (m *model) addStation(ser, n int) (*station, error) {
	s, e := m.addSeries(ser)
	if e != nil {
		return nil, e
	}
	a, e := grow(m, s.stations, n)
	if e != nil {
		return nil, e
	}
	s.stations = a
	if a[n] == nil {
		a[n] = new(station)
		if n > s.length {
			if s.endSer == ser && s.endNum == s.length {
				s.endNum = n
			}
			s.length = n
		}
	}
	return a[n], nil
}
func (m *model) addReference(ser, num int) (*reference, error) {
	for i := 1; i < len(m.refs); i++ {
		if e := m.tick(); e != nil {
			return nil, e
		}
		r := m.refs[i]
		if r != nil && r.ser == ser && r.num == num {
			return r, nil
		}
	}
	idx := len(m.refs)
	if idx == 0 {
		idx = 1
	}
	a, e := grow(m, m.refs, idx)
	if e != nil {
		return nil, e
	}
	m.refs = a
	r := new(reference)
	r.ser = ser
	r.num = num
	a[idx] = r
	return r, nil
}
func (m *model) addTrip(t trip) (int, error) {
	for i := 1; i < len(m.trips); i++ {
		if e := m.tick(); e != nil {
			return 0, e
		}
		v := m.trips[i]
		if v != nil && v.day == t.day && v.month == t.month && v.year == t.year && v.name1 == t.name1 && v.name2 == t.name2 {
			return i, nil
		}
	}
	idx := len(m.trips)
	if idx == 0 {
		idx = 1
	}
	a, e := grow(m, m.trips, idx)
	if e != nil {
		return 0, e
	}
	m.trips = a
	a[idx] = new(trip)
	*a[idx] = t
	return idx, nil
}
func (m *model) addCode(c code) (int, error) {
	for i := 1; i < len(m.codes); i++ {
		if e := m.tick(); e != nil {
			return 0, e
		}
		v := m.codes[i]
		if v != nil && v.aziUnit == c.aziUnit && v.inclUnit == c.inclUnit && v.deltaLen == c.deltaLen && v.deltaAzi == c.deltaAzi && v.deltaIncl == c.deltaIncl && v.ratio == c.ratio && v.limit == c.limit {
			return i, nil
		}
	}
	idx := len(m.codes)
	if idx == 0 {
		idx = 1
	}
	a, e := grow(m, m.codes, idx)
	if e != nil {
		return 0, e
	}
	m.codes = a
	a[idx] = new(code)
	*a[idx] = c
	return idx, nil
}
func (m *model) addLine(s *station, l inline) error {
	for i := range s.lines {
		if e := m.tick(); e != nil {
			return e
		}
		v := s.lines[i]
		if math.Abs(float64(v.dist-l.dist)) < 0.1 && math.Abs(float64(v.az-l.az)) < 0.1 && math.Abs(float64(v.incl-l.incl)) < 0.1 {
			s.lines[i].keys = append(s.lines[i].keys, l.keys...)
			return nil
		}
	}
	s.lines = append(s.lines, l)
	return nil
}
func encode(s string) []byte {
	b := []byte{}
	for _, r := range s {
		if r <= 65535 {
			b = append(b, mac[r])
		} else {
			a, z := utf16.EncodeRune(r)
			b = append(b, mac[a], mac[z])
		}
	}
	return b
}

const special = "ÄÅÇÉÑÖÜáàâäãåçéèêëíìîïñóòôöõúùûü?°¢£§?¶ß®©?´¨?ÆØ?±??¥µ?????ªº?æø¿¡¬????«»??ÀÃÕ????????÷?ÿ??€?????·???ÂÊÁËÈÍÎÏÌÓÔ?ÒÚÛÙ???¯???¸???"

func decode(b []byte) string {
	r := []rune(special)
	var out strings.Builder
	for _, c := range b {
		if c < 128 {
			out.WriteByte(c)
		} else {
			out.WriteRune(r[int(c)-128])
		}
	}
	return out.String()
}
func units(s string) []uint16 { return utf16.Encode([]rune(s)) }
func limit(s string, n int) string {
	u := units(s)
	if len(u) <= n {
		return s
	}
	return string(utf16.Decode(u[:n]))
}
func textPointer(s string) *string { v := new(string); *v = s; return v }

func trim(s string) string {
	return strings.TrimFunc(s, func(r rune) bool { return r <= 65535 && whitespace[r] != 0 })
}

func stringFromUnits(u []uint16) string { return string(utf16.Decode(u)) }
