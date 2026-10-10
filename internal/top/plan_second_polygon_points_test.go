package top_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
	"os"
	"pockettopo-exporter/internal/source"
	"pockettopo-exporter/internal/top"
	"reflect"
	"runtime"
	"testing"
)

func requirePlanSecondPolygonPointsError(t *testing.T, data []byte, limits top.PolygonCountLimits, code, field string, offset int) {
	t.Helper()
	before := bytes.Clone(data)
	p, err := top.ReadV3PlanSecondPolygonPointsPrefixWithLimits(data, limits)
	var pe *top.ParseError
	if !errors.As(err, &pe) || pe.Code != code || pe.Field != field || pe.Offset != offset || !reflect.DeepEqual(p, source.PlanSecondPolygonPointsPrefix{}) || !bytes.Equal(before, data) {
		t.Fatalf("wrong/partial failure: %+v %v", p, err)
	}
}

func assertSecondPolygonPoints(t *testing.T, p source.PlanSecondPolygonPointsPrefix, want [][2]int32, start int) {
	t.Helper()
	points := p.SecondPoints()
	if len(points) != len(want) {
		t.Fatalf("point count %d != %d", len(points), len(want))
	}
	for i, v := range want {
		off := start + 8*i
		spans := source.PolygonPointOffsets{Record: source.Span{Start: off, End: off + 8}, X: source.Span{Start: off, End: off + 4}, Y: source.Span{Start: off + 4, End: off + 8}}
		if points[i].XRaw() != v[0] || points[i].YRaw() != v[1] || points[i].Offsets() != spans {
			t.Fatalf("point %d: %+v, want %v %+v", i, points[i], v, spans)
		}
	}
	if p.Offsets().SecondPoints != (source.Span{Start: start, End: start + 8*len(want)}) || p.ConsumedOffset() != start+8*len(want) {
		t.Fatal("point table span/stop changed")
	}
}

func TestPlanSecondPolygonPointsPrefixValuesAndTails(t *testing.T) {
	for _, want := range [][][2]int32{nil, {{0, 0}}, {{math.MinInt32, math.MaxInt32}}, {{-4000, -2000}, {-3500, -1000}, {-2500, -1500}}, {{66051, -16777217}, {16777217, -1}, {math.MaxInt32, math.MinInt32}}} {
		prefix := append(secondPointsCountData(int32(len(want))), polygonPointBytes(want)...)
		tails := [][]byte{nil, {0, 1, 3, 255, 128}, planMappingRecord()}
		for b := 0; b < 256; b++ {
			tails = append(tails, []byte{byte(b)})
		}
		for _, tail := range tails {
			data := append(bytes.Clone(prefix), tail...)
			before := bytes.Clone(data)
			p, err := top.ReadV3PlanSecondPolygonPointsPrefix(data)
			if err != nil {
				t.Fatal(err)
			}
			assertSecondPolygonPoints(t, p, want, 51)
			base, err := top.ReadV3PlanSecondPolygonCountPrefix(data)
			assertSecondPointsBase(t, p, base)
			if err != nil || p.SecondPointCountRaw() != int32(len(want)) || p.MarkerRaw() != 1 || p.Offsets().PlanSecondPolygonCountPrefixOffsets != base.Offsets() || p.OverviewMapping() != base.OverviewMapping() || p.PlanMapping() != base.PlanMapping() || p.Header() != base.Header() || p.Version() != 3 || p.TripCountRaw() != 0 || p.MeasurementCountRaw() != 0 || p.ReferenceCountRaw() != 0 || len(p.Trips()) != 0 || len(p.Measurements()) != 0 || len(p.References()) != 0 || p.UnparsedTailSize() != len(tail) || !bytes.Equal(p.Bytes(), prefix) || !bytes.Equal(data, before) {
				t.Fatal("inherited values, input or tail changed")
			}
		}
	}
	// Literal bytes establish signed little-endian X/Y independently of the encoder.
	p, err := top.ReadV3PlanSecondPolygonPointsPrefix(append(secondPointsCountData(1), 3, 2, 1, 0, 253, 253, 254, 255))
	if err != nil {
		t.Fatal(err)
	}
	assertSecondPolygonPoints(t, p, [][2]int32{{66051, -66051}}, 51)
}

