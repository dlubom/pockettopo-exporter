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

func thirdPolygonCountData(points [][2]int32, count int32) []byte {
	return binary.LittleEndian.AppendUint32(append(thirdCountColorData(points, points), 1), uint32(count))
}

func thirdCountColorData(first, second [][2]int32) []byte {
	data := append(secondPolygonCountData(first, int32(len(second))), polygonPointBytes(second)...)
	return append(data, 255)
}

func requireThirdPolygonCountError(t *testing.T, data []byte, limits top.PolygonCountLimits, code, field string, offset int) {
	t.Helper()
	before := bytes.Clone(data)
	p, err := top.ReadV3PlanThirdPolygonCountPrefixWithLimits(data, limits)
	var pe *top.ParseError
	if !errors.As(err, &pe) || pe.Code != code || pe.Field != field || pe.Offset != offset || !reflect.DeepEqual(p, source.PlanThirdPolygonCountPrefix{}) || !bytes.Equal(data, before) {
		t.Fatalf("wrong/partial failure: %+v %v; want %s at %s/%d", p, err, code, field, offset)
	}
}

func assertThirdCountBase(t *testing.T, p source.PlanThirdPolygonCountPrefix, base source.PlanFollowingMarkerPrefix) {
	t.Helper()
	if p.Header() != base.Header() || p.Version() != base.Version() || p.TripCountRaw() != base.TripCountRaw() || p.MeasurementCountRaw() != base.MeasurementCountRaw() || p.ReferenceCountRaw() != base.ReferenceCountRaw() || p.MarkerRaw() != base.MarkerRaw() || p.PointCountRaw() != base.PointCountRaw() || p.ColorRaw() != base.ColorRaw() || p.NextMarkerRaw() != base.NextMarkerRaw() || p.OverviewMapping() != base.OverviewMapping() || p.PlanMapping() != base.PlanMapping() || p.FollowingMarkerRaw() != base.FollowingMarkerRaw() || p.SecondColorRaw() != base.SecondColorRaw() || p.SecondPointCountRaw() != base.SecondPointCountRaw() || !reflect.DeepEqual(p.SecondPoints(), base.SecondPoints()) || p.Offsets().PlanFollowingMarkerPrefixOffsets != base.Offsets() || !reflect.DeepEqual(p.Trips(), base.Trips()) || !reflect.DeepEqual(p.Measurements(), base.Measurements()) || !reflect.DeepEqual(p.References(), base.References()) || !reflect.DeepEqual(p.Points(), base.Points()) {
		t.Fatal("inherited fields/records/spans changed")
	}
}

func TestPlanThirdPolygonCountPrefixCountsAndTails(t *testing.T) {
	for _, points := range [][][2]int32{nil, {{66051, -66051}}, {{math.MinInt32, math.MaxInt32}, {-1, 0}, {16777217, -16777217}}} {
		start := 53 + 16*len(points)
		for _, count := range []int32{0, 1, 3, 255, 256, 66051, 999999, 1000000} {
			prefix := thirdPolygonCountData(points, count)
			for _, tail := range [][]byte{nil, {0}, {1}, {3}, {255, 128}, planMappingRecord()} {
				data := append(bytes.Clone(prefix), tail...)
				before := bytes.Clone(data)
				p, err := top.ReadV3PlanThirdPolygonCountPrefix(data)
				if err != nil {
					t.Fatal(err)
				}
				base, _ := top.ReadV3PlanFollowingMarkerPrefix(data)
				assertThirdCountBase(t, p, base)
				if p.ThirdPointCountRaw() != count || p.Offsets().ThirdPointCount != (source.Span{Start: start, End: start + 4}) || p.ConsumedOffset() != start+4 || p.UnparsedTailSize() != len(tail) || !bytes.Equal(p.Bytes(), prefix) || !bytes.Equal(data, before) {
					t.Fatalf("count %d: raw/span/stop/tail changed", count)
				}
			}
		}
		for _, count := range []int32{math.MinInt32, -66051, -1, 1000001, math.MaxInt32} {
			code := "resource_limit"
			if count < 0 {
				code = "negative_count"
			}
			requireThirdPolygonCountError(t, thirdPolygonCountData(points, count), top.DefaultPolygonCountLimits(), code, "plan.elements[2].point_count", start)
		}
	}
	// Independent little-endian scalar bytes, also exercised by the native probe.
	p, err := top.ReadV3PlanThirdPolygonCountPrefix(append(append(thirdCountColorData(nil, nil), 1), 3, 2, 1, 0))
	if err != nil || p.ThirdPointCountRaw() != 66051 || p.SecondColorRaw() != 255 {
		t.Fatal("literal endian count or color changed")
	}
	for tail := 0; tail < 256; tail++ {
		p, err := top.ReadV3PlanThirdPolygonCountPrefix(append(thirdPolygonCountData(nil, 0), byte(tail)))
		if err != nil || p.ConsumedOffset() != 57 || p.UnparsedTailSize() != 1 {
			t.Fatal("tail interpreted")
		}
	}
}

