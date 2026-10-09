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

// These literal bytes also run through the original assembly's endian probe.
func planMappingBase() []byte {
	return append(referenceTable(0), []byte{0xd2, 4, 0, 0, 0x2e, 0x16, 0, 0, 0xf5, 1, 0, 0}...)
}

func planMappingRecord() []byte {
	return []byte{1, 2, 3, 0x84, 8, 7, 6, 5, 0xfe, 0xfd, 0xfc, 0xfb}
}

func requirePlanMappingError(t *testing.T, data []byte, limits top.ReferenceLimits, code, field string, offset int) {
	t.Helper()
	before := bytes.Clone(data)
	p, err := top.ReadV3PlanMappingPrefixWithLimits(data, limits)
	var pe *top.ParseError
	if !errors.As(err, &pe) || pe.Code != code || pe.Field != field || pe.Offset != offset {
		t.Fatalf("got %v; want %s at %d (%s)", err, code, offset, field)
	}
	if !reflect.DeepEqual(p, source.PlanMappingPrefix{}) || !bytes.Equal(data, before) {
		t.Fatal("partial result or input mutation on failure")
	}
}

func TestPlanMappingPrefixLiteralFieldsAndSpans(t *testing.T) {
	data := append(planMappingBase(), planMappingRecord()...)
	before := bytes.Clone(data)
	data = append(data, 0xff, 0x80, 0, 3)
	p, err := top.ReadV3PlanMappingPrefix(data)
	if err != nil {
		t.Fatal(err)
	}
	m := p.PlanMapping()
	want := source.MappingOffsets{
		Record: source.Span{Start: 28, End: 40}, X0: source.Span{Start: 28, End: 32},
		Y0: source.Span{Start: 32, End: 36}, Scale: source.Span{Start: 36, End: 40},
	}
	if m.X0Raw() != -0x7bfcfdff || m.Y0Raw() != 0x05060708 || m.ScaleRaw() != -0x04030202 || m.Offsets() != want || p.Offsets().Plan != want || p.ConsumedOffset() != 40 || p.UnparsedTailSize() != 4 || !bytes.Equal(p.Bytes(), before) {
		t.Fatalf("mapping fields or accounting: %+v, %+v", m, p)
	}
	o := p.OverviewMapping()
	if o.X0Raw() != 1234 || o.Y0Raw() != 5678 || o.ScaleRaw() != 501 || o.Offsets().Record != (source.Span{Start: 16, End: 28}) || p.Offsets().Overview != o.Offsets() {
		t.Fatal("overview mapping was replaced by plan mapping")
	}
	if p.Header() != [4]byte{'T', 'o', 'p', 3} || p.Version() != 3 || p.TripCountRaw() != 0 || p.MeasurementCountRaw() != 0 || p.ReferenceCountRaw() != 0 || len(p.Trips()) != 0 || len(p.Measurements()) != 0 || len(p.References()) != 0 || p.Offsets().TripCount != (source.Span{Start: 4, End: 8}) || p.Offsets().MeasurementCount != (source.Span{Start: 8, End: 12}) || p.Offsets().ReferenceCount != (source.Span{Start: 12, End: 16}) {
		t.Fatal("lost inherited prefix metadata")
	}
}

func TestPlanMappingPrefixRawBoundaries(t *testing.T) {
	for field := 0; field < 3; field++ {
		for _, value := range []int32{math.MinInt32, -16777217, -501, -11, -10, -6, -5, -4, -1, 0, 1, 4, 5, 6, 9, 10, 11, 499, 500, 501, 16777217, math.MaxInt32} {
			b := planMappingRecord()
			binary.LittleEndian.PutUint32(b[4*field:4*field+4], uint32(value))
			p, err := top.ReadV3PlanMappingPrefix(append(planMappingBase(), b...))
			if err != nil {
				t.Fatalf("field %d value %d: %v", field, value, err)
			}
			m := p.PlanMapping()
			want := [3]int32{-0x7bfcfdff, 0x05060708, -0x04030202}
			want[field] = value
			if [3]int32{m.X0Raw(), m.Y0Raw(), m.ScaleRaw()} != want || !bytes.Equal(p.Bytes()[28:], b) {
				t.Fatalf("raw fields changed for field %d value %d", field, value)
			}
		}
	}
}

