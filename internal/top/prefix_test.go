package top_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"os"
	"reflect"
	"strings"
	"testing"

	"pockettopo-exporter/internal/source"
	"pockettopo-exporter/internal/top"
)

func fixture(count int32, records ...[]byte) []byte {
	b := []byte{'T', 'o', 'p', 3}
	b = binary.LittleEndian.AppendUint32(b, uint32(count))
	for _, record := range records {
		b = append(b, record...)
	}
	return b
}

func record(ticks int64, comment string, declination int16) []byte {
	b := binary.LittleEndian.AppendUint64(nil, uint64(ticks))
	n := uint32(len(comment))
	for n >= 128 {
		b = append(b, byte(n)|128)
		n >>= 7
	}
	b = append(b, byte(n))
	b = append(b, comment...)
	return binary.LittleEndian.AppendUint16(b, uint16(declination))
}

func requireError(t *testing.T, data []byte, limits top.Limits, code, field string, offset int) {
	t.Helper()
	before := bytes.Clone(data)
	p, err := top.ReadV3TripPrefixWithLimits(data, limits)
	var pe *top.ParseError
	if !errors.As(err, &pe) || pe.Code != code || pe.Field != field || pe.Offset != offset {
		t.Fatalf("got %v; want %s at %d (%s)", err, code, offset, field)
	}
	if pe.Error() != fmt.Sprintf("%s at byte %d (%s)", code, offset, field) {
		t.Fatalf("unexpected diagnostic: %s", pe.Error())
	}
	if len(p.Bytes()) != 0 || len(p.Trips()) != 0 || !bytes.Equal(data, before) {
		t.Fatal("failure returned a partial result or changed the input")
	}
}

func TestPrefixZeroTripsAndTail(t *testing.T) {
	for _, tail := range [][]byte{nil, {0xff, 0x80, 0, 3}} {
		data := append(fixture(0), tail...)
		p, err := top.ReadV3TripPrefix(data)
		if err != nil {
			t.Fatal(err)
		}
		if p.Header() != [4]byte{'T', 'o', 'p', 3} || p.Version() != 3 || p.TripCountRaw() != 0 || len(p.Trips()) != 0 || p.ConsumedOffset() != 8 || p.UnparsedTailSize() != len(tail) || !bytes.Equal(p.Bytes(), fixture(0)) {
			t.Fatalf("unexpected zero-trip prefix: %+v", p)
		}
	}
}

func TestPrefixOrderedTripsAndOffsets(t *testing.T) {
	// Hand-authored C#/IL-derived bytes; record spans are [8,19) and [19,33).
	data, _ := hex.DecodeString("546f7003020000000000000000000000000080ff3f37f47528ca2b0341c485ff7fdeadbeef")
	p, err := top.ReadV3TripPrefix(data)
	if err != nil {
		t.Fatal(err)
	}
	trips := p.Trips()
	if p.TripCountRaw() != 2 || len(trips) != 2 || p.ConsumedOffset() != 33 || p.UnparsedTailSize() != 4 || !bytes.Equal(p.Bytes(), data[:33]) {
		t.Fatalf("unexpected prefix: %+v", p)
	}
	want := []struct {
		ticks   int64
		comment string
		decl    int16
		auto    bool
		offsets source.TripOffsets
	}{
		{0, "", -32768, true, source.TripOffsets{Record: source.Span{Start: 8, End: 19}, Ticks: source.Span{Start: 8, End: 16}, CommentLength: source.Span{Start: 16, End: 17}, Comment: source.Span{Start: 17, End: 17}, Declination: source.Span{Start: 17, End: 19}}},
		{3155378975999999999, "Aą", 32767, false, source.TripOffsets{Record: source.Span{Start: 19, End: 33}, Ticks: source.Span{Start: 19, End: 27}, CommentLength: source.Span{Start: 27, End: 28}, Comment: source.Span{Start: 28, End: 31}, Declination: source.Span{Start: 31, End: 33}}},
	}
	for i, w := range want {
		trip := trips[i]
		if trip.Ticks() != w.ticks || trip.Comment() != w.comment || !bytes.Equal(trip.CommentBytes(), []byte(w.comment)) || trip.DeclinationRaw() != w.decl || trip.AutoDeclination() != w.auto || trip.Offsets() != w.offsets {
			t.Fatalf("trip %d: %+v, want %+v", i, trip, w)
		}
	}
}

