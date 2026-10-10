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

func requirePlanThirdPolygonColorError(t *testing.T, data []byte, limits top.PolygonCountLimits, code, field string, offset int) {
	t.Helper()
	before := bytes.Clone(data)
	p, err := top.ReadV3PlanThirdPolygonColorPrefixWithLimits(data, limits)
	var pe *top.ParseError
	if !errors.As(err, &pe) || pe.Code != code || pe.Field != field || pe.Offset != offset || !reflect.DeepEqual(p, source.PlanThirdPolygonColorPrefix{}) || !bytes.Equal(before, data) {
		t.Fatalf("wrong/partial failure: %+v %v", p, err)
	}
}

func assertThirdColorBase(t *testing.T, p source.PlanThirdPolygonColorPrefix, base source.PlanThirdPolygonPointsPrefix) {
	t.Helper()
	if p.Header() != base.Header() || p.Version() != base.Version() || p.TripCountRaw() != base.TripCountRaw() || p.MeasurementCountRaw() != base.MeasurementCountRaw() || p.ReferenceCountRaw() != base.ReferenceCountRaw() || p.MarkerRaw() != base.MarkerRaw() || p.PointCountRaw() != base.PointCountRaw() || p.ColorRaw() != base.ColorRaw() || p.NextMarkerRaw() != base.NextMarkerRaw() || p.FollowingMarkerRaw() != base.FollowingMarkerRaw() || p.SecondColorRaw() != base.SecondColorRaw() || p.SecondPointCountRaw() != base.SecondPointCountRaw() || !reflect.DeepEqual(p.SecondPoints(), base.SecondPoints()) || p.ThirdPointCountRaw() != base.ThirdPointCountRaw() || !reflect.DeepEqual(p.ThirdPoints(), base.ThirdPoints()) || p.OverviewMapping() != base.OverviewMapping() || p.PlanMapping() != base.PlanMapping() || p.Offsets().PlanThirdPolygonPointsPrefixOffsets != base.Offsets() || !reflect.DeepEqual(p.Trips(), base.Trips()) || !reflect.DeepEqual(p.Measurements(), base.Measurements()) || !reflect.DeepEqual(p.References(), base.References()) || !reflect.DeepEqual(p.Points(), base.Points()) {
		t.Fatal("inherited raw records/mappings/spans changed")
	}
}

func TestPlanThirdPolygonColorPrefixAllValuesAndTails(t *testing.T) {
	tables := [][][2]int32{nil, {{66051, -66051}}, {{math.MinInt32, math.MaxInt32}, {-1, 0}, {16777217, -16777217}}}
	for _, first := range tables {
		for _, second := range tables {
			for _, third := range tables {
				points := append(thirdPointsCountTables(first, second, int32(len(third))), polygonPointBytes(third)...)
				for color := 0; color < 256; color++ {
					prefix := append(bytes.Clone(points), byte(color))
					for _, tail := range [][]byte{nil, {0}, {1}, {3}, {255, 128}, planMappingRecord()} {
						data := append(bytes.Clone(prefix), tail...)
						before := bytes.Clone(data)
						p, err := top.ReadV3PlanThirdPolygonColorPrefix(data)
						if err != nil {
							t.Fatal(err)
						}
						base, _ := top.ReadV3PlanThirdPolygonPointsPrefix(data)
						assertThirdPolygonPoints(t, base, third, 57+8*(len(first)+len(second)))
						assertThirdColorBase(t, p, base)
						if p.ThirdColorRaw() != byte(color) || p.Offsets().ThirdColor != (source.Span{Start: len(points), End: len(prefix)}) || p.ConsumedOffset() != len(prefix) || p.UnparsedTailSize() != len(tail) || !bytes.Equal(p.Bytes(), prefix) || !bytes.Equal(data, before) {
							t.Fatalf("color %d: raw/span/stop/tail/input changed", color)
						}
					}
				}
				requirePlanThirdPolygonColorError(t, points, top.DefaultPolygonCountLimits(), "truncated", "plan.elements[2].color", len(points))
				base, err := top.ReadV3PlanThirdPolygonPointsPrefix(points)
				if err != nil || base.ConsumedOffset() != len(points) {
					t.Fatal("P04c11 no-color stop changed")
				}
			}
		}
	}
	for next := 0; next < 256; next++ {
		p, err := top.ReadV3PlanThirdPolygonColorPrefix(append(thirdPointsCountData(0), 129, byte(next)))
		if err != nil || p.ThirdColorRaw() != 129 || p.ConsumedOffset() != 58 || p.UnparsedTailSize() != 1 {
			t.Fatal("later byte was interpreted")
		}
	}
}

