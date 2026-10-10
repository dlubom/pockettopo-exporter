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

func requirePlanSecondPolygonColorError(t *testing.T, data []byte, limits top.PolygonCountLimits, code, field string, offset int) {
	t.Helper()
	before := bytes.Clone(data)
	p, err := top.ReadV3PlanSecondPolygonColorPrefixWithLimits(data, limits)
	var pe *top.ParseError
	if !errors.As(err, &pe) || pe.Code != code || pe.Field != field || pe.Offset != offset || !reflect.DeepEqual(p, source.PlanSecondPolygonColorPrefix{}) || !bytes.Equal(before, data) {
		t.Fatalf("wrong/partial failure: %+v %v", p, err)
	}
}

func assertSecondColorBase(t *testing.T, p source.PlanSecondPolygonColorPrefix, base source.PlanSecondPolygonPointsPrefix) {
	t.Helper()
	if p.Header() != base.Header() || p.Version() != base.Version() || p.TripCountRaw() != base.TripCountRaw() || p.MeasurementCountRaw() != base.MeasurementCountRaw() || p.ReferenceCountRaw() != base.ReferenceCountRaw() || p.MarkerRaw() != base.MarkerRaw() || p.PointCountRaw() != base.PointCountRaw() || p.ColorRaw() != base.ColorRaw() || p.NextMarkerRaw() != base.NextMarkerRaw() || p.SecondPointCountRaw() != base.SecondPointCountRaw() || !reflect.DeepEqual(p.SecondPoints(), base.SecondPoints()) || p.OverviewMapping() != base.OverviewMapping() || p.PlanMapping() != base.PlanMapping() || p.Offsets().PlanSecondPolygonPointsPrefixOffsets != base.Offsets() || !reflect.DeepEqual(p.Trips(), base.Trips()) || !reflect.DeepEqual(p.Measurements(), base.Measurements()) || !reflect.DeepEqual(p.References(), base.References()) || !reflect.DeepEqual(p.Points(), base.Points()) {
		t.Fatal("inherited raw records/mappings/spans changed")
	}
}

func TestPlanSecondPolygonColorPrefixAllValuesAndTails(t *testing.T) {
	for _, first := range [][][2]int32{nil, {{1, -2}}, {{66051, -66051}, {-1, 0}, {16777217, -16777217}}} {
		for _, want := range [][][2]int32{nil, {{66051, -66051}}, {{math.MinInt32, math.MaxInt32}, {-1, 0}, {16777217, -16777217}}} {
			points := append(secondPolygonCountData(first, int32(len(want))), polygonPointBytes(want)...)
			for color := 0; color < 256; color++ {
				prefix := append(bytes.Clone(points), byte(color))
				for _, tail := range [][]byte{nil, {0}, {1}, {3}, {255, 128}, planMappingRecord()} {
					data := append(bytes.Clone(prefix), tail...)
					before := bytes.Clone(data)
					p, err := top.ReadV3PlanSecondPolygonColorPrefix(data)
					if err != nil {
						t.Fatal(err)
					}
					base, _ := top.ReadV3PlanSecondPolygonPointsPrefix(data)
					assertSecondPolygonPoints(t, base, want, 51+8*len(first))
					assertSecondColorBase(t, p, base)
					if p.SecondColorRaw() != byte(color) || p.Offsets().SecondColor != (source.Span{Start: len(points), End: len(prefix)}) || p.ConsumedOffset() != len(prefix) || p.UnparsedTailSize() != len(tail) || !bytes.Equal(p.Bytes(), prefix) || !bytes.Equal(data, before) {
						t.Fatalf("color %d: raw value, span, stop, tail or input changed", color)
					}
				}
			}
			// P04c7 still requires no second color, even with zero points.
			requirePlanSecondPolygonColorError(t, points, top.DefaultPolygonCountLimits(), "truncated", "plan.elements[1].color", len(points))
			base, err := top.ReadV3PlanSecondPolygonPointsPrefix(points)
			if err != nil || base.ConsumedOffset() != len(points) {
				t.Fatal("P04c7 no-color stop changed")
			}
		}
	}
	for next := 0; next < 256; next++ {
		p, err := top.ReadV3PlanSecondPolygonColorPrefix(append(secondPointsCountData(0), 129, byte(next)))
		if err != nil || p.SecondColorRaw() != 129 || p.ConsumedOffset() != 52 || p.UnparsedTailSize() != 1 {
			t.Fatal("next byte was interpreted")
		}
	}
}

