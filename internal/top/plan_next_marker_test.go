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

func requirePlanNextMarkerError(t *testing.T, data []byte, limits top.PolygonCountLimits, code, field string, offset int) {
	t.Helper()
	before := bytes.Clone(data)
	p, err := top.ReadV3PlanNextMarkerPrefixWithLimits(data, limits)
	var pe *top.ParseError
	if !errors.As(err, &pe) || pe.Code != code || pe.Field != field || pe.Offset != offset || !reflect.DeepEqual(p, source.PlanNextMarkerPrefix{}) || !bytes.Equal(before, data) {
		t.Fatalf("wrong/partial failure: %+v %v", p, err)
	}
}

func assertNextMarkerBase(t *testing.T, p source.PlanNextMarkerPrefix, base source.PlanPolygonColorPrefix) {
	t.Helper()
	if p.Header() != base.Header() || p.Version() != base.Version() || p.TripCountRaw() != base.TripCountRaw() || p.MeasurementCountRaw() != base.MeasurementCountRaw() || p.ReferenceCountRaw() != base.ReferenceCountRaw() || p.MarkerRaw() != base.MarkerRaw() || p.PointCountRaw() != base.PointCountRaw() || p.ColorRaw() != base.ColorRaw() || p.OverviewMapping() != base.OverviewMapping() || p.PlanMapping() != base.PlanMapping() || p.Offsets().PlanPolygonColorPrefixOffsets != base.Offsets() || !reflect.DeepEqual(p.Trips(), base.Trips()) || !reflect.DeepEqual(p.Measurements(), base.Measurements()) || !reflect.DeepEqual(p.References(), base.References()) || !reflect.DeepEqual(p.Points(), base.Points()) {
		t.Fatal("inherited fields/records/spans changed")
	}
}

func nextMarkerColorData(points [][2]int32, color byte) []byte {
	return append(append(planPolygonCountData(int32(len(points))), polygonPointBytes(points)...), color)
}

func TestPlanNextMarkerPrefixAllBytesAndTails(t *testing.T) {
	for _, points := range [][][2]int32{nil, {{66051, -66051}}, {{math.MinInt32, math.MaxInt32}, {-1, 0}, {16777217, -16777217}}} {
		for value := 0; value < 256; value++ {
			// Complementary values distinguish marker from color and cover every raw byte.
			color := nextMarkerColorData(points, byte(255-value))
			base, err := top.ReadV3PlanPolygonColorPrefix(color)
			if err != nil {
				t.Fatal(err)
			}
			assertPolygonPoints(t, mustPoints(t, color), points, 45)
			prefix := append(bytes.Clone(color), byte(value))
			for _, tail := range [][]byte{nil, {0}, {1}, {3}, {255, 128}, planMappingRecord()} {
				data := append(bytes.Clone(prefix), tail...)
				before := bytes.Clone(data)
				p, err := top.ReadV3PlanNextMarkerPrefix(data)
				if err != nil {
					t.Fatal(err)
				}
				assertNextMarkerBase(t, p, base)
				if p.NextMarkerRaw() != byte(value) || p.Offsets().NextMarker != (source.Span{Start: len(color), End: len(prefix)}) || p.ConsumedOffset() != len(prefix) || p.UnparsedTailSize() != len(tail) || !bytes.Equal(p.Bytes(), prefix) || !bytes.Equal(data, before) {
					t.Fatalf("marker %d: raw byte/span/stop/tail changed", value)
				}
			}
			requirePlanNextMarkerError(t, color, top.DefaultPolygonCountLimits(), "truncated", "plan.elements[1].kind", len(color))
			if base.ConsumedOffset() != len(color) {
				t.Fatal("P04c4 stop changed")
			}
		}
	}
	// Independently cover all colors for known, terminal and unknown markers.
	for color := 0; color < 256; color++ {
		for _, marker := range []byte{0, 1, 3, 128, 255} {
			p, err := top.ReadV3PlanNextMarkerPrefix(append(nextMarkerColorData(nil, byte(color)), marker))
			if err != nil || p.ColorRaw() != byte(color) || p.NextMarkerRaw() != marker || p.ConsumedOffset() != 47 {
				t.Fatal("color or marker normalized")
			}
		}
	}
}

