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
	"testing"
)

func polygonPointBytes(points [][2]int32) []byte {
	var data []byte
	for _, p := range points {
		data = binary.LittleEndian.AppendUint32(data, uint32(p[0]))
		data = binary.LittleEndian.AppendUint32(data, uint32(p[1]))
	}
	return data
}

func requirePlanPolygonPointsError(t *testing.T, data []byte, limits top.PolygonCountLimits, code, field string, offset int) {
	t.Helper()
	before := bytes.Clone(data)
	p, err := top.ReadV3PlanPolygonPointsPrefixWithLimits(data, limits)
	var pe *top.ParseError
	if !errors.As(err, &pe) || pe.Code != code || pe.Field != field || pe.Offset != offset || !reflect.DeepEqual(p, source.PlanPolygonPointsPrefix{}) || !bytes.Equal(before, data) {
		t.Fatalf("wrong/partial failure: %+v %v", p, err)
	}
}

func assertPolygonPoints(t *testing.T, p source.PlanPolygonPointsPrefix, want [][2]int32, start int) {
	t.Helper()
	points := p.Points()
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
	if p.Offsets().Points != (source.Span{Start: start, End: start + 8*len(want)}) || p.ConsumedOffset() != start+8*len(want) {
		t.Fatal("point table span/stop changed")
	}
}

func TestPlanPolygonPointsPrefixValuesAndTails(t *testing.T) {
	for _, want := range [][][2]int32{nil, {{0, 0}}, {{math.MinInt32, math.MaxInt32}}, {{-6000, -2000}, {-5500, -1000}, {-4500, -1500}}, {{66051, -16777217}, {16777217, -1}, {math.MaxInt32, math.MinInt32}}} {
		prefix := append(planPolygonCountData(int32(len(want))), polygonPointBytes(want)...)
		tails := [][]byte{nil, {0, 1, 3, 255, 128}, planMappingRecord()}
		for b := 0; b < 256; b++ {
			tails = append(tails, []byte{byte(b)})
		}
		for _, tail := range tails {
			data := append(bytes.Clone(prefix), tail...)
			before := bytes.Clone(data)
			p, err := top.ReadV3PlanPolygonPointsPrefix(data)
			if err != nil {
				t.Fatal(err)
			}
			assertPolygonPoints(t, p, want, 45)
			base, err := top.ReadV3PlanPolygonCountPrefix(data)
			if err != nil || p.PointCountRaw() != int32(len(want)) || p.MarkerRaw() != 1 || p.Offsets().PlanPolygonCountPrefixOffsets != base.Offsets() || p.OverviewMapping() != base.OverviewMapping() || p.PlanMapping() != base.PlanMapping() || p.Header() != base.Header() || p.Version() != 3 || p.TripCountRaw() != 0 || p.MeasurementCountRaw() != 0 || p.ReferenceCountRaw() != 0 || len(p.Trips()) != 0 || len(p.Measurements()) != 0 || len(p.References()) != 0 || p.UnparsedTailSize() != len(tail) || !bytes.Equal(p.Bytes(), prefix) || !bytes.Equal(data, before) {
				t.Fatal("inherited values, input or tail changed")
			}
		}
	}
	// Literal bytes establish signed little-endian X/Y independently of the encoder.
	p, err := top.ReadV3PlanPolygonPointsPrefix(append(planPolygonCountData(1), 3, 2, 1, 0, 253, 253, 254, 255))
	if err != nil {
		t.Fatal(err)
	}
	assertPolygonPoints(t, p, [][2]int32{{66051, -66051}}, 45)
}

func TestPlanPolygonPointsPrefixTruncationAndPrecedence(t *testing.T) {
	data := append(planPolygonCountData(3), polygonPointBytes([][2]int32{{1, 2}, {3, 4}, {5, 6}})...)
	for n := 0; n < len(data); n++ {
		if n >= 45 {
			requirePlanPolygonPointsError(t, data[:n], top.DefaultPolygonCountLimits(), "truncated", "plan.elements[0].points", 45)
			continue
		}
		_, earlier := top.ReadV3PlanPolygonCountPrefix(data[:n])
		p, err := top.ReadV3PlanPolygonPointsPrefix(data[:n])
		if earlier == nil || !reflect.DeepEqual(err, earlier) || !reflect.DeepEqual(p, source.PlanPolygonPointsPrefix{}) {
			t.Fatalf("earlier error at %d changed", n)
		}
	}
	for _, count := range []int32{1, 3, 1000000} {
		requirePlanPolygonPointsError(t, planPolygonCountData(count), top.DefaultPolygonCountLimits(), "truncated", "plan.elements[0].points", 45)
		p, err := top.ReadV3PlanPolygonCountPrefix(planPolygonCountData(count))
		if err != nil || p.ConsumedOffset() != 45 {
			t.Fatal("P04c2 stop changed")
		}
	}
	for marker := 0; marker < 256; marker++ {
		if marker != 1 {
			requirePlanPolygonPointsError(t, append(planMarkerBase(), byte(marker)), top.DefaultPolygonCountLimits(), "unsupported_element", "plan.elements[0].kind", 40)
		}
	}
	for _, count := range []int32{-1, math.MinInt32, 1000001, math.MaxInt32} {
		code := "resource_limit"
		if count < 0 {
			code = "negative_count"
		}
		requirePlanPolygonPointsError(t, planPolygonCountData(count), top.DefaultPolygonCountLimits(), code, "plan.elements[0].point_count", 41)
	}
	// Existing test exercises every earlier API's independent stop; no new payload required.
	TestPlanPolygonCountPrefixTruncationAndEarlierStops(t)
}

