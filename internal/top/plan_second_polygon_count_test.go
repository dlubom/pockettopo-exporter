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

func secondPolygonCountData(points [][2]int32, count int32) []byte {
	return binary.LittleEndian.AppendUint32(append(nextMarkerColorData(points, 129), 1), uint32(count))
}

func requireSecondPolygonCountError(t *testing.T, data []byte, limits top.PolygonCountLimits, code, field string, offset int) {
	t.Helper()
	before := bytes.Clone(data)
	p, err := top.ReadV3PlanSecondPolygonCountPrefixWithLimits(data, limits)
	var pe *top.ParseError
	if !errors.As(err, &pe) || pe.Code != code || pe.Field != field || pe.Offset != offset || !reflect.DeepEqual(p, source.PlanSecondPolygonCountPrefix{}) || !bytes.Equal(data, before) {
		t.Fatalf("wrong/partial failure: %+v %v; want %s at %s/%d", p, err, code, field, offset)
	}
}

func assertSecondCountBase(t *testing.T, p source.PlanSecondPolygonCountPrefix, base source.PlanNextMarkerPrefix) {
	t.Helper()
	if p.Header() != base.Header() || p.Version() != base.Version() || p.TripCountRaw() != base.TripCountRaw() || p.MeasurementCountRaw() != base.MeasurementCountRaw() || p.ReferenceCountRaw() != base.ReferenceCountRaw() || p.MarkerRaw() != base.MarkerRaw() || p.PointCountRaw() != base.PointCountRaw() || p.ColorRaw() != base.ColorRaw() || p.NextMarkerRaw() != base.NextMarkerRaw() || p.OverviewMapping() != base.OverviewMapping() || p.PlanMapping() != base.PlanMapping() || p.Offsets().PlanNextMarkerPrefixOffsets != base.Offsets() || !reflect.DeepEqual(p.Trips(), base.Trips()) || !reflect.DeepEqual(p.Measurements(), base.Measurements()) || !reflect.DeepEqual(p.References(), base.References()) || !reflect.DeepEqual(p.Points(), base.Points()) {
		t.Fatal("inherited fields/records/spans changed")
	}
}

func TestPlanSecondPolygonCountPrefixCountsAndTails(t *testing.T) {
	for _, points := range [][][2]int32{nil, {{66051, -66051}}, {{math.MinInt32, math.MaxInt32}, {-1, 0}, {16777217, -16777217}}} {
		start := 47 + 8*len(points)
		for _, count := range []int32{0, 1, 3, 255, 256, 66051, 999999, 1000000} {
			prefix := secondPolygonCountData(points, count)
			for _, tail := range [][]byte{nil, {0}, {1}, {3}, {255, 128}, planMappingRecord()} {
				data := append(bytes.Clone(prefix), tail...)
				before := bytes.Clone(data)
				p, err := top.ReadV3PlanSecondPolygonCountPrefix(data)
				if err != nil {
					t.Fatal(err)
				}
				base, _ := top.ReadV3PlanNextMarkerPrefix(data)
				assertSecondCountBase(t, p, base)
				if p.SecondPointCountRaw() != count || p.Offsets().SecondPointCount != (source.Span{Start: start, End: start + 4}) || p.ConsumedOffset() != start+4 || p.UnparsedTailSize() != len(tail) || !bytes.Equal(p.Bytes(), prefix) || !bytes.Equal(data, before) {
					t.Fatalf("count %d: raw/span/stop/tail changed", count)
				}
			}
		}
		for _, count := range []int32{math.MinInt32, -66051, -1, 1000001, math.MaxInt32} {
			code := "resource_limit"
			if count < 0 {
				code = "negative_count"
			}
			requireSecondPolygonCountError(t, secondPolygonCountData(points, count), top.DefaultPolygonCountLimits(), code, "plan.elements[1].point_count", start)
		}
	}
	// Independent little-endian scalar bytes, also exercised by the native probe.
	p, err := top.ReadV3PlanSecondPolygonCountPrefix(append(append(nextMarkerColorData(nil, 255), 1), 3, 2, 1, 0))
	if err != nil || p.SecondPointCountRaw() != 66051 || p.ColorRaw() != 255 {
		t.Fatal("literal endian count or color changed")
	}
	for tail := 0; tail < 256; tail++ {
		p, err := top.ReadV3PlanSecondPolygonCountPrefix(append(secondPolygonCountData(nil, 0), byte(tail)))
		if err != nil || p.ConsumedOffset() != 51 || p.UnparsedTailSize() != 1 {
			t.Fatal("tail interpreted")
		}
	}
}

