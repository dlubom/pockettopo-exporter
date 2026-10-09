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
func overviewRecord() []byte {
	return []byte{1, 2, 3, 0x84, 8, 7, 6, 5, 0xfe, 0xfd, 0xfc, 0xfb}
}

func requireOverviewError(t *testing.T, data []byte, limits top.ReferenceLimits, code, field string, offset int) {
	t.Helper()
	before := bytes.Clone(data)
	p, err := top.ReadV3OverviewPrefixWithLimits(data, limits)
	var pe *top.ParseError
	if !errors.As(err, &pe) || pe.Code != code || pe.Field != field || pe.Offset != offset {
		t.Fatalf("got %v; want %s at %d (%s)", err, code, offset, field)
	}
	if !reflect.DeepEqual(p, source.OverviewPrefix{}) || !bytes.Equal(data, before) {
		t.Fatal("partial result or input mutation on failure")
	}
}

func TestOverviewPrefixLiteralFieldsAndSpans(t *testing.T) {
	data := append(referenceTable(0), overviewRecord()...)
	before := bytes.Clone(data)
	data = append(data, 0xff, 0x80, 0, 3)
	p, err := top.ReadV3OverviewPrefix(data)
	if err != nil {
		t.Fatal(err)
	}
	m := p.OverviewMapping()
	want := source.MappingOffsets{
		Record: source.Span{Start: 16, End: 28}, X0: source.Span{Start: 16, End: 20},
		Y0: source.Span{Start: 20, End: 24}, Scale: source.Span{Start: 24, End: 28},
	}
	if m.X0Raw() != -0x7bfcfdff || m.Y0Raw() != 0x05060708 || m.ScaleRaw() != -0x04030202 || m.Offsets() != want || p.Offsets().Overview != want || p.ConsumedOffset() != 28 || p.UnparsedTailSize() != 4 || !bytes.Equal(p.Bytes(), before) {
		t.Fatalf("mapping fields or accounting: %+v, %+v", m, p)
	}
	if p.Header() != [4]byte{'T', 'o', 'p', 3} || p.Version() != 3 || p.TripCountRaw() != 0 || p.MeasurementCountRaw() != 0 || p.ReferenceCountRaw() != 0 || len(p.Trips()) != 0 || len(p.Measurements()) != 0 || len(p.References()) != 0 || p.Offsets().TripCount != (source.Span{Start: 4, End: 8}) || p.Offsets().MeasurementCount != (source.Span{Start: 8, End: 12}) || p.Offsets().ReferenceCount != (source.Span{Start: 12, End: 16}) {
		t.Fatal("lost inherited prefix metadata")
	}
}

func TestOverviewPrefixRawBoundaries(t *testing.T) {
	for field := 0; field < 3; field++ {
		for _, value := range []int32{math.MinInt32, -16777217, -501, -11, -10, -6, -5, -4, -1, 0, 1, 4, 5, 6, 9, 10, 11, 499, 500, 501, 16777217, math.MaxInt32} {
			b := overviewRecord()
			binary.LittleEndian.PutUint32(b[4*field:4*field+4], uint32(value))
			p, err := top.ReadV3OverviewPrefix(append(referenceTable(0), b...))
			if err != nil {
				t.Fatalf("field %d value %d: %v", field, value, err)
			}
			m := p.OverviewMapping()
			want := [3]int32{-0x7bfcfdff, 0x05060708, -0x04030202}
			want[field] = value
			if [3]int32{m.X0Raw(), m.Y0Raw(), m.ScaleRaw()} != want || !bytes.Equal(p.Bytes()[16:], b) {
				t.Fatalf("raw fields changed for field %d value %d", field, value)
			}
		}
	}
}

func TestOverviewPrefixTruncationEveryRequiredByte(t *testing.T) {
	limits := top.DefaultReferenceLimits()
	for _, prefix := range [][]byte{referenceTable(0), referenceTable(2, referenceRecord("Aą"), referenceRecord(""))} {
		data := append(bytes.Clone(prefix), overviewRecord()...)
		for n := len(prefix); n < len(data); n++ {
			index := (n - len(prefix)) / 4
			requireOverviewError(t, data[:n], limits, "truncated", []string{"overview.x0", "overview.y0", "overview.scale"}[index], len(prefix)+4*index)
		}
		if p, err := top.ReadV3OverviewPrefix(data); err != nil || p.ConsumedOffset() != len(data) || p.UnparsedTailSize() != 0 {
			t.Fatalf("exact prefix without drawing bytes: %v", err)
		}
	}
	for n := 0; n < 16; n++ {
		field := []string{"header", "trip_count", "measurement_count", "reference_count"}[n/4]
		requireOverviewError(t, referenceTable(0)[:n], limits, "truncated", field, n/4*4)
	}
	// The new API never moves the existing readers' stopping positions.
	for _, tail := range [][]byte{nil, {0xff}, overviewRecord(), {0xff, 0xff, 0xff, 0xff}} {
		if p, err := top.ReadV3TripPrefix(append(fixture(0), tail...)); err != nil || p.ConsumedOffset() != 8 || p.UnparsedTailSize() != len(tail) {
			t.Fatal("P03b contract changed")
		}
		if p, err := top.ReadV3MeasurementPrefix(append(measurementTable(0), tail...)); err != nil || p.ConsumedOffset() != 12 || p.UnparsedTailSize() != len(tail) {
			t.Fatal("P03c1 contract changed")
		}
		if p, err := top.ReadV3ReferencePrefix(append(referenceTable(0), tail...)); err != nil || p.ConsumedOffset() != 16 || p.UnparsedTailSize() != len(tail) {
			t.Fatal("P03c2 contract changed")
		}
	}
}

