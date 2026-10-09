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

func measurementTable(count int32, records ...[]byte) []byte {
	b := binary.LittleEndian.AppendUint32(fixture(0), uint32(count))
	for _, record := range records {
		b = append(b, record...)
	}
	return b
}

func measurementRecord(flags byte, comment string) []byte {
	// Distinct reserved IDs, asymmetric signed fields, all roll bits, trip -2.
	b := []byte{0xff, 0xff, 0x0f, 0x80, 0, 0, 0, 0x80,
		0xfe, 0xff, 0xff, 0xff, 0x23, 0x81, 0x67, 0x45, flags, 0xff, 0xfe, 0xff}
	if flags&2 != 0 {
		// The trip helper encodes the same BinaryWriter string representation.
		encoded := record(0, comment, 0)
		b = append(b, encoded[8:len(encoded)-2]...)
	}
	return b
}

func requireMeasurementError(t *testing.T, data []byte, limits top.MeasurementLimits, code, field string, offset int) {
	t.Helper()
	before := bytes.Clone(data)
	p, err := top.ReadV3MeasurementPrefixWithLimits(data, limits)
	var pe *top.ParseError
	if !errors.As(err, &pe) || pe.Code != code || pe.Field != field || pe.Offset != offset {
		t.Fatalf("got %v; want %s at %d (%s)", err, code, offset, field)
	}
	if !reflect.DeepEqual(p, source.MeasurementPrefix{}) || !bytes.Equal(data, before) {
		t.Fatal("failure returned a partial result or changed input")
	}
}

func TestMeasurementPrefixZeroAndTripContract(t *testing.T) {
	for _, tail := range [][]byte{nil, {0xff}, {0xff, 0xff, 0xff, 0xff, 0x80}} {
		data := append(measurementTable(0), tail...)
		p, err := top.ReadV3MeasurementPrefix(data)
		if err != nil {
			t.Fatal(err)
		}
		if p.Header() != [4]byte{'T', 'o', 'p', 3} || p.Version() != 3 || p.TripCountRaw() != 0 || len(p.Trips()) != 0 || p.MeasurementCountRaw() != 0 || len(p.Measurements()) != 0 || p.ConsumedOffset() != 12 || p.UnparsedTailSize() != len(tail) || !bytes.Equal(p.Bytes(), data[:12]) {
			t.Fatalf("wrong zero-table prefix: %+v", p)
		}
		want := source.MeasurementPrefixOffsets{PrefixOffsets: source.PrefixOffsets{
			Header: source.Span{Start: 0, End: 4}, Magic: source.Span{Start: 0, End: 3},
			Version: source.Span{Start: 3, End: 4}, TripCount: source.Span{Start: 4, End: 8}},
			MeasurementCount: source.Span{Start: 8, End: 12}}
		if p.Offsets() != want {
			t.Fatal("wrong fixed spans")
		}
	}
	// P03b still accepts a trip-only prefix, even with an invalid measurement tail.
	for _, tail := range [][]byte{nil, {0xff}, {0xff, 0xff, 0xff, 0xff}} {
		data := append(fixture(1, record(17, "abc", -32768)), tail...)
		p, err := top.ReadV3TripPrefix(data)
		if err != nil || p.ConsumedOffset() != 22 || p.UnparsedTailSize() != len(tail) {
			t.Fatalf("P03b contract changed: %v", err)
		}
	}
}