func TestPlanSecondPolygonCountPrefixMarkersAndTruncations(t *testing.T) {
	for _, points := range [][][2]int32{nil, {{1, -2}}, {{1, -2}, {3, -4}, {5, -6}}} {
		color := nextMarkerColorData(points, 255)
		for marker := 0; marker < 256; marker++ {
			if marker == 1 {
				continue
			}
			for _, tail := range [][]byte{nil, {255}, {255, 255, 255, 255}, planMappingRecord()} {
				data := append(append(bytes.Clone(color), byte(marker)), tail...)
				requireSecondPolygonCountError(t, data, top.DefaultPolygonCountLimits(), "unsupported_element", "plan.elements[1].kind", len(color))
				base, err := top.ReadV3PlanNextMarkerPrefix(data[:len(color)+1])
				if err != nil || base.NextMarkerRaw() != byte(marker) || base.ConsumedOffset() != len(color)+1 {
					t.Fatal("P04c5 acceptance/stop changed")
				}
			}
		}
		for _, count := range []int32{0, 3, 1000000, -1} {
			data := secondPolygonCountData(points, count)
			start := len(data) - 4
			for n := 0; n < len(data); n++ {
				if n >= start {
					requireSecondPolygonCountError(t, data[:n], top.DefaultPolygonCountLimits(), "truncated", "plan.elements[1].point_count", start)
					continue
				}
				_, earlier := top.ReadV3PlanNextMarkerPrefix(data[:n])
				p, err := top.ReadV3PlanSecondPolygonCountPrefix(data[:n])
				if earlier == nil || !reflect.DeepEqual(err, earlier) || !reflect.DeepEqual(p, source.PlanSecondPolygonCountPrefix{}) {
					t.Fatalf("inherited truncation %d changed", n)
				}
			}
		}
	}
	// Existing tests establish all earlier stops, validation, raw records and limits.
	TestPlanPolygonCountPrefixTruncationAndEarlierStops(t)
	TestPlanNextMarkerPrefixInheritedErrorsAndLimits(t)
	TestPlanNextMarkerPrefixAllBytesAndTails(t)
}

func TestPlanSecondPolygonCountPrefixLimitsAndPrecedence(t *testing.T) {
	for _, max := range []int{0, 1, 3, 32, 1000000} {
		limits := top.DefaultPolygonCountLimits()
		limits.MaxPoints = max
		// The same ceiling applies separately, rather than summing both counts.
		first := max
		if first > 3 {
			first = 3
		}
		points := [][2]int32{{1, -2}, {3, -4}, {5, -6}}[:first]
		data := secondPolygonCountData(points, int32(max))
		limits.MaxInputBytes = len(data)
		p, err := top.ReadV3PlanSecondPolygonCountPrefixWithLimits(data, limits)
		if err != nil || p.SecondPointCountRaw() != int32(max) || p.PointCountRaw() != int32(first) {
			t.Fatalf("inclusive/separate ceiling %d: %v", max, err)
		}
		start := len(data) - 4
		requireSecondPolygonCountError(t, secondPolygonCountData(points, int32(max+1)), limits, "resource_limit", "plan.elements[1].point_count", start)
		requireSecondPolygonCountError(t, secondPolygonCountData(points, -1), limits, "negative_count", "plan.elements[1].point_count", start)
		requireSecondPolygonCountError(t, append(data, 128), limits, "resource_limit", "input", 0)
	}
	limits := top.PolygonCountLimits{}
	limits.MaxInputBytes = 51
	if _, err := top.ReadV3PlanSecondPolygonCountPrefixWithLimits(secondPolygonCountData(nil, 0), limits); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []int{-1, 1000001, math.MaxInt} {
		limits := top.DefaultPolygonCountLimits()
		limits.MaxPoints, limits.MaxReferences = bad, -1
		requireSecondPolygonCountError(t, nil, limits, "invalid_limit", "limits.max_points", 0)
	}
	for _, bad := range [][]byte{nil, {255}, fixture(-1), fixture(1, record(-1, "", 0)), fixture(1, record(17, "A\xffB", 0)), measurementTable(-1), referenceTable(-1), planMappingBase(), planPolygonCountData(-1), planPolygonCountData(1000001), planPolygonCountData(3), nextMarkerColorData(nil, 1)} {
		_, earlier := top.ReadV3PlanNextMarkerPrefix(bad)
		p, err := top.ReadV3PlanSecondPolygonCountPrefix(bad)
		if earlier == nil || !reflect.DeepEqual(err, earlier) || !reflect.DeepEqual(p, source.PlanSecondPolygonCountPrefix{}) {
			t.Fatal("inherited error/precedence changed")
		}
	}
	for _, field := range []string{"max_references", "max_measurements", "max_trips", "max_input_bytes", "max_comment_bytes"} {
		for _, bad := range []int{-1, math.MaxInt} {
			limits := top.DefaultPolygonCountLimits()
			switch field {
			case "max_references":
				limits.MaxReferences = bad
			case "max_measurements":
				limits.MaxMeasurements = bad
			case "max_trips":
				limits.MaxTrips = bad
			case "max_input_bytes":
				limits.MaxInputBytes = bad
			case "max_comment_bytes":
				limits.MaxCommentBytes = bad
			}
			requireSecondPolygonCountError(t, nil, limits, "invalid_limit", "limits."+field, 0)
		}
	}
}