func TestPlanSecondPolygonPointsPrefixTruncationAndPrecedence(t *testing.T) {
	data := append(secondPointsCountData(3), polygonPointBytes([][2]int32{{1, 2}, {3, 4}, {5, 6}})...)
	for n := 0; n < len(data); n++ {
		if n >= 51 {
			requirePlanSecondPolygonPointsError(t, data[:n], top.DefaultPolygonCountLimits(), "truncated", "plan.elements[1].points", 51)
			continue
		}
		_, earlier := top.ReadV3PlanSecondPolygonCountPrefix(data[:n])
		p, err := top.ReadV3PlanSecondPolygonPointsPrefix(data[:n])
		if earlier == nil || !reflect.DeepEqual(err, earlier) || !reflect.DeepEqual(p, source.PlanSecondPolygonPointsPrefix{}) {
			t.Fatalf("earlier error at %d changed", n)
		}
	}
	for _, count := range []int32{1, 3, 1000000} {
		requirePlanSecondPolygonPointsError(t, secondPointsCountData(count), top.DefaultPolygonCountLimits(), "truncated", "plan.elements[1].points", 51)
		p, err := top.ReadV3PlanSecondPolygonCountPrefix(secondPointsCountData(count))
		if err != nil || p.ConsumedOffset() != 51 {
			t.Fatal("P04c6 stop changed")
		}
	}
	for marker := 0; marker < 256; marker++ {
		if marker != 1 {
			requirePlanSecondPolygonPointsError(t, append(nextMarkerColorData(nil, 129), byte(marker)), top.DefaultPolygonCountLimits(), "unsupported_element", "plan.elements[1].kind", 46)
		}
	}
	for _, count := range []int32{-1, math.MinInt32, 1000001, math.MaxInt32} {
		code := "resource_limit"
		if count < 0 {
			code = "negative_count"
		}
		requirePlanSecondPolygonPointsError(t, secondPointsCountData(count), top.DefaultPolygonCountLimits(), code, "plan.elements[1].point_count", 47)
	}
	// Existing test exercises every earlier API's independent stop; no new payload required.
	TestPlanSecondPolygonCountPrefixMarkersAndTruncations(t)
	TestPlanSecondPolygonCountPrefixLimitsAndPrecedence(t)
}

func TestPlanSecondPolygonPointsPrefixLimits(t *testing.T) {
	for _, n := range []int{0, 1, 3, 1000000} {
		limits := top.DefaultPolygonCountLimits()
		limits.MaxPoints = n
		data := append(secondPointsCountData(int32(n)), make([]byte, 8*n)...)
		limits.MaxInputBytes = len(data)
		p, err := top.ReadV3PlanSecondPolygonPointsPrefixWithLimits(data, limits)
		if err != nil || len(p.SecondPoints()) != n || p.ConsumedOffset() != len(data) {
			t.Fatalf("exact limit %d: %v", n, err)
		}
		requirePlanSecondPolygonPointsError(t, append(data, 255), limits, "resource_limit", "input", 0)
		requirePlanSecondPolygonPointsError(t, secondPointsCountData(int32(n+1)), limits, "resource_limit", "plan.elements[1].point_count", 47)
	}
	for _, bad := range []int{-1, 1000001, math.MaxInt} {
		limits := top.DefaultPolygonCountLimits()
		limits.MaxPoints = bad
		limits.MaxReferences = -1
		requirePlanSecondPolygonPointsError(t, nil, limits, "invalid_limit", "limits.max_points", 0)
	}
	limits := top.PolygonCountLimits{}
	requirePlanSecondPolygonPointsError(t, nil, limits, "truncated", "header", 0)
	limits.MaxInputBytes = 51
	if _, err := top.ReadV3PlanSecondPolygonPointsPrefixWithLimits(secondPointsCountData(0), limits); err != nil {
		t.Fatal(err)
	}
	requirePlanSecondPolygonPointsError(t, secondPointsCountData(-1), limits, "negative_count", "plan.elements[1].point_count", 47)
	for _, bad := range [][]byte{nil, {255}, fixture(-1), fixture(1, record(-1, "", 0)), fixture(1, record(17, "A\xffB", 0)), measurementTable(-1), referenceTable(-1), referenceTable(1, referenceRecord("")[:24])} {
		_, earlier := top.ReadV3PlanSecondPolygonCountPrefix(bad)
		p, err := top.ReadV3PlanSecondPolygonPointsPrefix(bad)
		if earlier == nil || !reflect.DeepEqual(err, earlier) || !reflect.DeepEqual(p, source.PlanSecondPolygonPointsPrefix{}) {
			t.Fatal("inherited failure changed")
		}
	}
}