func TestPlanMappingPrefixStopsBeforeEveryMarker(t *testing.T) {
	prefix := append(planMappingBase(), planMappingRecord()...)
	for marker := 0; marker < 256; marker++ {
		// These include terminators, known/unknown markers, and a misleading
		// adjacent scalar record. None belongs to this prefix contract.
		for _, tail := range [][]byte{{byte(marker)}, append([]byte{byte(marker)}, planMappingRecord()...)} {
			data := append(bytes.Clone(prefix), tail...)
			p, err := top.ReadV3PlanMappingPrefix(data)
			if err != nil || p.ConsumedOffset() != 40 || p.UnparsedTailSize() != len(tail) || !bytes.Equal(p.Bytes(), prefix) {
				t.Fatalf("interpreted tail starting with %d: %v", marker, err)
			}
		}
	}
}

func TestPlanMappingPrefixTruncationEveryRequiredByte(t *testing.T) {
	limits := top.DefaultReferenceLimits()
	for _, prefix := range [][]byte{referenceTable(0), referenceTable(2, referenceRecord("Aą"), referenceRecord(""))} {
		prefix = append(bytes.Clone(prefix), planMappingRecord()...)
		data := append(bytes.Clone(prefix), planMappingRecord()...)
		for n := len(prefix); n < len(data); n++ {
			index := (n - len(prefix)) / 4
			requirePlanMappingError(t, data[:n], limits, "truncated", []string{"plan.mapping.x0", "plan.mapping.y0", "plan.mapping.scale"}[index], len(prefix)+4*index)
		}
		if p, err := top.ReadV3PlanMappingPrefix(data); err != nil || p.ConsumedOffset() != len(data) || p.UnparsedTailSize() != 0 {
			t.Fatalf("exact prefix without drawing bytes: %v", err)
		}
	}
	for n := 0; n < 16; n++ {
		field := []string{"header", "trip_count", "measurement_count", "reference_count"}[n/4]
		requirePlanMappingError(t, referenceTable(0)[:n], limits, "truncated", field, n/4*4)
	}
	for n := 16; n < 28; n++ {
		requirePlanMappingError(t, planMappingBase()[:n], limits, "truncated", []string{"overview.x0", "overview.y0", "overview.scale"}[(n-16)/4], 16+(n-16)/4*4)
	}
	// The new API never moves the existing readers' stopping positions.
	for _, tail := range [][]byte{nil, {0xff}, planMappingRecord(), {0xff, 0xff, 0xff, 0xff}} {
		if p, err := top.ReadV3TripPrefix(append(fixture(0), tail...)); err != nil || p.ConsumedOffset() != 8 || p.UnparsedTailSize() != len(tail) {
			t.Fatal("P03b contract changed")
		}
		if p, err := top.ReadV3MeasurementPrefix(append(measurementTable(0), tail...)); err != nil || p.ConsumedOffset() != 12 || p.UnparsedTailSize() != len(tail) {
			t.Fatal("P03c1 contract changed")
		}
		if p, err := top.ReadV3ReferencePrefix(append(referenceTable(0), tail...)); err != nil || p.ConsumedOffset() != 16 || p.UnparsedTailSize() != len(tail) {
			t.Fatal("P03c2 contract changed")
		}
		if p, err := top.ReadV3OverviewPrefix(append(planMappingBase(), tail...)); err != nil || p.ConsumedOffset() != 28 || p.UnparsedTailSize() != len(tail) {
			t.Fatal("P04a contract changed")
		}
	}
}