func TestPlanSecondPolygonColorPrefixTruncationAndInheritedErrors(t *testing.T) {
	data := append(secondPolygonCountData([][2]int32{{7, -8}, {9, -10}, {11, -12}}, 3), polygonPointBytes([][2]int32{{1, -2}, {3, -4}, {5, -6}})...)
	data = append(data, 255)
	for n := 0; n < len(data); n++ {
		if n == len(data)-1 {
			requirePlanSecondPolygonColorError(t, data[:n], top.DefaultPolygonCountLimits(), "truncated", "plan.elements[1].color", n)
			continue
		}
		_, earlier := top.ReadV3PlanSecondPolygonPointsPrefix(data[:n])
		p, err := top.ReadV3PlanSecondPolygonColorPrefix(data[:n])
		if earlier == nil || !reflect.DeepEqual(err, earlier) || !reflect.DeepEqual(p, source.PlanSecondPolygonColorPrefix{}) {
			t.Fatalf("inherited error/preflight at %d changed", n)
		}
	}
	for marker := 0; marker < 256; marker++ {
		if marker != 1 {
			requirePlanSecondPolygonColorError(t, append(nextMarkerColorData(nil, 129), byte(marker)), top.DefaultPolygonCountLimits(), "unsupported_element", "plan.elements[1].kind", 46)
		}
	}
	for _, count := range []int32{-1, math.MinInt32, 1000001, math.MaxInt32} {
		code := "resource_limit"
		if count < 0 {
			code = "negative_count"
		}
		requirePlanSecondPolygonColorError(t, secondPointsCountData(count), top.DefaultPolygonCountLimits(), code, "plan.elements[1].point_count", 47)
	}
	for _, bad := range [][]byte{nil, {255}, fixture(-1), fixture(1, record(-1, "", 0)), fixture(1, record(17, "A\xffB", 0)), measurementTable(-1), referenceTable(-1), referenceTable(1, referenceRecord("")[:24])} {
		_, earlier := top.ReadV3PlanSecondPolygonPointsPrefix(bad)
		p, err := top.ReadV3PlanSecondPolygonColorPrefix(bad)
		if earlier == nil || !reflect.DeepEqual(err, earlier) || !reflect.DeepEqual(p, source.PlanSecondPolygonColorPrefix{}) {
			t.Fatal("inherited failure changed")
		}
	}
	// Every earlier entry point still succeeds at its own independent stop.
	TestPlanSecondPolygonCountPrefixMarkersAndTruncations(t)
}

func TestPlanSecondPolygonColorPrefixLimits(t *testing.T) {
	for _, n := range []int{0, 1, 3, 1000000} {
		limits := top.DefaultPolygonCountLimits()
		limits.MaxPoints = n
		data := append(secondPointsCountData(int32(n)), make([]byte, 8*n)...)
		data = append(data, 255)
		limits.MaxInputBytes = len(data)
		p, err := top.ReadV3PlanSecondPolygonColorPrefixWithLimits(data, limits)
		if err != nil || len(p.SecondPoints()) != n || p.SecondColorRaw() != 255 || p.ConsumedOffset() != len(data) {
			t.Fatalf("exact limit %d: %v", n, err)
		}
		requirePlanSecondPolygonColorError(t, append(data, 128), limits, "resource_limit", "input", 0)
		requirePlanSecondPolygonColorError(t, secondPointsCountData(int32(n+1)), limits, "resource_limit", "plan.elements[1].point_count", 47)
	}
	for _, bad := range []int{-1, 1000001, math.MaxInt} {
		limits := top.DefaultPolygonCountLimits()
		limits.MaxPoints, limits.MaxReferences = bad, -1
		requirePlanSecondPolygonColorError(t, nil, limits, "invalid_limit", "limits.max_points", 0)
	}
	limits := top.PolygonCountLimits{}
	requirePlanSecondPolygonColorError(t, nil, limits, "truncated", "header", 0)
	limits.MaxInputBytes = 52
	if _, err := top.ReadV3PlanSecondPolygonColorPrefixWithLimits(append(secondPointsCountData(0), 0), limits); err != nil {
		t.Fatal(err)
	}
	requirePlanSecondPolygonColorError(t, secondPointsCountData(0), limits, "truncated", "plan.elements[1].color", 51)
}