func mustPoints(t *testing.T, data []byte) source.PlanPolygonPointsPrefix {
	t.Helper()
	p, err := top.ReadV3PlanPolygonPointsPrefix(data)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestPlanNextMarkerPrefixInheritedErrorsAndLimits(t *testing.T) {
	data := append(nextMarkerColorData([][2]int32{{1, -2}, {3, -4}, {5, -6}}, 255), 3)
	for n := 0; n < len(data); n++ {
		if n == len(data)-1 {
			requirePlanNextMarkerError(t, data[:n], top.DefaultPolygonCountLimits(), "truncated", "plan.elements[1].kind", n)
			continue
		}
		_, earlier := top.ReadV3PlanPolygonColorPrefix(data[:n])
		p, err := top.ReadV3PlanNextMarkerPrefix(data[:n])
		if earlier == nil || !reflect.DeepEqual(err, earlier) || !reflect.DeepEqual(p, source.PlanNextMarkerPrefix{}) {
			t.Fatalf("inherited error at %d changed", n)
		}
	}
	for marker := 0; marker < 256; marker++ {
		if marker != 1 {
			requirePlanNextMarkerError(t, append(planMarkerBase(), byte(marker)), top.DefaultPolygonCountLimits(), "unsupported_element", "plan.elements[0].kind", 40)
		}
	}
	for _, count := range []int32{-1, math.MinInt32, 1000001, math.MaxInt32} {
		code := "resource_limit"
		if count < 0 {
			code = "negative_count"
		}
		requirePlanNextMarkerError(t, planPolygonCountData(count), top.DefaultPolygonCountLimits(), code, "plan.elements[0].point_count", 41)
	}
	for _, bad := range []int{-1, 1000001, math.MaxInt} {
		limits := top.DefaultPolygonCountLimits()
		limits.MaxPoints, limits.MaxReferences = bad, -1
		requirePlanNextMarkerError(t, nil, limits, "invalid_limit", "limits.max_points", 0)
	}
	for _, n := range []int{0, 1, 3, 1000000} {
		limits := top.DefaultPolygonCountLimits()
		limits.MaxPoints = n
		data := append(append(planPolygonCountData(int32(n)), make([]byte, 8*n)...), 255, 0)
		limits.MaxInputBytes = len(data)
		p, err := top.ReadV3PlanNextMarkerPrefixWithLimits(data, limits)
		if err != nil || len(p.Points()) != n || p.ConsumedOffset() != len(data) {
			t.Fatalf("exact bound %d: %v", n, err)
		}
		requirePlanNextMarkerError(t, append(data, 128), limits, "resource_limit", "input", 0)
		requirePlanNextMarkerError(t, planPolygonCountData(int32(n+1)), limits, "resource_limit", "plan.elements[0].point_count", 41)
	}
	limits := top.PolygonCountLimits{}
	requirePlanNextMarkerError(t, nil, limits, "truncated", "header", 0)
	limits.MaxInputBytes = 47
	if _, err := top.ReadV3PlanNextMarkerPrefixWithLimits(append(nextMarkerColorData(nil, 0), 0), limits); err != nil {
		t.Fatal(err)
	}
	TestPlanPolygonColorPrefixTruncationAndInheritedErrors(t)
}

func TestPlanNextMarkerPrefixVariableTablesAndCopies(t *testing.T) {
	data := binary.LittleEndian.AppendUint32(fixture(1, record(17, "trip", -32768)), 1)
	data = append(data, measurementRecord(255, "abc")...)
	data = binary.LittleEndian.AppendUint32(data, 1)
	data = append(data, referenceRecord("Aą")...)
	data = append(data, planMappingRecord()...)
	data = append(data, planMappingBase()[16:]...)
	data = binary.LittleEndian.AppendUint32(append(data, 1), 2)
	data = append(data, polygonPointBytes([][2]int32{{-16777217, 66051}, {123, -456}})...)
	data = append(data, 129, 255, 128, 0)
	before := bytes.Clone(data)
	p, err := top.ReadV3PlanNextMarkerPrefix(data)
	if err != nil {
		t.Fatal(err)
	}
	base, _ := top.ReadV3PlanPolygonColorPrefix(data)
	assertNextMarkerBase(t, p, base)
	if p.Offsets().NextMarker != (source.Span{Start: 129, End: 130}) {
		t.Fatal("hardcoded marker position")
	}
	for _, restrict := range []func(*top.PolygonCountLimits){func(l *top.PolygonCountLimits) { l.MaxTrips = 0 }, func(l *top.PolygonCountLimits) { l.MaxMeasurements = 0 }, func(l *top.PolygonCountLimits) { l.MaxReferences = 0 }, func(l *top.PolygonCountLimits) { l.MaxCommentBytes = 3 }, func(l *top.PolygonCountLimits) { l.MaxPoints = 1 }, func(l *top.PolygonCountLimits) { l.MaxReferences = -1 }} {
		limits := top.DefaultPolygonCountLimits()
		restrict(&limits)
		_, earlier := top.ReadV3PlanPolygonColorPrefixWithLimits(data, limits)
		q, err := top.ReadV3PlanNextMarkerPrefixWithLimits(data, limits)
		if earlier == nil || !reflect.DeepEqual(err, earlier) || !reflect.DeepEqual(q, source.PlanNextMarkerPrefix{}) {
			t.Fatal("inherited limit/error changed")
		}
	}
	points, raw, trips, shots, refs, offsets := p.Points(), p.Bytes(), p.Trips(), p.Measurements(), p.References(), p.Offsets()
	for i := range data {
		data[i] = 0
	}
	points[0], raw[129], trips[0], shots[0], refs[0], offsets.NextMarker.Start = source.PolygonPoint{}, 0, source.Trip{}, source.Measurement{}, source.Reference{}, 0
	assertNextMarkerBase(t, p, base)
	if p.NextMarkerRaw() != 255 || p.Offsets().NextMarker != (source.Span{Start: 129, End: 130}) || !bytes.Equal(p.Bytes(), before[:130]) || p.ConsumedOffset() != 130 || p.UnparsedTailSize() != 2 {
		t.Fatal("mutable alias or lost marker")
	}
}

func TestPlanNextMarkerPrefixNativeFixture(t *testing.T) {
	data, err := os.ReadFile("testdata/api-drawings.top")
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(data)) != "4a494ead03cade750f670d50aa38661abda9b3e1f131d67f27a252982752c2a5" {
		t.Fatal("native fixture changed")
	}
	base, _ := top.ReadV3PlanPolygonColorPrefix(data)
	for _, tail := range [][]byte{nil, data[177:], {255, 128}} {
		p, err := top.ReadV3PlanNextMarkerPrefix(append(bytes.Clone(data[:177]), tail...))
		if err != nil {
			t.Fatal(err)
		}
		assertNextMarkerBase(t, p, base)
		if p.NextMarkerRaw() != 1 || p.Offsets().NextMarker != (source.Span{Start: 176, End: 177}) || p.ConsumedOffset() != 177 || p.UnparsedTailSize() != len(tail) || !bytes.Equal(p.Bytes(), data[:177]) {
			t.Fatal("native marker/position changed")
		}
	}
	requirePlanNextMarkerError(t, data[:176], top.DefaultPolygonCountLimits(), "truncated", "plan.elements[1].kind", 176)
}

