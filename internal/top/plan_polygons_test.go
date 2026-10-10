package top_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
	"reflect"
	"testing"

	"pockettopo-exporter/internal/source"
	"pockettopo-exporter/internal/top"
)

func loopPolygon(points [][2]int32, color byte) []byte {
	b := []byte{1}
	b = binary.LittleEndian.AppendUint32(b, uint32(len(points)))
	b = append(b, polygonPointBytes(points)...)
	return append(b, color)
}
func loopData(n int, stop byte) []byte {
	b := planMarkerBase()
	for i := 0; i < n; i++ {
		table := make([][2]int32, i%4)
		for j := range table {
			table[j] = [2]int32{int32(i*66051 + j), -int32(i*16777217 + j)}
		}
		b = append(b, loopPolygon(table, byte(i*73))...)
	}
	return append(b, stop)
}
func requirePlanPolygonsError(t *testing.T, b []byte, limits top.PlanPolygonLimits, code, field string, offset int) {
	t.Helper()
	before := bytes.Clone(b)
	p, err := top.ReadV3PlanPolygonsPrefixWithLimits(b, limits)
	var pe *top.ParseError
	if !errors.As(err, &pe) || pe.Code != code || pe.Field != field || pe.Offset != offset || !reflect.DeepEqual(p, source.PlanPolygonsPrefix{}) || !bytes.Equal(b, before) {
		t.Fatalf("wrong failure: %+v %v; want %s %s %d", p, err, code, field, offset)
	}
}
func assertLoop(t *testing.T, b []byte, p source.PlanPolygonsPrefix) {
	t.Helper()
	base, err := top.ReadV3PlanMappingPrefix(b)
	if err != nil {
		t.Fatal(err)
	}
	if p.Header() != base.Header() || p.Version() != base.Version() || p.TripCountRaw() != base.TripCountRaw() || p.MeasurementCountRaw() != base.MeasurementCountRaw() || p.ReferenceCountRaw() != base.ReferenceCountRaw() || !reflect.DeepEqual(p.Trips(), base.Trips()) || !reflect.DeepEqual(p.Measurements(), base.Measurements()) || !reflect.DeepEqual(p.References(), base.References()) || p.OverviewMapping() != base.OverviewMapping() || p.PlanMapping() != base.PlanMapping() || p.Offsets().PlanMappingPrefixOffsets != base.Offsets() {
		t.Fatal("inherited values/spans changed")
	}
	offset := base.ConsumedOffset()
	total := 0
	for i, poly := range p.Polygons() {
		count := int32(binary.LittleEndian.Uint32(b[offset+1:]))
		colorOffset := offset + 5 + int(count)*8
		want := source.PlanPolygonOffsets{Record: source.Span{Start: offset, End: colorOffset + 1}, Marker: source.Span{Start: offset, End: offset + 1}, PointCount: source.Span{Start: offset + 1, End: offset + 5}, Points: source.Span{Start: offset + 5, End: colorOffset}, Color: source.Span{Start: colorOffset, End: colorOffset + 1}}
		if poly.MarkerRaw() != 1 || poly.PointCountRaw() != count || poly.ColorRaw() != b[colorOffset] || poly.Offsets() != want || len(poly.Points()) != int(count) {
			t.Fatalf("polygon %d values/spans", i)
		}
		for j, pt := range poly.Points() {
			start := offset + 5 + j*8
			if pt.XRaw() != int32(binary.LittleEndian.Uint32(b[start:])) || pt.YRaw() != int32(binary.LittleEndian.Uint32(b[start+4:])) || pt.Offsets() != (source.PolygonPointOffsets{Record: source.Span{Start: start, End: start + 8}, X: source.Span{Start: start, End: start + 4}, Y: source.Span{Start: start + 4, End: start + 8}}) {
				t.Fatal("point raw/order/span changed")
			}
		}
		total += int(count)
		offset = colorOffset + 1
	}
	reason := source.PlanStopUnsupportedElement
	if b[offset] == 0 {
		reason = source.PlanStopTerminated
	}
	if p.StopReason() != reason || p.StopMarkerRaw() != b[offset] || p.Offsets().StopMarker != (source.Span{Start: offset, End: offset + 1}) || p.TotalPoints() != total || p.ConsumedOffset() != offset+1 || p.UnparsedTailSize() != len(b)-offset-1 || !bytes.Equal(p.Bytes(), b[:offset+1]) {
		t.Fatal("stop/accounting changed")
	}
}
func TestPlanPolygonsEmptyManyAndStops(t *testing.T) {
	for _, n := range []int{0, 1, 2, 3, 4, 17, 257} {
		for marker := 0; marker < 256; marker++ {
			if marker == 1 {
				continue
			}
			prefix := loopData(n, byte(marker))
			for _, tail := range [][]byte{nil, {0}, {1, 255, 128}, planMappingRecord()} {
				b := append(bytes.Clone(prefix), tail...)
				before := bytes.Clone(b)
				p, err := top.ReadV3PlanPolygonsPrefix(b)
				if err != nil || len(p.Polygons()) != n {
					t.Fatal("loop failed", err)
				}
				assertLoop(t, b, p)
				if !bytes.Equal(b, before) {
					t.Fatal("input mutated")
				}
			}
		}
	}
	// Independent endian/sign values and all raw colors, including zero-point color.
	for c := 0; c < 256; c++ {
		b := append(planMarkerBase(), loopPolygon([][2]int32{{math.MinInt32, math.MaxInt32}, {-1, 0}, {66051, -16777217}}, byte(c))...)
		b = append(b, loopPolygon(nil, byte(c))...)
		b = append(b, 0)
		p, e := top.ReadV3PlanPolygonsPrefix(b)
		if e != nil {
			t.Fatal(e)
		}
		assertLoop(t, b, p)
	}
}
func TestPlanPolygonsTruncation(t *testing.T) {
	b := loopData(7, 0)
	for n := 0; n < len(b); n++ {
		if n < 40 {
			_, want := top.ReadV3PlanMappingPrefix(b[:n])
			p, e := top.ReadV3PlanPolygonsPrefix(b[:n])
			if want == nil || !reflect.DeepEqual(e, want) || !reflect.DeepEqual(p, source.PlanPolygonsPrefix{}) {
				t.Fatalf("base failure %d", n)
			}
			continue
		}
		offset := 40
		for i := 0; ; i++ {
			field := fmt.Sprintf("plan.elements[%d]", i)
			if n <= offset {
				requirePlanPolygonsError(t, b[:n], top.DefaultPlanPolygonLimits(), "truncated", field+".kind", offset)
				break
			}
			count := int(binary.LittleEndian.Uint32(b[offset+1:]))
			if n < offset+5 {
				requirePlanPolygonsError(t, b[:n], top.DefaultPlanPolygonLimits(), "truncated", field+".point_count", offset+1)
				break
			}
			color := offset + 5 + count*8
			if n < color {
				requirePlanPolygonsError(t, b[:n], top.DefaultPlanPolygonLimits(), "truncated", field+".points", offset+5)
				break
			}
			if n == color {
				requirePlanPolygonsError(t, b[:n], top.DefaultPlanPolygonLimits(), "truncated", field+".color", color)
				break
			}
			offset = color + 1
		}
	}
	for _, n := range []int32{-1, math.MinInt32, 1000001, math.MaxInt32} {
		b := loopData(5, 1)
		start := len(b)
		b = binary.LittleEndian.AppendUint32(b, uint32(n))
		code := "resource_limit"
		if n < 0 {
			code = "negative_count"
		}
		requirePlanPolygonsError(t, b, top.DefaultPlanPolygonLimits(), code, "plan.elements[5].point_count", start)
	}
}
func TestPlanPolygonsLimitsAndPrecedence(t *testing.T) {
	d := top.DefaultPlanPolygonLimits()
	if d.MaxElements != 100000 || d.MaxTotalPoints != 1000000 || d.PolygonCountLimits != top.DefaultPolygonCountLimits() {
		t.Fatal("defaults changed")
	}
	for _, field := range []string{"limits.max_points", "limits.max_elements", "limits.max_total_points"} {
		maximum := 1000000
		if field == "limits.max_elements" {
			maximum = 100000
		}
		for _, v := range []int{-1, maximum + 1} {
			l := d
			switch field {
			case "limits.max_points":
				l.MaxPoints = v
			case "limits.max_elements":
				l.MaxElements = v
			case "limits.max_total_points":
				l.MaxTotalPoints = v
			}
			requirePlanPolygonsError(t, nil, l, "invalid_limit", field, 0)
		}
	}
	l := d
	l.MaxElements = 0
	l.MaxTotalPoints = 0
	l.MaxPoints = 0
	for _, stop := range []byte{0, 3, 255} {
		p, e := top.ReadV3PlanPolygonsPrefixWithLimits(loopData(0, stop), l)
		if e != nil {
			t.Fatal(e)
		}
		assertLoop(t, loopData(0, stop), p)
	}
	requirePlanPolygonsError(t, append(planMarkerBase(), 1), l, "resource_limit", "plan.elements[0].kind", 40)
	l.MaxElements = 2
	b := append(planMarkerBase(), loopPolygon(nil, 255)...)
	b = append(b, loopPolygon(nil, 128)...)
	b = append(b, 0)
	p, e := top.ReadV3PlanPolygonsPrefixWithLimits(b, l)
	if e != nil {
		t.Fatal(e)
	}
	assertLoop(t, b, p)
	b[len(b)-1] = 1
	requirePlanPolygonsError(t, b, l, "resource_limit", "plan.elements[2].kind", 52)
	l = d
	l.MaxElements = 2
	l.MaxPoints = 3
	l.MaxTotalPoints = 4
	b = append(planMarkerBase(), loopPolygon([][2]int32{{1, 2}, {3, 4}, {5, 6}}, 129)...)
	b = append(b, loopPolygon([][2]int32{{-1, -2}}, 255)...)
	b = append(b, 0)
	p, e = top.ReadV3PlanPolygonsPrefixWithLimits(b, l)
	if e != nil {
		t.Fatal(e)
	}
	assertLoop(t, b, p)
	l.MaxElements = 1
	requirePlanPolygonsError(t, b, l, "resource_limit", "plan.elements[1].kind", 70)
	l.MaxElements = 2
	l.MaxTotalPoints = 3
	requirePlanPolygonsError(t, b, l, "resource_limit", "plan.elements[1].point_count", 71)
	l.MaxTotalPoints = 4
	l.MaxPoints = 2
	requirePlanPolygonsError(t, b, l, "resource_limit", "plan.elements[0].point_count", 41)
	// Total budget rejects before checking available points; negative count wins.
	l = d
	l.MaxTotalPoints = 0
	requirePlanPolygonsError(t, planPolygonCountData(1), l, "resource_limit", "plan.elements[0].point_count", 41)
	requirePlanPolygonsError(t, planPolygonCountData(-1), l, "negative_count", "plan.elements[0].point_count", 41)
	l = d
	l.MaxInputBytes = len(b) - 1
	requirePlanPolygonsError(t, b, l, "resource_limit", "input", 0)
	// Each inherited bound/error is delegated without replacing user limits.
	for _, edit := range []func(*top.PlanPolygonLimits){func(l *top.PlanPolygonLimits) { l.MaxTrips = -1 }, func(l *top.PlanPolygonLimits) { l.MaxMeasurements = -1 }, func(l *top.PlanPolygonLimits) { l.MaxReferences = -1 }} {
		l = d
		edit(&l)
		_, want := top.ReadV3PlanMappingPrefixWithLimits(b, l.ReferenceLimits)
		p, e := top.ReadV3PlanPolygonsPrefixWithLimits(b, l)
		if !reflect.DeepEqual(e, want) || !reflect.DeepEqual(p, source.PlanPolygonsPrefix{}) {
			t.Fatal("inherited error changed")
		}
	}
}
func TestPlanPolygonsNativeAndCopies(t *testing.T) {
	b, e := os.ReadFile("testdata/api-drawings.top")
	if e != nil {
		t.Fatal(e)
	}
	p, e := top.ReadV3PlanPolygonsPrefix(b)
	if e != nil {
		t.Fatal(e)
	}
	assertLoop(t, b, p)
	// Pinned helper emits eight Polygons, then native XSection marker 3.
	if len(p.Polygons()) != 8 || p.StopReason() != source.PlanStopUnsupportedElement || p.ConsumedOffset() != 371 || p.UnparsedTailSize() != 309 || p.TotalPoints() != 22 {
		t.Fatalf("native loop %+v", p)
	}
	saved := p.Bytes()
	polys := p.Polygons()
	pts := polys[0].Points()
	pts[0] = source.PolygonPoint{}
	polys[0] = source.PlanPolygon{}
	b[0] = 0
	out := p.Bytes()
	out[0] = 0
	if !bytes.Equal(p.Bytes(), saved) || p.Polygons()[0].Points()[0].XRaw() != -6000 || p.Polygons()[0].MarkerRaw() != 1 {
		t.Fatal("output/input alias")
	}
}
func FuzzReadV3PlanPolygonsPrefix(f *testing.F) {
	for _, n := range []int{0, 1, 4, 17} {
		for _, stop := range []byte{0, 3, 255} {
			f.Add(loopData(n, stop))
		}
	}
	f.Add(planPolygonCountData(-1))
	f.Fuzz(func(t *testing.T, b []byte) {
		if len(b) > 4096 {
			return
		}
		before := bytes.Clone(b)
		l := top.DefaultPlanPolygonLimits()
		l.MaxInputBytes = 4096
		l.MaxTrips = 32
		l.MaxMeasurements = 32
		l.MaxReferences = 32
		l.MaxCommentBytes = 256
		l.MaxPoints = 32
		l.MaxElements = 32
		l.MaxTotalPoints = 128
		p, e := top.ReadV3PlanPolygonsPrefixWithLimits(b, l)
		q, e2 := top.ReadV3PlanPolygonsPrefixWithLimits(b, l)
		if !bytes.Equal(b, before) || !reflect.DeepEqual(p, q) || !reflect.DeepEqual(e, e2) {
			t.Fatal("mutation/nondeterminism")
		}
		if e != nil {
			if !reflect.DeepEqual(p, source.PlanPolygonsPrefix{}) {
				t.Fatal("partial failure")
			}
			return
		}
		assertLoop(t, b, p)
		if len(p.Polygons()) > 32 || p.TotalPoints() > 128 {
			t.Fatal("budget exceeded")
		}
	})
}
