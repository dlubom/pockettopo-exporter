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
	"strings"
	"testing"

	"pockettopo-exporter/internal/source"
	"pockettopo-exporter/internal/top"
)

func referenceTable(count int32, records ...[]byte) []byte {
	b := binary.LittleEndian.AppendUint32(measurementTable(0), uint32(count))
	for _, record := range records {
		b = append(b, record...)
	}
	return b
}

func referenceRecord(comment string) []byte {
	// Literal fields also read by the original assembly's signed-endian probe.
	b := []byte{0xff, 0xff, 0x0f, 0x80,
		1, 2, 3, 4, 5, 6, 7, 0x88, 8, 7, 6, 5, 4, 3, 2, 1,
		0xfe, 0xff, 0xff, 0xff}
	encoded := record(0, comment, 0)
	return append(b, encoded[8:len(encoded)-2]...)
}

func requireReferenceError(t *testing.T, data []byte, limits top.ReferenceLimits, code, field string, offset int) {
	t.Helper()
	before := bytes.Clone(data)
	p, err := top.ReadV3ReferencePrefixWithLimits(data, limits)
	var pe *top.ParseError
	if !errors.As(err, &pe) || pe.Code != code || pe.Field != field || pe.Offset != offset {
		t.Fatalf("got %v; want %s at %d (%s)", err, code, offset, field)
	}
	if !reflect.DeepEqual(p, source.ReferencePrefix{}) || !bytes.Equal(data, before) {
		t.Fatal("partial result or input mutation on failure")
	}
}

func TestReferencePrefixLiteralFieldsAndSpans(t *testing.T) {
	data := referenceTable(2, referenceRecord("Aą"), referenceRecord(""))
	before := bytes.Clone(data)
	data = append(data, 0xff, 0x80, 0, 3)
	p, err := top.ReadV3ReferencePrefix(data)
	if err != nil {
		t.Fatal(err)
	}
	if p.ReferenceCountRaw() != 2 || len(p.References()) != 2 || p.ConsumedOffset() != 69 || p.UnparsedTailSize() != 4 || !bytes.Equal(p.Bytes(), before) {
		t.Fatalf("wrong prefix accounting: %+v", p)
	}
	for i, r := range p.References() {
		comment, start, end, commentStart := "Aą", 16, 44, 41
		if i == 1 {
			comment, start, end, commentStart = "", 44, 69, 69
		}
		want := source.ReferenceOffsets{
			Record: source.Span{Start: start, End: end}, Station: source.Span{Start: start, End: start + 4},
			East: source.Span{Start: start + 4, End: start + 12}, North: source.Span{Start: start + 12, End: start + 20},
			Altitude: source.Span{Start: start + 20, End: start + 24}, CommentLength: source.Span{Start: start + 24, End: commentStart},
			Comment: source.Span{Start: commentStart, End: end},
		}
		if r.Station().Raw() != 0x800fffff || r.EastMM() != -0x77f8f9fafbfcfdff || r.NorthMM() != 0x0102030405060708 || r.AltitudeMM() != -2 || r.Comment() != comment || !bytes.Equal(r.CommentBytes(), []byte(comment)) || r.Offsets() != want {
			t.Fatalf("reference %d: %+v, offsets %+v", i, r, r.Offsets())
		}
	}
	if p.Header() != [4]byte{'T', 'o', 'p', 3} || p.Version() != 3 || p.TripCountRaw() != 0 || p.MeasurementCountRaw() != 0 || len(p.Trips()) != 0 || len(p.Measurements()) != 0 || p.Offsets().ReferenceCount != (source.Span{Start: 12, End: 16}) || p.Offsets().MeasurementCount != (source.Span{Start: 8, End: 12}) || p.Offsets().TripCount != (source.Span{Start: 4, End: 8}) {
		t.Fatal("lost inherited prefix metadata")
	}
}