func TestMeasurementPrefixLiteralFieldsAndOffsets(t *testing.T) {
	// Literal C#/IL layout. Record spans: [12,32), [32,53), [53,77).
	data, err := hex.DecodeString("546f70030000000003000000" +
		"ffff0f8000001080000000800080ff7ffdff0080" +
		"0100000002001080ffffff7fff7f00800281ff7f00" +
		"00000080ffff0f80ffffffffffffffffff00feff0341c485" + "deadbeef")
	if err != nil {
		t.Fatal(err)
	}
	p, err := top.ReadV3MeasurementPrefix(data)
	if err != nil {
		t.Fatal(err)
	}
	if p.MeasurementCountRaw() != 3 || len(p.Measurements()) != 3 || p.ConsumedOffset() != 77 || p.UnparsedTailSize() != 4 || !bytes.Equal(p.Bytes(), data[:77]) {
		t.Fatalf("wrong prefix accounting: %+v", p)
	}
	want := []struct {
		from, to    uint32
		dist        int32
		az, incl    int16
		flags, roll byte
		trip        int16
		comment     string
		start, end  int
	}{
		{0x800fffff, 0x80100000, math.MinInt32, math.MinInt16, math.MaxInt16, 253, 255, math.MinInt16, "", 12, 32},
		{1, 0x80100002, math.MaxInt32, math.MaxInt16, math.MinInt16, 2, 129, math.MaxInt16, "", 32, 53},
		{0x80000000, 0x800fffff, -1, -1, -1, 255, 0, -2, "Aą", 53, 77},
	}
	for i, m := range p.Measurements() {
		w := want[i]
		if m.From().Raw() != w.from || m.To().Raw() != w.to || m.DistanceMM() != w.dist || m.AzimuthRaw() != w.az || m.InclinationRaw() != w.incl || m.FlagsRaw() != w.flags || m.RollRaw() != w.roll || m.TripIndexRaw() != w.trip || m.HasComment() != (w.flags&2 != 0) || m.Comment() != w.comment || !bytes.Equal(m.CommentBytes(), []byte(w.comment)) || m.Offsets().Record != (source.Span{Start: w.start, End: w.end}) {
			t.Fatalf("measurement %d: %+v, want %+v", i, m, w)
		}
	}
	rows := p.Measurements()
	if rows[0].From().NativeValue() != -2 || rows[0].To().NativeValue() != -1 || rows[0].From().String() != "" || rows[0].From().SameIdentity(rows[0].To()) || !rows[1].From().SameIdentity(rows[1].To()) {
		t.Fatal("source IDs lost reserved distinction or alias identity")
	}
	wantOffsets := source.MeasurementOffsets{
		Record: source.Span{Start: 12, End: 32}, From: source.Span{Start: 12, End: 16},
		To: source.Span{Start: 16, End: 20}, Distance: source.Span{Start: 20, End: 24},
		Azimuth: source.Span{Start: 24, End: 26}, Inclination: source.Span{Start: 26, End: 28},
		Flags: source.Span{Start: 28, End: 29}, Roll: source.Span{Start: 29, End: 30},
		TripIndex: source.Span{Start: 30, End: 32},
	}
	if rows[0].Offsets() != wantOffsets || rows[1].Offsets().CommentLength != (source.Span{Start: 52, End: 53}) || rows[1].Offsets().Comment != (source.Span{Start: 53, End: 53}) || rows[2].Offsets().CommentLength != (source.Span{Start: 73, End: 74}) || rows[2].Offsets().Comment != (source.Span{Start: 74, End: 77}) {
		t.Fatal("wrong source field spans or absent/empty comment distinction")
	}
}

func TestMeasurementPrefixFlagsAndRawBoundaries(t *testing.T) {
	// Native Station.Read accepts every byte, even undeclared bit 0x40.
	for flags := 0; flags < 256; flags++ {
		data := measurementTable(1, measurementRecord(byte(flags), ""))
		p, err := top.ReadV3MeasurementPrefix(data)
		if err != nil {
			t.Fatal(err)
		}
		m := p.Measurements()[0]
		if m.FlagsRaw() != byte(flags) || m.HasComment() != (flags&2 != 0) || m.Comment() != "" || p.ConsumedOffset() != len(data) {
			t.Fatalf("flags %d lost bits/presence/boundary", flags)
		}
	}
	for _, distance := range []int32{math.MinInt32, -1, 0, 1, math.MaxInt32} {
		for _, index := range []int16{math.MinInt16, -2, -1, 0, 1, 2, math.MaxInt16} {
			b := measurementRecord(0, "")
			binary.LittleEndian.PutUint32(b[8:12], uint32(distance))
			binary.LittleEndian.PutUint16(b[18:20], uint16(index))
			// No trips: even nonnegative out-of-table indices remain raw source data.
			p, err := top.ReadV3MeasurementPrefix(measurementTable(1, b))
			if err != nil || p.Measurements()[0].DistanceMM() != distance || p.Measurements()[0].TripIndexRaw() != index {
				t.Fatalf("distance %d, trip %d: %v", distance, index, err)
			}
		}
	}
}

