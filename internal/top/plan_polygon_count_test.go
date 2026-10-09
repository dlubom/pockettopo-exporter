package top_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"math"
	"os"
	"reflect"
	"testing"

	"pockettopo-exporter/internal/source"
	"pockettopo-exporter/internal/top"
)

func planPolygonCountData(count int32) []byte {
	return binary.LittleEndian.AppendUint32(append(planMarkerBase(), 1), uint32(count))
}

func requirePlanPolygonCountError(t *testing.T, data []byte, limits top.PolygonCountLimits, code, field string, offset int) {
	t.Helper()
	before := bytes.Clone(data)
	p, err := top.ReadV3PlanPolygonCountPrefixWithLimits(data, limits)
	var pe *top.ParseError
	if !errors.As(err, &pe) || pe.Code != code || pe.Field != field || pe.Offset != offset {
		t.Fatalf("got %v; want %s at %d (%s)", err, code, offset, field)
	}
	if !reflect.DeepEqual(p, source.PlanPolygonCountPrefix{}) || !bytes.Equal(data, before) {
		t.Fatal("partial result or input mutation on failure")
	}
}

func TestPlanPolygonCountPrefixCountsAndTails(t *testing.T) {
	for _, count := range []int32{0, 1, 3, 255, 256, 66051, 999999, 1000000} {
		prefix := planPolygonCountData(count)
		for _, tail := range [][]byte{nil, {0xff}, {0, 1, 3, 0xff, 0x80}, planMappingRecord()} {
			data := append(bytes.Clone(prefix), tail...)
			before := bytes.Clone(data)
			p, err := top.ReadV3PlanPolygonCountPrefix(data)
			if err != nil || p.PointCountRaw() != count || p.MarkerRaw() != 1 || p.Offsets().PointCount != (source.Span{Start: 41, End: 45}) || p.ConsumedOffset() != 45 || p.UnparsedTailSize() != len(tail) || !bytes.Equal(p.Bytes(), prefix) || !bytes.Equal(data, before) {
				t.Fatalf("count %d tail %x: %+v, %v", count, tail, p, err)
			}
			base, err := top.ReadV3PlanMarkerPrefix(data)
			if err != nil || p.OverviewMapping() != base.OverviewMapping() || p.PlanMapping() != base.PlanMapping() || p.Offsets().PlanMarkerPrefixOffsets != base.Offsets() || p.Header() != base.Header() || p.Version() != 3 || p.TripCountRaw() != 0 || p.MeasurementCountRaw() != 0 || p.ReferenceCountRaw() != 0 || len(p.Trips()) != 0 || len(p.Measurements()) != 0 || len(p.References()) != 0 {
				t.Fatal("count prefix changed earlier source data")
			}
		}
	}
	// Literal native-probed endian bytes, independent of AppendUint32 above.
	p, err := top.ReadV3PlanPolygonCountPrefix(append(append(planMarkerBase(), 1), 3, 2, 1, 0))
	if err != nil || p.PointCountRaw() != 66051 {
		t.Fatalf("literal endian count: %v", err)
	}
	for _, count := range []int32{math.MinInt32, -66051, -1} {
		requirePlanPolygonCountError(t, planPolygonCountData(count), top.DefaultPolygonCountLimits(), "negative_count", "plan.elements[0].point_count", 41)
	}
	for _, count := range []int32{1000001, math.MaxInt32} {
		requirePlanPolygonCountError(t, planPolygonCountData(count), top.DefaultPolygonCountLimits(), "resource_limit", "plan.elements[0].point_count", 41)
	}
}