func TestReferencePrefixRawBoundariesAndIDs(t *testing.T) {
	for _, coordinate := range []int64{math.MinInt64, -9007199254740993, -1, 0, 1, 9007199254740993, math.MaxInt64} {
		for _, altitude := range []int32{math.MinInt32, -1, 0, 1, math.MaxInt32} {
			b := referenceRecord("")
			binary.LittleEndian.PutUint64(b[4:12], uint64(coordinate))
			binary.LittleEndian.PutUint64(b[12:20], uint64(coordinate))
			binary.LittleEndian.PutUint32(b[20:24], uint32(altitude))
			p, err := top.ReadV3ReferencePrefix(referenceTable(1, b))
			if err != nil || p.References()[0].EastMM() != coordinate || p.References()[0].NorthMM() != coordinate || p.References()[0].AltitudeMM() != altitude {
				t.Fatalf("coordinate %d, altitude %d: %v", coordinate, altitude, err)
			}
		}
	}
	var records [][]byte
	for _, raw := range []uint32{1, 0x80100002, 0x800fffff, 0x80000000, 0x80100000} {
		b := referenceRecord("")
		binary.LittleEndian.PutUint32(b[:4], raw)
		records = append(records, b)
	}
	p, err := top.ReadV3ReferencePrefix(referenceTable(5, records...))
	if err != nil {
		t.Fatal(err)
	}
	rows := p.References()
	for i, raw := range []uint32{1, 0x80100002, 0x800fffff, 0x80000000, 0x80100000} {
		if rows[i].Station().Raw() != raw {
			t.Fatal("lost raw station pattern or record order")
		}
	}
	if !rows[0].Station().SameIdentity(rows[1].Station()) || rows[2].Station().SameIdentity(rows[3].Station()) || rows[2].Station().NativeValue() != -2 || rows[3].Station().NativeValue() != -1 || rows[2].Station().String() != "" || rows[3].Station().String() != "" {
		t.Fatal("lost aliases or reserved identity")
	}
}

func TestReferencePrefixCountsAndInheritedErrors(t *testing.T) {
	limits := top.DefaultReferenceLimits()
	for n := 0; n < 16; n++ {
		field, offset := "header", 0
		if n >= 4 {
			field, offset = "trip_count", 4
		}
		if n >= 8 {
			field, offset = "measurement_count", 8
		}
		if n >= 12 {
			field, offset = "reference_count", 12
		}
		requireReferenceError(t, referenceTable(0)[:n], limits, "truncated", field, offset)
	}
	bad := referenceTable(0)
	bad[0] = 0
	requireReferenceError(t, bad, limits, "bad_magic", "header.magic", 0)
	bad[0], bad[3] = 'T', 2
	requireReferenceError(t, bad, limits, "unsupported_version", "header.version", 3)
	requireReferenceError(t, fixture(-1), limits, "negative_count", "trip_count", 4)
	requireReferenceError(t, fixture(1, record(-1, "", 0)), limits, "ticks_out_of_range", "trips[0].ticks", 8)
	requireReferenceError(t, measurementTable(-1), limits, "negative_count", "measurement_count", 8)
	requireReferenceError(t, measurementTable(1, measurementRecord(2, "A\xffB")), limits, "invalid_utf8", "measurements[0].comment", 33)
	for _, count := range []int32{-1, math.MinInt32} {
		requireReferenceError(t, referenceTable(count), limits, "negative_count", "reference_count", 12)
	}
	for _, count := range []int32{1_000_001, math.MaxInt32} {
		requireReferenceError(t, referenceTable(count), limits, "resource_limit", "reference_count", 12)
	}
	for _, count := range []int32{1, 1_000_000} {
		requireReferenceError(t, referenceTable(count), limits, "truncated", "reference_count", 12)
	}
	withTables := binary.LittleEndian.AppendUint32(fixture(1, record(17, "trip", -32768)), 1)
	withTables = append(withTables, measurementRecord(255, "abc")...)
	withTables = binary.LittleEndian.AppendUint32(withTables, 0xffffffff)
	requireReferenceError(t, withTables, limits, "negative_count", "reference_count", 51)
	// Earlier APIs continue to succeed with absent or malformed later tables.
	for _, tail := range [][]byte{nil, {0xff}, {0xff, 0xff, 0xff, 0xff}} {
		if p, err := top.ReadV3TripPrefix(append(fixture(0), tail...)); err != nil || p.ConsumedOffset() != 8 {
			t.Fatal("P03b contract changed")
		}
		if p, err := top.ReadV3MeasurementPrefix(append(measurementTable(0), tail...)); err != nil || p.ConsumedOffset() != 12 {
			t.Fatal("P03c1 contract changed")
		}
		if p, err := top.ReadV3ReferencePrefix(append(referenceTable(0), tail...)); err != nil || p.ConsumedOffset() != 16 || p.UnparsedTailSize() != len(tail) {
			t.Fatal("reference reader interpreted drawing tail")
		}
	}
}