func TestMeasurementPrefixCountsAndInheritedErrors(t *testing.T) {
	limits := top.DefaultMeasurementLimits()
	for n := 0; n < 12; n++ {
		field, offset := "header", 0
		if n >= 4 {
			field, offset = "trip_count", 4
		}
		if n >= 8 {
			field, offset = "measurement_count", 8
		}
		requireMeasurementError(t, measurementTable(0)[:n], limits, "truncated", field, offset)
	}
	bad := measurementTable(0)
	bad[0] = 0
	requireMeasurementError(t, bad, limits, "bad_magic", "header.magic", 0)
	bad[0], bad[3] = 'T', 2
	requireMeasurementError(t, bad, limits, "unsupported_version", "header.version", 3)
	requireMeasurementError(t, fixture(-1), limits, "negative_count", "trip_count", 4)
	requireMeasurementError(t, fixture(1, record(-1, "", 0)), limits, "ticks_out_of_range", "trips[0].ticks", 8)
	for _, count := range []int32{-1, math.MinInt32} {
		requireMeasurementError(t, measurementTable(count), limits, "negative_count", "measurement_count", 8)
	}
	for _, count := range []int32{1_000_001, math.MaxInt32} {
		requireMeasurementError(t, measurementTable(count), limits, "resource_limit", "measurement_count", 8)
	}
	for _, count := range []int32{1, 1_000_000} {
		requireMeasurementError(t, measurementTable(count), limits, "truncated", "measurement_count", 8)
	}
	// Count offset follows the actual trip table, not a fixed header offset.
	withTrips := binary.LittleEndian.AppendUint32(fixture(1, record(17, "abc", 0)), 0xffffffff)
	requireMeasurementError(t, withTrips, limits, "negative_count", "measurement_count", 22)
}

func TestMeasurementPrefixTruncationEveryRequiredByte(t *testing.T) {
	limits := top.DefaultMeasurementLimits()
	data := measurementTable(2, measurementRecord(2, strings.Repeat("x", 128)), measurementRecord(2, "abc"))
	// First comment makes the minimum-byte preflight pass before the last record.
	names := []string{"from_id", "to_id", "distance_mm", "azimuth_raw", "inclination_raw", "flags_raw", "roll_raw", "trip_index_raw", "comment.length", "comment"}
	starts := []int{162, 166, 170, 174, 176, 178, 179, 180, 182, 183}
	ends := []int{166, 170, 174, 176, 178, 179, 180, 182, 183, 186}
	for n := 12; n < len(data); n++ {
		field, offset := "measurement_count", 8
		if n >= 52 && n < 162 {
			field, offset = "measurements[0].comment", 34
		}
		if n >= 162 {
			for i, end := range ends {
				if n < end {
					field, offset = "measurements[1]."+names[i], starts[i]
					break
				}
			}
		}
		requireMeasurementError(t, data[:n], limits, "truncated", field, offset)
	}
	base := measurementTable(1, measurementRecord(2, "")[:20])
	for n := 0; n < 5; n++ {
		b := append(bytes.Clone(base), bytes.Repeat([]byte{0x80}, n)...)
		requireMeasurementError(t, b, limits, "truncated", "measurements[0].comment.length", 32+n)
	}
}

func TestMeasurementPrefixCommentRules(t *testing.T) {
	limits := top.DefaultMeasurementLimits()
	for _, n := range []int{0, 1, 127, 128, 16383, 16384, 1024 * 1024} {
		comment := strings.Repeat("x", n)
		data := measurementTable(1, measurementRecord(2, comment))
		p, err := top.ReadV3MeasurementPrefix(data)
		if err != nil || p.Measurements()[0].Comment() != comment || !bytes.Equal(p.Bytes(), data) {
			t.Fatalf("length %d: %v", n, err)
		}
	}
	for _, comment := range []string{"\x00", "\ufeff", "�", "😀", "Zażółć gęślą jaźń\r\n"} {
		p, err := top.ReadV3MeasurementPrefix(measurementTable(1, measurementRecord(2, comment)))
		if err != nil || p.Measurements()[0].Comment() != comment {
			t.Fatalf("valid UTF-8: %v", err)
		}
	}
	for _, comment := range []string{"A\xffB", "\xc0\x80", "\xed\xa0\x80", "\xf4\x90\x80\x80", "\xe2\x82", "\x80"} {
		requireMeasurementError(t, measurementTable(1, measurementRecord(2, comment)), limits, "invalid_utf8", "measurements[0].comment", 33)
	}
	base := measurementTable(1, measurementRecord(2, "")[:20])
	for _, length := range [][]byte{{0x80, 0}, {0x80, 0x80, 0x80, 0x80, 0}} {
		data := append(bytes.Clone(base), length...)
		p, err := top.ReadV3MeasurementPrefix(data)
		if err != nil || !bytes.Equal(p.Bytes(), data) || p.Measurements()[0].Offsets().CommentLength != (source.Span{Start: 32, End: len(data)}) {
			t.Fatalf("nonminimal zero: %v", err)
		}
	}
	for _, length := range [][]byte{{0xff, 0xff, 0xff, 0xff, 8}, {0x80, 0x80, 0x80, 0x80, 0x80}, {0xff, 0xff, 0xff, 0xff, 0x7f}, {0x80, 0x80, 0x80, 0x80, 0x10}} {
		requireMeasurementError(t, append(bytes.Clone(base), length...), limits, "string_length_overflow", "measurements[0].comment.length", 32)
	}
	for _, length := range [][]byte{{0xff, 0xff, 0xff, 0xff, 7}, {0x81, 0x80, 0x40}} {
		requireMeasurementError(t, append(bytes.Clone(base), length...), limits, "resource_limit", "measurements[0].comment.length", 32)
	}
}