func TestPlanThirdPolygonCountPrefixMarkersAndTruncations(t *testing.T) {
	for _, points := range [][][2]int32{nil, {{1, -2}}, {{1, -2}, {3, -4}, {5, -6}}} {
		color := thirdCountColorData(points, points)
		for marker := 0; marker < 256; marker++ {
			if marker == 1 {
				continue
			}
			for _, tail := range [][]byte{nil, {255}, {255, 255, 255, 255}, planMappingRecord()} {
				data := append(append(bytes.Clone(color), byte(marker)), tail...)
				requireThirdPolygonCountError(t, data, top.DefaultPolygonCountLimits(), "unsupported_element", "plan.elements[2].kind", len(color))
				base, err := top.ReadV3PlanFollowingMarkerPrefix(data[:len(color)+1])
				if err != nil || base.FollowingMarkerRaw() != byte(marker) || base.ConsumedOffset() != len(color)+1 {
					t.Fatal("P04c9 acceptance/stop changed")
				}
			}
		}
		for _, count := range []int32{0, 3, 1000000, -1} {
			data := thirdPolygonCountData(points, count)
			start := len(data) - 4
			for n := 0; n < len(data); n++ {
				if n >= start {
					requireThirdPolygonCountError(t, data[:n], top.DefaultPolygonCountLimits(), "truncated", "plan.elements[2].point_count", start)
					continue
				}
				_, earlier := top.ReadV3PlanFollowingMarkerPrefix(data[:n])
				p, err := top.ReadV3PlanThirdPolygonCountPrefix(data[:n])
				if earlier == nil || !reflect.DeepEqual(err, earlier) || !reflect.DeepEqual(p, source.PlanThirdPolygonCountPrefix{}) {
					t.Fatalf("inherited truncation %d changed", n)
				}
			}
		}
	}
	// Existing tests establish all earlier stops, validation, raw records and limits.
	TestPlanPolygonCountPrefixTruncationAndEarlierStops(t)
	TestPlanFollowingMarkerPrefixTruncationAndInheritedErrors(t)
	TestPlanFollowingMarkerPrefixAllValuesAndTails(t)
}

func TestPlanThirdPolygonCountPrefixLimitsAndPrecedence(t *testing.T) {
	for _, max := range []int{0, 1, 3, 32, 1000000} {
		limits := top.DefaultPolygonCountLimits()
		limits.MaxPoints = max
		// The same ceiling applies separately, rather than summing both counts.
		first := max
		if first > 3 {
			first = 3
		}
		points := [][2]int32{{1, -2}, {3, -4}, {5, -6}}[:first]
		data := thirdPolygonCountData(points, int32(max))
		limits.MaxInputBytes = len(data)
		p, err := top.ReadV3PlanThirdPolygonCountPrefixWithLimits(data, limits)
		if err != nil || p.ThirdPointCountRaw() != int32(max) || p.PointCountRaw() != int32(first) {
			t.Fatalf("inclusive/separate ceiling %d: %v", max, err)
		}
		start := len(data) - 4
		requireThirdPolygonCountError(t, thirdPolygonCountData(points, int32(max+1)), limits, "resource_limit", "plan.elements[2].point_count", start)
		requireThirdPolygonCountError(t, thirdPolygonCountData(points, -1), limits, "negative_count", "plan.elements[2].point_count", start)
		requireThirdPolygonCountError(t, append(data, 128), limits, "resource_limit", "input", 0)
	}
	limits := top.PolygonCountLimits{}
	limits.MaxInputBytes = 57
	if _, err := top.ReadV3PlanThirdPolygonCountPrefixWithLimits(thirdPolygonCountData(nil, 0), limits); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []int{-1, 1000001, math.MaxInt} {
		limits := top.DefaultPolygonCountLimits()
		limits.MaxPoints, limits.MaxReferences = bad, -1
		requireThirdPolygonCountError(t, nil, limits, "invalid_limit", "limits.max_points", 0)
	}
	for _, bad := range [][]byte{nil, {255}, fixture(-1), fixture(1, record(-1, "", 0)), fixture(1, record(17, "A\xffB", 0)), measurementTable(-1), referenceTable(-1), planMappingBase(), planPolygonCountData(-1), planPolygonCountData(1000001), planPolygonCountData(3), nextMarkerColorData(nil, 1)} {
		_, earlier := top.ReadV3PlanFollowingMarkerPrefix(bad)
		p, err := top.ReadV3PlanThirdPolygonCountPrefix(bad)
		if earlier == nil || !reflect.DeepEqual(err, earlier) || !reflect.DeepEqual(p, source.PlanThirdPolygonCountPrefix{}) {
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
			requireThirdPolygonCountError(t, nil, limits, "invalid_limit", "limits."+field, 0)
		}
	}
}