func TestPlanSecondPolygonColorPrefixVariableTablesAndCopies(t *testing.T) {
	data := binary.LittleEndian.AppendUint32(fixture(1, record(17, "trip", -32768)), 1)
	data = append(data, measurementRecord(255, "abc")...)
	data = binary.LittleEndian.AppendUint32(data, 1)
	data = append(data, referenceRecord("Aą")...)
	data = append(data, planMappingRecord()...)
	data = append(data, planMappingBase()[16:]...)
	data = binary.LittleEndian.AppendUint32(append(data, 1), 2)
	data = append(data, polygonPointBytes([][2]int32{{-16777217, 66051}, {123, -456}})...)
	data = binary.LittleEndian.AppendUint32(append(data, 129, 1), 1)
	data = append(data, polygonPointBytes([][2]int32{{-789, 1011}})...)
	data = append(data, 255, 255, 128)
	before := bytes.Clone(data)
	p, err := top.ReadV3PlanSecondPolygonColorPrefix(data)
	if err != nil {
		t.Fatal(err)
	}
	base, _ := top.ReadV3PlanSecondPolygonPointsPrefix(data)
	assertSecondColorBase(t, p, base)
	if p.Offsets().SecondColor != (source.Span{Start: 142, End: 143}) {
		t.Fatal("hardcoded color position")
	}
	for _, restrict := range []func(*top.PolygonCountLimits){func(l *top.PolygonCountLimits) { l.MaxTrips = 0 }, func(l *top.PolygonCountLimits) { l.MaxMeasurements = 0 }, func(l *top.PolygonCountLimits) { l.MaxReferences = 0 }, func(l *top.PolygonCountLimits) { l.MaxCommentBytes = 3 }, func(l *top.PolygonCountLimits) { l.MaxPoints = 1 }, func(l *top.PolygonCountLimits) { l.MaxReferences = -1 }} {
		limits := top.DefaultPolygonCountLimits()
		restrict(&limits)
		_, earlier := top.ReadV3PlanSecondPolygonPointsPrefixWithLimits(data, limits)
		q, err := top.ReadV3PlanSecondPolygonColorPrefixWithLimits(data, limits)
		if earlier == nil || !reflect.DeepEqual(err, earlier) || !reflect.DeepEqual(q, source.PlanSecondPolygonColorPrefix{}) {
			t.Fatal("inherited caller bound/error changed")
		}
	}
	points, raw, trips, shots, refs, offsets := p.SecondPoints(), p.Bytes(), p.Trips(), p.Measurements(), p.References(), p.Offsets()
	firstPoints := p.Points()
	firstPoints[0] = source.PolygonPoint{}
	for i := range data {
		data[i] = 0
	}
	points[0], raw[142], trips[0], shots[0], refs[0], offsets.SecondColor.Start = source.PolygonPoint{}, 0, source.Trip{}, source.Measurement{}, source.Reference{}, 0
	assertSecondColorBase(t, p, base)
	if p.SecondColorRaw() != 255 || p.Offsets().SecondColor != (source.Span{Start: 142, End: 143}) || !bytes.Equal(p.Bytes(), before[:143]) || p.ConsumedOffset() != 143 || p.UnparsedTailSize() != 2 {
		t.Fatal("mutable aliases or lost color")
	}
}

func TestPlanSecondPolygonColorPrefixNativeFixture(t *testing.T) {
	data, err := os.ReadFile("testdata/api-drawings.top")
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(data)) != "4a494ead03cade750f670d50aa38661abda9b3e1f131d67f27a252982752c2a5" {
		t.Fatal("native fixture changed")
	}
	// Pinned helper second Polygon is gray; original scalar probe confirms code 2.
	for _, tail := range [][]byte{nil, data[206:], {255, 128}} {
		p, err := top.ReadV3PlanSecondPolygonColorPrefix(append(bytes.Clone(data[:206]), tail...))
		if err != nil {
			t.Fatal(err)
		}
		base, _ := top.ReadV3PlanSecondPolygonPointsPrefix(data)
		assertSecondPolygonPoints(t, base, [][2]int32{{-4000, -2000}, {-3500, -1000}, {-2500, -1500}}, 181)
		assertSecondColorBase(t, p, base)
		if p.SecondColorRaw() != 2 || p.Offsets().SecondColor != (source.Span{Start: 205, End: 206}) || p.ConsumedOffset() != 206 || p.UnparsedTailSize() != len(tail) || !bytes.Equal(p.Bytes(), data[:206]) {
			t.Fatal("native color or stop changed")
		}
	}
	requirePlanSecondPolygonColorError(t, data[:205], top.DefaultPolygonCountLimits(), "truncated", "plan.elements[1].color", 205)
}