func TestMeasurementPrefixLimits(t *testing.T) {
	defaults := top.DefaultMeasurementLimits()
	if defaults != (top.MeasurementLimits{Limits: top.DefaultLimits(), MaxMeasurements: 1_000_000}) {
		t.Fatal("wrong defaults")
	}
	for _, value := range []int{-1, 1_000_001, math.MaxInt} {
		limits := defaults
		limits.MaxMeasurements = value
		requireMeasurementError(t, measurementTable(0), limits, "invalid_limit", "limits.max_measurements", 0)
	}
	invalid := defaults
	invalid.MaxTrips = -1
	requireMeasurementError(t, measurementTable(0), invalid, "invalid_limit", "limits.max_trips", 0)
	data := measurementTable(1, measurementRecord(2, "abc"))
	limits := top.MeasurementLimits{Limits: top.Limits{MaxInputBytes: len(data), MaxTrips: 0, MaxCommentBytes: 3}, MaxMeasurements: 1}
	if _, err := top.ReadV3MeasurementPrefixWithLimits(data, limits); err != nil {
		t.Fatal(err)
	}
	limits.MaxInputBytes--
	requireMeasurementError(t, data, limits, "resource_limit", "input", 0)
	limits.MaxInputBytes++
	limits.MaxMeasurements = 0
	requireMeasurementError(t, data, limits, "resource_limit", "measurement_count", 8)
	limits.MaxMeasurements = 1
	limits.MaxCommentBytes = 2
	requireMeasurementError(t, data, limits, "resource_limit", "measurements[0].comment.length", 32)
	limits.MaxCommentBytes = 0
	if _, err := top.ReadV3MeasurementPrefixWithLimits(measurementTable(1, measurementRecord(2, "")), limits); err != nil {
		t.Fatal(err)
	}
	limits.MaxMeasurements = 0
	if _, err := top.ReadV3MeasurementPrefixWithLimits(measurementTable(0), limits); err != nil {
		t.Fatal(err)
	}
	requireMeasurementError(t, measurementTable(0), top.MeasurementLimits{}, "resource_limit", "input", 0)
	large := make([]byte, defaults.MaxInputBytes+1)
	copy(large, measurementTable(0))
	if p, err := top.ReadV3MeasurementPrefix(large[:defaults.MaxInputBytes]); err != nil || p.UnparsedTailSize() != defaults.MaxInputBytes-12 {
		t.Fatalf("input boundary: %v", err)
	}
	requireMeasurementError(t, large, defaults, "resource_limit", "input", 0)
	many := append(measurementTable(1_000_000), make([]byte, 20_000_000)...)
	if p, err := top.ReadV3MeasurementPrefix(many); err != nil || len(p.Measurements()) != 1_000_000 {
		t.Fatalf("count boundary: %v", err)
	}
}

func TestMeasurementPrefixCopies(t *testing.T) {
	data := binary.LittleEndian.AppendUint32(fixture(1, record(17, "trip", -32768)), 1)
	data = append(data, measurementRecord(255, "abc")...)
	before := bytes.Clone(data)
	p, err := top.ReadV3MeasurementPrefix(data)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, before) {
		t.Fatal("reader changed input")
	}
	rows, trips, raw := p.Measurements(), p.Trips(), p.Bytes()
	comment, offsets, prefixOffsets, header := rows[0].CommentBytes(), rows[0].Offsets(), p.Offsets(), p.Header()
	for i := range data {
		data[i] = 0xff
	}
	rows[0], trips[0] = source.Measurement{}, source.Trip{}
	raw[0], comment[0], header[0] = 0, 0, 0
	offsets.From.Start, prefixOffsets.MeasurementCount.Start = 100, 100
	if !bytes.Equal(p.Bytes(), before) || p.Measurements()[0].Comment() != "abc" || p.Measurements()[0].FlagsRaw() != 255 || p.Measurements()[0].Offsets().From.Start != 27 || p.Offsets().MeasurementCount != (source.Span{Start: 23, End: 27}) || p.Trips()[0].Ticks() != 17 || p.Trips()[0].Comment() != "trip" || p.Header()[0] != 'T' {
		t.Fatal("mutable aliases")
	}
}