func TestPlanThirdPolygonCountPrefixVariableTablesAndCopies(t *testing.T) {
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
	data = binary.LittleEndian.AppendUint32(append(data, 255, 1), 66051)
	data = append(data, 255, 128)
	before := bytes.Clone(data)
	p, err := top.ReadV3PlanThirdPolygonCountPrefix(data)
	if err != nil {
		t.Fatal(err)
	}
	base, _ := top.ReadV3PlanFollowingMarkerPrefix(data)
	assertThirdCountBase(t, p, base)
	for _, restrict := range []func(*top.PolygonCountLimits){func(l *top.PolygonCountLimits) { l.MaxTrips = 0 }, func(l *top.PolygonCountLimits) { l.MaxMeasurements = 0 }, func(l *top.PolygonCountLimits) { l.MaxReferences = 0 }, func(l *top.PolygonCountLimits) { l.MaxCommentBytes = 3 }, func(l *top.PolygonCountLimits) { l.MaxPoints = 1 }} {
		limits := top.DefaultPolygonCountLimits()
		restrict(&limits)
		_, earlier := top.ReadV3PlanFollowingMarkerPrefixWithLimits(data, limits)
		q, err := top.ReadV3PlanThirdPolygonCountPrefixWithLimits(data, limits)
		if earlier == nil || !reflect.DeepEqual(err, earlier) || !reflect.DeepEqual(q, source.PlanThirdPolygonCountPrefix{}) {
			t.Fatal("lower inherited limit lost")
		}
	}
	secondPoints := p.SecondPoints()
	secondPoints[0] = source.PolygonPoint{}
	points, raw, trips, shots, refs, offsets := p.Points(), p.Bytes(), p.Trips(), p.Measurements(), p.References(), p.Offsets()
	for i := range data {
		data[i] = 0
	}
	points[0], raw[144], trips[0], shots[0], refs[0], offsets.ThirdPointCount.Start = source.PolygonPoint{}, 0, source.Trip{}, source.Measurement{}, source.Reference{}, 0
	assertThirdCountBase(t, p, base)
	if p.ThirdPointCountRaw() != 66051 || p.Offsets().ThirdPointCount != (source.Span{Start: 144, End: 148}) || p.ConsumedOffset() != 148 || p.UnparsedTailSize() != 2 || !bytes.Equal(p.Bytes(), before[:148]) {
		t.Fatal("copies/variable span lost")
	}
}

func TestPlanThirdPolygonCountPrefixNativeFixture(t *testing.T) {
	data, err := os.ReadFile("testdata/api-drawings.top")
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(data)) != "4a494ead03cade750f670d50aa38661abda9b3e1f131d67f27a252982752c2a5" {
		t.Fatal("native fixture changed")
	}
	base, _ := top.ReadV3PlanFollowingMarkerPrefix(data)
	// Pinned native helper: third brown Polygon has 3 points; scalar count is [207,211).
	for _, tail := range [][]byte{nil, data[211:], {255, 128}} {
		p, err := top.ReadV3PlanThirdPolygonCountPrefix(append(bytes.Clone(data[:211]), tail...))
		if err != nil {
			t.Fatal(err)
		}
		assertThirdCountBase(t, p, base)
		if p.ThirdPointCountRaw() != 3 || p.Offsets().ThirdPointCount != (source.Span{Start: 207, End: 211}) || p.ConsumedOffset() != 211 || p.UnparsedTailSize() != len(tail) || !bytes.Equal(p.Bytes(), data[:211]) {
			t.Fatal("native count/span/stop changed")
		}
	}
	for n := 207; n < 211; n++ {
		requireThirdPolygonCountError(t, data[:n], top.DefaultPolygonCountLimits(), "truncated", "plan.elements[2].point_count", 207)
	}
}

func TestPlanThirdPolygonCountPrefixNoThirdPointAllocation(t *testing.T) {
	allocations := func(data []byte) float64 {
		return testing.AllocsPerRun(20, func() {
			p, err := top.ReadV3PlanThirdPolygonCountPrefix(data)
			if err != nil || p.ConsumedOffset() != 57 {
				t.Fatal("requires third payload")
			}
		})
	}
	if allocations(thirdPolygonCountData(nil, 0)) != allocations(thirdPolygonCountData(nil, 1000000)) {
		t.Fatal("allocations depend on third count")
	}
}

