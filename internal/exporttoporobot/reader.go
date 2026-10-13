package exporttoporobot

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Read keeps native fixed UTF-16 offsets and sequential station insertion.
// Resource/nonfinite failures are explicit before any output publication.
func read(b []byte) (*model, error) {
	return readBudget(b, 10_000_000, 32*1024*1024)
}
func readBudget(b []byte, workLimit, byteLimit int) (*model, error) {
	if len(b) > byteLimit {
		return nil, fmt.Errorf("resource_limit (Toporobot merge bytes)")
	}
	text := decode(b)
	switch {
	case bytes.HasPrefix(b, []byte{0xef, 0xbb, 0xbf}):
		text = strings.ToValidUTF8(string(b[3:]), "�")
	case bytes.HasPrefix(b, []byte{0, 0, 0xfe, 0xff}):
		return nil, fmt.Errorf("native_failure (Toporobot UTF-32BE BOM)")
	case bytes.HasPrefix(b, []byte{0xff, 0xfe, 0, 0}):
		if (len(b)-4)%4 != 0 {
			return nil, fmt.Errorf("invalid incomplete Toporobot UTF-32")
		}
		order := binary.ByteOrder(binary.LittleEndian)
		var s strings.Builder
		for i := 4; i+3 < len(b); i += 4 {
			r := rune(order.Uint32(b[i:]))
			if !utf8.ValidRune(r) {
				r = utf8.RuneError
			}
			s.WriteRune(r)
		}
		text = s.String()
	case bytes.HasPrefix(b, []byte{0xff, 0xfe}) || bytes.HasPrefix(b, []byte{0xfe, 0xff}):
		if (len(b)-2)%2 != 0 {
			return nil, fmt.Errorf("invalid incomplete Toporobot UTF-16")
		}
		order := binary.ByteOrder(binary.LittleEndian)
		if b[0] == 0xfe {
			order = binary.BigEndian
		}
		u := []uint16{}
		for i := 2; i+1 < len(b); i += 2 {
			u = append(u, order.Uint16(b[i:]))
		}
		text = stringFromUnits(u)
	}
	m := newModel()
	m.maxWork = workLimit
	var actual any
	lines := strings.Split(strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n"), "\n")
	for line, s := range lines {
		if s == "" {
			continue
		}
		if e := m.tick(); e != nil {
			return nil, e
		}
		var err error
		u := units(s)
		field := func(start, n int) string {
			if start < 0 || n < 0 || start > len(u) || n > len(u)-start {
				err = fmt.Errorf("truncated fixed field")
				return ""
			}
			return stringFromUnits(u[start : start+n])
		}
		integer := func(start, n int) int {
			v, e := strconv.Atoi(strings.TrimSpace(field(start, n)))
			if e != nil {
				err = e
			}
			return v
		}
		number := func(start, n, bits int) float64 {
			v, e := parseNumber(field(start, n), bits)
			if e != nil || math.IsNaN(v) || math.IsInf(v, 0) {
				err = fmt.Errorf("invalid finite number")
			}
			return v
		}
		ser, num, codeIndex, tripIndex := integer(1, 5), integer(6, 6), integer(12, 4), integer(20, 4)
		owner := fmt.Sprintf("existing:line:%d", line+1)
		if err != nil {
			return nil, fmt.Errorf("invalid Toporobot line %d: %w", line+1, err)
		}
		if s[0] == '(' {
			if len(units(s)) < 26 || actual == nil {
				continue
			}
			body := field(25, len(units(s))-25)
			sep := "\r\n"
			if field(24, 1) == "-" {
				sep = ""
			}
			addOwner(actual, owner)
			var comment *string
			var setComment func(*string)
			switch v := actual.(type) {
			case *station:
				if strings.HasPrefix(body, "[") && strings.HasSuffix(body, "]") {
					fields := strings.SplitN(body[1:len(body)-1], " ", 3)
					if len(fields) != 3 {
						return nil, fmt.Errorf("invalid Toporobot inline line %d", line+1)
					}
					vals := [3]float32{}
					for i, f := range fields {
						n, e := parseNumber(f, 32)
						if e != nil || math.IsNaN(n) || math.IsInf(n, 0) {
							return nil, fmt.Errorf("invalid Toporobot inline number")
						}
						vals[i] = float32(n)
					}
					v.lines = append(v.lines, inline{dist: vals[0], az: vals[1], incl: vals[2], keys: []string{owner}})
					continue
				}
				comment = v.comment
				setComment = func(value *string) { v.comment = value }
			case *series:
				comment = v.comment
				setComment = func(value *string) { v.comment = value }
			case *code:
				comment = v.comment
				setComment = func(value *string) { v.comment = value }
			case *reference:
				comment = v.comment
				setComment = func(value *string) { v.comment = value }
			case *trip:
				if v.comment == nil && sep == "" && (strings.HasSuffix(body, "1") || strings.HasSuffix(body, "2")) {
					m.work += len(units(v.name1)) + len(units(v.name2)) + len(units(body))
					if e := m.tick(); e != nil {
						return nil, e
					}
					part := strings.TrimSuffix(body[:len(body)-1], "¸")
					if strings.HasSuffix(body, "1") {
						v.name1 = strings.Join([]string{v.name1, part}, "")
					} else {
						v.name2 = strings.Join([]string{v.name2, part}, "")
					}
					continue
				}
				comment = v.comment
				setComment = func(value *string) { v.comment = value }
			}
			if setComment != nil {
				copied := len(units(body)) + len(units(sep))
				if comment != nil {
					copied += len(units(*comment))
				}
				m.work += copied
				if e := m.tick(); e != nil {
					return nil, e
				}
				if comment == nil {
					setComment(textPointer(body))
				} else {
					setComment(textPointer(strings.Join([]string{*comment, sep, body}, "")))
				}
			}
			continue
		}
		switch -ser {
		case 6:
			a, e := grow(m, m.refs, num)
			if e != nil {
				return nil, e
			}
			m.refs = a
			r := new(reference)
			if len(units(s)) >= 26 {
				r.name = textPointer(field(25, len(units(s))-25))
			}
			r.keys = []string{owner}
			a[num] = r
			actual = nil
		case 5:
			if num < 0 || num >= len(m.refs) || m.refs[num] == nil {
				return nil, fmt.Errorf("missing Toporobot reference")
			}
			r := m.refs[num]
			r.e = number(24, 12, 64)
			r.n = number(36, 12, 64)
			r.h = number(48, 12, 64)
			r.ser = integer(60, 6)
			r.num = integer(66, 6)
			actual = r
		case 4, 3:
			actual = nil
		case 2:
			a, e := grow(m, m.trips, num)
			if e != nil {
				return nil, e
			}
			m.trips = a
			t := new(trip)
			t.day = integer(25, 2)
			if field(27, 1) == "/" {
				t.month = integer(28, 2)
				t.year = integer(31, 2) + 1900
				if t.year < 1920 {
					t.year += 100
				}
			} else {
				t.month = integer(27, 2)
				t.year = integer(29, 4)
			}
			t.name1 = field(35, 13)
			t.name2 = field(49, 13)
			t.auto = integer(62, 2) != 0
			t.azi = float32(number(64, 8, 32))
			t.incl = float32(integer(72, 4)) / 10
			t.color = integer(76, 4)
			a[num] = t
			actual = t
		case 1:
			a, e := grow(m, m.codes, num)
			if e != nil {
				return nil, e
			}
			m.codes = a
			c := new(code)
			c.aziUnit = int(math.RoundToEven(number(24, 8, 64)))
			c.inclUnit = int(math.RoundToEven(number(32, 8, 64)))
			if c.aziUnit < math.MinInt32 || c.aziUnit > math.MaxInt32 || c.inclUnit < math.MinInt32 || c.inclUnit > math.MaxInt32 {
				return nil, fmt.Errorf("invalid Toporobot Int32 angular unit")
			}
			c.deltaLen = float32(number(40, 8, 32))
			c.deltaAzi = float32(number(48, 8, 32))
			c.deltaIncl = float32(number(56, 8, 32))
			c.ratio = float32(number(64, 8, 32))
			c.limit = float32(number(72, 8, 32))
			a[num] = c
			actual = c
		default:
			if ser < 0 {
				return nil, fmt.Errorf("native_failure (negative Toporobot series)")
			}
			if num == -2 {
				a, e := grow(m, m.series, ser)
				if e != nil {
					return nil, e
				}
				m.series = a
				v := new(series)
				if len(units(s)) >= 26 {
					v.name = textPointer(field(25, len(units(s))-25))
				}
				a[ser] = v
				v.keys = []string{owner}
				actual = nil
			} else {
				if ser >= len(m.series) || m.series[ser] == nil {
					return nil, fmt.Errorf("missing Toporobot series")
				}
				v := m.series[ser]
				if num == -1 {
					v.startSer = integer(24, 8)
					v.startNum = integer(32, 8)
					v.endSer = integer(40, 8)
					v.endNum = integer(48, 8)
					v.length = integer(56, 8)
					v.chance = integer(64, 8)
					v.obst = integer(72, 8)
					if v.length < -1 || v.length > 1000000 {
						return nil, fmt.Errorf("resource_limit (Toporobot series length)")
					}
					v.stations = []*station{}
					actual = v
				} else {
					t := new(station)
					t.trip = tripIndex
					t.code = codeIndex
					t.dist = float32(number(24, 8, 32))
					t.az = float32(number(32, 8, 32))
					t.incl = float32(number(40, 8, 32))
					t.left = float32(number(48, 8, 32))
					t.right = float32(number(56, 8, 32))
					t.up = float32(number(64, 8, 32))
					t.down = float32(number(72, 8, 32))
					a, e := grow(m, v.stations, len(v.stations))
					if e != nil {
						return nil, e
					}
					v.stations = a
					a[len(a)-1] = t
					actual = t
				}
			}
		}
		if err != nil {
			return nil, fmt.Errorf("invalid Toporobot line %d: %w", line+1, err)
		}

		addOwner(actual, owner)
	}
	for i := 1; i < len(m.trips); i++ {
		t := m.trips[i]
		if t != nil {
			t.name1 = trim(t.name1)
			t.name2 = trim(t.name2)
		}
	}
	return m, nil
}
func addOwner(actual any, owner string) {
	switch v := actual.(type) {
	case *reference:
		v.keys = append(v.keys, owner)
	case *trip:
		v.keys = append(v.keys, owner)
	case *code:
		v.keys = append(v.keys, owner)
	case *series:
		v.keys = append(v.keys, owner)
	case *station:
		v.keys = append(v.keys, owner)
	}
}

func parseNumber(s string, bits int) (float64, error) {
	s = strings.Trim(s, " \t\r\n\v\f")
	integerPart, digitSeen := true, false
	for _, r := range s {
		if r >= '0' && r <= '9' {
			digitSeen = true
		}
		if r == ',' && (!integerPart || !digitSeen) {
			return 0, fmt.Errorf("invalid Toporobot decimal separator")
		}
		if r == '.' || r == 'e' || r == 'E' {
			integerPart = false
		}
		if !(r >= '0' && r <= '9') && r != '+' && r != '-' && r != '.' && r != 'e' && r != 'E' && r != ',' {
			return 0, fmt.Errorf("invalid Toporobot decimal number")
		}
	}
	return strconv.ParseFloat(strings.ReplaceAll(s, ",", ""), bits)
}
