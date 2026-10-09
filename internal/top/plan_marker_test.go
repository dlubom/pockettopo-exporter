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

func planMarkerBase() []byte {
	return append(planMappingBase(), planMappingRecord()...)
}

func requirePlanMarkerError(t *testing.T, data []byte, limits top.ReferenceLimits, code, field string, offset int) {
	t.Helper()
	before := bytes.Clone(data)
	p, err := top.ReadV3PlanMarkerPrefixWithLimits(data, limits)
	var pe *top.ParseError
	if !errors.As(err, &pe) || pe.Code != code || pe.Field != field || pe.Offset != offset {
		t.Fatalf("got %v; want %s at %d (%s)", err, code, offset, field)
	}
	if !reflect.DeepEqual(p, source.PlanMarkerPrefix{}) || !bytes.Equal(data, before) {
		t.Fatal("partial result or input mutation on failure")
	}
}

func TestPlanMarkerPrefixEveryByteAndTail(t *testing.T) {
	for marker := 0; marker < 256; marker++ {
		prefix := append(planMarkerBase(), byte(marker))
		// No payload is required for known or unknown markers. Even zero stops
		// here, before a misleading side mapping or malformed remaining bytes.
		for _, tail := range [][]byte{nil, {0xff}, {0, 1, 3, 0xff, 0x80}, planMappingRecord()} {
			data := append(bytes.Clone(prefix), tail...)
			before := bytes.Clone(data)
			p, err := top.ReadV3PlanMarkerPrefix(data)
			if err != nil || p.MarkerRaw() != byte(marker) || p.Offsets().Marker != (source.Span{Start: 40, End: 41}) || p.ConsumedOffset() != 41 || p.UnparsedTailSize() != len(tail) || !bytes.Equal(p.Bytes(), prefix) || !bytes.Equal(data, before) {
				t.Fatalf("marker %d tail %x: %+v, %v", marker, tail, p, err)
			}
			base, err := top.ReadV3PlanMappingPrefix(data)
			if err != nil || p.OverviewMapping() != base.OverviewMapping() || p.PlanMapping() != base.PlanMapping() || p.Offsets().PlanMappingPrefixOffsets != base.Offsets() || p.Header() != base.Header() || p.Version() != 3 || p.TripCountRaw() != 0 || p.MeasurementCountRaw() != 0 || p.ReferenceCountRaw() != 0 || len(p.Trips()) != 0 || len(p.Measurements()) != 0 || len(p.References()) != 0 {
				t.Fatal("marker prefix changed earlier source data")
			}
		}
	}
}

func TestPlanMarkerPrefixTruncationAndEarlierStops(t *testing.T) {
	data := append(planMarkerBase(), 3)
	for n := 0; n < 40; n++ {
		_, earlier := top.ReadV3PlanMappingPrefix(data[:n])
		p, err := top.ReadV3PlanMarkerPrefix(data[:n])
		if earlier == nil || !reflect.DeepEqual(err, earlier) || !reflect.DeepEqual(p, source.PlanMarkerPrefix{}) {
			t.Fatalf("inherited truncation at %d changed: %v, %v", n, err, earlier)
		}
	}
	requirePlanMarkerError(t, data[:40], top.DefaultReferenceLimits(), "truncated", "plan.elements[0].kind", 40)
	for _, tail := range [][]byte{nil, {0xff}, {0}, planMappingRecord()} {
		if p, err := top.ReadV3TripPrefix(append(fixture(0), tail...)); err != nil || p.ConsumedOffset() != 8 || p.UnparsedTailSize() != len(tail) {
			t.Fatal("P03b stop changed")
		}
		if p, err := top.ReadV3MeasurementPrefix(append(measurementTable(0), tail...)); err != nil || p.ConsumedOffset() != 12 || p.UnparsedTailSize() != len(tail) {
			t.Fatal("P03c1 stop changed")
		}
		if p, err := top.ReadV3ReferencePrefix(append(referenceTable(0), tail...)); err != nil || p.ConsumedOffset() != 16 || p.UnparsedTailSize() != len(tail) {
			t.Fatal("P03c2 stop changed")
		}
		if p, err := top.ReadV3OverviewPrefix(append(planMappingBase(), tail...)); err != nil || p.ConsumedOffset() != 28 || p.UnparsedTailSize() != len(tail) {
			t.Fatal("P04a stop changed")
		}
		if p, err := top.ReadV3PlanMappingPrefix(append(planMarkerBase(), tail...)); err != nil || p.ConsumedOffset() != 40 || p.UnparsedTailSize() != len(tail) {
			t.Fatal("P04b stop changed")
		}
	}
}