func TestPlanPolygonCountPrefixUnsupportedMarkers(t *testing.T) {
	for marker := 0; marker < 256; marker++ {
		if marker == 1 {
			continue
		}
		for _, tail := range [][]byte{nil, {0xff}, {255, 255, 255, 255}, planMappingRecord()} {
			data := append(append(planMarkerBase(), byte(marker)), tail...)
			requirePlanPolygonCountError(t, data, top.DefaultPolygonCountLimits(), "unsupported_element", "plan.elements[0].kind", 40)
			earlier, err := top.ReadV3PlanMarkerPrefix(data)
			if err != nil || earlier.MarkerRaw() != byte(marker) || earlier.ConsumedOffset() != 41 || earlier.UnparsedTailSize() != len(tail) {
				t.Fatal("P04c1 marker acceptance or stop changed")
			}
		}
	}
}

func TestPlanPolygonCountPrefixTruncationAndEarlierStops(t *testing.T) {
	data := planPolygonCountData(3)
	for n := 0; n < 41; n++ {
		_, earlier := top.ReadV3PlanMarkerPrefix(data[:n])
		p, err := top.ReadV3PlanPolygonCountPrefix(data[:n])
		if earlier == nil || !reflect.DeepEqual(err, earlier) || !reflect.DeepEqual(p, source.PlanPolygonCountPrefix{}) {
			t.Fatalf("inherited truncation at %d changed: %v, %v", n, err, earlier)
		}
	}
	for _, count := range []int32{0, 3, 1000000, -1} {
		for n := 41; n < 45; n++ {
			requirePlanPolygonCountError(t, planPolygonCountData(count)[:n], top.DefaultPolygonCountLimits(), "truncated", "plan.elements[0].point_count", 41)
		}
	}
	if p, err := top.ReadV3TripPrefix(data[:8]); err != nil || p.ConsumedOffset() != 8 {
		t.Fatal("P03b stop changed")
	}
	if p, err := top.ReadV3MeasurementPrefix(data[:12]); err != nil || p.ConsumedOffset() != 12 {
		t.Fatal("P03c1 stop changed")
	}
	if p, err := top.ReadV3ReferencePrefix(data[:16]); err != nil || p.ConsumedOffset() != 16 {
		t.Fatal("P03c2 stop changed")
	}
	if p, err := top.ReadV3OverviewPrefix(data[:28]); err != nil || p.ConsumedOffset() != 28 {
		t.Fatal("P04a stop changed")
	}
	if p, err := top.ReadV3PlanMappingPrefix(data[:40]); err != nil || p.ConsumedOffset() != 40 {
		t.Fatal("P04b stop changed")
	}
	if p, err := top.ReadV3PlanMarkerPrefix(data[:41]); err != nil || p.ConsumedOffset() != 41 {
		t.Fatal("P04c1 stop changed")
	}
}

