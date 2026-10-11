package top_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
	"pockettopo-exporter/internal/source"
	"pockettopo-exporter/internal/top"
	"reflect"
	"testing"
)

func loopXSection(x, y int32, id uint32, direction int32) []byte {
	b := []byte{3}
	for _, v := range []uint32{uint32(x), uint32(y), id, uint32(direction)} {
		b = binary.LittleEndian.AppendUint32(b, v)
	}
	return b
}
func mixedData(n int, stop byte) []byte {
	b := planMarkerBase()
	for i := 0; i < n; i++ {
		if i%3 == 0 {
			b = append(b, loopPolygon([][2]int32{{int32(i), -int32(i + 1)}}, byte(i*73))...)
		} else {
			b = append(b, loopXSection(int32(i*66051), -int32(i*16777217), uint32(i)|0x80000000, int32(i-10))...)
		}
	}
	return append(b, stop)
}
func requirePlanElementsError(t *testing.T, b []byte, l top.PlanElementLimits, code, field string, offset int) {
	t.Helper()
	before := bytes.Clone(b)
	p, e := top.ReadV3PlanElementsPrefixWithLimits(b, l)
	var pe *top.ParseError
	if !errors.As(e, &pe) || pe.Code != code || pe.Field != field || pe.Offset != offset || !reflect.DeepEqual(p, source.PlanElementsPrefix{}) || !bytes.Equal(b, before) {
		t.Fatalf("failure %+v %v want %s %s %d", p, e, code, field, offset)
	}
}
func assertElements(t *testing.T, b []byte, p source.PlanElementsPrefix) {
	t.Helper()
	base, e := top.ReadV3PlanMappingPrefix(b)
	if e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(p.PlanMappingPrefix, base) || p.Offsets().PlanMappingPrefixOffsets != base.Offsets() {
		t.Fatal("inherited metadata/spans changed")
	}
	offset := base.ConsumedOffset()
	total := 0
	for i, elem := range p.Elements() {
		if elem.MarkerRaw() != b[offset] {
			t.Fatalf("element %d marker", i)
		}
		poly, isPoly := elem.Polygon()
		xs, isXS := elem.XSection()
		if b[offset] == 1 {
			if !isPoly || isXS || xs != (source.PlanXSection{}) {
				t.Fatal("Polygon variant")
			}
			n := int(int32(binary.LittleEndian.Uint32(b[offset+1:])))
			color := offset + 5 + n*8
			if poly.MarkerRaw() != 1 || int(poly.PointCountRaw()) != n || poly.ColorRaw() != b[color] || len(poly.Points()) != n || poly.Offsets() != (source.PlanPolygonOffsets{Record: source.Span{Start: offset, End: color + 1}, Marker: source.Span{Start: offset, End: offset + 1}, PointCount: source.Span{Start: offset + 1, End: offset + 5}, Points: source.Span{Start: offset + 5, End: color}, Color: source.Span{Start: color, End: color + 1}}) {
				t.Fatal("Polygon fields/spans")
			}
			for j, pt := range poly.Points() {
				s := offset + 5 + j*8
				if pt.XRaw() != int32(binary.LittleEndian.Uint32(b[s:])) || pt.YRaw() != int32(binary.LittleEndian.Uint32(b[s+4:])) || pt.Offsets() != (source.PolygonPointOffsets{Record: source.Span{Start: s, End: s + 8}, X: source.Span{Start: s, End: s + 4}, Y: source.Span{Start: s + 4, End: s + 8}}) {
					t.Fatal("Polygon point")
				}
			}
			total += n
			offset = color + 1
		} else {
			if !isXS || isPoly || !reflect.DeepEqual(poly, source.PlanPolygon{}) {
				t.Fatal("XSection variant")
			}
			want := source.PlanXSectionOffsets{Record: source.Span{Start: offset, End: offset + 17}, Marker: source.Span{Start: offset, End: offset + 1}, CenterX: source.Span{Start: offset + 1, End: offset + 5}, CenterY: source.Span{Start: offset + 5, End: offset + 9}, StationID: source.Span{Start: offset + 9, End: offset + 13}, Direction: source.Span{Start: offset + 13, End: offset + 17}}
			if xs.MarkerRaw() != 3 || xs.CenterXRaw() != int32(binary.LittleEndian.Uint32(b[offset+1:])) || xs.CenterYRaw() != int32(binary.LittleEndian.Uint32(b[offset+5:])) || xs.StationID().Raw() != binary.LittleEndian.Uint32(b[offset+9:]) || xs.DirectionRaw() != int32(binary.LittleEndian.Uint32(b[offset+13:])) || xs.Offsets() != want {
				t.Fatal("XSection fields/spans")
			}
			offset += 17
		}
	}
	reason := source.PlanStopUnsupportedElement
	if b[offset] == 0 {
		reason = source.PlanStopTerminated
	}
	if p.StopReason() != reason || p.StopMarkerRaw() != b[offset] || p.Offsets().StopMarker != (source.Span{Start: offset, End: offset + 1}) || p.TotalPoints() != total || p.ConsumedOffset() != offset+1 || p.UnparsedTailSize() != len(b)-offset-1 || !bytes.Equal(p.Bytes(), b[:offset+1]) {
		t.Fatal("stop/accounting")
	}
}
func TestPlanElementsMixedStopsAndRaw(t *testing.T) {
	for _, n := range []int{0, 1, 2, 3, 4, 17, 257} {
		for m := 0; m < 256; m++ {
			if m == 1 || m == 3 {
				continue
			}
			b := mixedData(n, byte(m))
			for _, tail := range [][]byte{nil, {255, 128}, planMappingRecord()} {
				input := append(bytes.Clone(b), tail...)
				before := bytes.Clone(input)
				p, e := top.ReadV3PlanElementsPrefix(input)
				if e != nil || len(p.Elements()) != n {
					t.Fatal(e)
				}
				assertElements(t, input, p)
				if !bytes.Equal(input, before) {
					t.Fatal("input changed")
				}
			}
		}
	}
	for _, v := range []int32{math.MinInt32, -16777217, -2, -1, 0, 66051, 65535, 65536, math.MaxInt32} {
		for _, id := range []uint32{0, 0x7fffffff, 0x80000000, 0x800fffff, 0x80100000, 0xffffffff} {
			b := append(planMarkerBase(), loopXSection(v, ^v, id, v)...)
			b = append(b, loopXSection(^v, v, ^id, ^v)...)
			b = append(b, loopPolygon(nil, 255)...)
			b = append(b, 0)
			p, e := top.ReadV3PlanElementsPrefix(b)
			if e != nil {
				t.Fatal(e)
			}
			assertElements(t, b, p)
		}
	}
	for c := 0; c < 256; c++ {
		b := append(planMarkerBase(), loopPolygon([][2]int32{{math.MinInt32, math.MaxInt32}, {66051, -16777217}}, byte(c))...)
		b = append(b, 0)
		p, e := top.ReadV3PlanElementsPrefix(b)
		if e != nil {
			t.Fatal(e)
		}
		assertElements(t, b, p)
	}
}
func TestPlanElementsTruncation(t *testing.T) {
	b := mixedData(7, 0)
	for n := 0; n < len(b); n++ {
		if n < 40 {
			_, want := top.ReadV3PlanMappingPrefix(b[:n])
			p, e := top.ReadV3PlanElementsPrefix(b[:n])
			if want == nil || !reflect.DeepEqual(e, want) || !reflect.DeepEqual(p, source.PlanElementsPrefix{}) {
				t.Fatal("inherited failure", n)
			}
			continue
		}
		offset := 40
		for i := 0; ; i++ {
			field := fmt.Sprintf("plan.elements[%d]", i)
			if n <= offset {
				requirePlanElementsError(t, b[:n], top.DefaultPlanElementLimits(), "truncated", field+".kind", offset)
				break
			}
			if b[offset] == 3 {
				failed := false
				for j, f := range []string{".center.x", ".center.y", ".station_id", ".direction"} {
					start := offset + 1 + j*4
					if n < start+4 {
						requirePlanElementsError(t, b[:n], top.DefaultPlanElementLimits(), "truncated", field+f, start)
						failed = true
						break
					}
				}
				if failed {
					break
				}
				offset += 17
				continue
			}
			count := int(binary.LittleEndian.Uint32(b[offset+1:]))
			color := offset + 5 + count*8
			if n < offset+5 {
				requirePlanElementsError(t, b[:n], top.DefaultPlanElementLimits(), "truncated", field+".point_count", offset+1)
				break
			}
			if n < color {
				requirePlanElementsError(t, b[:n], top.DefaultPlanElementLimits(), "truncated", field+".points", offset+5)
				break
			}
			if n == color {
				requirePlanElementsError(t, b[:n], top.DefaultPlanElementLimits(), "truncated", field+".color", color)
				break
			}
			offset = color + 1
		}
	}
	for _, v := range []int32{-1, math.MinInt32, 1000001, math.MaxInt32} {
		b := append(mixedData(5, 1), binary.LittleEndian.AppendUint32(nil, uint32(v))...)
		code := "resource_limit"
		if v < 0 {
			code = "negative_count"
		}
		requirePlanElementsError(t, b, top.DefaultPlanElementLimits(), code, "plan.elements[5].point_count", len(b)-4)
	}
}
func TestPlanElementsLimits(t *testing.T) {
	d := top.DefaultPlanElementLimits()
	if top.PlanPolygonLimits(d) != top.DefaultPlanPolygonLimits() {
		t.Fatal("defaults")
	}
	for _, f := range []string{"limits.max_points", "limits.max_elements", "limits.max_total_points"} {
		max := 1000000
		if f == "limits.max_elements" {
			max = 100000
		}
		for _, v := range []int{-1, max + 1} {
			l := d
			switch f {
			case "limits.max_points":
				l.MaxPoints = v
			case "limits.max_elements":
				l.MaxElements = v
			default:
				l.MaxTotalPoints = v
			}
			requirePlanElementsError(t, nil, l, "invalid_limit", f, 0)
		}
	}
	// Simultaneously invalid limits pin the documented validation order, before
	// any input/header error or inherited bound validation.
	for _, c := range []struct {
		points, elements, total int
		field                   string
	}{
		{-1, -1, -1, "limits.max_points"},
		{0, -1, -1, "limits.max_elements"},
		{0, 0, -1, "limits.max_total_points"},
	} {
		l := d
		l.MaxPoints = c.points
		l.MaxElements = c.elements
		l.MaxTotalPoints = c.total
		l.MaxTrips = -1
		requirePlanElementsError(t, nil, l, "invalid_limit", c.field, 0)
	}
	l := d
	l.MaxElements = 0
	l.MaxTotalPoints = 0
	l.MaxPoints = 0
	for _, m := range []byte{0, 2, 255} {
		b := mixedData(0, m)
		p, e := top.ReadV3PlanElementsPrefixWithLimits(b, l)
		if e != nil {
			t.Fatal(e)
		}
		assertElements(t, b, p)
	}
	for _, m := range []byte{1, 3} {
		requirePlanElementsError(t, append(planMarkerBase(), m), l, "resource_limit", "plan.elements[0].kind", 40)
	}
	l.MaxElements = 2
	b := append(planMarkerBase(), loopXSection(1, 2, 3, 4)...)
	b = append(b, loopXSection(-1, -2, 0xffffffff, -1)...)
	b = append(b, 0)
	p, e := top.ReadV3PlanElementsPrefixWithLimits(b, l)
	if e != nil {
		t.Fatal(e)
	}
	assertElements(t, b, p)
	b[len(b)-1] = 3
	requirePlanElementsError(t, b, l, "resource_limit", "plan.elements[2].kind", 74)
	l = d
	l.MaxElements = 3
	l.MaxPoints = 1
	l.MaxTotalPoints = 2
	b = mixedData(4, 0)
	requirePlanElementsError(t, b, l, "resource_limit", "plan.elements[3].kind", 88)
	l.MaxElements = 4
	p, e = top.ReadV3PlanElementsPrefixWithLimits(b, l)
	if e != nil {
		t.Fatal(e)
	}
	assertElements(t, b, p)
	l.MaxTotalPoints = 1
	requirePlanElementsError(t, b, l, "resource_limit", "plan.elements[3].point_count", 89)
	l = d
	l.MaxPoints = 0
	requirePlanElementsError(t, b, l, "resource_limit", "plan.elements[0].point_count", 41)
	l = d
	l.MaxTotalPoints = 0
	requirePlanElementsError(t, planPolygonCountData(1), l, "resource_limit", "plan.elements[0].point_count", 41)
	requirePlanElementsError(t, planPolygonCountData(-1), l, "negative_count", "plan.elements[0].point_count", 41)
	for _, edit := range []func(*top.PlanElementLimits){func(l *top.PlanElementLimits) { l.MaxInputBytes = len(b) - 1 }, func(l *top.PlanElementLimits) { l.MaxTrips = -1 }, func(l *top.PlanElementLimits) { l.MaxMeasurements = -1 }, func(l *top.PlanElementLimits) { l.MaxReferences = -1 }} {
		l = d
		edit(&l)
		_, want := top.ReadV3PlanMappingPrefixWithLimits(b, l.ReferenceLimits)
		p, e := top.ReadV3PlanElementsPrefixWithLimits(b, l)
		if want == nil || !reflect.DeepEqual(e, want) || !reflect.DeepEqual(p, source.PlanElementsPrefix{}) {
			t.Fatal("inherited bound")
		}
	}
}
func TestPlanElementsNativeAndCopies(t *testing.T) {
	b, e := os.ReadFile("testdata/api-drawings.top")
	if e != nil {
		t.Fatal(e)
	}
	p, e := top.ReadV3PlanElementsPrefix(b)
	if e != nil {
		t.Fatal(e)
	}
	assertElements(t, b, p)
	if len(p.Elements()) != 10 || p.TotalPoints() != 22 || p.StopReason() != source.PlanStopTerminated || p.ConsumedOffset() != 405 {
		t.Fatalf("native accounting %d %d", len(p.Elements()), p.ConsumedOffset())
	}
	old, e := top.ReadV3PlanPolygonsPrefix(b)
	if e != nil || old.ConsumedOffset() != 371 || old.StopMarkerRaw() != 3 {
		t.Fatal("old contract")
	}
	saved := p.Bytes()
	elems := p.Elements()
	elems[0] = source.PlanElement{}
	out := p.Bytes()
	out[0] = 0
	b[0] = 0
	poly, _ := p.Elements()[0].Polygon()
	pts := poly.Points()
	pts[0] = source.PolygonPoint{}
	if !bytes.Equal(saved, p.Bytes()) || p.Elements()[0].MarkerRaw() != 1 || poly.Points()[0].XRaw() != -6000 {
		t.Fatal("aliases")
	}
}
func FuzzReadV3PlanElementsPrefix(f *testing.F) {
	for _, n := range []int{0, 1, 2, 4, 17} {
		for _, m := range []byte{0, 2, 255} {
			f.Add(mixedData(n, m))
		}
	}
	f.Add(planPolygonCountData(-1))
	f.Fuzz(func(t *testing.T, b []byte) {
		if len(b) > 4096 {
			return
		}
		before := bytes.Clone(b)
		l := top.DefaultPlanElementLimits()
		l.MaxInputBytes = 4096
		l.MaxTrips = 32
		l.MaxMeasurements = 32
		l.MaxReferences = 32
		l.MaxCommentBytes = 256
		l.MaxPoints = 32
		l.MaxElements = 32
		l.MaxTotalPoints = 128
		p, e := top.ReadV3PlanElementsPrefixWithLimits(b, l)
		q, e2 := top.ReadV3PlanElementsPrefixWithLimits(b, l)
		if !bytes.Equal(b, before) || !reflect.DeepEqual(p, q) || !reflect.DeepEqual(e, e2) {
			t.Fatal("mutation/determinism")
		}
		if e != nil {
			if !reflect.DeepEqual(p, source.PlanElementsPrefix{}) {
				t.Fatal("partial failure")
			}
			return
		}
		assertElements(t, b, p)
		if len(p.Elements()) > 32 || p.TotalPoints() > 128 {
			t.Fatal("budgets")
		}
	})
}