func TestPlanMarkerPrefixInheritedErrorsAndLimits(t *testing.T) {
	defaults := top.DefaultReferenceLimits()
	for _, bad := range [][]byte{nil, {0xff}, fixture(-1), fixture(1, record(-1, "", 0)), fixture(1, record(17, "A\xffB", 0)), measurementTable(-1), measurementTable(1, measurementRecord(2, "A\xffB")), referenceTable(-1), referenceTable(1, referenceRecord("A\xffB")), referenceTable(1, referenceRecord("")[:24]), planMappingBase()} {
		_, earlier := top.ReadV3PlanMappingPrefixWithLimits(bad, defaults)
		p, err := top.ReadV3PlanMarkerPrefixWithLimits(bad, defaults)
		if earlier == nil || !reflect.DeepEqual(err, earlier) || !reflect.DeepEqual(p, source.PlanMarkerPrefix{}) {
			t.Fatalf("inherited error changed: %v, %v", err, earlier)
		}
	}
	data := append(referenceTable(1, referenceRecord("abc")), planMappingRecord()...)
	data = append(data, planMappingRecord()...)
	data = append(data, 1, 0xff)
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
			requirePlanMarkerError(t, data, limits, "invalid_limit", "limits."+field, 0)
		}
	}
	limits := top.ReferenceLimits{MeasurementLimits: top.MeasurementLimits{Limits: top.Limits{MaxInputBytes: len(data), MaxTrips: 0, MaxCommentBytes: 3}, MaxMeasurements: 0}, MaxReferences: 1}
	if _, err := top.ReadV3PlanMarkerPrefixWithLimits(data, limits); err != nil {
		t.Fatal(err)
	}
	limits.MaxInputBytes--
	requirePlanMarkerError(t, data, limits, "resource_limit", "input", 0)
	limits.MaxInputBytes++
	requirePlanMarkerError(t, append(bytes.Clone(data), 0), limits, "resource_limit", "input", 0)
	limits.MaxReferences = 0
	requirePlanMarkerError(t, data, limits, "resource_limit", "reference_count", 12)
	limits.MaxReferences = 1
	limits.MaxCommentBytes = 2
	requirePlanMarkerError(t, data, limits, "resource_limit", "references[0].comment.length", 40)
	limits.MaxCommentBytes = 0
	limits.MaxReferences = 0
	if _, err := top.ReadV3PlanMarkerPrefixWithLimits(append(planMarkerBase(), 0), limits); err != nil {
		t.Fatal(err)
	}
	requirePlanMarkerError(t, data, top.ReferenceLimits{}, "resource_limit", "input", 0)
	// Reference limit precedence remains ahead of all inherited validations.
	limits = top.ReferenceLimits{}
	limits.MaxReferences, limits.MaxMeasurements = -1, -1
	requirePlanMarkerError(t, nil, limits, "invalid_limit", "limits.max_references", 0)
}

func TestPlanMarkerPrefixCopiesAndVariableTables(t *testing.T) {
	data := binary.LittleEndian.AppendUint32(fixture(1, record(17, "trip", -32768)), 1)
	data = append(data, measurementRecord(255, "abc")...)
	data = binary.LittleEndian.AppendUint32(data, 1)
	data = append(data, referenceRecord("Aą")...)
	data = append(data, planMappingRecord()...)
	data = append(data, planMappingBase()[16:]...)
	requirePlanMarkerError(t, data, top.DefaultReferenceLimits(), "truncated", "plan.elements[0].kind", 107)
	data = append(data, 0xe7, 0xff, 0)
	before := bytes.Clone(data)
	p, err := top.ReadV3PlanMarkerPrefix(data)
	if err != nil || !bytes.Equal(data, before) {
		t.Fatalf("variable prefix: %v", err)
	}
	base, err := top.ReadV3PlanMappingPrefix(data)
	if err != nil || p.PlanMapping() != base.PlanMapping() || p.OverviewMapping() != base.OverviewMapping() || p.Offsets().PlanMappingPrefixOffsets != base.Offsets() {
		t.Fatal("mapping separation lost")
	}
	rows, trips, shots, raw, offsets := p.References(), p.Trips(), p.Measurements(), p.Bytes(), p.Offsets()
	comment := rows[0].CommentBytes()
	for i := range data {
		data[i] = 0xff
	}
	rows[0], trips[0], shots[0], raw[107], offsets.Marker.Start, comment[0] = source.Reference{}, source.Trip{}, source.Measurement{}, 0, 0, 0
	if p.MarkerRaw() != 231 || p.Offsets().Marker != (source.Span{Start: 107, End: 108}) || p.ConsumedOffset() != 108 || p.UnparsedTailSize() != 2 || !bytes.Equal(p.Bytes(), before[:108]) || p.Trips()[0].Ticks() != 17 || p.Trips()[0].Comment() != "trip" || p.Measurements()[0].FlagsRaw() != 255 || p.Measurements()[0].Comment() != "abc" || p.References()[0].Comment() != "Aą" {
		t.Fatal("mutable aliases, incorrect offset or lost source fields")
	}
}