func TestPlanPolygonCountPrefixLimitsAndPrecedence(t *testing.T) {
	defaults := top.DefaultPolygonCountLimits()
	if defaults.MaxPoints != 1000000 || defaults.ReferenceLimits != top.DefaultReferenceLimits() {
		t.Fatal("defaults changed")
	}
	limits := defaults
	limits.MaxPoints = 0
	if top.DefaultPolygonCountLimits().MaxPoints != 1000000 {
		t.Fatal("mutable defaults")
	}
	if _, err := top.ReadV3PlanPolygonCountPrefixWithLimits(planPolygonCountData(0), limits); err != nil {
		t.Fatal(err)
	}
	requirePlanPolygonCountError(t, planPolygonCountData(1), limits, "resource_limit", "plan.elements[0].point_count", 41)
	// A negative count takes precedence over a real zero operational bound.
	requirePlanPolygonCountError(t, planPolygonCountData(-1), limits, "negative_count", "plan.elements[0].point_count", 41)
	limits.MaxPoints = 3
	if _, err := top.ReadV3PlanPolygonCountPrefixWithLimits(planPolygonCountData(3), limits); err != nil {
		t.Fatal(err)
	}
	requirePlanPolygonCountError(t, planPolygonCountData(4), limits, "resource_limit", "plan.elements[0].point_count", 41)
	for _, bad := range []int{-1, 1000001, math.MaxInt} {
		limits = defaults
		limits.MaxPoints, limits.MaxReferences = bad, -1
		requirePlanPolygonCountError(t, nil, limits, "invalid_limit", "limits.max_points", 0)
		requirePlanPolygonCountError(t, append(planMarkerBase(), 0), limits, "invalid_limit", "limits.max_points", 0)
	}
	for _, field := range []string{"max_references", "max_measurements", "max_trips", "max_input_bytes", "max_comment_bytes"} {
		for _, bad := range []int{-1, math.MaxInt} {
			limits = defaults
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
			requirePlanPolygonCountError(t, nil, limits, "invalid_limit", "limits."+field, 0)
		}
	}
	limits = top.PolygonCountLimits{}
	requirePlanPolygonCountError(t, nil, limits, "truncated", "header", 0)
	requirePlanPolygonCountError(t, planPolygonCountData(0), limits, "resource_limit", "input", 0)
	limits.MaxInputBytes = 45
	if _, err := top.ReadV3PlanPolygonCountPrefixWithLimits(planPolygonCountData(0), limits); err != nil {
		t.Fatal(err)
	}
	requirePlanPolygonCountError(t, append(planPolygonCountData(0), 0xff), limits, "resource_limit", "input", 0)
	limits.MaxInputBytes = 44
	requirePlanPolygonCountError(t, planPolygonCountData(0), limits, "resource_limit", "input", 0)
	limits = defaults
	limits.MaxReferences, limits.MaxMeasurements, limits.MaxTrips = -1, -1, -1
	requirePlanPolygonCountError(t, nil, limits, "invalid_limit", "limits.max_references", 0)
	for _, bad := range [][]byte{nil, {0xff}, fixture(-1), fixture(1, record(-1, "", 0)), fixture(1, record(17, "A\xffB", 0)), measurementTable(-1), measurementTable(1, measurementRecord(2, "A\xffB")), referenceTable(-1), referenceTable(1, referenceRecord("A\xffB")), referenceTable(1, referenceRecord("")[:24]), planMappingBase()} {
		_, earlier := top.ReadV3PlanMarkerPrefix(bad)
		p, err := top.ReadV3PlanPolygonCountPrefix(bad)
		if earlier == nil || !reflect.DeepEqual(err, earlier) || !reflect.DeepEqual(p, source.PlanPolygonCountPrefix{}) {
			t.Fatalf("inherited error changed: %v, %v", err, earlier)
		}
	}
	// Lower inherited table/comment bounds must also reach the earlier reader.
	data := binary.LittleEndian.AppendUint32(fixture(1, record(17, "trip", 0)), 1)
	data = append(data, measurementRecord(2, "shot")...)
	data = binary.LittleEndian.AppendUint32(data, 1)
	data = append(data, referenceRecord("ref")...)
	data = append(data, planMappingRecord()...)
	data = append(data, planMappingRecord()...)
	data = binary.LittleEndian.AppendUint32(append(data, 1), 0)
	for _, restrict := range []func(*top.PolygonCountLimits){
		func(l *top.PolygonCountLimits) { l.MaxTrips = 0 },
		func(l *top.PolygonCountLimits) { l.MaxMeasurements = 0 },
		func(l *top.PolygonCountLimits) { l.MaxReferences = 0 },
		func(l *top.PolygonCountLimits) { l.MaxCommentBytes = 3 },
	} {
		limits = defaults
		restrict(&limits)
		_, earlier := top.ReadV3PlanMarkerPrefixWithLimits(data, limits.ReferenceLimits)
		p, err := top.ReadV3PlanPolygonCountPrefixWithLimits(data, limits)
		if earlier == nil || !reflect.DeepEqual(err, earlier) || !reflect.DeepEqual(p, source.PlanPolygonCountPrefix{}) {
			t.Fatal("lower inherited bounds lost")
		}
	}
}