func TestOverviewPrefixInheritedErrorsAndLimits(t *testing.T) {
	defaults := top.DefaultReferenceLimits()
	for _, bad := range [][]byte{nil, {0xff}, fixture(-1), fixture(1, record(-1, "", 0)), fixture(1, record(17, "A\xffB", 0)), measurementTable(-1), measurementTable(1, measurementRecord(2, "A\xffB")), referenceTable(-1), referenceTable(1, referenceRecord("A\xffB")), referenceTable(1, referenceRecord("")[:24])} {
		_, earlier := top.ReadV3ReferencePrefixWithLimits(bad, defaults)
		p, err := top.ReadV3OverviewPrefixWithLimits(bad, defaults)
		if earlier == nil || !reflect.DeepEqual(err, earlier) || !reflect.DeepEqual(p, source.OverviewPrefix{}) {
			t.Fatalf("inherited error changed: %v, earlier %v", err, earlier)
		}
	}
	data := append(referenceTable(1, referenceRecord("abc")), overviewRecord()...)
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
			requireOverviewError(t, data, limits, "invalid_limit", "limits."+field, 0)
		}
	}
	limits := top.ReferenceLimits{MeasurementLimits: top.MeasurementLimits{Limits: top.Limits{MaxInputBytes: len(data), MaxTrips: 0, MaxCommentBytes: 3}, MaxMeasurements: 0}, MaxReferences: 1}
	if _, err := top.ReadV3OverviewPrefixWithLimits(data, limits); err != nil {
		t.Fatal(err)
	}
	limits.MaxInputBytes--
	requireOverviewError(t, data, limits, "resource_limit", "input", 0)
	limits.MaxInputBytes++
	requireOverviewError(t, append(bytes.Clone(data), 0xff), limits, "resource_limit", "input", 0)
	limits.MaxReferences = 0
	requireOverviewError(t, data, limits, "resource_limit", "reference_count", 12)
	limits.MaxReferences = 1
	limits.MaxCommentBytes = 2
	requireOverviewError(t, data, limits, "resource_limit", "references[0].comment.length", 40)
	limits.MaxCommentBytes = 0
	limits.MaxReferences = 0
	if _, err := top.ReadV3OverviewPrefixWithLimits(append(referenceTable(0), overviewRecord()...), limits); err != nil {
		t.Fatal(err)
	}
	requireOverviewError(t, data, top.ReferenceLimits{}, "resource_limit", "input", 0)
}

func TestOverviewPrefixCopiesAndVariableTables(t *testing.T) {
	data := binary.LittleEndian.AppendUint32(fixture(1, record(17, "trip", -32768)), 1)
	data = append(data, measurementRecord(255, "abc")...)
	data = binary.LittleEndian.AppendUint32(data, 1)
	data = append(data, referenceRecord("Aą")...)
	data = append(data, overviewRecord()...)
	before := bytes.Clone(data)
	p, err := top.ReadV3OverviewPrefix(data)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, before) {
		t.Fatal("reader changed input")
	}
	rows, trips, shots, raw := p.References(), p.Trips(), p.Measurements(), p.Bytes()
	mapping, offsets, comment, header := p.OverviewMapping(), p.Offsets(), rows[0].CommentBytes(), p.Header()
	for i := range data {
		data[i] = 0xff
	}
	rows[0], trips[0], shots[0], raw[0], mapping, offsets.Overview.X0.Start, comment[0], header[0] = source.Reference{}, source.Trip{}, source.Measurement{}, 0, source.NewMapping(0, mapping.Y0Raw(), mapping.ScaleRaw(), mapping.Offsets()), 0, 0, 0
	want := source.MappingOffsets{Record: source.Span{Start: 83, End: 95}, X0: source.Span{Start: 83, End: 87}, Y0: source.Span{Start: 87, End: 91}, Scale: source.Span{Start: 91, End: 95}}
	if !bytes.Equal(p.Bytes(), before) || p.OverviewMapping() == mapping || p.OverviewMapping().Offsets() != want || p.Offsets().Overview != want || p.Offsets().ReferenceCount != (source.Span{Start: 51, End: 55}) || p.Offsets().MeasurementCount != (source.Span{Start: 23, End: 27}) || p.ConsumedOffset() != 95 || p.Trips()[0].Ticks() != 17 || p.Trips()[0].Comment() != "trip" || p.Measurements()[0].FlagsRaw() != 255 || p.Measurements()[0].Comment() != "abc" || p.References()[0].Comment() != "Aą" || p.Header()[0] != 'T' {
		t.Fatal("mutable aliases or lost previous source data")
	}
}