func FuzzReadV3PlanNextMarkerPrefix(f *testing.F) {
	for _, seed := range [][]byte{nil, {255}, planPolygonCountData(-1), planPolygonCountData(1000000)} {
		f.Add(seed)
	}
	for _, n := range []int{0, 1, 3} {
		points := [][2]int32{{66051, -66051}, {math.MinInt32, math.MaxInt32}, {16777217, -16777217}}[:n]
		for _, marker := range []byte{0, 1, 3, 128, 255} {
			color := nextMarkerColorData(points, 255-marker)
			f.Add(color)
			f.Add(append(bytes.Clone(color), marker))
			f.Add(append(bytes.Clone(color), marker, 255, 128))
		}
	}
	limits := top.PolygonCountLimits{ReferenceLimits: top.ReferenceLimits{MeasurementLimits: top.MeasurementLimits{Limits: top.Limits{MaxInputBytes: 4096, MaxTrips: 32, MaxCommentBytes: 256}, MaxMeasurements: 32}, MaxReferences: 32}, MaxPoints: 32}
	f.Fuzz(func(t *testing.T, data []byte) {
		before := bytes.Clone(data)
		p, err := top.ReadV3PlanNextMarkerPrefixWithLimits(data, limits)
		q, again := top.ReadV3PlanNextMarkerPrefixWithLimits(data, limits)
		if !bytes.Equal(data, before) || !reflect.DeepEqual(p, q) || !reflect.DeepEqual(err, again) {
			t.Fatal("mutation/nondeterminism")
		}
		base, earlier := top.ReadV3PlanPolygonColorPrefixWithLimits(data, limits)
		if earlier != nil {
			if !reflect.DeepEqual(err, earlier) || !reflect.DeepEqual(p, source.PlanNextMarkerPrefix{}) {
				t.Fatal("inherited error changed")
			}
			return
		}
		start := base.ConsumedOffset()
		if start == len(data) {
			requirePlanNextMarkerError(t, data, limits, "truncated", "plan.elements[1].kind", start)
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		assertNextMarkerBase(t, p, base)
		if p.NextMarkerRaw() != data[start] || p.Offsets().NextMarker != (source.Span{Start: start, End: start + 1}) || p.ConsumedOffset() != start+1 || p.ConsumedOffset()+p.UnparsedTailSize() != len(data) || !bytes.Equal(p.Bytes(), data[:start+1]) {
			t.Fatal("raw marker/span/one-byte accounting changed")
		}
	})
}