func TestPlanPolygonPointsPrefixLimits(t *testing.T) {
	for _, n := range []int{0, 1, 3, 1000000} {
		limits := top.DefaultPolygonCountLimits()
		limits.MaxPoints = n
		data := append(planPolygonCountData(int32(n)), make([]byte, 8*n)...)
		limits.MaxInputBytes = len(data)
		p, err := top.ReadV3PlanPolygonPointsPrefixWithLimits(data, limits)
		if err != nil || len(p.Points()) != n || p.ConsumedOffset() != len(data) {
			t.Fatalf("exact limit %d: %v", n, err)
		}
		requirePlanPolygonPointsError(t, append(data, 255), limits, "resource_limit", "input", 0)
		requirePlanPolygonPointsError(t, planPolygonCountData(int32(n+1)), limits, "resource_limit", "plan.elements[0].point_count", 41)
	}
	for _, bad := range []int{-1, 1000001, math.MaxInt} {
		limits := top.DefaultPolygonCountLimits()
		limits.MaxPoints = bad
		limits.MaxReferences = -1
		requirePlanPolygonPointsError(t, nil, limits, "invalid_limit", "limits.max_points", 0)
	}
	limits := top.PolygonCountLimits{}
	requirePlanPolygonPointsError(t, nil, limits, "truncated", "header", 0)
	limits.MaxInputBytes = 45
	if _, err := top.ReadV3PlanPolygonPointsPrefixWithLimits(planPolygonCountData(0), limits); err != nil {
		t.Fatal(err)
	}
	requirePlanPolygonPointsError(t, planPolygonCountData(-1), limits, "negative_count", "plan.elements[0].point_count", 41)
	for _, bad := range [][]byte{nil, {255}, fixture(-1), fixture(1, record(-1, "", 0)), fixture(1, record(17, "A\xffB", 0)), measurementTable(-1), referenceTable(-1), referenceTable(1, referenceRecord("")[:24])} {
		_, earlier := top.ReadV3PlanPolygonCountPrefix(bad)
		p, err := top.ReadV3PlanPolygonPointsPrefix(bad)
		if earlier == nil || !reflect.DeepEqual(err, earlier) || !reflect.DeepEqual(p, source.PlanPolygonPointsPrefix{}) {
			t.Fatal("inherited failure changed")
		}
	}
}

func TestPlanPolygonPointsPrefixVariableTablesAndCopies(t *testing.T) {
	data := binary.LittleEndian.AppendUint32(fixture(1, record(17, "trip", -32768)), 1)
	data = append(data, measurementRecord(255, "abc")...)
	data = binary.LittleEndian.AppendUint32(data, 1)
	data = append(data, referenceRecord("Aą")...)
	data = append(data, planMappingRecord()...)
	data = append(data, planMappingBase()[16:]...)
	data = binary.LittleEndian.AppendUint32(append(data, 1), 2)
	want := [][2]int32{{-16777217, 66051}, {123, -456}}
	data = append(data, polygonPointBytes(want)...)
	data = append(data, 255, 128)
	before := bytes.Clone(data)
	p, err := top.ReadV3PlanPolygonPointsPrefix(data)
	if err != nil {
		t.Fatal(err)
	}
	assertPolygonPoints(t, p, want, 112)
	base, _ := top.ReadV3PlanPolygonCountPrefix(data)
	if p.Offsets().PlanPolygonCountPrefixOffsets != base.Offsets() || p.PlanMapping() != base.PlanMapping() || p.OverviewMapping() != base.OverviewMapping() {
		t.Fatal("inherited spans/mappings lost")
	}
	for _, restrict := range []func(*top.PolygonCountLimits){func(l *top.PolygonCountLimits) { l.MaxTrips = 0 }, func(l *top.PolygonCountLimits) { l.MaxMeasurements = 0 }, func(l *top.PolygonCountLimits) { l.MaxReferences = 0 }, func(l *top.PolygonCountLimits) { l.MaxCommentBytes = 3 }, func(l *top.PolygonCountLimits) { l.MaxPoints = 1 }} {
		limits := top.DefaultPolygonCountLimits()
		restrict(&limits)
		_, earlier := top.ReadV3PlanPolygonCountPrefixWithLimits(data, limits)
		q, e := top.ReadV3PlanPolygonPointsPrefixWithLimits(data, limits)
		if earlier == nil || !reflect.DeepEqual(e, earlier) || !reflect.DeepEqual(q, source.PlanPolygonPointsPrefix{}) {
			t.Fatal("lower inherited bound lost")
		}
	}
	points, raw, trips, shots, refs, off := p.Points(), p.Bytes(), p.Trips(), p.Measurements(), p.References(), p.Offsets()
	for i := range data {
		data[i] = 0
	}
	points[0], raw[112], trips[0], shots[0], refs[0], off.Points.Start = source.PolygonPoint{}, 0, source.Trip{}, source.Measurement{}, source.Reference{}, 0
	assertPolygonPoints(t, p, want, 112)
	if !bytes.Equal(p.Bytes(), before[:128]) || p.UnparsedTailSize() != 2 || p.Trips()[0].Comment() != "trip" || p.Measurements()[0].FlagsRaw() != 255 || p.Measurements()[0].Comment() != "abc" || p.References()[0].Comment() != "Aą" {
		t.Fatal("mutable aliases or lost records")
	}
}