func TestPrefixLengthAndSignedBoundaries(t *testing.T) {
	for _, n := range []int{0, 1, 127, 128, 16383, 16384, 1024 * 1024} {
		for _, decl := range []int16{-32768, -32767, -1, 0, 1, 32767} {
			comment := strings.Repeat("x", n)
			data := fixture(1, record(630822816000000007, comment, decl))
			p, err := top.ReadV3TripPrefix(data)
			if err != nil {
				t.Fatalf("length %d, decl %d: %v", n, decl, err)
			}
			trip := p.Trips()[0]
			lengthBytes := 1
			if n >= 128 {
				lengthBytes = 2
			}
			if n >= 16384 {
				lengthBytes = 3
			}
			if trip.Comment() != comment || trip.DeclinationRaw() != decl || trip.AutoDeclination() != (decl == -32768) || trip.Ticks() != 630822816000000007 || trip.Offsets().CommentLength != (source.Span{Start: 16, End: 16 + lengthBytes}) || p.ConsumedOffset() != len(data) {
				t.Fatalf("length %d, decl %d: wrong decoded record", n, decl)
			}
		}
	}
	// Nonminimal encodings are accepted, as by BinaryReader; retain their bytes.
	for _, length := range [][]byte{{0x80, 0}, {0x80, 0x80, 0x80, 0x80, 0}} {
		data := fixture(1, append(append(make([]byte, 8), length...), 0, 0))
		p, err := top.ReadV3TripPrefix(data)
		if err != nil || !bytes.Equal(p.Bytes(), data) {
			t.Fatalf("nonminimal length: %v", err)
		}
	}
}

func TestPrefixHeaderAndCounts(t *testing.T) {
	limits := top.DefaultLimits()
	for n := 0; n < 8; n++ {
		field, offset := "header", 0
		if n >= 4 {
			field, offset = "trip_count", 4
		}
		requireError(t, fixture(0)[:n], limits, "truncated", field, offset)
	}
	for i := 0; i < 3; i++ {
		b := fixture(0)
		b[i] ^= 1
		requireError(t, b, limits, "bad_magic", "header.magic", 0)
	}
	for version := 0; version < 256; version++ {
		if version == 3 {
			continue
		}
		b := fixture(0)
		b[3] = byte(version)
		requireError(t, b, limits, "unsupported_version", "header.version", 3)
	}
	for _, count := range []int32{-1, math.MinInt32} {
		requireError(t, fixture(count), limits, "negative_count", "trip_count", 4)
	}
	for _, count := range []int32{1_000_001, math.MaxInt32} {
		requireError(t, fixture(count), limits, "resource_limit", "trip_count", 4)
	}
	for _, count := range []int32{1, 1_000_000} {
		requireError(t, fixture(count), limits, "truncated", "trip_count", 4)
	}
}

func TestPrefixTruncationEveryRequiredByte(t *testing.T) {
	limits := top.DefaultLimits()
	data := fixture(2, record(1, strings.Repeat("x", 128), -1), record(2, "abc", 1))
	for n := 8; n < len(data); n++ {
		field, offset := "trip_count", 4 // minimum-byte preflight before allocation
		if n >= 30 {
			switch {
			case n < 18:
				field, offset = "trips[0].comment.length", n
			case n < 146:
				field, offset = "trips[0].comment", 18
			case n < 148:
				field, offset = "trips[0].declination_raw", 146
			case n < 156:
				field, offset = "trips[1].ticks", 148
			case n < 157:
				field, offset = "trips[1].comment.length", 156
			case n < 160:
				field, offset = "trips[1].comment", 157
			default:
				field, offset = "trips[1].declination_raw", 160
			}
		}
		requireError(t, data[:n], limits, "truncated", field, offset)
	}
	// Truncate each byte of a five-byte length while passing the preflight.
	base := fixture(2, record(0, "abcdef", 0), make([]byte, 8)) // second length at 33
	for n := 0; n < 5; n++ {
		b := append(bytes.Clone(base), bytes.Repeat([]byte{0x80}, n)...)
		requireError(t, b, limits, "truncated", "trips[1].comment.length", 33+n)
	}
}

