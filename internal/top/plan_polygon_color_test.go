package top_test

import (
	"bytes"
	"crypto/sha256"
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

func requirePlanPolygonColorError(t *testing.T, data []byte, limits top.PolygonCountLimits, code, field string, offset int) {
	t.Helper()
	before := bytes.Clone(data)
	p, err := top.ReadV3PlanPolygonColorPrefixWithLimits(data, limits)
	var pe *top.ParseError
	if !errors.As(err, &pe) || pe.Code != code || pe.Field != field || pe.Offset != offset || !reflect.DeepEqual(p, source.PlanPolygonColorPrefix{}) || !bytes.Equal(before, data) {
		t.Fatalf("wrong/partial failure: %+v %v", p, err)
	}
}

func assertColorBase(t *testing.T, p source.PlanPolygonColorPrefix, base source.PlanPolygonPointsPrefix) {
	t.Helper()
	if p.Header() != base.Header() || p.Version() != base.Version() || p.TripCountRaw() != base.TripCountRaw() || p.MeasurementCountRaw() != base.MeasurementCountRaw() || p.ReferenceCountRaw() != base.ReferenceCountRaw() || p.MarkerRaw() != base.MarkerRaw() || p.PointCountRaw() != base.PointCountRaw() || p.OverviewMapping() != base.OverviewMapping() || p.PlanMapping() != base.PlanMapping() || p.Offsets().PlanPolygonPointsPrefixOffsets != base.Offsets() || !reflect.DeepEqual(p.Trips(), base.Trips()) || !reflect.DeepEqual(p.Measurements(), base.Measurements()) || !reflect.DeepEqual(p.References(), base.References()) || !reflect.DeepEqual(p.Points(), base.Points()) {
		t.Fatal("inherited raw records/mappings/spans changed")
	}
}

func TestPlanPolygonColorPrefixAllValuesAndTails(t *testing.T) {
	for _, want := range [][][2]int32{nil, {{66051, -66051}}, {{math.MinInt32, math.MaxInt32}, {-1, 0}, {16777217, -16777217}}} {
		points := append(planPolygonCountData(int32(len(want))), polygonPointBytes(want)...)
		for color := 0; color < 256; color++ {
			prefix := append(bytes.Clone(points), byte(color))
			for _, tail := range [][]byte{nil, {0}, {1}, {3}, {255, 128}, planMappingRecord()} {
				data := append(bytes.Clone(prefix), tail...)
				before := bytes.Clone(data)
				p, err := top.ReadV3PlanPolygonColorPrefix(data)
				if err != nil {
					t.Fatal(err)
				}
				base, _ := top.ReadV3PlanPolygonPointsPrefix(data)
				assertPolygonPoints(t, base, want, 45)
				assertColorBase(t, p, base)
				if p.ColorRaw() != byte(color) || p.Offsets().Color != (source.Span{Start: len(points), End: len(prefix)}) || p.ConsumedOffset() != len(prefix) || p.UnparsedTailSize() != len(tail) || !bytes.Equal(p.Bytes(), prefix) || !bytes.Equal(data, before) {
					t.Fatalf("color %d: raw value, span, stop, tail or input changed", color)
				}
			}
		}
		// No color is required by the unchanged P04c3 API, even with zero points.
		requirePlanPolygonColorError(t, points, top.DefaultPolygonCountLimits(), "truncated", "plan.elements[0].color", len(points))
		base, err := top.ReadV3PlanPolygonPointsPrefix(points)
		if err != nil || base.ConsumedOffset() != len(points) {
			t.Fatal("P04c3 no-color stop changed")
		}
	}
	for next := 0; next < 256; next++ {
		p, err := top.ReadV3PlanPolygonColorPrefix(append(planPolygonCountData(0), 129, byte(next)))
		if err != nil || p.ColorRaw() != 129 || p.ConsumedOffset() != 46 || p.UnparsedTailSize() != 1 {
			t.Fatal("next byte was interpreted")
		}
	}
}

func TestPlanPolygonColorPrefixTruncationAndInheritedErrors(t *testing.T) {
	data := append(planPolygonCountData(3), polygonPointBytes([][2]int32{{1, -2}, {3, -4}, {5, -6}})...)
	data = append(data, 255)
	for n := 0; n < len(data); n++ {
		if n == len(data)-1 {
			requirePlanPolygonColorError(t, data[:n], top.DefaultPolygonCountLimits(), "truncated", "plan.elements[0].color", n)
			continue
		}
		_, earlier := top.ReadV3PlanPolygonPointsPrefix(data[:n])
		p, err := top.ReadV3PlanPolygonColorPrefix(data[:n])
		if earlier == nil || !reflect.DeepEqual(err, earlier) || !reflect.DeepEqual(p, source.PlanPolygonColorPrefix{}) {
			t.Fatalf("inherited error/preflight at %d changed", n)
		}
	}
	for marker := 0; marker < 256; marker++ {
		if marker != 1 {
			requirePlanPolygonColorError(t, append(planMarkerBase(), byte(marker)), top.DefaultPolygonCountLimits(), "unsupported_element", "plan.elements[0].kind", 40)
		}
	}
	for _, count := range []int32{-1, math.MinInt32, 1000001, math.MaxInt32} {
		code := "resource_limit"
		if count < 0 {
			code = "negative_count"
		}
		requirePlanPolygonColorError(t, planPolygonCountData(count), top.DefaultPolygonCountLimits(), code, "plan.elements[0].point_count", 41)
	}
	for _, bad := range [][]byte{nil, {255}, fixture(-1), fixture(1, record(-1, "", 0)), fixture(1, record(17, "A\xffB", 0)), measurementTable(-1), referenceTable(-1), referenceTable(1, referenceRecord("")[:24])} {
		_, earlier := top.ReadV3PlanPolygonPointsPrefix(bad)
		p, err := top.ReadV3PlanPolygonColorPrefix(bad)
		if earlier == nil || !reflect.DeepEqual(err, earlier) || !reflect.DeepEqual(p, source.PlanPolygonColorPrefix{}) {
			t.Fatal("inherited failure changed")
		}
	}
	// Every earlier entry point still succeeds at its own independent stop.
	TestPlanPolygonCountPrefixTruncationAndEarlierStops(t)
}

func TestPlanPolygonColorPrefixLimits(t *testing.T) {
	for _, n := range []int{0, 1, 3, 1000000} {
		limits := top.DefaultPolygonCountLimits()
		limits.MaxPoints = n
		data := append(planPolygonCountData(int32(n)), make([]byte, 8*n)...)
		data = append(data, 255)
		limits.MaxInputBytes = len(data)
		p, err := top.ReadV3PlanPolygonColorPrefixWithLimits(data, limits)
		if err != nil || len(p.Points()) != n || p.ColorRaw() != 255 || p.ConsumedOffset() != len(data) {
			t.Fatalf("exact limit %d: %v", n, err)
		}
		requirePlanPolygonColorError(t, append(data, 128), limits, "resource_limit", "input", 0)
		requirePlanPolygonColorError(t, planPolygonCountData(int32(n+1)), limits, "resource_limit", "plan.elements[0].point_count", 41)
	}
	for _, bad := range []int{-1, 1000001, math.MaxInt} {
		limits := top.DefaultPolygonCountLimits()
		limits.MaxPoints, limits.MaxReferences = bad, -1
		requirePlanPolygonColorError(t, nil, limits, "invalid_limit", "limits.max_points", 0)
	}
	limits := top.PolygonCountLimits{}
	requirePlanPolygonColorError(t, nil, limits, "truncated", "header", 0)
	limits.MaxInputBytes = 46
	if _, err := top.ReadV3PlanPolygonColorPrefixWithLimits(append(planPolygonCountData(0), 0), limits); err != nil {
		t.Fatal(err)
	}
	requirePlanPolygonColorError(t, planPolygonCountData(0), limits, "truncated", "plan.elements[0].color", 45)
}

func TestPlanPolygonColorPrefixVariableTablesAndCopies(t *testing.T) {
	data := binary.LittleEndian.AppendUint32(fixture(1, record(17, "trip", -32768)), 1)
	data = append(data, measurementRecord(255, "abc")...)
	data = binary.LittleEndian.AppendUint32(data, 1)
	data = append(data, referenceRecord("Aą")...)
	data = append(data, planMappingRecord()...)
	data = append(data, planMappingBase()[16:]...)
	data = binary.LittleEndian.AppendUint32(append(data, 1), 2)
	data = append(data, polygonPointBytes([][2]int32{{-16777217, 66051}, {123, -456}})...)
	data = append(data, 129, 255, 128)
	before := bytes.Clone(data)
	p, err := top.ReadV3PlanPolygonColorPrefix(data)
	if err != nil {
		t.Fatal(err)
	}
	base, _ := top.ReadV3PlanPolygonPointsPrefix(data)
	assertColorBase(t, p, base)
	if p.Offsets().Color != (source.Span{Start: 128, End: 129}) {
		t.Fatal("hardcoded color position")
	}
	for _, restrict := range []func(*top.PolygonCountLimits){func(l *top.PolygonCountLimits) { l.MaxTrips = 0 }, func(l *top.PolygonCountLimits) { l.MaxMeasurements = 0 }, func(l *top.PolygonCountLimits) { l.MaxReferences = 0 }, func(l *top.PolygonCountLimits) { l.MaxCommentBytes = 3 }, func(l *top.PolygonCountLimits) { l.MaxPoints = 1 }, func(l *top.PolygonCountLimits) { l.MaxReferences = -1 }} {
		limits := top.DefaultPolygonCountLimits()
		restrict(&limits)
		_, earlier := top.ReadV3PlanPolygonPointsPrefixWithLimits(data, limits)
		q, err := top.ReadV3PlanPolygonColorPrefixWithLimits(data, limits)
		if earlier == nil || !reflect.DeepEqual(err, earlier) || !reflect.DeepEqual(q, source.PlanPolygonColorPrefix{}) {
			t.Fatal("inherited caller bound/error changed")
		}
	}
	points, raw, trips, shots, refs, offsets := p.Points(), p.Bytes(), p.Trips(), p.Measurements(), p.References(), p.Offsets()
	for i := range data {
		data[i] = 0
	}
	points[0], raw[128], trips[0], shots[0], refs[0], offsets.Color.Start = source.PolygonPoint{}, 0, source.Trip{}, source.Measurement{}, source.Reference{}, 0
	assertColorBase(t, p, base)
	if p.ColorRaw() != 129 || p.Offsets().Color != (source.Span{Start: 128, End: 129}) || !bytes.Equal(p.Bytes(), before[:129]) || p.ConsumedOffset() != 129 || p.UnparsedTailSize() != 2 {
		t.Fatal("mutable aliases or lost color")
	}
}

func TestPlanPolygonColorPrefixNativeFixture(t *testing.T) {
	data, err := os.ReadFile("testdata/api-drawings.top")
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(data)) != "4a494ead03cade750f670d50aa38661abda9b3e1f131d67f27a252982752c2a5" {
		t.Fatal("native fixture changed")
	}
	// Pinned helper first Polygon is black; original scalar probe confirms code 1.
	for _, tail := range [][]byte{nil, data[176:], {255, 128}} {
		p, err := top.ReadV3PlanPolygonColorPrefix(append(bytes.Clone(data[:176]), tail...))
		if err != nil {
			t.Fatal(err)
		}
		base, _ := top.ReadV3PlanPolygonPointsPrefix(data)
		assertPolygonPoints(t, base, [][2]int32{{-6000, -2000}, {-5500, -1000}, {-4500, -1500}}, 151)
		assertColorBase(t, p, base)
		if p.ColorRaw() != 1 || p.Offsets().Color != (source.Span{Start: 175, End: 176}) || p.ConsumedOffset() != 176 || p.UnparsedTailSize() != len(tail) || !bytes.Equal(p.Bytes(), data[:176]) {
			t.Fatal("native color or stop changed")
		}
	}
	requirePlanPolygonColorError(t, data[:175], top.DefaultPolygonCountLimits(), "truncated", "plan.elements[0].color", 175)
}