func TestPlanPolygonPointsPrefixNativeFixture(t *testing.T) {
	data, err := os.ReadFile("testdata/api-drawings.top")
	if err != nil {
		t.Fatal(err)
	}
	// Pinned original helper literal points, verified by the separate native scalar probe.
	want := [][2]int32{{-6000, -2000}, {-5500, -1000}, {-4500, -1500}}
	for _, tail := range [][]byte{nil, data[175:], {255, 128}} {
		p, err := top.ReadV3PlanPolygonPointsPrefix(append(bytes.Clone(data[:175]), tail...))
		if err != nil {
			t.Fatal(err)
		}
		assertPolygonPoints(t, p, want, 151)
		if p.UnparsedTailSize() != len(tail) || !bytes.Equal(p.Bytes(), data[:175]) {
			t.Fatal("native stop changed")
		}
	}
	for n := 151; n < 175; n++ {
		requirePlanPolygonPointsError(t, data[:n], top.DefaultPolygonCountLimits(), "truncated", "plan.elements[0].points", 151)
	}
}

func FuzzReadV3PlanPolygonPointsPrefix(f *testing.F) {
	for _, seed := range [][]byte{planPolygonCountData(0), planPolygonCountData(3), planPolygonCountData(-1), planPolygonCountData(1000000), append(planPolygonCountData(1), 3, 2, 1, 0, 253, 253, 254, 255), append(planPolygonCountData(2), polygonPointBytes([][2]int32{{1, -2}, {math.MinInt32, math.MaxInt32}})...), {255}} {
		f.Add(seed)
	}
	limits := top.PolygonCountLimits{ReferenceLimits: top.ReferenceLimits{MeasurementLimits: top.MeasurementLimits{Limits: top.Limits{MaxInputBytes: 4096, MaxTrips: 32, MaxCommentBytes: 256}, MaxMeasurements: 32}, MaxReferences: 32}, MaxPoints: 32}
	f.Fuzz(func(t *testing.T, data []byte) {
		before := bytes.Clone(data)
		p, err := top.ReadV3PlanPolygonPointsPrefixWithLimits(data, limits)
		q, again := top.ReadV3PlanPolygonPointsPrefixWithLimits(data, limits)
		if !bytes.Equal(data, before) || !reflect.DeepEqual(p, q) || !reflect.DeepEqual(err, again) {
			t.Fatal("mutation or nondeterminism")
		}
		base, earlier := top.ReadV3PlanPolygonCountPrefixWithLimits(data, limits)
		if earlier != nil {
			if !reflect.DeepEqual(err, earlier) || !reflect.DeepEqual(p, source.PlanPolygonPointsPrefix{}) {
				t.Fatal("inherited error changed")
			}
			return
		}
		start := base.ConsumedOffset()
		n := int(base.PointCountRaw())
		if n*8 > len(data)-start {
			requirePlanPolygonPointsError(t, data, limits, "truncated", "plan.elements[0].points", start)
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
		assertPolygonPoints(t, p, want, start)
		if p.PointCountRaw() != base.PointCountRaw() || p.Offsets().PlanPolygonCountPrefixOffsets != base.Offsets() || p.PlanMapping() != base.PlanMapping() || p.OverviewMapping() != base.OverviewMapping() || p.ConsumedOffset()+p.UnparsedTailSize() != len(data) || !bytes.Equal(p.Bytes(), data[:p.ConsumedOffset()]) {
			t.Fatal("inherited metadata or tail accounting changed")
		}
	})
}