func TestMeasurementPrefixNativeFixture(t *testing.T) {
	data, err := os.ReadFile("testdata/api-trips-ids.top")
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(data)
	if hex.EncodeToString(hash[:]) != "adb83280b6d5a70f383b5542727f675b3ff8a5740058b601d3d49ed27b14895b" {
		t.Fatal("fixture hash changed")
	}
	p, err := top.ReadV3MeasurementPrefix(data)
	if err != nil {
		t.Fatal(err)
	}
	if p.TripCountRaw() != 3 || len(p.Trips()) != 3 || p.MeasurementCountRaw() != 4 || len(p.Measurements()) != 4 || p.ConsumedOffset() != 496 || p.UnparsedTailSize() != 46 || p.Offsets().MeasurementCount != (source.Span{Start: 171, End: 175}) {
		t.Fatalf("native prefix: %+v", p)
	}
	from := []uint32{0x80000001, 0, 851967, 851967}
	to := []uint32{0, 851967, 0x80000000, 851968}
	az := []int16{0, 16384, -32768, -16384}
	incl := []int16{0, 1820, -1820, 0}
	flags := []byte{2, 3, 2, 2}
	trip := []int16{0, 1, 2, -1}
	// Frozen native helper input has exactly 70 repeated letters after the prefix.
	comments := []string{"plain zero to major.minor zero", "Zażółć gęślą jaźń — " + strings.Repeat("ą", 70), "automatic trip splay", "unassigned trip"}
	starts, ends := []int{175, 226, 419, 460}, []int{226, 419, 460, 496}
	for i, m := range p.Measurements() {
		if m.From().Raw() != from[i] || m.To().Raw() != to[i] || m.DistanceMM() != int32((i+1)*1000) || m.AzimuthRaw() != az[i] || m.InclinationRaw() != incl[i] || m.FlagsRaw() != flags[i] || m.RollRaw() != 0 || m.TripIndexRaw() != trip[i] || !m.HasComment() || m.Comment() != comments[i] || m.Offsets().Record != (source.Span{Start: starts[i], End: ends[i]}) {
			t.Fatalf("native measurement %d: %+v", i, m)
		}
	}
}

func FuzzReadV3MeasurementPrefix(f *testing.F) {
	for _, seed := range [][]byte{measurementTable(0), measurementTable(1, measurementRecord(255, "Zażółć 😀")), measurementTable(-1), measurementTable(1, measurementRecord(2, "A\xffB")), {0xff}} {
		f.Add(seed)
	}
	limits := top.MeasurementLimits{Limits: top.Limits{MaxInputBytes: 4096, MaxTrips: 32, MaxCommentBytes: 256}, MaxMeasurements: 32}
	f.Fuzz(func(t *testing.T, data []byte) {
		before := bytes.Clone(data)
		p, err := top.ReadV3MeasurementPrefixWithLimits(data, limits)
		q, again := top.ReadV3MeasurementPrefixWithLimits(data, limits)
		if !bytes.Equal(data, before) || !reflect.DeepEqual(p, q) || !reflect.DeepEqual(err, again) {
			t.Fatal("mutation or nondeterministic result")
		}
		if err != nil {
			var pe *top.ParseError
			if !errors.As(err, &pe) || pe.Offset < 0 || pe.Offset > len(data) || !reflect.DeepEqual(p, source.MeasurementPrefix{}) {
				t.Fatalf("invalid failure: %v", err)
			}
			return
		}
		if p.ConsumedOffset() > len(data) || p.ConsumedOffset()+p.UnparsedTailSize() != len(data) || !bytes.Equal(p.Bytes(), data[:p.ConsumedOffset()]) || int(p.MeasurementCountRaw()) != len(p.Measurements()) || int(p.TripCountRaw()) != len(p.Trips()) {
			t.Fatal("invalid accounting")
		}
	})
}