func TestReferencePrefixTruncationEveryRequiredByte(t *testing.T) {
	limits := top.DefaultReferenceLimits()
	data := referenceTable(2, referenceRecord(strings.Repeat("x", 128)), referenceRecord("abc"))
	names := []string{"station_id", "east_mm", "north_mm", "altitude_mm", "comment.length", "comment"}
	starts := []int{170, 174, 182, 190, 194, 195}
	ends := []int{174, 182, 190, 194, 195, 198}
	for n := 16; n < len(data); n++ {
		field, offset := "reference_count", 12
		if n >= 66 && n < 170 {
			field, offset = "references[0].comment", 42
		}
		if n >= 170 {
			for i, end := range ends {
				if n < end {
					field, offset = "references[1]."+names[i], starts[i]
					break
				}
			}
		}
		requireReferenceError(t, data[:n], limits, "truncated", field, offset)
	}
	base := referenceTable(1, referenceRecord("")[:24])
	for n := 1; n < 5; n++ {
		requireReferenceError(t, append(bytes.Clone(base), bytes.Repeat([]byte{0x80}, n)...), limits, "truncated", "references[0].comment.length", 40+n)
	}
}

func TestReferencePrefixCommentRules(t *testing.T) {
	limits := top.DefaultReferenceLimits()
	for _, n := range []int{0, 1, 127, 128, 16383, 16384, 1024 * 1024} {
		comment := strings.Repeat("x", n)
		data := referenceTable(1, referenceRecord(comment))
		p, err := top.ReadV3ReferencePrefix(data)
		if err != nil || p.References()[0].Comment() != comment || !bytes.Equal(p.Bytes(), data) {
			t.Fatalf("length %d: %v", n, err)
		}
	}
	for _, comment := range []string{"\x00", "\ufeff", "�", "😀", "Zażółć gęślą jaźń\r\n"} {
		p, err := top.ReadV3ReferencePrefix(referenceTable(1, referenceRecord(comment)))
		if err != nil || p.References()[0].Comment() != comment {
			t.Fatalf("valid UTF-8: %v", err)
		}
	}
	for _, comment := range []string{"A\xffB", "\xc0\x80", "\xed\xa0\x80", "\xf4\x90\x80\x80", "\xe2\x82", "\x80"} {
		requireReferenceError(t, referenceTable(1, referenceRecord(comment)), limits, "invalid_utf8", "references[0].comment", 41)
	}
	base := referenceTable(1, referenceRecord("")[:24])
	for _, length := range [][]byte{{0x80, 0}, {0x80, 0x80, 0x80, 0x80, 0}} {
		data := append(bytes.Clone(base), length...)
		p, err := top.ReadV3ReferencePrefix(data)
		if err != nil || p.References()[0].Comment() != "" || !bytes.Equal(p.Bytes(), data) || p.References()[0].Offsets().CommentLength != (source.Span{Start: 40, End: len(data)}) || p.References()[0].Offsets().Comment != (source.Span{Start: len(data), End: len(data)}) {
			t.Fatalf("nonminimal empty comment: %v", err)
		}
	}
	data := append(bytes.Clone(base), 0x81, 0, 65)
	if p, err := top.ReadV3ReferencePrefix(data); err != nil || p.References()[0].Comment() != "A" || !bytes.Equal(p.Bytes(), data) {
		t.Fatalf("nonminimal one: %v", err)
	}
	for _, length := range [][]byte{{0xff, 0xff, 0xff, 0xff, 8}, {0x80, 0x80, 0x80, 0x80, 0x80}, {0xff, 0xff, 0xff, 0xff, 0x7f}, {0x80, 0x80, 0x80, 0x80, 0x10}} {
		requireReferenceError(t, append(bytes.Clone(base), length...), limits, "string_length_overflow", "references[0].comment.length", 40)
	}
	for _, length := range [][]byte{{0xff, 0xff, 0xff, 0xff, 7}, {0x81, 0x80, 0x40}} {
		requireReferenceError(t, append(bytes.Clone(base), length...), limits, "resource_limit", "references[0].comment.length", 40)
	}
}