func TestPlanSecondPolygonPointsPrefixVariableTablesAndCopies(t *testing.T) {
	data := binary.LittleEndian.AppendUint32(fixture(1, record(17, "trip", -32768)), 1)
	data = append(data, measurementRecord(255, "abc")...)
	data = binary.LittleEndian.AppendUint32(data, 1)
	data = append(data, referenceRecord("Aą")...)
	data = append(data, planMappingRecord()...)
	data = append(data, planMappingBase()[16:]...)
	data = binary.LittleEndian.AppendUint32(append(data, 1), 2)
	want := [][2]int32{{-16777217, 66051}, {123, -456}}
	data = append(data, polygonPointBytes(want)...)
	data = binary.LittleEndian.AppendUint32(append(data, 129, 1), 2)
	data = append(data, polygonPointBytes(want)...)
	data = append(data, 255, 128)
	before := bytes.Clone(data)
	p, err := top.ReadV3PlanSecondPolygonPointsPrefix(data)
	if err != nil {
		t.Fatal(err)
	}
	assertSecondPolygonPoints(t, p, want, 134)
	base, _ := top.ReadV3PlanSecondPolygonCountPrefix(data)
	assertSecondPointsBase(t, p, base)
	if p.Offsets().PlanSecondPolygonCountPrefixOffsets != base.Offsets() || p.PlanMapping() != base.PlanMapping() || p.OverviewMapping() != base.OverviewMapping() {
		t.Fatal("inherited spans/mappings lost")
	}
	for _, restrict := range []func(*top.PolygonCountLimits){func(l *top.PolygonCountLimits) { l.MaxTrips = 0 }, func(l *top.PolygonCountLimits) { l.MaxMeasurements = 0 }, func(l *top.PolygonCountLimits) { l.MaxReferences = 0 }, func(l *top.PolygonCountLimits) { l.MaxCommentBytes = 3 }, func(l *top.PolygonCountLimits) { l.MaxPoints = 1 }} {
		limits := top.DefaultPolygonCountLimits()
		restrict(&limits)
		_, earlier := top.ReadV3PlanSecondPolygonCountPrefixWithLimits(data, limits)
		q, e := top.ReadV3PlanSecondPolygonPointsPrefixWithLimits(data, limits)
		if earlier == nil || !reflect.DeepEqual(e, earlier) || !reflect.DeepEqual(q, source.PlanSecondPolygonPointsPrefix{}) {
			t.Fatal("lower inherited bound lost")
		}
	}
	firstPoints := p.Points()
	firstPoints[0] = source.PolygonPoint{}
	points, raw, trips, shots, refs, off := p.SecondPoints(), p.Bytes(), p.Trips(), p.Measurements(), p.References(), p.Offsets()
	for i := range data {
		data[i] = 0
	}
	points[0], raw[134], trips[0], shots[0], refs[0], off.SecondPoints.Start = source.PolygonPoint{}, 0, source.Trip{}, source.Measurement{}, source.Reference{}, 0
	assertSecondPolygonPoints(t, p, want, 134)
	assertSecondPointsBase(t, p, base)
	if !bytes.Equal(p.Bytes(), before[:150]) || p.UnparsedTailSize() != 2 || p.Trips()[0].Comment() != "trip" || p.Measurements()[0].FlagsRaw() != 255 || p.Measurements()[0].Comment() != "abc" || p.References()[0].Comment() != "Aą" {
		t.Fatal("mutable aliases or lost records")
	}
}