func TestPlanMappingPrefixInheritedErrorsAndLimits(t *testing.T) {
	defaults := top.DefaultReferenceLimits()
	for _, bad := range [][]byte{nil, {0xff}, fixture(-1), fixture(1, record(-1, "", 0)), fixture(1, record(17, "A\xffB", 0)), measurementTable(-1), measurementTable(1, measurementRecord(2, "A\xffB")), referenceTable(-1), referenceTable(1, referenceRecord("A\xffB")), referenceTable(1, referenceRecord("")[:24])} {
		_, earlier := top.ReadV3OverviewPrefixWithLimits(bad, defaults)
		p, err := top.ReadV3PlanMappingPrefixWithLimits(bad, defaults)
		if earlier == nil || !reflect.DeepEqual(err, earlier) || !reflect.DeepEqual(p, source.PlanMappingPrefix{}) {
			t.Fatalf("inherited error changed: %v, earlier %v", err, earlier)
		}
	}
	data := append(referenceTable(1, referenceRecord("abc")), planMappingRecord()...)
	data = append(data, planMappingRecord()...)
	for _, field := range []string{"max_references", "max_measurements", "max_trips", "max_input_bytes", "max_comment_bytes"} {
		for _, value := range []int{-1, math.MaxInt} {
			limits := defaults
			switch field {
			case "max_references":
				limits.MaxReferences = value
			case "max_measurements":
				limits.MaxMeasurements = value
			case "max_trips":
				limits.MaxTrips = value
			case "max_input_bytes":
				limits.MaxInputBytes = value
			case "max_comment_bytes":
				limits.MaxCommentBytes = value
			}
			requirePlanMappingError(t, data, limits, "invalid_limit", "limits."+field, 0)
		}
	}
	limits := top.ReferenceLimits{MeasurementLimits: top.MeasurementLimits{Limits: top.Limits{MaxInputBytes: len(data), MaxTrips: 0, MaxCommentBytes: 3}, MaxMeasurements: 0}, MaxReferences: 1}
	if _, err := top.ReadV3PlanMappingPrefixWithLimits(data, limits); err != nil {
		t.Fatal(err)
	}
	limits.MaxInputBytes--
	requirePlanMappingError(t, data, limits, "resource_limit", "input", 0)
	limits.MaxInputBytes++
	requirePlanMappingError(t, append(bytes.Clone(data), 0xff), limits, "resource_limit", "input", 0)
	limits.MaxReferences = 0
	requirePlanMappingError(t, data, limits, "resource_limit", "reference_count", 12)
	limits.MaxReferences = 1
	limits.MaxCommentBytes = 2
	requirePlanMappingError(t, data, limits, "resource_limit", "references[0].comment.length", 40)
	limits.MaxCommentBytes = 0
	limits.MaxReferences = 0
	if _, err := top.ReadV3PlanMappingPrefixWithLimits(append(planMappingBase(), planMappingRecord()...), limits); err != nil {
		t.Fatal(err)
	}
	requirePlanMappingError(t, data, top.ReferenceLimits{}, "resource_limit", "input", 0)
}

