package top_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
	"os"
	"reflect"
	"testing"

	"pockettopo-exporter/internal/source"
	"pockettopo-exporter/internal/top"
)

func sideRecord(x, y, scale int32) []byte {
	var b []byte
	for _, v := range []int32{x, y, scale} {
		b = binary.LittleEndian.AppendUint32(b, uint32(v))
	}
	return b
}
func sideData(n int) []byte { return append(mixedData(n, 0), sideRecord(-66051, 16777217, -501)...) }
func sideOffsets(start int) source.MappingOffsets {
	return source.MappingOffsets{Record: source.Span{Start: start, End: start + 12}, X0: source.Span{Start: start, End: start + 4}, Y0: source.Span{Start: start + 4, End: start + 8}, Scale: source.Span{Start: start + 8, End: start + 12}}
}
func assertSide(t *testing.T, b []byte, p source.SideMappingPrefix) {
	t.Helper()
	plan, e := top.ReadV3PlanElementsPrefix(b)
	if e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(p.PlanElementsPrefix, plan) || p.Offsets().PlanElementsPrefixOffsets != plan.Offsets() {
		t.Fatal("lost plan metadata")
	}
	m, present := p.SideMapping()
	end := plan.ConsumedOffset()
	if plan.StopReason() == source.PlanStopTerminated {
		if !present || m.Offsets() != sideOffsets(end) || p.Offsets().Side != m.Offsets() {
			t.Fatal("side spans/presence")
		}
		for i, v := range []int32{m.X0Raw(), m.Y0Raw(), m.ScaleRaw()} {
			if v != int32(binary.LittleEndian.Uint32(b[end+4*i:])) {
				t.Fatal("raw scalar")
			}
		}
		end += 12
	} else if present || m != (source.Mapping{}) || p.Offsets().Side != (source.MappingOffsets{}) {
		t.Fatal("mapping after unsupported stop")
	}
	if p.ConsumedOffset() != end || p.UnparsedTailSize() != len(b)-end || !bytes.Equal(p.Bytes(), b[:end]) {
		t.Fatal("prefix/tail accounting")
	}
}
func TestSideMappingPrefixFieldsStopsAndCopies(t *testing.T) {
	for _, n := range []int{0, 1, 2, 3, 4, 17, 257} {
		tails := []byte{255}
		if n == 4 {
			tails = make([]byte, 256)
			for i := range tails {
				tails[i] = byte(i)
			}
		}
		for _, tail := range tails {
			b := sideData(n)
			b = append(b, tail, 3, 1, 0xff)
			before := bytes.Clone(b)
			p, e := top.ReadV3SideMappingPrefix(b)
			if e != nil {
				t.Fatal(e)
			}
			assertSide(t, b, p)
			if !bytes.Equal(before, b) {
				t.Fatal("mutated input")
			}
			raw := p.Bytes()
			raw[0] = 0
			b[0] = 0
			elems := p.Elements()
			if len(elems) > 0 {
				elems[0] = source.PlanElement{}
				if p.Elements()[0].MarkerRaw() != 1 {
					t.Fatal("element alias")
				}
			}
			offsets := p.Offsets()
			offsets.Side.Record.Start = 0
			if !bytes.Equal(p.Bytes(), before[:len(before)-4]) || p.Offsets() == offsets {
				t.Fatal("copy alias")
			}
		}
	}
	for m := 0; m < 256; m++ {
		if m == 0 || m == 1 || m == 3 {
			continue
		}
		for _, tail := range [][]byte{nil, {0}, sideRecord(1, 2, 3)} {
			b := append(mixedData(4, byte(m)), tail...)
			p, e := top.ReadV3SideMappingPrefix(b)
			if e != nil {
				t.Fatal(e)
			}
			assertSide(t, b, p)
		}
	}
}
func TestSideMappingPrefixRawBoundaries(t *testing.T) {
	for field := 0; field < 3; field++ {
		for _, v := range []int32{math.MinInt32, -16777217, -501, -11, -5, -1, 0, 1, 4, 5, 9, 10, 501, 16777217, math.MaxInt32} {
			values := [3]int32{-66051, 16777217, -501}
			values[field] = v
			b := append(mixedData(3, 0), sideRecord(values[0], values[1], values[2])...)
			p, e := top.ReadV3SideMappingPrefix(b)
			if e != nil {
				t.Fatal(e)
			}
			assertSide(t, b, p)
		}
	}
	p, e := top.ReadV3SideMappingPrefix(append(mixedData(0, 0), make([]byte, 12)...))
	m, ok := p.SideMapping()
	if e != nil || !ok || m.X0Raw() != 0 || m.Y0Raw() != 0 || m.ScaleRaw() != 0 {
		t.Fatal("zero mapping absent")
	}
}
func TestSideMappingPrefixEveryCutAndInheritedErrors(t *testing.T) {
	limits := top.DefaultPlanElementLimits()
	for _, n := range []int{0, 4, 17} {
		b := sideData(n)
		start := len(b) - 12
		for cut := 0; cut < len(b); cut++ {
			before := bytes.Clone(b[:cut])
			p, e := top.ReadV3SideMappingPrefixWithLimits(b[:cut], limits)
			if e == nil || !reflect.DeepEqual(p, source.SideMappingPrefix{}) || !bytes.Equal(before, b[:cut]) {
				t.Fatal("partial failure")
			}
			if cut < start {
				_, want := top.ReadV3PlanElementsPrefixWithLimits(b[:cut], limits)
				if !reflect.DeepEqual(e, want) {
					t.Fatal("inherited error")
				}
			} else {
				var pe *top.ParseError
				i := (cut - start) / 4
				if !errors.As(e, &pe) || pe.Code != "truncated" || pe.Field != []string{"side.mapping.x0", "side.mapping.y0", "side.mapping.scale"}[i] || pe.Offset != start+4*i {
					t.Fatalf("cut %d: %v", cut, e)
				}
			}
		}
		p, e := top.ReadV3SideMappingPrefix(b)
		if e != nil {
			t.Fatal(e)
		}
		assertSide(t, b, p)
	}
	for _, bad := range [][]byte{{0xff}, fixture(-1), referenceTable(-1), append(planMarkerBase(), 1, 255, 255, 255, 255), append(planMarkerBase(), 3)} {
		_, want := top.ReadV3PlanElementsPrefixWithLimits(bad, limits)
		p, e := top.ReadV3SideMappingPrefixWithLimits(bad, limits)
		if want == nil || !reflect.DeepEqual(want, e) || !reflect.DeepEqual(p, source.SideMappingPrefix{}) {
			t.Fatal("inherited malformed error")
		}
	}
}
func TestSideMappingPrefixLimitsAndPrecedence(t *testing.T) {
	b := sideData(4)
	defaults := top.DefaultPlanElementLimits()
	cases := []top.PlanElementLimits{}
	for i := 0; i < 8; i++ {
		for _, v := range []int{-1, math.MaxInt, 0} {
			l := defaults
			switch i {
			case 0:
				l.MaxPoints = v
			case 1:
				l.MaxElements = v
			case 2:
				l.MaxTotalPoints = v
			case 3:
				l.MaxReferences = v
			case 4:
				l.MaxMeasurements = v
			case 5:
				l.MaxTrips = v
			case 6:
				l.MaxInputBytes = v
			case 7:
				l.MaxCommentBytes = v
			}
			cases = append(cases, l)
		}
	}
	l := defaults
	l.MaxPoints = -1
	l.MaxElements = -1
	l.MaxTotalPoints = -1
	l.MaxInputBytes = -1
	cases = append(cases, l)
	for _, l := range cases {
		plan, want := top.ReadV3PlanElementsPrefixWithLimits(b, l)
		p, e := top.ReadV3SideMappingPrefixWithLimits(b, l)
		if !reflect.DeepEqual(want, e) {
			t.Fatal("limit order changed")
		}
		if want != nil {
			if !reflect.DeepEqual(p, source.SideMappingPrefix{}) {
				t.Fatal("partial limit failure")
			}
		} else if !reflect.DeepEqual(plan, p.PlanElementsPrefix) {
			t.Fatal("changed inherited plan")
		}
	}
	l = defaults
	l.MaxElements = 4
	l.MaxPoints = 1
	l.MaxTotalPoints = 2
	l.MaxInputBytes = len(b)
	if _, e := top.ReadV3SideMappingPrefixWithLimits(b, l); e != nil {
		t.Fatal(e)
	}
	l.MaxInputBytes--
	if _, e := top.ReadV3SideMappingPrefixWithLimits(b, l); e == nil {
		t.Fatal("whole input bound")
	}
	l = defaults
	l.MaxElements = 0
	l.MaxPoints = 0
	l.MaxTotalPoints = 0
	for _, m := range []byte{0, 2} {
		b := mixedData(0, m)
		if m == 0 {
			b = append(b, make([]byte, 12)...)
		}
		if _, e := top.ReadV3SideMappingPrefixWithLimits(b, l); e != nil {
			t.Fatal(e)
		}
	}
}
func TestSideMappingPrefixVariableTablesAndNativeFixtures(t *testing.T) {
	b := binary.LittleEndian.AppendUint32(fixture(1, record(17, "trip", -32768)), 1)
	b = append(b, measurementRecord(255, "shot")...)
	b = binary.LittleEndian.AppendUint32(b, 1)
	b = append(b, referenceRecord("Aą")...)
	b = append(b, planMappingRecord()...)
	b = append(b, planMappingBase()[16:]...)
	b = append(b, loopXSection(-1, 2, 0x800fffff, -2)...)
	b = append(b, 0)
	b = append(b, sideRecord(-1234, 5678, 0)...)
	p, e := top.ReadV3SideMappingPrefix(b)
	if e != nil {
		t.Fatal(e)
	}
	assertSide(t, b, p)
	raw := bytes.Clone(b)
	trips, shots, refs := p.Trips(), p.Measurements(), p.References()
	trips[0] = source.Trip{}
	shots[0] = source.Measurement{}
	refs[0] = source.Reference{}
	for i := range b {
		b[i] = 255
	}
	if !bytes.Equal(p.Bytes(), raw) || p.Trips()[0].Comment() != "trip" || p.Measurements()[0].Comment() != "shot" || p.References()[0].Comment() != "Aą" {
		t.Fatal("table alias")
	}
	for _, f := range []struct {
		name        string
		start, tail int
		x, y, scale int32
	}{{"api-references.top", 231, 5, 0, 0, 500}, {"api-drawings.top", 405, 263, 300, -400, 500}} {
		b, e := os.ReadFile("testdata/" + f.name)
		if e != nil {
			t.Fatal(e)
		}
		p, e := top.ReadV3SideMappingPrefix(b)
		if e != nil {
			t.Fatal(e)
		}
		assertSide(t, b, p)
		m, ok := p.SideMapping()
		if !ok || m.X0Raw() != f.x || m.Y0Raw() != f.y || m.ScaleRaw() != f.scale || m.Offsets() != sideOffsets(f.start) || p.UnparsedTailSize() != f.tail {
			t.Fatalf("fixture %s: %+v", f.name, m)
		}
		for _, tail := range [][]byte{nil, {255, 3, 1, 0}} {
			b = append(bytes.Clone(b[:f.start+12]), tail...)
			p, e := top.ReadV3SideMappingPrefix(b)
			if e != nil {
				t.Fatal(e)
			}
			assertSide(t, b, p)
		}
	}
}
func FuzzReadV3SideMappingPrefix(f *testing.F) {
	for _, b := range [][]byte{sideData(0), sideData(4), mixedData(3, 2), sideData(3)[:53], {255}} {
		f.Add(b)
	}
	l := top.DefaultPlanElementLimits()
	l.MaxInputBytes = 4096
	l.MaxTrips = 32
	l.MaxMeasurements = 32
	l.MaxReferences = 32
	l.MaxCommentBytes = 256
	l.MaxPoints = 32
	l.MaxElements = 32
	l.MaxTotalPoints = 128
	f.Fuzz(func(t *testing.T, b []byte) {
		before := bytes.Clone(b)
		p, e := top.ReadV3SideMappingPrefixWithLimits(b, l)
		q, again := top.ReadV3SideMappingPrefixWithLimits(b, l)
		if !bytes.Equal(before, b) || !reflect.DeepEqual(p, q) || !reflect.DeepEqual(e, again) {
			t.Fatal("mutation/determinism")
		}
		if e != nil {
			var pe *top.ParseError
			if !errors.As(e, &pe) || pe.Offset < 0 || pe.Offset > len(b) || !reflect.DeepEqual(p, source.SideMappingPrefix{}) {
				t.Fatal("invalid error")
			}
			return
		}
		assertSide(t, b, p)
	})
}
