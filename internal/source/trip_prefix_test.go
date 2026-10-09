package source_test

import (
	"bytes"
	"testing"

	"pockettopo-exporter/internal/source"
)

func TestTripPrefixConstructorCopies(t *testing.T) {
	offsets := source.TripOffsets{Record: source.Span{Start: 8, End: 22}}
	trip := source.NewTrip(123, "abc", -17, offsets)
	trips := []source.Trip{trip}
	raw := []byte{'T', 'o', 'p', 3, 1, 0, 0, 0}
	p := source.NewTripPrefix([4]byte{'T', 'o', 'p', 3}, 1, trips, raw, 99)
	raw[0] = 0
	trips[0] = source.Trip{}
	offsets.Record.Start = 0
	if p.Header() != [4]byte{'T', 'o', 'p', 3} || p.Version() != 3 || p.TripCountRaw() != 1 || p.ConsumedOffset() != 8 || p.UnparsedTailSize() != 99 || !bytes.Equal(p.Bytes(), []byte{'T', 'o', 'p', 3, 1, 0, 0, 0}) || p.Trips()[0] != trip || trip.Offsets().Record.Start != 8 {
		t.Fatal("constructor did not preserve immutable source values")
	}
	if trip.Ticks() != 123 || trip.Comment() != "abc" || !bytes.Equal(trip.CommentBytes(), []byte("abc")) || trip.DeclinationRaw() != -17 || trip.AutoDeclination() {
		t.Fatal("wrong source fields")
	}

	wantOffsets := source.PrefixOffsets{
		Header: source.Span{Start: 0, End: 4}, Magic: source.Span{Start: 0, End: 3},
		Version: source.Span{Start: 3, End: 4}, TripCount: source.Span{Start: 4, End: 8},
	}
	offsetsCopy := p.Offsets()
	if offsetsCopy != wantOffsets {
		t.Fatal("wrong fixed header/count offsets")
	}
	offsetsCopy.Header.Start = 99
	if p.Offsets() != wantOffsets {
		t.Fatal("mutable header/count offsets")
	}
}

func TestTripPrefixAutoSentinel(t *testing.T) {
	for _, declination := range []int16{-32768, -32767, -1, 0, 1, 32767} {
		trip := source.NewTrip(7, "", declination, source.TripOffsets{})
		if trip.DeclinationRaw() != declination || trip.AutoDeclination() != (declination == -32768) {
			t.Fatalf("raw %d: wrong Auto mode or changed raw value", declination)
		}
	}
}