func TestPlanSecondPolygonPointsPrefixNativeFixture(t *testing.T) {
	data, err := os.ReadFile("testdata/api-drawings.top")
	if err != nil {
		t.Fatal(err)
	}
	// Pinned original helper literal points, verified by the separate native scalar probe.
	want := [][2]int32{{-4000, -2000}, {-3500, -1000}, {-2500, -1500}}
	for _, tail := range [][]byte{nil, data[205:], {255, 128}} {
		p, err := top.ReadV3PlanSecondPolygonPointsPrefix(append(bytes.Clone(data[:205]), tail...))
		if err != nil {
			t.Fatal(err)
		}
		assertSecondPolygonPoints(t, p, want, 181)
		if p.UnparsedTailSize() != len(tail) || !bytes.Equal(p.Bytes(), data[:205]) {
			t.Fatal("native stop changed")
		}
	}
	for n := 181; n < 205; n++ {
		requirePlanSecondPolygonPointsError(t, data[:n], top.DefaultPolygonCountLimits(), "truncated", "plan.elements[1].points", 181)
	}
}

func FuzzReadV3PlanSecondPolygonPointsPrefix(f *testing.F) {
	for _, seed := range [][]byte{secondPointsCountData(0), secondPointsCountData(3), secondPointsCountData(-1), secondPointsCountData(1000000), append(secondPointsCountData(1), 3, 2, 1, 0, 253, 253, 254, 255), append(secondPointsCountData(2), polygonPointBytes([][2]int32{{1, -2}, {math.MinInt32, math.MaxInt32}})...), {255}} {
		f.Add(seed)
	}
	for _, first := range [][][2]int32{nil, {{1, -2}}, {{66051, -66051}, {-1, 0}, {16777217, -16777217}}} {
		for _, second := range [][][2]int32{nil, {{3, -4}}, {{-1, 0}, {-2147483648, 2147483647}, {16777217, -16777217}}} {
			data := append(secondPolygonCountData(first, int32(len(second))), polygonPointBytes(second)...)
			f.Add(data)
			f.Add(append(bytes.Clone(data), 255, 128))
			for cut := 1; cut <= 8*len(second); cut++ {
				f.Add(data[:len(data)-cut])
			}
		}
	}
	native, e := os.ReadFile("testdata/api-drawings.top")
	if e != nil {
		f.Fatal(e)
	}
	f.Add(native)
	f.Add(native[:205])
	limits := top.PolygonCountLimits{ReferenceLimits: top.ReferenceLimits{MeasurementLimits: top.MeasurementLimits{Limits: top.Limits{MaxInputBytes: 4096, MaxTrips: 32, MaxCommentBytes: 256}, MaxMeasurements: 32}, MaxReferences: 32}, MaxPoints: 32}
	f.Fuzz(func(t *testing.T, data []byte) {
		before := bytes.Clone(data)
		p, err := top.ReadV3PlanSecondPolygonPointsPrefixWithLimits(data, limits)
		q, again := top.ReadV3PlanSecondPolygonPointsPrefixWithLimits(data, limits)
		if !bytes.Equal(data, before) || !reflect.DeepEqual(p, q) || !reflect.DeepEqual(err, again) {
			t.Fatal("mutation or nondeterminism")
		}
		base, earlier := top.ReadV3PlanSecondPolygonCountPrefixWithLimits(data, limits)
		if earlier != nil {
			if !reflect.DeepEqual(err, earlier) || !reflect.DeepEqual(p, source.PlanSecondPolygonPointsPrefix{}) {
				t.Fatal("inherited error changed")
			}
			return
		}
		start := base.ConsumedOffset()
		n := int(base.SecondPointCountRaw())
		if n*8 > len(data)-start {
			requirePlanSecondPolygonPointsError(t, data, limits, "truncated", "plan.elements[1].points", start)
			return
		}
		want := make([][2]int32, n)
		for i := range want {
			b := data[start+i*8 : start+i*8+8]
			want[i] = [2]int32{int32(uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24), int32(uint32(b[4]) | uint32(b[5])<<8 | uint32(b[6])<<16 | uint32(b[7])<<24)}
		}
		if err != nil {
			t.Fatal(err)
		}
		assertSecondPolygonPoints(t, p, want, start)
		assertSecondPointsBase(t, p, base)
		if p.SecondPointCountRaw() != base.SecondPointCountRaw() || p.Offsets().PlanSecondPolygonCountPrefixOffsets != base.Offsets() || p.PlanMapping() != base.PlanMapping() || p.OverviewMapping() != base.OverviewMapping() || p.ConsumedOffset()+p.UnparsedTailSize() != len(data) || !bytes.Equal(p.Bytes(), data[:p.ConsumedOffset()]) {
			t.Fatal("inherited metadata or tail accounting changed")
		}
	})
}