func TestPlanPolygonCountPrefixCopiesAndVariableTables(t *testing.T) {
	data := binary.LittleEndian.AppendUint32(fixture(1, record(17, "trip", -32768)), 1)
	data = append(data, measurementRecord(255, "abc")...)
	data = binary.LittleEndian.AppendUint32(data, 1)
	data = append(data, referenceRecord("Aą")...)
	data = append(data, planMappingRecord()...)
	data = append(data, planMappingBase()[16:]...)
	requirePlanPolygonCountError(t, data, top.DefaultPolygonCountLimits(), "truncated", "plan.elements[0].kind", 107)
	data = append(data, 1)
	requirePlanPolygonCountError(t, data, top.DefaultPolygonCountLimits(), "truncated", "plan.elements[0].point_count", 108)
	data = binary.LittleEndian.AppendUint32(data, 66051)
	data = append(data, 0xff, 0)
	before := bytes.Clone(data)
	p, err := top.ReadV3PlanPolygonCountPrefix(data)
	if err != nil || !bytes.Equal(data, before) {
		t.Fatalf("variable prefix: %v", err)
	}
	base, err := top.ReadV3PlanMarkerPrefix(data)
	if err != nil || p.PlanMapping() != base.PlanMapping() || p.OverviewMapping() != base.OverviewMapping() || p.Offsets().PlanMarkerPrefixOffsets != base.Offsets() {
		t.Fatal("inherited mapping/span separation lost")
	}
	rows, trips, shots, raw, offsets := p.References(), p.Trips(), p.Measurements(), p.Bytes(), p.Offsets()
	comment := rows[0].CommentBytes()
	for i := range data {
		data[i] = 0xff
	}
	rows[0], trips[0], shots[0], raw[108], offsets.PointCount.Start, comment[0] = source.Reference{}, source.Trip{}, source.Measurement{}, 0, 0, 0
	if p.PointCountRaw() != 66051 || p.MarkerRaw() != 1 || p.Offsets().PointCount != (source.Span{Start: 108, End: 112}) || p.ConsumedOffset() != 112 || p.UnparsedTailSize() != 2 || !bytes.Equal(p.Bytes(), before[:112]) || p.Trips()[0].Ticks() != 17 || p.Trips()[0].Comment() != "trip" || p.Measurements()[0].FlagsRaw() != 255 || p.Measurements()[0].Comment() != "abc" || p.References()[0].Comment() != "Aą" {
		t.Fatal("mutable aliases, incorrect offset or lost source fields")
	}
}

func TestPlanPolygonCountPrefixNativeFixture(t *testing.T) {
	data, err := os.ReadFile("testdata/api-drawings.top")
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(data)
	if hex.EncodeToString(hash[:]) != "4a494ead03cade750f670d50aa38661abda9b3e1f131d67f27a252982752c2a5" {
		t.Fatal("native fixture hash changed")
	}
	// Independently pinned helper: first Polygon has three points. Fresh
	// original FileReader/BinaryReader probe establishes count span [147,151).
	for _, tail := range [][]byte{nil, data[151:], {0xff, 0x80}} {
		p, err := top.ReadV3PlanPolygonCountPrefix(append(bytes.Clone(data[:151]), tail...))
		if err != nil || p.PointCountRaw() != 3 || p.MarkerRaw() != 1 || p.Offsets().PointCount != (source.Span{Start: 147, End: 151}) || p.ConsumedOffset() != 151 || p.UnparsedTailSize() != len(tail) || !bytes.Equal(p.Bytes(), data[:151]) {
			t.Fatalf("native count: %+v, %v", p, err)
		}
	}
	if len(data)-151 != 529 {
		t.Fatal("native tail size changed")
	}
	for n := 147; n < 151; n++ {
		requirePlanPolygonCountError(t, data[:n], top.DefaultPolygonCountLimits(), "truncated", "plan.elements[0].point_count", 147)
	}
	references, err := os.ReadFile("testdata/api-references.top")
	if err != nil {
		t.Fatal(err)
	}
	requirePlanPolygonCountError(t, references, top.DefaultPolygonCountLimits(), "unsupported_element", "plan.elements[0].kind", 230)
}