func TestPlanSecondPolygonCountPrefixVariableTablesAndCopies(t *testing.T) {
	data := binary.LittleEndian.AppendUint32(fixture(1, record(17, "trip", -32768)), 1)
	data = append(data, measurementRecord(255, "abc")...)
	data = binary.LittleEndian.AppendUint32(data, 1)
	data = append(data, referenceRecord("Aą")...)
	data = append(data, planMappingRecord()...)
	data = append(data, planMappingBase()[16:]...)
	data = binary.LittleEndian.AppendUint32(append(data, 1), 2)
	data = append(data, polygonPointBytes([][2]int32{{-16777217, 66051}, {123, -456}})...)
	data = binary.LittleEndian.AppendUint32(append(data, 129, 1), 66051)
	data = append(data, 255, 128)
	before := bytes.Clone(data)
	p, err := top.ReadV3PlanSecondPolygonCountPrefix(data)
	if err != nil {
		t.Fatal(err)
	}
	base, _ := top.ReadV3PlanNextMarkerPrefix(data)
	assertSecondCountBase(t, p, base)
	for _, restrict := range []func(*top.PolygonCountLimits){func(l *top.PolygonCountLimits) { l.MaxTrips = 0 }, func(l *top.PolygonCountLimits) { l.MaxMeasurements = 0 }, func(l *top.PolygonCountLimits) { l.MaxReferences = 0 }, func(l *top.PolygonCountLimits) { l.MaxCommentBytes = 3 }, func(l *top.PolygonCountLimits) { l.MaxPoints = 1 }} {
		limits := top.DefaultPolygonCountLimits()
		restrict(&limits)
		_, earlier := top.ReadV3PlanNextMarkerPrefixWithLimits(data, limits)
		q, err := top.ReadV3PlanSecondPolygonCountPrefixWithLimits(data, limits)
		if earlier == nil || !reflect.DeepEqual(err, earlier) || !reflect.DeepEqual(q, source.PlanSecondPolygonCountPrefix{}) {
			t.Fatal("lower inherited limit lost")
		}
	}
	points, raw, trips, shots, refs, offsets := p.Points(), p.Bytes(), p.Trips(), p.Measurements(), p.References(), p.Offsets()
	for i := range data {
		data[i] = 0
	}
	points[0], raw[130], trips[0], shots[0], refs[0], offsets.SecondPointCount.Start = source.PolygonPoint{}, 0, source.Trip{}, source.Measurement{}, source.Reference{}, 0
	assertSecondCountBase(t, p, base)
	if p.SecondPointCountRaw() != 66051 || p.Offsets().SecondPointCount != (source.Span{Start: 130, End: 134}) || p.ConsumedOffset() != 134 || p.UnparsedTailSize() != 2 || !bytes.Equal(p.Bytes(), before[:134]) {
		t.Fatal("copies/variable span lost")
	}
}