func FuzzReadV3PlanPolygonColorPrefix(f *testing.F) {
	for _, seed := range [][]byte{planPolygonCountData(0), planPolygonCountData(3), planPolygonCountData(-1), planPolygonCountData(1000000), {255}} {
		f.Add(seed)
	}
	for _, count := range []int32{0, 1, 3} {
		prefix := append(planPolygonCountData(count), polygonPointBytes([][2]int32{{66051, -66051}, {math.MinInt32, math.MaxInt32}, {16777217, -16777217}}[:count])...)
		for _, color := range []byte{0, 1, 2, 3, 4, 5, 6, 7, 128, 255} {
			f.Add(append(bytes.Clone(prefix), color))
			f.Add(append(bytes.Clone(prefix), color, 255, 128))
		}
	}
	limits := top.PolygonCountLimits{ReferenceLimits: top.ReferenceLimits{MeasurementLimits: top.MeasurementLimits{Limits: top.Limits{MaxInputBytes: 4096, MaxTrips: 32, MaxCommentBytes: 256}, MaxMeasurements: 32}, MaxReferences: 32}, MaxPoints: 32}
	f.Fuzz(func(t *testing.T, data []byte) {
		before := bytes.Clone(data)
		p, err := top.ReadV3PlanPolygonColorPrefixWithLimits(data, limits)
		q, again := top.ReadV3PlanPolygonColorPrefixWithLimits(data, limits)
		if !bytes.Equal(data, before) || !reflect.DeepEqual(p, q) || !reflect.DeepEqual(err, again) {
			t.Fatal("mutation or nondeterminism")
		}
		base, earlier := top.ReadV3PlanPolygonPointsPrefixWithLimits(data, limits)
		if earlier != nil {
			if !reflect.DeepEqual(err, earlier) || !reflect.DeepEqual(p, source.PlanPolygonColorPrefix{}) {
				t.Fatal("inherited error/preflight changed")
			}
			return
		}
		start := base.ConsumedOffset()
		if start == len(data) {
			requirePlanPolygonColorError(t, data, limits, "truncated", "plan.elements[0].color", start)
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		assertColorBase(t, p, base)
		if p.ColorRaw() != data[start] || p.Offsets().Color != (source.Span{Start: start, End: start + 1}) || p.ConsumedOffset() != start+1 || p.ConsumedOffset()+p.UnparsedTailSize() != len(data) || !bytes.Equal(p.Bytes(), data[:start+1]) {
			t.Fatal("raw color, span or exact prefix/tail accounting changed")
		}
	})
}