func TestPlanThirdPolygonColorPrefixTruncationAndInheritedErrors(t *testing.T) {
	data := append(thirdPolygonCountData([][2]int32{{7, -8}, {9, -10}, {11, -12}}, 3), polygonPointBytes([][2]int32{{1, -2}, {3, -4}, {5, -6}})...)
	data = append(data, 255)
	for n := 0; n < len(data); n++ {
		if n == len(data)-1 {
			requirePlanThirdPolygonColorError(t, data[:n], top.DefaultPolygonCountLimits(), "truncated", "plan.elements[2].color", n)
			continue
		}
		_, earlier := top.ReadV3PlanThirdPolygonPointsPrefix(data[:n])
		p, err := top.ReadV3PlanThirdPolygonColorPrefix(data[:n])
		if earlier == nil || !reflect.DeepEqual(err, earlier) || !reflect.DeepEqual(p, source.PlanThirdPolygonColorPrefix{}) {
			t.Fatalf("inherited error/preflight at %d changed", n)
		}
	}
	for marker := 0; marker < 256; marker++ {
		if marker != 1 {
			requirePlanThirdPolygonColorError(t, append(thirdCountColorData(nil, nil), byte(marker)), top.DefaultPolygonCountLimits(), "unsupported_element", "plan.elements[2].kind", 52)
		}
	}
	for _, count := range []int32{-1, math.MinInt32, 1000001, math.MaxInt32} {
		code := "resource_limit"
		if count < 0 {
			code = "negative_count"
		}
		requirePlanThirdPolygonColorError(t, thirdPointsCountData(count), top.DefaultPolygonCountLimits(), code, "plan.elements[2].point_count", 53)
	}
	for _, bad := range [][]byte{nil, {255}, fixture(-1), fixture(1, record(-1, "", 0)), fixture(1, record(17, "A\xffB", 0)), measurementTable(-1), referenceTable(-1), referenceTable(1, referenceRecord("")[:24])} {
		_, earlier := top.ReadV3PlanThirdPolygonPointsPrefix(bad)
		p, err := top.ReadV3PlanThirdPolygonColorPrefix(bad)
		if earlier == nil || !reflect.DeepEqual(err, earlier) || !reflect.DeepEqual(p, source.PlanThirdPolygonColorPrefix{}) {
			t.Fatal("inherited failure changed")
		}
	}
	// Every earlier entry point still succeeds at its own independent stop.
	TestPlanThirdPolygonPointsPrefixTruncationAndPrecedence(t)
}

func TestPlanThirdPolygonColorPrefixLimits(t *testing.T) {
	for _, n := range []int{0, 1, 3, 1000000} {
		limits := top.DefaultPolygonCountLimits()
		limits.MaxPoints = n
		data := append(thirdPointsCountData(int32(n)), make([]byte, 8*n)...)
		data = append(data, 255)
		limits.MaxInputBytes = len(data)
		p, err := top.ReadV3PlanThirdPolygonColorPrefixWithLimits(data, limits)
		if err != nil || len(p.ThirdPoints()) != n || p.ThirdColorRaw() != 255 || p.ConsumedOffset() != len(data) {
			t.Fatalf("exact limit %d: %v", n, err)
		}
		requirePlanThirdPolygonColorError(t, append(data, 128), limits, "resource_limit", "input", 0)
		requirePlanThirdPolygonColorError(t, thirdPointsCountData(int32(n+1)), limits, "resource_limit", "plan.elements[2].point_count", 53)
	}
	for _, bad := range []int{-1, 1000001, math.MaxInt} {
		limits := top.DefaultPolygonCountLimits()
		limits.MaxPoints, limits.MaxReferences = bad, -1
		requirePlanThirdPolygonColorError(t, nil, limits, "invalid_limit", "limits.max_points", 0)
	}
	limits := top.PolygonCountLimits{}
	requirePlanThirdPolygonColorError(t, nil, limits, "truncated", "header", 0)
	limits.MaxInputBytes = 58
	if _, err := top.ReadV3PlanThirdPolygonColorPrefixWithLimits(append(thirdPointsCountData(0), 0), limits); err != nil {
		t.Fatal(err)
	}
	requirePlanThirdPolygonColorError(t, thirdPointsCountData(0), limits, "truncated", "plan.elements[2].color", 57)
}