func FuzzReadV3PlanThirdPolygonCountPrefix(f *testing.F) {
	for _, n := range []int{0, 1, 3} {
		points := [][2]int32{{66051, -66051}, {math.MinInt32, math.MaxInt32}, {16777217, -16777217}}[:n]
		for _, count := range []int32{math.MinInt32, -1, 0, 1, 3, 32, 33, 66051, 1000000, 1000001, math.MaxInt32} {
			data := thirdPolygonCountData(points, count)
			f.Add(data)
			f.Add(append(bytes.Clone(data), 255, 128))
			for cut := 1; cut <= 4; cut++ {
				f.Add(data[:len(data)-cut])
			}
		}
		for _, marker := range []byte{0, 3, 128, 255} {
			f.Add(append(thirdCountColorData(points, points), marker))
		}
	}
	f.Add([]byte(nil))
	f.Add(planPolygonCountData(-1))
	limits := top.PolygonCountLimits{ReferenceLimits: top.ReferenceLimits{MeasurementLimits: top.MeasurementLimits{Limits: top.Limits{MaxInputBytes: 4096, MaxTrips: 32, MaxCommentBytes: 256}, MaxMeasurements: 32}, MaxReferences: 32}, MaxPoints: 32}
	f.Fuzz(func(t *testing.T, data []byte) {
		before := bytes.Clone(data)
		p, err := top.ReadV3PlanThirdPolygonCountPrefixWithLimits(data, limits)
		q, again := top.ReadV3PlanThirdPolygonCountPrefixWithLimits(data, limits)
		if !bytes.Equal(data, before) || !reflect.DeepEqual(p, q) || !reflect.DeepEqual(err, again) {
			t.Fatal("mutation/nondeterminism")
		}
		base, earlier := top.ReadV3PlanFollowingMarkerPrefixWithLimits(data, limits)
		if earlier != nil {
			if !reflect.DeepEqual(err, earlier) || !reflect.DeepEqual(p, source.PlanThirdPolygonCountPrefix{}) {
				t.Fatal("inherited error changed")
			}
			return
		}
		code, field, start := "", "plan.elements[2].point_count", base.ConsumedOffset()
		var count int32
		switch {
		case base.FollowingMarkerRaw() != 1:
			code, field, start = "unsupported_element", "plan.elements[2].kind", base.Offsets().FollowingMarker.Start
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
			requireThirdPolygonCountError(t, data, limits, code, field, start)
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		assertThirdCountBase(t, p, base)
		if p.ThirdPointCountRaw() != count || p.Offsets().ThirdPointCount != (source.Span{Start: start, End: start + 4}) || p.ConsumedOffset() != start+4 || p.ConsumedOffset()+p.UnparsedTailSize() != len(data) || !bytes.Equal(p.Bytes(), data[:start+4]) {
			t.Fatal("raw count/span/four-byte accounting changed")
		}
	})
}

func TestPlanThirdPolygonCountPrefixIndependentTables(t *testing.T) {
	tables := [][][2]int32{nil, {{66051, -66051}}, {{-1, 0}, {16777217, -16777217}, {math.MinInt32, math.MaxInt32}}}
	for _, first := range tables {
		for _, second := range tables {
			for color := 0; color < 256; color++ {
				prefix := thirdCountColorData(first, second)
				prefix[45+8*len(first)] = byte(color)
				prefix[len(prefix)-1] = byte(255 - color)
				data := binary.LittleEndian.AppendUint32(append(prefix, 1), 66051)
				p, err := top.ReadV3PlanThirdPolygonCountPrefix(data)
				if err != nil {
					t.Fatal(err)
				}
				base, _ := top.ReadV3PlanFollowingMarkerPrefix(data)
				assertThirdCountBase(t, p, base)
				if p.ThirdPointCountRaw() != 66051 || p.Offsets().ThirdPointCount.Start != 53+8*(len(first)+len(second)) {
					t.Fatal("independent tables/count position")
				}
			}
		}
	}
	for _, bad := range [][]byte{secondPolygonCountData(nil, 3), append(secondPolygonCountData(nil, 0), 129), append(thirdCountColorData(nil, nil), 1)} {
		_, earlier := top.ReadV3PlanFollowingMarkerPrefix(bad)
		if earlier == nil {
			continue
		}
		p, err := top.ReadV3PlanThirdPolygonCountPrefix(bad)
		if !reflect.DeepEqual(err, earlier) || !reflect.DeepEqual(p, source.PlanThirdPolygonCountPrefix{}) {
			t.Fatal("inherited preflight/color/marker changed")
		}
	}
	// A lower ceiling must reject second points even when first and third are zero.
	limits := top.DefaultPolygonCountLimits()
	limits.MaxPoints = 0
	requireThirdPolygonCountError(t, binary.LittleEndian.AppendUint32(append(thirdCountColorData(nil, tables[1]), 1), 0), limits, "resource_limit", "plan.elements[1].point_count", 47)
}