func TestPlanSecondPolygonCountPrefixNativeFixture(t *testing.T) {
	data, err := os.ReadFile("testdata/api-drawings.top")
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(data)) != "4a494ead03cade750f670d50aa38661abda9b3e1f131d67f27a252982752c2a5" {
		t.Fatal("native fixture changed")
	}
	base, _ := top.ReadV3PlanNextMarkerPrefix(data)
	// Pinned native helper: second gray Polygon has 3 points; scalar count is [177,181).
	for _, tail := range [][]byte{nil, data[181:], {255, 128}} {
		p, err := top.ReadV3PlanSecondPolygonCountPrefix(append(bytes.Clone(data[:181]), tail...))
		if err != nil {
			t.Fatal(err)
		}
		assertSecondCountBase(t, p, base)
		if p.SecondPointCountRaw() != 3 || p.Offsets().SecondPointCount != (source.Span{Start: 177, End: 181}) || p.ConsumedOffset() != 181 || p.UnparsedTailSize() != len(tail) || !bytes.Equal(p.Bytes(), data[:181]) {
			t.Fatal("native count/span/stop changed")
		}
	}
	for n := 177; n < 181; n++ {
		requireSecondPolygonCountError(t, data[:n], top.DefaultPolygonCountLimits(), "truncated", "plan.elements[1].point_count", 177)
	}
}

func TestPlanSecondPolygonCountPrefixNoSecondPointAllocation(t *testing.T) {
	allocations := func(data []byte) float64 {
		return testing.AllocsPerRun(20, func() {
			p, err := top.ReadV3PlanSecondPolygonCountPrefix(data)
			if err != nil || p.ConsumedOffset() != 51 {
				t.Fatal("requires second payload")
			}
		})
	}
	if allocations(secondPolygonCountData(nil, 0)) != allocations(secondPolygonCountData(nil, 1000000)) {
		t.Fatal("allocations depend on second count")
	}
}

func FuzzReadV3PlanSecondPolygonCountPrefix(f *testing.F) {
	for _, n := range []int{0, 1, 3} {
		points := [][2]int32{{66051, -66051}, {math.MinInt32, math.MaxInt32}, {16777217, -16777217}}[:n]
		for _, count := range []int32{math.MinInt32, -1, 0, 1, 3, 32, 33, 66051, 1000000, 1000001, math.MaxInt32} {
			data := secondPolygonCountData(points, count)
			f.Add(data)
			f.Add(append(bytes.Clone(data), 255, 128))
			for cut := 1; cut <= 4; cut++ {
				f.Add(data[:len(data)-cut])
			}
		}
		for _, marker := range []byte{0, 3, 128, 255} {
			f.Add(append(nextMarkerColorData(points, 255-marker), marker))
		}
	}
	f.Add([]byte(nil))
	f.Add(planPolygonCountData(-1))
	limits := top.PolygonCountLimits{ReferenceLimits: top.ReferenceLimits{MeasurementLimits: top.MeasurementLimits{Limits: top.Limits{MaxInputBytes: 4096, MaxTrips: 32, MaxCommentBytes: 256}, MaxMeasurements: 32}, MaxReferences: 32}, MaxPoints: 32}
	f.Fuzz(func(t *testing.T, data []byte) {
		before := bytes.Clone(data)
		p, err := top.ReadV3PlanSecondPolygonCountPrefixWithLimits(data, limits)
		q, again := top.ReadV3PlanSecondPolygonCountPrefixWithLimits(data, limits)
		if !bytes.Equal(data, before) || !reflect.DeepEqual(p, q) || !reflect.DeepEqual(err, again) {
			t.Fatal("mutation/nondeterminism")
		}
		base, earlier := top.ReadV3PlanNextMarkerPrefixWithLimits(data, limits)
		if earlier != nil {
			if !reflect.DeepEqual(err, earlier) || !reflect.DeepEqual(p, source.PlanSecondPolygonCountPrefix{}) {
				t.Fatal("inherited error changed")
			}
			return
		}
		code, field, start := "", "plan.elements[1].point_count", base.ConsumedOffset()
		var count int32
		switch {
		case base.NextMarkerRaw() != 1:
			code, field, start = "unsupported_element", "plan.elements[1].kind", base.Offsets().NextMarker.Start
		case len(data)-start < 4:
			code = "truncated"
		default:
			count = int32(binary.LittleEndian.Uint32(data[start : start+4]))
			if count < 0 {
				code = "negative_count"
			} else if int(count) > limits.MaxPoints {
				code = "resource_limit"
			}
		}
		if code != "" {
			requireSecondPolygonCountError(t, data, limits, code, field, start)
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		assertSecondCountBase(t, p, base)
		if p.SecondPointCountRaw() != count || p.Offsets().SecondPointCount != (source.Span{Start: start, End: start + 4}) || p.ConsumedOffset() != start+4 || p.ConsumedOffset()+p.UnparsedTailSize() != len(data) || !bytes.Equal(p.Bytes(), data[:start+4]) {
			t.Fatal("raw count/span/four-byte accounting changed")
		}
	})
}