func TestPlanThirdPolygonColorPrefixVariableTablesAndCopies(t *testing.T) {
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
	data = binary.LittleEndian.AppendUint32(append(data, 255, 1), 3)
	data = append(data, polygonPointBytes([][2]int32{{-2000, -2000}, {-1500, -1000}, {-500, -1500}})...)
	data = append(data, 254, 255, 128)
	before := bytes.Clone(data)
	p, err := top.ReadV3PlanThirdPolygonColorPrefix(data)
	if err != nil {
		t.Fatal(err)
	}
	base, _ := top.ReadV3PlanThirdPolygonPointsPrefix(data)
	assertThirdColorBase(t, p, base)
	if p.Offsets().ThirdColor != (source.Span{Start: 172, End: 173}) {
		t.Fatal("hardcoded color position")
	}
	for _, restrict := range []func(*top.PolygonCountLimits){func(l *top.PolygonCountLimits) { l.MaxTrips = 0 }, func(l *top.PolygonCountLimits) { l.MaxMeasurements = 0 }, func(l *top.PolygonCountLimits) { l.MaxReferences = 0 }, func(l *top.PolygonCountLimits) { l.MaxCommentBytes = 3 }, func(l *top.PolygonCountLimits) { l.MaxPoints = 1 }, func(l *top.PolygonCountLimits) { l.MaxReferences = -1 }} {
		limits := top.DefaultPolygonCountLimits()
		restrict(&limits)
		_, earlier := top.ReadV3PlanThirdPolygonPointsPrefixWithLimits(data, limits)
		q, err := top.ReadV3PlanThirdPolygonColorPrefixWithLimits(data, limits)
		if earlier == nil || !reflect.DeepEqual(err, earlier) || !reflect.DeepEqual(q, source.PlanThirdPolygonColorPrefix{}) {
			t.Fatal("inherited caller bound/error changed")
		}
	}
	points, raw, trips, shots, refs, offsets := p.ThirdPoints(), p.Bytes(), p.Trips(), p.Measurements(), p.References(), p.Offsets()
	firstPoints := p.Points()
	firstPoints[0] = source.PolygonPoint{}
	secondPoints := p.SecondPoints()
	secondPoints[0] = source.PolygonPoint{}
	for i := range data {
		data[i] = 0
	}
	points[0], raw[172], trips[0], shots[0], refs[0], offsets.ThirdColor.Start = source.PolygonPoint{}, 0, source.Trip{}, source.Measurement{}, source.Reference{}, 0
	assertThirdColorBase(t, p, base)
	if p.ThirdColorRaw() != 254 || p.Offsets().ThirdColor != (source.Span{Start: 172, End: 173}) || !bytes.Equal(p.Bytes(), before[:173]) || p.ConsumedOffset() != 173 || p.UnparsedTailSize() != 2 {
		t.Fatal("mutable aliases or lost color")
	}
}

func TestPlanThirdPolygonColorPrefixNativeFixture(t *testing.T) {
	data, err := os.ReadFile("testdata/api-drawings.top")
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(data)) != "4a494ead03cade750f670d50aa38661abda9b3e1f131d67f27a252982752c2a5" {
		t.Fatal("native fixture changed")
	}
	// Pinned helper third Polygon is brown; fresh original scalar probe confirms code 3.
	for _, tail := range [][]byte{nil, data[236:], {255, 128}} {
		p, err := top.ReadV3PlanThirdPolygonColorPrefix(append(bytes.Clone(data[:236]), tail...))
		if err != nil {
			t.Fatal(err)
		}
		base, _ := top.ReadV3PlanThirdPolygonPointsPrefix(data)
		assertThirdPolygonPoints(t, base, [][2]int32{{-2000, -2000}, {-1500, -1000}, {-500, -1500}}, 211)
		assertThirdColorBase(t, p, base)
		if p.ThirdColorRaw() != 3 || p.Offsets().ThirdColor != (source.Span{Start: 235, End: 236}) || p.ConsumedOffset() != 236 || p.UnparsedTailSize() != len(tail) || !bytes.Equal(p.Bytes(), data[:236]) {
			t.Fatal("native color or stop changed")
		}
	}
	requirePlanThirdPolygonColorError(t, data[:235], top.DefaultPolygonCountLimits(), "truncated", "plan.elements[2].color", 235)
}