func TestPrefixMalformedFields(t *testing.T) {
	limits := top.DefaultLimits()
	for _, ticks := range []int64{-1, math.MinInt64, 3155378976000000000, math.MaxInt64} {
		requireError(t, fixture(1, record(ticks, "", 0)), limits, "ticks_out_of_range", "trips[0].ticks", 8)
	}
	for _, length := range [][]byte{{0xff, 0xff, 0xff, 0xff, 8}, {0x80, 0x80, 0x80, 0x80, 0x80}, {0xff, 0xff, 0xff, 0xff, 0x7f}, {0x80, 0x80, 0x80, 0x80, 0x10}} {
		b := fixture(1, append(append(make([]byte, 8), length...), 0, 0))
		requireError(t, b, limits, "string_length_overflow", "trips[0].comment.length", 16)
	}
	for _, length := range [][]byte{{0xff, 0xff, 0xff, 0xff, 7}, {0x81, 0x80, 0x40}} {
		b := fixture(1, append(append(make([]byte, 8), length...), 0, 0))
		requireError(t, b, limits, "resource_limit", "trips[0].comment.length", 16)
	}
	for _, comment := range []string{"a\xff", "\xc0\x80", "\xed\xa0\x80", "\xf4\x90\x80\x80", "\xe2\x82", "\x80"} {
		requireError(t, fixture(1, record(0, comment, 0)), limits, "invalid_utf8", "trips[0].comment", 17)
	}
	for _, comment := range []string{"\x00", "\ufeff", "�", "😀"} {
		p, err := top.ReadV3TripPrefix(fixture(1, record(0, comment, 0)))
		if err != nil || p.Trips()[0].Comment() != comment {
			t.Fatalf("valid UTF-8 rejected: %v", err)
		}
	}
}

func TestPrefixLimits(t *testing.T) {
	defaults := top.DefaultLimits()
	if defaults != (top.Limits{MaxInputBytes: 64 * 1024 * 1024, MaxTrips: 1_000_000, MaxCommentBytes: 1024 * 1024}) {
		t.Fatal("wrong defaults")
	}
	for i, field := range []string{"limits.max_input_bytes", "limits.max_trips", "limits.max_comment_bytes"} {
		for _, value := range []int{-1, math.MaxInt} {
			limits := defaults
			switch i {
			case 0:
				limits.MaxInputBytes = value
			case 1:
				limits.MaxTrips = value
			case 2:
				limits.MaxCommentBytes = value
			}
			requireError(t, fixture(0), limits, "invalid_limit", field, 0)
		}
	}
	data := fixture(1, record(0, "abc", 0))
	limits := top.Limits{MaxInputBytes: len(data), MaxTrips: 1, MaxCommentBytes: 3}
	if _, err := top.ReadV3TripPrefixWithLimits(data, limits); err != nil {
		t.Fatal(err)
	}
	limits.MaxInputBytes--
	requireError(t, data, limits, "resource_limit", "input", 0)
	limits.MaxInputBytes++
	limits.MaxTrips = 0
	requireError(t, data, limits, "resource_limit", "trip_count", 4)
	limits.MaxTrips = 1
	limits.MaxCommentBytes = 2
	requireError(t, data, limits, "resource_limit", "trips[0].comment.length", 16)
	if _, err := top.ReadV3TripPrefixWithLimits(fixture(0), top.Limits{MaxInputBytes: 8, MaxTrips: 0, MaxCommentBytes: 0}); err != nil {
		t.Fatal(err)
	}
	requireError(t, fixture(0), top.Limits{}, "resource_limit", "input", 0)
	if _, err := top.ReadV3TripPrefixWithLimits(fixture(1, record(0, "", 0)), top.Limits{MaxInputBytes: 19, MaxTrips: 1, MaxCommentBytes: 0}); err != nil {
		t.Fatal(err)
	}
	// Exact default input boundary, with the large tail still uninterpreted.
	large := make([]byte, defaults.MaxInputBytes+1)
	copy(large, fixture(0))
	if p, err := top.ReadV3TripPrefix(large[:defaults.MaxInputBytes]); err != nil || p.UnparsedTailSize() != defaults.MaxInputBytes-8 {
		t.Fatalf("exact input bound: %v", err)
	}
	requireError(t, large, defaults, "resource_limit", "input", 0)
	// Exact default count boundary using minimal empty-comment records.
	many := fixture(1_000_000)
	many = append(many, make([]byte, 11_000_000)...)
	if p, err := top.ReadV3TripPrefix(many); err != nil || len(p.Trips()) != 1_000_000 {
		t.Fatalf("exact count bound: %v", err)
	}
}