func TestPlanMappingPrefixCopiesAndVariableTables(t *testing.T) {
	data := binary.LittleEndian.AppendUint32(fixture(1, record(17, "trip", -32768)), 1)
	data = append(data, measurementRecord(255, "abc")...)
	data = binary.LittleEndian.AppendUint32(data, 1)
	data = append(data, referenceRecord("Aą")...)
	data = append(data, planMappingRecord()...)
	data = append(data, planMappingBase()[16:]...)
	before := bytes.Clone(data)
	p, err := top.ReadV3PlanMappingPrefix(data)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, before) {
		t.Fatal("reader changed input")
	}
	rows, trips, shots, raw := p.References(), p.Trips(), p.Measurements(), p.Bytes()
	mapping, offsets, comment, header := p.PlanMapping(), p.Offsets(), rows[0].CommentBytes(), p.Header()
	for i := range data {
		data[i] = 0xff
	}
	rows[0], trips[0], shots[0], raw[0], mapping, offsets.Plan.X0.Start, comment[0], header[0] = source.Reference{}, source.Trip{}, source.Measurement{}, 0, source.NewMapping(0, mapping.Y0Raw(), mapping.ScaleRaw(), mapping.Offsets()), 0, 0, 0
	want := source.MappingOffsets{Record: source.Span{Start: 95, End: 107}, X0: source.Span{Start: 95, End: 99}, Y0: source.Span{Start: 99, End: 103}, Scale: source.Span{Start: 103, End: 107}}
	if p.PlanMapping().X0Raw() != 1234 || p.PlanMapping().Y0Raw() != 5678 || p.PlanMapping().ScaleRaw() != 501 || p.OverviewMapping().X0Raw() != -0x7bfcfdff || p.OverviewMapping().Offsets().Record != (source.Span{Start: 83, End: 95}) || p.Offsets().Overview != p.OverviewMapping().Offsets() {
		t.Fatal("variable tables conflated mapping fields or spans")
	}
	if !bytes.Equal(p.Bytes(), before) || p.PlanMapping() == mapping || p.PlanMapping().Offsets() != want || p.Offsets().Plan != want || p.Offsets().ReferenceCount != (source.Span{Start: 51, End: 55}) || p.Offsets().MeasurementCount != (source.Span{Start: 23, End: 27}) || p.ConsumedOffset() != 107 || p.Trips()[0].Ticks() != 17 || p.Trips()[0].Comment() != "trip" || p.Measurements()[0].FlagsRaw() != 255 || p.Measurements()[0].Comment() != "abc" || p.References()[0].Comment() != "Aą" || p.Header()[0] != 'T' {
		t.Fatal("mutable aliases or lost previous source data")
	}
}

func TestPlanMappingPrefixNativeFixtures(t *testing.T) {
	for _, fixture := range []struct {
		name, hash            string
		x0, y0, scale         int32
		start, consumed, tail int
	}{
		{"api-references.top", "9034cf5e52f92a8713c7823bd9be52bdb50b294966a9e9713c538b937b7feb5f", 0, 0, 500, 218, 230, 18},
		{"api-drawings.top", "4a494ead03cade750f670d50aa38661abda9b3e1f131d67f27a252982752c2a5", -100, 200, 500, 134, 146, 534},
	} {
		data, err := os.ReadFile("testdata/" + fixture.name)
		if err != nil {
			t.Fatal(err)
		}
		hash := sha256.Sum256(data)
		if hex.EncodeToString(hash[:]) != fixture.hash {
			t.Fatal("native fixture hash changed")
		}
		p, err := top.ReadV3PlanMappingPrefix(data)
		if err != nil {
			t.Fatal(err)
		}
		m := p.PlanMapping()
		o, earlier := top.ReadV3OverviewPrefix(data)
		if earlier != nil || p.OverviewMapping() != o.OverviewMapping() || p.Offsets().OverviewPrefixOffsets != o.Offsets() {
			t.Fatal("native plan prefix changed overview data")
		}
		want := source.MappingOffsets{Record: source.Span{Start: fixture.start, End: fixture.consumed}, X0: source.Span{Start: fixture.start, End: fixture.start + 4}, Y0: source.Span{Start: fixture.start + 4, End: fixture.start + 8}, Scale: source.Span{Start: fixture.start + 8, End: fixture.consumed}}
		if m.X0Raw() != fixture.x0 || m.Y0Raw() != fixture.y0 || m.ScaleRaw() != fixture.scale || m.Offsets() != want || p.Offsets().Plan != want || p.ConsumedOffset() != fixture.consumed || p.UnparsedTailSize() != fixture.tail || !bytes.Equal(p.Bytes(), data[:fixture.consumed]) {
			t.Fatalf("native plan mapping %s: %+v, %+v", fixture.name, m, p)
		}
		// Expectations come from pinned helper inputs and native stream positions.
		if q, err := top.ReadV3PlanMappingPrefix(data[:fixture.consumed]); err != nil || q.PlanMapping() != m || q.UnparsedTailSize() != 0 {
			t.Fatalf("native prefix without drawing bytes: %v", err)
		}
		arbitrary := append(bytes.Clone(data[:fixture.consumed]), 0xff, 0xff, 0x80)
		if q, err := top.ReadV3PlanMappingPrefix(arbitrary); err != nil || q.PlanMapping() != m || q.UnparsedTailSize() != 3 {
			t.Fatalf("reader interpreted drawing tail: %v", err)
		}
	}
}