func TestPlanMarkerPrefixNativeFixtures(t *testing.T) {
	// Pinned native helper fields plus fresh original reader positions;
	// these constants do not come from this Go reader.
	for _, fixture := range []struct {
		name, hash            string
		marker                byte
		start, consumed, tail int
	}{
		{"api-references.top", "9034cf5e52f92a8713c7823bd9be52bdb50b294966a9e9713c538b937b7feb5f", 0, 230, 231, 17},
		{"api-drawings.top", "4a494ead03cade750f670d50aa38661abda9b3e1f131d67f27a252982752c2a5", 1, 146, 147, 533},
	} {
		data, err := os.ReadFile("testdata/" + fixture.name)
		if err != nil {
			t.Fatal(err)
		}
		hash := sha256.Sum256(data)
		if hex.EncodeToString(hash[:]) != fixture.hash {
			t.Fatal("native fixture hash changed")
		}
		for _, tail := range [][]byte{nil, data[fixture.consumed:], {0xff, 0x80}} {
			input := append(bytes.Clone(data[:fixture.consumed]), tail...)
			p, err := top.ReadV3PlanMarkerPrefix(input)
			if err != nil || p.MarkerRaw() != fixture.marker || p.Offsets().Marker != (source.Span{Start: fixture.start, End: fixture.consumed}) || p.ConsumedOffset() != fixture.consumed || p.UnparsedTailSize() != len(tail) || !bytes.Equal(p.Bytes(), data[:fixture.consumed]) {
				t.Fatalf("native marker %s: %+v, %v", fixture.name, p, err)
			}
		}
		if len(data)-fixture.consumed != fixture.tail {
			t.Fatal("native tail size changed")
		}
		requirePlanMarkerError(t, data[:fixture.start], top.DefaultReferenceLimits(), "truncated", "plan.elements[0].kind", fixture.start)
	}
}

func FuzzReadV3PlanMarkerPrefix(f *testing.F) {
	for _, seed := range [][]byte{planMarkerBase(), append(planMarkerBase(), 0), append(planMarkerBase(), 1), append(planMarkerBase(), 3, 0xff), append(planMarkerBase(), 255), {0xff}} {
		f.Add(seed)
	}
	limits := top.ReferenceLimits{MeasurementLimits: top.MeasurementLimits{Limits: top.Limits{MaxInputBytes: 4096, MaxTrips: 32, MaxCommentBytes: 256}, MaxMeasurements: 32}, MaxReferences: 32}
	f.Fuzz(func(t *testing.T, data []byte) {
		before := bytes.Clone(data)
		p, err := top.ReadV3PlanMarkerPrefixWithLimits(data, limits)
		q, again := top.ReadV3PlanMarkerPrefixWithLimits(data, limits)
		if !bytes.Equal(data, before) || !reflect.DeepEqual(p, q) || !reflect.DeepEqual(err, again) {
			t.Fatal("mutation or nondeterministic result")
		}
		if err != nil {
			var pe *top.ParseError
			if !errors.As(err, &pe) || pe.Offset < 0 || pe.Offset > len(data) || !reflect.DeepEqual(p, source.PlanMarkerPrefix{}) {
				t.Fatalf("invalid failure: %v", err)
			}
			return
		}
		base, earlier := top.ReadV3PlanMappingPrefixWithLimits(data, limits)
		span := p.Offsets().Marker
		if earlier != nil || p.PlanMapping() != base.PlanMapping() || p.OverviewMapping() != base.OverviewMapping() || p.Offsets().PlanMappingPrefixOffsets != base.Offsets() || span.Start != base.ConsumedOffset() || span.End != span.Start+1 || span.End != p.ConsumedOffset() || p.ConsumedOffset() > len(data) || p.ConsumedOffset()+p.UnparsedTailSize() != len(data) || !bytes.Equal(p.Bytes(), data[:p.ConsumedOffset()]) || p.MarkerRaw() != data[span.Start] {
			t.Fatal("invalid marker, inherited fields, span or prefix/tail accounting")
		}
	})
}