func secondPointsCountData(n int32) []byte { return secondPolygonCountData(nil, n) }

func assertSecondPointsBase(t *testing.T, p source.PlanSecondPolygonPointsPrefix, base source.PlanSecondPolygonCountPrefix) {
	t.Helper()
	if p.Header() != base.Header() || p.Version() != base.Version() || p.TripCountRaw() != base.TripCountRaw() || p.MeasurementCountRaw() != base.MeasurementCountRaw() || p.ReferenceCountRaw() != base.ReferenceCountRaw() || p.MarkerRaw() != base.MarkerRaw() || p.PointCountRaw() != base.PointCountRaw() || p.ColorRaw() != base.ColorRaw() || p.NextMarkerRaw() != base.NextMarkerRaw() || p.SecondPointCountRaw() != base.SecondPointCountRaw() || p.OverviewMapping() != base.OverviewMapping() || p.PlanMapping() != base.PlanMapping() || p.Offsets().PlanSecondPolygonCountPrefixOffsets != base.Offsets() || !reflect.DeepEqual(p.Trips(), base.Trips()) || !reflect.DeepEqual(p.Measurements(), base.Measurements()) || !reflect.DeepEqual(p.References(), base.References()) || !reflect.DeepEqual(p.Points(), base.Points()) {
		t.Fatal("inherited fields/records/spans changed")
	}
}

func TestPlanSecondPolygonPointsPrefixSeparateLimitsAndPreflightAllocation(t *testing.T) {
	limits := top.DefaultPolygonCountLimits()
	limits.MaxPoints = 3
	first := [][2]int32{{1, -2}, {3, -4}, {5, -6}}
	second := [][2]int32{{7, -8}, {9, -10}, {11, -12}}
	data := append(secondPolygonCountData(first, 3), polygonPointBytes(second)...)
	p, err := top.ReadV3PlanSecondPolygonPointsPrefixWithLimits(data, limits)
	if err != nil || len(p.Points()) != 3 || len(p.SecondPoints()) != 3 {
		t.Fatal("separate inclusive limits lost", err)
	}
	// Failed preflight must not allocate in proportion to the untrusted count.
	// Measure allocated bytes, since allocation counts alone cannot distinguish
	// an early one-point allocation from an early million-point allocation.
	allocatedBytes := func(n int32) uint64 {
		data := secondPolygonCountData(nil, n)
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		for i := 0; i < 4; i++ {
			requirePlanSecondPolygonPointsError(t, data, top.DefaultPolygonCountLimits(), "truncated", "plan.elements[1].points", 51)
		}
		runtime.ReadMemStats(&after)
		return after.TotalAlloc - before.TotalAlloc
	}
	small, large := allocatedBytes(1), allocatedBytes(1000000)
	if large > small+65536 {
		t.Fatalf("preflight allocated in proportion to count: %d vs %d bytes", large, small)
	}
}