func FuzzReadV3PlanMappingPrefix(f *testing.F) {
	for _, seed := range [][]byte{append(planMappingBase(), planMappingRecord()...), append(append(referenceTable(2, referenceRecord("Aą"), referenceRecord("")), planMappingRecord()...), planMappingRecord()...), referenceTable(0), {0xff}} {
		f.Add(seed)
	}
	limits := top.ReferenceLimits{MeasurementLimits: top.MeasurementLimits{Limits: top.Limits{MaxInputBytes: 4096, MaxTrips: 32, MaxCommentBytes: 256}, MaxMeasurements: 32}, MaxReferences: 32}
	f.Fuzz(func(t *testing.T, data []byte) {
		before := bytes.Clone(data)
		p, err := top.ReadV3PlanMappingPrefixWithLimits(data, limits)
		q, again := top.ReadV3PlanMappingPrefixWithLimits(data, limits)
		if !bytes.Equal(data, before) || !reflect.DeepEqual(p, q) || !reflect.DeepEqual(err, again) {
			t.Fatal("mutation or nondeterministic result")
		}
		if err != nil {
			var pe *top.ParseError
			if !errors.As(err, &pe) || pe.Offset < 0 || pe.Offset > len(data) || !reflect.DeepEqual(p, source.PlanMappingPrefix{}) {
				t.Fatalf("invalid failure: %v", err)
			}
			return
		}
		m := p.PlanMapping().Offsets()
		if p.ConsumedOffset() > len(data) || p.ConsumedOffset()+p.UnparsedTailSize() != len(data) || !bytes.Equal(p.Bytes(), data[:p.ConsumedOffset()]) || int(p.ReferenceCountRaw()) != len(p.References()) || int(p.MeasurementCountRaw()) != len(p.Measurements()) || int(p.TripCountRaw()) != len(p.Trips()) || m.Record.End != p.ConsumedOffset() || m.Record.End-m.Record.Start != 12 || m.X0 != (source.Span{Start: m.Record.Start, End: m.Record.Start + 4}) || m.Y0 != (source.Span{Start: m.Record.Start + 4, End: m.Record.Start + 8}) || m.Scale != (source.Span{Start: m.Record.Start + 8, End: m.Record.End}) || p.Offsets().Plan != m {
			t.Fatal("invalid accounting or mapping spans")
		}
		o, earlier := top.ReadV3OverviewPrefixWithLimits(data, limits)
		if earlier != nil || m.Record.Start != o.ConsumedOffset() || p.OverviewMapping() != o.OverviewMapping() || p.Offsets().OverviewPrefixOffsets != o.Offsets() {
			t.Fatal("invalid separation from overview prefix")
		}
		mapping := p.PlanMapping()
		if mapping.X0Raw() != int32(binary.LittleEndian.Uint32(data[m.X0.Start:m.X0.End])) || mapping.Y0Raw() != int32(binary.LittleEndian.Uint32(data[m.Y0.Start:m.Y0.End])) || mapping.ScaleRaw() != int32(binary.LittleEndian.Uint32(data[m.Scale.Start:m.Scale.End])) {
			t.Fatal("plan mapping changed raw bits")
		}
	})
}