func TestReferencePrefixLimits(t *testing.T) {
	defaults := top.DefaultReferenceLimits()
	if defaults != (top.ReferenceLimits{MeasurementLimits: top.DefaultMeasurementLimits(), MaxReferences: 1_000_000}) {
		t.Fatal("wrong defaults")
	}
	for _, value := range []int{-1, 1_000_001, math.MaxInt} {
		limits := defaults
		limits.MaxReferences = value
		requireReferenceError(t, referenceTable(0), limits, "invalid_limit", "limits.max_references", 0)
	}
	for _, field := range []string{"max_measurements", "max_trips", "max_input_bytes", "max_comment_bytes"} {
		limits := defaults
		switch field {
		case "max_measurements":
			limits.MaxMeasurements = -1
		case "max_trips":
			limits.MaxTrips = -1
		case "max_input_bytes":
			limits.MaxInputBytes = -1
		case "max_comment_bytes":
			limits.MaxCommentBytes = -1
		}
		requireReferenceError(t, referenceTable(0), limits, "invalid_limit", "limits."+field, 0)
	}
	data := referenceTable(1, referenceRecord("abc"))
	limits := top.ReferenceLimits{MeasurementLimits: top.MeasurementLimits{Limits: top.Limits{MaxInputBytes: len(data), MaxTrips: 0, MaxCommentBytes: 3}, MaxMeasurements: 0}, MaxReferences: 1}
	if _, err := top.ReadV3ReferencePrefixWithLimits(data, limits); err != nil {
		t.Fatal(err)
	}
	limits.MaxInputBytes--
	requireReferenceError(t, data, limits, "resource_limit", "input", 0)
	limits.MaxInputBytes++
	limits.MaxReferences = 0
	requireReferenceError(t, data, limits, "resource_limit", "reference_count", 12)
	limits.MaxReferences = 1
	limits.MaxCommentBytes = 2
	requireReferenceError(t, data, limits, "resource_limit", "references[0].comment.length", 40)
	limits.MaxCommentBytes = 0
	if _, err := top.ReadV3ReferencePrefixWithLimits(referenceTable(1, referenceRecord("")), limits); err != nil {
		t.Fatal(err)
	}
	limits.MaxReferences = 0
	if _, err := top.ReadV3ReferencePrefixWithLimits(referenceTable(0), limits); err != nil {
		t.Fatal(err)
	}
	requireReferenceError(t, referenceTable(0), top.ReferenceLimits{}, "resource_limit", "input", 0)
	large := make([]byte, defaults.MaxInputBytes+1)
	copy(large, referenceTable(0))
	if p, err := top.ReadV3ReferencePrefix(large[:defaults.MaxInputBytes]); err != nil || p.UnparsedTailSize() != defaults.MaxInputBytes-16 {
		t.Fatalf("input boundary: %v", err)
	}
	requireReferenceError(t, large, defaults, "resource_limit", "input", 0)
	many := append(referenceTable(1_000_000), make([]byte, 25_000_000)...)
	if p, err := top.ReadV3ReferencePrefix(many); err != nil || p.ReferenceCountRaw() != 1_000_000 || p.ConsumedOffset() != len(many) {
		t.Fatalf("count boundary: %v", err)
	}
}

func TestReferencePrefixCopies(t *testing.T) {
	data := binary.LittleEndian.AppendUint32(fixture(1, record(17, "trip", -32768)), 1)
	data = append(data, measurementRecord(255, "abc")...)
	data = binary.LittleEndian.AppendUint32(data, 1)
	data = append(data, referenceRecord("Aą")...)
	before := bytes.Clone(data)
	p, err := top.ReadV3ReferencePrefix(data)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, before) {
		t.Fatal("reader changed input")
	}
	rows, trips, shots, raw := p.References(), p.Trips(), p.Measurements(), p.Bytes()
	comment, offsets, prefixOffsets, header := rows[0].CommentBytes(), rows[0].Offsets(), p.Offsets(), p.Header()
	for i := range data {
		data[i] = 0xff
	}
	rows[0], trips[0], shots[0] = source.Reference{}, source.Trip{}, source.Measurement{}
	raw[0], comment[0], header[0] = 0, 0, 0
	offsets.Station.Start, prefixOffsets.ReferenceCount.Start = 0, 0
	if !bytes.Equal(p.Bytes(), before) || p.References()[0].Comment() != "Aą" || p.References()[0].Offsets().Station.Start != 55 || p.Offsets().ReferenceCount != (source.Span{Start: 51, End: 55}) || p.Offsets().MeasurementCount != (source.Span{Start: 23, End: 27}) || p.Trips()[0].Ticks() != 17 || p.Trips()[0].Comment() != "trip" || p.Measurements()[0].FlagsRaw() != 255 || p.Measurements()[0].Comment() != "abc" || p.Header()[0] != 'T' {
		t.Fatal("mutable aliases or lost previous source data")
	}
}