func TestPrefixCopies(t *testing.T) {
	data := fixture(1, record(17, "abc", -32768))
	before := bytes.Clone(data)
	p, err := top.ReadV3TripPrefix(data)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, before) {
		t.Fatal("reader changed input")
	}
	trips := p.Trips()
	raw := p.Bytes()
	comment := trips[0].CommentBytes()
	offsets := trips[0].Offsets()
	header := p.Header()
	for i := range data {
		data[i] = 0xff
	}
	raw[0] = 0
	comment[0] = 0
	offsets.Ticks.Start = 100
	header[0] = 0
	trips[0] = source.Trip{}
	trip := p.Trips()[0]
	if !bytes.Equal(p.Bytes(), before) || trip.Ticks() != 17 || trip.Comment() != "abc" || trip.DeclinationRaw() != -32768 || !trip.AutoDeclination() || trip.Offsets().Ticks.Start != 8 || p.Header()[0] != 'T' {
		t.Fatal("model exposed mutable aliases")
	}
}

func TestPrefixNativeFixture(t *testing.T) {
	// Literal native API inputs from the pinned JKTZ fixture, not this reader.
	data, err := os.ReadFile("testdata/api-trips-ids.top")
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(data)) != "adb83280b6d5a70f383b5542727f675b3ff8a5740058b601d3d49ed27b14895b" {
		t.Fatal("native fixture hash changed")
	}
	p, err := top.ReadV3TripPrefix(data)
	if err != nil {
		t.Fatal(err)
	}
	if p.Header() != [4]byte{'T', 'o', 'p', 3} || p.TripCountRaw() != 3 || p.ConsumedOffset() != 171 || p.UnparsedTailSize() != 371 {
		t.Fatalf("unexpected native prefix: %+v", p)
	}
	ticks := []int64{630822816000000007, 639259776000000009, 639260640000000001}
	comments := []string{"Synthetic reset-looking date, not field evidence", "Unicode trip: Zażółć gęślą jaźń", "Auto declination sentinel; no CRS asserted"}
	decls := []int16{910, -637, -32768}
	starts := []int{8, 67, 118}
	ends := []int{67, 118, 171}
	if len(p.Trips()) != 3 {
		t.Fatal("wrong native trip order/count")
	}
	for i, trip := range p.Trips() {
		if trip.Ticks() != ticks[i] || trip.Comment() != comments[i] || trip.DeclinationRaw() != decls[i] || trip.AutoDeclination() != (i == 2) || trip.Offsets().Record != (source.Span{Start: starts[i], End: ends[i]}) {
			t.Fatalf("native trip %d mismatch: %+v", i, trip)
		}
	}
}

func FuzzReadV3TripPrefix(f *testing.F) {
	for _, seed := range [][]byte{fixture(0), fixture(1, record(17, "Zażółć 😀", -32768)), fixture(-1), fixture(1, record(-1, "", 0)), {0xff}} {
		f.Add(seed)
	}
	limits := top.Limits{MaxInputBytes: 4096, MaxTrips: 32, MaxCommentBytes: 256}
	f.Fuzz(func(t *testing.T, data []byte) {
		before := bytes.Clone(data)
		p, err := top.ReadV3TripPrefixWithLimits(data, limits)
		q, again := top.ReadV3TripPrefixWithLimits(data, limits)
		if !bytes.Equal(data, before) || !reflect.DeepEqual(p, q) || !reflect.DeepEqual(err, again) {
			t.Fatal("mutation or nondeterministic result")
		}
		if err != nil {
			var pe *top.ParseError
			if !errors.As(err, &pe) || pe.Offset < 0 || pe.Offset > len(data) {
				t.Fatalf("unstructured error: %v", err)
			}
			return
		}
		if p.ConsumedOffset() > len(data) || p.ConsumedOffset()+p.UnparsedTailSize() != len(data) || !bytes.Equal(p.Bytes(), data[:p.ConsumedOffset()]) || int(p.TripCountRaw()) != len(p.Trips()) {
			t.Fatal("invalid prefix accounting")
		}
	})
}