func FuzzReadV3PlanThirdPolygonColorPrefix(f *testing.F) {
	for _, seed := range [][]byte{thirdPointsCountData(0), thirdPointsCountData(3), thirdPointsCountData(-1), thirdPointsCountData(1000000), {255}} {
		f.Add(seed)
	}
	for _, first := range [][][2]int32{nil, {{66051, -66051}}, {{1, -2}, {3, -4}, {5, -6}}} {
		for _, count := range []int32{0, 1, 3} {
			prefix := append(thirdPointsCountTables(first, nil, count), polygonPointBytes([][2]int32{{66051, -66051}, {math.MinInt32, math.MaxInt32}, {16777217, -16777217}}[:count])...)
			for _, color := range []byte{0, 1, 2, 3, 4, 5, 6, 7, 128, 255} {
				f.Add(append(bytes.Clone(prefix), color))
				f.Add(append(bytes.Clone(prefix), color, 255, 128))
			}
		}
	}
	for _, first := range [][][2]int32{nil, {{1, -2}}, {{3, -4}, {5, -6}, {7, -8}}} {
		for _, second := range [][][2]int32{nil, {{9, -10}}, {{11, -12}, {13, -14}, {15, -16}}} {
			for _, third := range [][][2]int32{nil, {{17, -18}}, {{19, -20}, {21, -22}, {23, -24}}} {
				prefix := append(thirdPointsCountTables(first, second, int32(len(third))), polygonPointBytes(third)...)
				f.Add(prefix)
				f.Add(append(bytes.Clone(prefix), 254))
				f.Add(append(bytes.Clone(prefix), 128, 255, 0))
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
		p, err := top.ReadV3PlanThirdPolygonColorPrefixWithLimits(data, limits)
		q, again := top.ReadV3PlanThirdPolygonColorPrefixWithLimits(data, limits)
		if !bytes.Equal(data, before) || !reflect.DeepEqual(p, q) || !reflect.DeepEqual(err, again) {
			t.Fatal("mutation or nondeterminism")
		}
		base, earlier := top.ReadV3PlanThirdPolygonPointsPrefixWithLimits(data, limits)
		if earlier != nil {
			if !reflect.DeepEqual(err, earlier) || !reflect.DeepEqual(p, source.PlanThirdPolygonColorPrefix{}) {
				t.Fatal("inherited error/preflight changed")
			}
			return
		}
		start := base.ConsumedOffset()
		if start == len(data) {
			requirePlanThirdPolygonColorError(t, data, limits, "truncated", "plan.elements[2].color", start)
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		assertThirdColorBase(t, p, base)
		if p.ThirdColorRaw() != data[start] || p.Offsets().ThirdColor != (source.Span{Start: start, End: start + 1}) || p.ConsumedOffset() != start+1 || p.ConsumedOffset()+p.UnparsedTailSize() != len(data) || !bytes.Equal(p.Bytes(), data[:start+1]) {
			t.Fatal("raw color, span or exact prefix/tail accounting changed")
		}
	})
}

func TestPlanThirdPolygonColorPrefixSeparateLimits(t *testing.T) {
	limits := top.DefaultPolygonCountLimits()
	limits.MaxPoints = 3
	first := [][2]int32{{1, -2}, {3, -4}, {5, -6}}
	second := [][2]int32{{7, -8}, {9, -10}, {11, -12}}
	third := [][2]int32{{13, -14}, {15, -16}, {17, -18}}
	data := append(thirdPointsCountTables(first, second, 3), polygonPointBytes(third)...)
	data = append(data, 254)
	p, err := top.ReadV3PlanThirdPolygonColorPrefixWithLimits(data, limits)
	if err != nil || len(p.Points()) != 3 || len(p.SecondPoints()) != 3 || len(p.ThirdPoints()) != 3 || p.ColorRaw() != 129 || p.SecondColorRaw() != 255 || p.ThirdColorRaw() != 254 {
		t.Fatal("separate inclusive limits/colors lost", err)
	}
	limits.MaxPoints = 2
	requirePlanThirdPolygonColorError(t, data, limits, "resource_limit", "plan.elements[0].point_count", 41)
	limits.MaxPoints = 3
	requirePlanThirdPolygonColorError(t, thirdPointsCountTables(first, append(second, [2]int32{0, 0}), 0), limits, "resource_limit", "plan.elements[1].point_count", 71)
	requirePlanThirdPolygonColorError(t, thirdPointsCountTables(first, second, 4), limits, "resource_limit", "plan.elements[2].point_count", 101)
	for _, bad := range [][]byte{planPolygonCountData(-1), planPolygonCountData(1000001), planPolygonCountData(1), planPolygonCountData(0), append(planMarkerBase(), 3), secondPointsCountData(-1), secondPointsCountData(1), secondPointsCountData(0)} {
		_, earlier := top.ReadV3PlanThirdPolygonPointsPrefix(bad)
		q, e := top.ReadV3PlanThirdPolygonColorPrefix(bad)
		if earlier == nil || !reflect.DeepEqual(e, earlier) || !reflect.DeepEqual(q, source.PlanThirdPolygonColorPrefix{}) {
			t.Fatal("earlier Polygon error precedence changed")
		}
	}
}