func TestReferencePrefixNativeFixtures(t *testing.T) {
	for _, fixture := range []struct {
		name, hash                 string
		count                      int32
		consumed, tail, countStart int
	}{
		{"api-trips-ids.top", "adb83280b6d5a70f383b5542727f675b3ff8a5740058b601d3d49ed27b14895b", 0, 500, 42, 496},
		{"api-references.top", "9034cf5e52f92a8713c7823bd9be52bdb50b294966a9e9713c538b937b7feb5f", 2, 206, 42, 83},
	} {
		data, err := os.ReadFile("testdata/" + fixture.name)
		if err != nil {
			t.Fatal(err)
		}
		hash := sha256.Sum256(data)
		if hex.EncodeToString(hash[:]) != fixture.hash {
			t.Fatal("native fixture hash changed")
		}
		p, err := top.ReadV3ReferencePrefix(data)
		if err != nil {
			t.Fatal(err)
		}
		if p.ReferenceCountRaw() != fixture.count || len(p.References()) != int(fixture.count) || p.ConsumedOffset() != fixture.consumed || p.UnparsedTailSize() != fixture.tail || p.Offsets().ReferenceCount != (source.Span{Start: fixture.countStart, End: fixture.countStart + 4}) {
			t.Fatalf("native prefix: %+v", p)
		}
		if fixture.count == 0 {
			continue
		}
		if p.TripCountRaw() != 1 || p.Trips()[0].Ticks() != 632401344000000000 || p.MeasurementCountRaw() != 1 || p.Measurements()[0].DistanceMM() != 1000 {
			t.Fatal("native inherited tables changed")
		}
		// Expectations are literal native helper inputs, not outputs from Go.
		want := []struct {
			east, north int64
			altitude    int32
			comment     string
			start, end  int
		}{
			{-4000000001, -5000000002, -1250, "ujemne E/N/Z; żadnego przypisania CRS", 87, 150},
			{6000000003, 7000000004, 1500250, "positive int64 E/N beyond int32", 150, 206},
		}
		for i, r := range p.References() {
			w := want[i]
			if r.Station().Raw() != 0x80000001 || r.EastMM() != w.east || r.NorthMM() != w.north || r.AltitudeMM() != w.altitude || r.Comment() != w.comment || !bytes.Equal(r.CommentBytes(), []byte(w.comment)) || r.Offsets() != (source.ReferenceOffsets{Record: source.Span{Start: w.start, End: w.end}, Station: source.Span{Start: w.start, End: w.start + 4}, East: source.Span{Start: w.start + 4, End: w.start + 12}, North: source.Span{Start: w.start + 12, End: w.start + 20}, Altitude: source.Span{Start: w.start + 20, End: w.start + 24}, CommentLength: source.Span{Start: w.start + 24, End: w.start + 25}, Comment: source.Span{Start: w.start + 25, End: w.end}}) {
				t.Fatalf("native reference %d: %+v", i, r)
			}
		}
	}
}

func FuzzReadV3ReferencePrefix(f *testing.F) {
	for _, seed := range [][]byte{referenceTable(0), referenceTable(2, referenceRecord(""), referenceRecord("Zażółć 😀")), referenceTable(-1), referenceTable(1, referenceRecord("A\xffB")), {0xff}} {
		f.Add(seed)
	}
	limits := top.ReferenceLimits{MeasurementLimits: top.MeasurementLimits{Limits: top.Limits{MaxInputBytes: 4096, MaxTrips: 32, MaxCommentBytes: 256}, MaxMeasurements: 32}, MaxReferences: 32}
	f.Fuzz(func(t *testing.T, data []byte) {
		before := bytes.Clone(data)
		p, err := top.ReadV3ReferencePrefixWithLimits(data, limits)
		q, again := top.ReadV3ReferencePrefixWithLimits(data, limits)
		if !bytes.Equal(data, before) || !reflect.DeepEqual(p, q) || !reflect.DeepEqual(err, again) {
			t.Fatal("mutation or nondeterministic result")
		}
		if err != nil {
			var pe *top.ParseError
			if !errors.As(err, &pe) || pe.Offset < 0 || pe.Offset > len(data) || !reflect.DeepEqual(p, source.ReferencePrefix{}) {
				t.Fatalf("invalid failure: %v", err)
			}
			return
		}
		if p.ConsumedOffset() > len(data) || p.ConsumedOffset()+p.UnparsedTailSize() != len(data) || !bytes.Equal(p.Bytes(), data[:p.ConsumedOffset()]) || int(p.ReferenceCountRaw()) != len(p.References()) || int(p.MeasurementCountRaw()) != len(p.Measurements()) || int(p.TripCountRaw()) != len(p.Trips()) {
			t.Fatal("invalid accounting")
		}
	})
}