func TestPlanPolygonCountPrefixNoPointAllocation(t *testing.T) {
	zero, large := planPolygonCountData(0), planPolygonCountData(1000000)
	allocations := func(data []byte) float64 {
		return testing.AllocsPerRun(20, func() {
			p, err := top.ReadV3PlanPolygonCountPrefix(data)
			if err != nil || p.ConsumedOffset() != 45 {
				t.Fatal("count-only prefix requires payload")
			}
		})
	}
	if allocations(zero) != allocations(large) {
		t.Fatal("allocation count depends on unread point count")
	}
}

func FuzzReadV3PlanPolygonCountPrefix(f *testing.F) {
	for _, seed := range [][]byte{planPolygonCountData(0), planPolygonCountData(3), planPolygonCountData(1000000), planPolygonCountData(-1), planPolygonCountData(math.MaxInt32), planPolygonCountData(3)[:44], append(planMarkerBase(), 0), append(planMarkerBase(), 3), {0xff}} {
		f.Add(seed)
	}
	limits := top.PolygonCountLimits{ReferenceLimits: top.ReferenceLimits{MeasurementLimits: top.MeasurementLimits{Limits: top.Limits{MaxInputBytes: 4096, MaxTrips: 32, MaxCommentBytes: 256}, MaxMeasurements: 32}, MaxReferences: 32}, MaxPoints: 32}
	f.Fuzz(func(t *testing.T, data []byte) {
		before := bytes.Clone(data)
		p, err := top.ReadV3PlanPolygonCountPrefixWithLimits(data, limits)
		q, again := top.ReadV3PlanPolygonCountPrefixWithLimits(data, limits)
		if !bytes.Equal(data, before) || !reflect.DeepEqual(p, q) || !reflect.DeepEqual(err, again) {
			t.Fatal("mutation or nondeterministic result")
		}
		base, earlier := top.ReadV3PlanMarkerPrefixWithLimits(data, limits.ReferenceLimits)
		if earlier != nil {
			if !reflect.DeepEqual(err, earlier) || !reflect.DeepEqual(p, source.PlanPolygonCountPrefix{}) {
				t.Fatal("inherited failure changed")
			}
			return
		}
		code, field, offset := "", "plan.elements[0].point_count", base.ConsumedOffset()
		var count int32
		switch {
		case base.MarkerRaw() != 1:
			code, field, offset = "unsupported_element", "plan.elements[0].kind", base.Offsets().Marker.Start
		case len(data)-offset < 4:
			code = "truncated"
		default:
			count = int32(binary.LittleEndian.Uint32(data[offset : offset+4]))
			if count < 0 {
				code = "negative_count"
			} else if int(count) > limits.MaxPoints {
				code = "resource_limit"
			}
		}
		if code != "" {
			var pe *top.ParseError
			if !errors.As(err, &pe) || pe.Code != code || pe.Field != field || pe.Offset != offset || !reflect.DeepEqual(p, source.PlanPolygonCountPrefix{}) {
				t.Fatalf("invalid failure: %v", err)
			}
			return
		}
		span := p.Offsets().PointCount
		if err != nil || p.PointCountRaw() != count || p.MarkerRaw() != 1 || p.PlanMapping() != base.PlanMapping() || p.OverviewMapping() != base.OverviewMapping() || p.Offsets().PlanMarkerPrefixOffsets != base.Offsets() || span != (source.Span{Start: offset, End: offset + 4}) || span.End != p.ConsumedOffset() || p.ConsumedOffset()+p.UnparsedTailSize() != len(data) || !bytes.Equal(p.Bytes(), data[:p.ConsumedOffset()]) {
			t.Fatal("invalid count, inherited fields, span or prefix/tail accounting")
		}
	})
}