func TestOverviewPrefixNativeFixtures(t *testing.T) {
	for _, fixture := range []struct {
		name, hash            string
		x0, y0, scale         int32
		start, consumed, tail int
	}{
		{"api-references.top", "9034cf5e52f92a8713c7823bd9be52bdb50b294966a9e9713c538b937b7feb5f", 0, 0, 500, 206, 218, 30},
		{"api-drawings.top", "4a494ead03cade750f670d50aa38661abda9b3e1f131d67f27a252982752c2a5", -1234, 5678, 500, 122, 134, 546},
	} {
		data, err := os.ReadFile("testdata/" + fixture.name)
		if err != nil {
			t.Fatal(err)
		}
		hash := sha256.Sum256(data)
		if hex.EncodeToString(hash[:]) != fixture.hash {
			t.Fatal("native fixture hash changed")
		}
		p, err := top.ReadV3OverviewPrefix(data)
		if err != nil {
			t.Fatal(err)
		}
		m := p.OverviewMapping()
		want := source.MappingOffsets{Record: source.Span{Start: fixture.start, End: fixture.consumed}, X0: source.Span{Start: fixture.start, End: fixture.start + 4}, Y0: source.Span{Start: fixture.start + 4, End: fixture.start + 8}, Scale: source.Span{Start: fixture.start + 8, End: fixture.consumed}}
		if m.X0Raw() != fixture.x0 || m.Y0Raw() != fixture.y0 || m.ScaleRaw() != fixture.scale || m.Offsets() != want || p.Offsets().Overview != want || p.ConsumedOffset() != fixture.consumed || p.UnparsedTailSize() != fixture.tail || !bytes.Equal(p.Bytes(), data[:fixture.consumed]) {
			t.Fatalf("native overview %s: %+v, %+v", fixture.name, m, p)
		}
		// Expectations come from pinned helper inputs and native stream positions.
		if q, err := top.ReadV3OverviewPrefix(data[:fixture.consumed]); err != nil || q.OverviewMapping() != m || q.UnparsedTailSize() != 0 {
			t.Fatalf("native prefix without drawing bytes: %v", err)
		}
		arbitrary := append(bytes.Clone(data[:fixture.consumed]), 0xff, 0xff, 0x80)
		if q, err := top.ReadV3OverviewPrefix(arbitrary); err != nil || q.OverviewMapping() != m || q.UnparsedTailSize() != 3 {
			t.Fatalf("reader interpreted drawing tail: %v", err)
		}
	}
}

func FuzzReadV3OverviewPrefix(f *testing.F) {
	for _, seed := range [][]byte{append(referenceTable(0), overviewRecord()...), append(referenceTable(2, referenceRecord("Aą"), referenceRecord("")), overviewRecord()...), referenceTable(0), {0xff}} {
		f.Add(seed)
	}
	limits := top.ReferenceLimits{MeasurementLimits: top.MeasurementLimits{Limits: top.Limits{MaxInputBytes: 4096, MaxTrips: 32, MaxCommentBytes: 256}, MaxMeasurements: 32}, MaxReferences: 32}
	f.Fuzz(func(t *testing.T, data []byte) {
		before := bytes.Clone(data)
		p, err := top.ReadV3OverviewPrefixWithLimits(data, limits)
		q, again := top.ReadV3OverviewPrefixWithLimits(data, limits)
		if !bytes.Equal(data, before) || !reflect.DeepEqual(p, q) || !reflect.DeepEqual(err, again) {
			t.Fatal("mutation or nondeterministic result")
		}
		if err != nil {
			var pe *top.ParseError
			if !errors.As(err, &pe) || pe.Offset < 0 || pe.Offset > len(data) || !reflect.DeepEqual(p, source.OverviewPrefix{}) {
				t.Fatalf("invalid failure: %v", err)
			}
			return
		}
		m := p.OverviewMapping().Offsets()
		if p.ConsumedOffset() > len(data) || p.ConsumedOffset()+p.UnparsedTailSize() != len(data) || !bytes.Equal(p.Bytes(), data[:p.ConsumedOffset()]) || int(p.ReferenceCountRaw()) != len(p.References()) || int(p.MeasurementCountRaw()) != len(p.Measurements()) || int(p.TripCountRaw()) != len(p.Trips()) || m.Record.End != p.ConsumedOffset() || m.Record.End-m.Record.Start != 12 || m.X0 != (source.Span{Start: m.Record.Start, End: m.Record.Start + 4}) || m.Y0 != (source.Span{Start: m.Record.Start + 4, End: m.Record.Start + 8}) || m.Scale != (source.Span{Start: m.Record.Start + 8, End: m.Record.End}) || p.Offsets().Overview != m {
			t.Fatal("invalid accounting or mapping spans")
		}
	})
}