func FuzzReadV3PlanSecondPolygonColorPrefix(f *testing.F) {
	for _, seed := range [][]byte{secondPointsCountData(0), secondPointsCountData(3), secondPointsCountData(-1), secondPointsCountData(1000000), {255}} {
		f.Add(seed)
	}
	for _, first := range [][][2]int32{nil, {{66051, -66051}}, {{1, -2}, {3, -4}, {5, -6}}} {
		for _, count := range []int32{0, 1, 3} {
			prefix := append(secondPolygonCountData(first, count), polygonPointBytes([][2]int32{{66051, -66051}, {math.MinInt32, math.MaxInt32}, {16777217, -16777217}}[:count])...)
			for _, color := range []byte{0, 1, 2, 3, 4, 5, 6, 7, 128, 255} {
				f.Add(append(bytes.Clone(prefix), color))
				f.Add(append(bytes.Clone(prefix), color, 255, 128))
			}
		}
	}
	native, e := os.ReadFile("testdata/api-drawings.top")
	if e != nil {
		f.Fatal(e)
	}
	f.Add(native)
	f.Add(native[:206])
	f.Add(native[:205])
	limits := top.PolygonCountLimits{ReferenceLimits: top.ReferenceLimits{MeasurementLimits: top.MeasurementLimits{Limits: top.Limits{MaxInputBytes: 4096, MaxTrips: 32, MaxCommentBytes: 256}, MaxMeasurements: 32}, MaxReferences: 32}, MaxPoints: 32}
	f.Fuzz(func(t *testing.T, data []byte) {
		before := bytes.Clone(data)
		p, err := top.ReadV3PlanSecondPolygonColorPrefixWithLimits(data, limits)
		q, again := top.ReadV3PlanSecondPolygonColorPrefixWithLimits(data, limits)
		if !bytes.Equal(data, before) || !reflect.DeepEqual(p, q) || !reflect.DeepEqual(err, again) {
			t.Fatal("mutation or nondeterminism")
		}
		base, earlier := top.ReadV3PlanSecondPolygonPointsPrefixWithLimits(data, limits)
		if earlier != nil {
			if !reflect.DeepEqual(err, earlier) || !reflect.DeepEqual(p, source.PlanSecondPolygonColorPrefix{}) {
				t.Fatal("inherited error/preflight changed")
			}
			return
		}
		start := base.ConsumedOffset()
		if start == len(data) {
			requirePlanSecondPolygonColorError(t, data, limits, "truncated", "plan.elements[1].color", start)
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		assertSecondColorBase(t, p, base)
		if p.SecondColorRaw() != data[start] || p.Offsets().SecondColor != (source.Span{Start: start, End: start + 1}) || p.ConsumedOffset() != start+1 || p.ConsumedOffset()+p.UnparsedTailSize() != len(data) || !bytes.Equal(p.Bytes(), data[:start+1]) {
			t.Fatal("raw color, span or exact prefix/tail accounting changed")
		}
	})
}

func TestPlanSecondPolygonColorPrefixSeparateLimits(t *testing.T) {
	limits := top.DefaultPolygonCountLimits()
	limits.MaxPoints = 3
	first := [][2]int32{{1, -2}, {3, -4}, {5, -6}}
	second := [][2]int32{{7, -8}, {9, -10}, {11, -12}}
	data := append(secondPolygonCountData(first, 3), polygonPointBytes(second)...)
	data = append(data, 255)
	p, err := top.ReadV3PlanSecondPolygonColorPrefixWithLimits(data, limits)
	if err != nil || len(p.Points()) != 3 || len(p.SecondPoints()) != 3 || p.ColorRaw() != 129 || p.SecondColorRaw() != 255 {
		t.Fatal("separate inclusive limits/colors lost", err)
	}
	limits.MaxPoints = 2
	requirePlanSecondPolygonColorError(t, data, limits, "resource_limit", "plan.elements[0].point_count", 41)
	limits.MaxPoints = 3
	requirePlanSecondPolygonColorError(t, secondPolygonCountData(first, 4), limits, "resource_limit", "plan.elements[1].point_count", 71)
	for _, bad := range [][]byte{planPolygonCountData(-1), planPolygonCountData(1000001), planPolygonCountData(1), planPolygonCountData(0), append(planMarkerBase(), 3)} {
		_, earlier := top.ReadV3PlanSecondPolygonPointsPrefix(bad)
		q, e := top.ReadV3PlanSecondPolygonColorPrefix(bad)
		if earlier == nil || !reflect.DeepEqual(e, earlier) || !reflect.DeepEqual(q, source.PlanSecondPolygonColorPrefix{}) {
			t.Fatal("first Polygon error precedence changed")
		}
	}
}
